package main

import (
	"fmt"
	"github.com/mitchellh/cli"
	"os"
)

var (
	cfgfile string = "/opt/conduit/etc/conduit.json"
	ui      *cli.BasicUi
	version string
)

func init() {
	ui = &cli.BasicUi{
		Reader:      os.Stdin,
		Writer:      os.Stdout,
		ErrorWriter: os.Stderr,
	}
}

func main() {

	c := cli.NewCLI("conduit", "0.0.1")
	c.Args = os.Args[1:]
	version = c.Version

	c.Commands = map[string]cli.CommandFactory{
		"server": serverCmdFactory,
		"agent":  agentCmdFactory,
		//		"telegraf": telegrafCmdFactory,
	}

	exitStatus, err := c.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}

	os.Exit(exitStatus)
}
