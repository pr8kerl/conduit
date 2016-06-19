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
