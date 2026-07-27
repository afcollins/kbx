.PHONY: build lint test clean

build:
	go build -o kbx .

lint:
	go vet ./...
	$(GOPATH)/bin/golangci-lint fmt
	$(GOPATH)/bin/golangci-lint run --fix 


test:
	go test ./...

clean:
	rm -f kbx
