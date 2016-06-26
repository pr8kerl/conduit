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
	"fmt"
	"github.com/mitchellh/cli"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type ServerCommand struct {
	Port int
	Ui   cli.Ui
}

func serverCmdFactory() (cli.Command, error) {
	return &ServerCommand{
		Ui: &cli.ColoredUi{
			Ui:          ui,
			OutputColor: cli.UiColorGreen,
		},
	}, nil
}

type conduitAgentConnection struct {
	addr     string
	conn     *grpc.ClientConn
	dopts    grpc.DialOption
	shutdown chan bool
}

func newAgentConnection(hostname string, port int, shut chan bool) (c *conduitAgentConnection, err error) {
	c = new(conduitAgentConnection)
	portstr := fmt.Sprintf("%d", port)
	c.addr = hostname + ":" + portstr
	c.dopts = grpc.WithInsecure()
	c.shutdown = shut
	c.conn, err = grpc.Dial(c.addr, c.dopts)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (c *conduitAgentConnection) Poll() (err error) {

	for {

		select {
		case <-c.shutdown:
			// shutdown signal
			grpclog.Printf("server: poll shutdown\n")
			return nil
		case <-time.After(time.Minute):
			grpclog.Printf("server: poll timeout\n")
			cagent := NewConduitAgentClient(c.conn)
			stream, err := cagent.Pull(context.Background())
			if err != nil {
				grpclog.Printf("error retrieving agent events: %s\n", err)
			}
			for {
				paket, err := stream.Recv()
				if err == io.EOF {
					grpclog.Printf("eom\n")
					break
				}
				if err != nil {
					fmt.Printf("error %v, %v\n", cagent, err)
					return err
				}
				msgs := paket.Msg
				length := len(msgs)
				for i, msg := range msgs {
					fmt.Printf("%d msg: %s\n", i, msg)
				}
				fmt.Printf("%d msgs received in packet\n", length)

			}
		}
	}
}

type conduitServer struct {
	sigs     chan os.Signal
	shutdown chan bool
	incoming chan []byte
	wait     sync.WaitGroup
	conn     grpc.ClientConn
}

func newServer() *conduitServer {
	s := new(conduitServer)
	s.sigs = make(chan os.Signal, 1)
	s.shutdown = make(chan bool, 1)
	s.incoming = make(chan []byte, 1024)
	return s
}

func (c *conduitServer) SigHandler() {
	defer c.wait.Done()
	signal.Notify(c.sigs,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	sig := <-c.sigs
	grpclog.Printf("signal received: %v\n", sig)
	return
}

func (c *conduitServer) Stop() {

	c.shutdown <- true
	c.conn.Close()
	c.wait.Wait()

	// Release all remaining resources.
	c.shutdown = nil
	c.incoming = nil
}

func (c *ServerCommand) Run(args []string) int {

	cmdFlags := flag.NewFlagSet("server", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.Ui.Output(c.Help()) }

	cmdFlags.IntVar(&c.Port, "port", 6666, "The port on which to run the console server")
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	// controller channel

	// signal channel

	// pull channel

	grpclog.Printf("starting server %s\n", version)

	server := newServer()
	conn, err := newAgentConnection("127.0.0.1", 6666, server.shutdown)
	if err != nil {
		grpclog.Printf("error connecting to agent: %s\n", err)
		return 1
	}
	server.wait.Add(1)
	go server.SigHandler()
	conn.Poll()
	grpclog.Printf("server: Poll is finished\n")
	server.Stop()

	return 0

}

func (c *ServerCommand) Help() string {
	return "Run as a server (detailed help information here)"
}

func (c *ServerCommand) Synopsis() string {
	return "Run as a server"
}
