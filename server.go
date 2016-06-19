package main

import (
	"flag"
	"github.com/mitchellh/cli"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	//	"log"
	//	"net"
	"fmt"
	"io"
)

const (
	serverAddr = "127.0.0.1:6666"
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

	msg := "starting server " + version
	c.Ui.Output(msg)

	conn, err := grpc.Dial(serverAddr, grpc.WithInsecure())
	if err != nil {
		c.Ui.Output(fmt.Sprintf("error connecting to server: %s\n", err))
		return 1
	}
	defer conn.Close()
	agent := NewConduitAgentClient(conn)
	stream, err := agent.Pull(context.Background(), &Token{Domain: "localhost"})
	if err != nil {
		c.Ui.Output(fmt.Sprintf("error retrieving agent events: %s\n", err))
	}
	for {
		paket, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Printf("%v.Msg: %v", agent, err)
			return 1
		}
		fmt.Println(paket.Msg)
	}

	return 0

}

func (c *ServerCommand) Help() string {
	return "Run as a server (detailed help information here)"
}

func (c *ServerCommand) Synopsis() string {
	return "Run as a server"
}