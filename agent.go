/*
  Copyright 2016 Ian Stahnke

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package main

import (
	"flag"
	"github.com/mitchellh/cli"
	//	"golang.org/x/net/context"
	"google.golang.org/grpc"
	//	"log"
	"net"
	//	"google.golang.org/grpc/credentials"
	"bufio"
	"bytes"
	"container/heap"
	"google.golang.org/grpc/grpclog"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

const (
	agentAddr       = "127.0.0.1:6666"
	MAX_UDP_PAYLOAD = 64 * 1024
)

type AgentCommand struct {
	Port int
	Ui   cli.Ui
}

func agentCmdFactory() (cli.Command, error) {
	return &AgentCommand{
		Ui: &cli.ColoredUi{
			Ui:          ui,
			OutputColor: cli.UiColorGreen,
		},
	}, nil
}

type PacketQ []*Packet

func (q PacketQ) Len() int           { return len(q) }
func (q PacketQ) Less(i, j int) bool { return i < j }
func (q PacketQ) Swap(i, j int)      { q[i], q[j] = q[j], q[i] }

func (q *PacketQ) Push(x interface{}) {
	// Push and Pop use pointer receivers because they modify the slice's length,
	// not just its contents.
	*q = append(*q, x.(*Packet))
}

func (q *PacketQ) Pop() interface{} {
	old := *q
	n := len(old)
	x := old[n-1]
	*q = old[0 : n-1]
	return x
}

type conduitAgentServer struct {
	pktq       *PacketQ
	msgmutex   *sync.Mutex
	sigs       chan os.Signal
	GrpcServer *grpc.Server
	shutdown   chan bool
	incoming   chan []byte
	wait       sync.WaitGroup
	conn       *net.UDPConn
	addr       *net.UDPAddr
}

func newAgentServer() *conduitAgentServer {
	a := new(conduitAgentServer)
	a.msgmutex = &sync.Mutex{}
	a.sigs = make(chan os.Signal, 1)
	a.GrpcServer = grpc.NewServer()
	a.shutdown = make(chan bool, 1)
	a.incoming = make(chan []byte, 1024)
	a.pktq = &PacketQ{}
	heap.Init(a.pktq)
	return a
}

func (c *conduitAgentServer) Pull(token *Token, stream ConduitAgent_PullServer) error {
	grpclog.Printf("pulled: %s\n", token.Domain)
	c.msgmutex.Lock()
	for c.pktq.Len() > 0 {
		pkt := heap.Pop(c.pktq).(*Packet)
		grpclog.Printf("msg: %s\n", pkt.Msg)
		if err := stream.Send(pkt); err != nil {
			return err
		}
	}
	c.msgmutex.Unlock()

	return nil
}

func (c *conduitAgentServer) SigHandler() {
	defer c.wait.Done()
	signal.Notify(c.sigs,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	sig := <-c.sigs
	grpclog.Printf("signal received: %v\n", sig)
	c.Stop()
	return
}

func (c *conduitAgentServer) Stop() {

	if c.conn == nil {
		grpclog.Printf("udp connection already closed.")
	}

	c.conn.Close()
	c.shutdown <- true
	c.wait.Wait()
	c.GrpcServer.Stop()

	// Release all remaining resources.
	c.shutdown = nil
	c.conn = nil
}

func (c *conduitAgentServer) ServeTelegraf() (err error) {
	defer c.wait.Done()

	c.addr, err = net.ResolveUDPAddr("udp", "127.0.0.1:8089")
	if err != nil {
		grpclog.Printf("error cannot resolve udp address: %s\n", err)
		return
	}

	c.conn, err = net.ListenUDP("udp", c.addr)
	if err != nil {
		grpclog.Printf("error creating udp listener: %s\n", err)
		return
	}

	err = c.conn.SetReadBuffer(MAX_UDP_PAYLOAD)
	if err != nil {
		grpclog.Printf("error setting udp read buffer: %s\n", err)
		return
	}

	buf := make([]byte, MAX_UDP_PAYLOAD)
	for {

		select {
		case <-c.shutdown:
			// shutdown signal
			return
		default:
			// read a message
			i, _, err := c.conn.ReadFromUDP(buf)
			if err != nil {
				grpclog.Printf("error: could not read udp msg: %s\n", err)
				continue
			}

			grpclog.Printf("telegraf listener read %d bytes\n", i)
			bufbuf := make([]byte, i)
			copy(bufbuf, buf[:i])
			c.incoming <- bufbuf

		}
	}
}

func (c *conduitAgentServer) parseTelegraf() {

	for {
		select {
		case <-c.shutdown:
			return
		case bites := <-c.incoming:

			rdr := bytes.NewReader(bites)
			scanner := bufio.NewScanner(rdr)
			c.msgmutex.Lock()
			for scanner.Scan() {
				msg := scanner.Text()
				pkt := &Packet{Id: 1, Msg: msg, Source: "telegraf"}
				heap.Push(c.pktq, pkt)
			}
			c.msgmutex.Unlock()

		}
	}

}

func (c *AgentCommand) Run(args []string) int {

	cmdFlags := flag.NewFlagSet("agent", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.Ui.Output(c.Help()) }

	cmdFlags.IntVar(&c.Port, "port", 6666, "The port on which to bind the conduit agent")
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	// controller channel

	// signal channel

	// pull channel

	grpclog.Printf("starting agent %s\n", version)

	lsnr, err := net.Listen("tcp", agentAddr)
	if err != nil {
		grpclog.Fatalf("failed to bind agent: %v", err)
	}

	/*
		var opts []grpc.ServerOption
		if *tls {
			creds, err := credentials.NewServerTLSFromFile(*certFile, *keyFile)
			if err != nil {
				grpclog.Fatalf("Failed to generate credentials %v", err)
			}
			opts = []grpc.ServerOption{grpc.Creds(creds)}
		}
		grpcServer := grpc.NewServer(opts...)
	*/

	agent := newAgentServer()
	RegisterConduitAgentServer(agent.GrpcServer, agent)
	agent.wait.Add(2)
	go agent.SigHandler()
	go agent.parseTelegraf()
	go agent.ServeTelegraf()
	agent.GrpcServer.Serve(lsnr)

	return 0

}

func (c *AgentCommand) Help() string {
	return "Run an agent"
}

func (c *AgentCommand) Synopsis() string {
	return "Run an agent"
}
