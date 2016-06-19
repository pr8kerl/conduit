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
	"time"
	//	"google.golang.org/grpc/credentials"
	"fmt"
	"google.golang.org/grpc/grpclog"
	"os"
	"os/signal"
	"syscall"
)

const (
	agentAddr = "127.0.0.1:6666"
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

type conduitAgentServer struct {
	messages   []string
	sigs       chan os.Signal
	GrpcServer *grpc.Server
}

func (c *conduitAgentServer) Pull(token *Token, stream ConduitAgent_PullServer) error {
	grpclog.Printf("domain: %s\n", token.Domain)
	for i := 0; i < 1024; i++ {
		msg := fmt.Sprintf("msg: %d\n", i)
		if err := stream.Send(&Packet{Id: 1, Msg: msg, Source: "module"}); err != nil {
			return err
		}
		time.Sleep(1 * time.Second)
	}

	return nil
}

func (c *conduitAgentServer) SigHandler() {
	signal.Notify(c.sigs,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	sig := <-c.sigs
	grpclog.Printf("signal received: %v\n", sig)
	c.GrpcServer.Stop()
	return
}

func newAgentServer() *conduitAgentServer {
	a := new(conduitAgentServer)
	a.messages = make([]string, 0, 1024)
	a.sigs = make(chan os.Signal, 1)
	a.GrpcServer = grpc.NewServer()
	return a
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

	msg := "starting agent " + version
	c.Ui.Output(msg)

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
	go agent.SigHandler()
	agent.GrpcServer.Serve(lsnr)

	return 0

}

func (c *AgentCommand) Help() string {
	return "Run an agent"
}

func (c *AgentCommand) Synopsis() string {
	return "Run an agent"
}
