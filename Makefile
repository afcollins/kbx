.PHONY: build lint test clean install
BINARY  := kbx
BUILD_DATE = $(shell date '+%Y-%m-%d-%H:%M:%S')
VERSION := $(shell git rev-parse --short HEAD)
LDFLAGS := -s -w -X main.Version=$(VERSION) -X main.BuildDate=$(BUILD_DATE) -X main.GitCommit=$(VERSION)

build:
	go build -ldflags="$(LDFLAGS)" -o $(BINARY) .

lint:
	go vet ./...
	$(GOPATH)/bin/golangci-lint fmt
	$(GOPATH)/bin/golangci-lint run --fix 

install: build
	go install

test:
	go test ./...

clean:
	rm -f $(BINARY)
