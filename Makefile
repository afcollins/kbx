.PHONY: build lint test clean install

build:
	go build -o kbx .

lint:
	go vet ./...
	$(GOPATH)/bin/golangci-lint fmt
	$(GOPATH)/bin/golangci-lint run --fix 

install: build
	go install

test:
	go test ./...

clean:
	rm -f kbx
