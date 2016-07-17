package main

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"google.golang.org/grpc/grpclog"
	"os"
	"path/filepath"
	"runtime"
)

var (
	config Config
)

type Config struct {
	homedir string
	Agent   *AgentConfig
	Aws     *AwsConfig
}

type AgentConfig struct {
	BindAddr         string `toml:"bindaddr"`
	BindPort         int32  `toml:"bindport"`
	IncomingBufferSz int32  `toml:"incoming_buffer_size"`
	TlsCertificate   string `toml:"certificate"`
	TlsKey           string `toml:"key"`
	UseTls           bool   `toml:"usetls"`
}

type AwsConfig struct {
	PollInterval int32 `toml:"poll_interval"`
	Agents       []string
	UseTls       bool   `toml:"usetls"`
	TrustedCert  string `toml:"customca"`
	CertName     string `toml:"certificatename"`
}

func getConfig(cfg *Config) (err error) {

	cfgfile, err := findConfigFile()
	if err != nil {
		return err
	}

	if _, err := toml.DecodeFile(cfgfile, cfg); err != nil {
		return err
	}

	return nil

}

func findConfigFile() (string, error) {
	cfiles := make([]string, 0, 4)
	home := os.Getenv("CONDUIT_HOME")
	if len(home) > 0 {
		grpclog.Printf("using CONDUIT_HOME: %s\n", home)
		homecf := os.ExpandEnv("${CONDUIT_HOME}/etc/conduit.conf")
		cfiles = append(cfiles, homecf)
		homecf = os.ExpandEnv("${CONDUIT_HOME}/conduit.conf")
		cfiles = append(cfiles, homecf)
	}
	pwd, err := getPwd()
	if err != nil {
		grpclog.Printf("warn: %s\n", err)
	} else {
		pwdcf := ""
		if runtime.GOOS == "windows" {
			pwdcf = pwd + "\\conduit.conf"
		} else {
			pwdcf = pwd + "/conduit.conf"
		}
		cfiles = append(cfiles, pwdcf)
	}
	defaultcf := "/opt/conduit/etc/conduit.conf"
	cfiles = append(cfiles, defaultcf)
	for _, path := range cfiles {
		if _, err := os.Stat(path); err == nil {
			grpclog.Printf("Using config file: %s", path)
			return path, nil
		}
	}

	err = fmt.Errorf("no config file found")
	return "", err
}

func getPwd() (string, error) {
	exe, err := filepath.Abs(os.Args[0])

	if err != nil {
		return "", fmt.Errorf("cannot find full path for this executable: %s", err)
	}

	path := filepath.Dir(exe)
	fexe, err := filepath.EvalSymlinks(exe)

	if err != nil {
		if _, err = os.Stat(exe + ".exe"); err == nil {
			fexe = filepath.Clean(exe + ".exe")
		}
	}

	if err == nil && fexe != exe {
		path = filepath.Dir(fexe)
	}

	return path, nil
}
