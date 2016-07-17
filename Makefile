GOROOT := /usr/local/go
GOPATH := $(shell pwd)
GOBIN  := $(GOPATH)/bin
PATH   := $(GOROOT)/bin:$(PATH)
DEPS   := google.golang.org/grpc github.com/mitchellh/cli github.com/BurntSushi/toml


all: conduit

deps: $(DEPS)
	GOPATH=$(GOPATH) go get -u $^

proto: packet.proto
		protoc $^ --go_out=plugins=grpc:.

conduit: main.go packet.pb.go aws.go agent.go config.go
    # always format code
		GOPATH=$(GOPATH) go fmt $^
		# vet it
		GOPATH=$(GOPATH) go tool vet $^
    # binary
		GOPATH=$(GOPATH) go build -o $@ -v $^
		touch $@

linux64: main.go packet.pb.go aws.go agent.go config.go
    # always format code
		GOPATH=$(GOPATH) go fmt $^
		# vet it
		GOPATH=$(GOPATH) go tool vet $^
    # binary
		GOOS=linux GOARCH=amd64 GOPATH=$(GOPATH) go build -o console-linux-amd64.bin -v $^
		touch conduit-linux-amd64.bin

win64: main.go packet.pb.go aws.go agent.go config.go
    # always format code
		GOPATH=$(GOPATH) go fmt $^
		# vet it
		GOPATH=$(GOPATH) go tool vet $^
    # binary
		GOOS=windows GOARCH=amd64 GOPATH=$(GOPATH) go build -o console-windows-amd64.bin -v $^
		touch conduit-windows-amd64.bin

.PHONY: $(DEPS) clean pipes

pipes: 
	rsync -av pipes/*.go src/bitbucket.org/pr8kerl/conduit/pipes/

clean:
	rm -f conduit conduit-linux-amd64.bin conduit-windows-amd64.bin

