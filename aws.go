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
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/grpclog"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type CollectorAwsCommand struct {
	Port int
	Ui   cli.Ui
}

func collectorAwsCmdFactory() (cli.Command, error) {
	return &CollectorAwsCommand{
		Ui: &cli.ColoredUi{
			Ui:          ui,
			OutputColor: cli.UiColorGreen,
		},
	}, nil
}

type conduitAgentConnection struct {
	addr     string
	conn     *grpc.ClientConn
	dopts    []grpc.DialOption
	shutdown chan bool
}

func (c *conduitCollecterAws) NewAgentConnection(hostname string, port int, shut chan bool) (a *conduitAgentConnection, err error) {
	a = new(conduitAgentConnection)
	portstr := fmt.Sprintf("%d", port)
	a.addr = hostname + ":" + portstr

	if c.config.UseTls {
		var creds credentials.TransportCredentials
		if c.config.CertName == "" {
			err := fmt.Errorf("error: common certificate name is missing from the config, required when using tls.\n")
			return nil, err
		}
		if c.config.TrustedCert != "" {
			var err error
			creds, err = credentials.NewClientTLSFromFile(c.config.TrustedCert, c.config.CertName)
			if err != nil {
				grpclog.Fatalf("Failed to create TLS credentials %v", err)
			}
		} else {
			creds = credentials.NewClientTLSFromCert(nil, c.config.CertName)
		}
		a.dopts = append(a.dopts, grpc.WithTransportCredentials(creds))
	} else {
		a.dopts = append(a.dopts, grpc.WithInsecure())
	}

	a.shutdown = shut
	a.conn, err = grpc.Dial(a.addr, a.dopts...)
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (c *conduitAgentConnection) Poll() (err error) {

	for {

		select {
		case <-c.shutdown:
			// shutdown signal
			grpclog.Printf("aws: poll shutdown\n")
			return nil
		case <-time.After(time.Minute):
			grpclog.Printf("aws: poll timeout\n")
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

type conduitCollecterAws struct {
	sigs     chan os.Signal
	shutdown chan bool
	incoming chan []byte
	wait     sync.WaitGroup
	config   *AwsConfig
}

func newCollector() (*conduitCollecterAws, error) {
	c := new(conduitCollecterAws)
	c.sigs = make(chan os.Signal, 1)
	c.shutdown = make(chan bool, 1)
	c.incoming = make(chan []byte, 1024)

	var cfg *Config
	cfg = &Config{}
	err := getConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("error reading config: %s", err)
	}
	c.config = cfg.Aws

	return c, nil
}

func (c *conduitCollecterAws) SigHandler() {
	defer c.wait.Done()
	signal.Notify(c.sigs,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	sig := <-c.sigs
	grpclog.Printf("signal received: %v\n", sig)
	c.Stop()
	return
}

func (c *conduitCollecterAws) Stop() {

	//c.conn.Close()
	close(c.shutdown)
	c.wait.Wait()

	// Release all remaining resources.
	c.shutdown = nil
	c.incoming = nil
}

func (c *CollectorAwsCommand) Run(args []string) int {

	cmdFlags := flag.NewFlagSet("aws", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.Ui.Output(c.Help()) }

	cmdFlags.IntVar(&c.Port, "port", 6666, "The port on which to run the conduit collector")
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	// controller channel

	// signal channel

	// pull channel

	grpclog.Printf("starting conduit aws %s\n", version)

	collector, err := newCollector()
	if err != nil {
		grpclog.Printf("error initialising aws collector: %s\n", err)
		return 1
	}

	conn, err := collector.NewAgentConnection("127.0.0.1", 6666, collector.shutdown)
	if err != nil {
		grpclog.Printf("error connecting to agent: %s\n", err)
		return 1
	}
	collector.wait.Add(1)
	go collector.SigHandler()
	conn.Poll()
	grpclog.Printf("aws: poll finished\n")
	collector.Stop()

	return 0

}

func (c *CollectorAwsCommand) Help() string {
	return "Run as a collector (detailed help information here)"
}

func (c *CollectorAwsCommand) Synopsis() string {
	return "Run as a collector"
}
