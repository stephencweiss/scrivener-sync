.PHONY: build clean test install fmt lint build-all

BINARY=scriv-sync
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

build:
	go build $(LDFLAGS) -o $(BINARY) ./cmd/scriv-sync

clean:
	rm -f $(BINARY) $(BINARY)-*

test:
	go test -v ./...

install: build
	cp $(BINARY) /usr/local/bin/

fmt:
	go fmt ./...

lint:
	golangci-lint run

# Cross-compilation
build-all: clean
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY)-darwin-amd64 ./cmd/scriv-sync
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY)-darwin-arm64 ./cmd/scriv-sync
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY)-linux-amd64 ./cmd/scriv-sync
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY)-windows-amd64.exe ./cmd/scriv-sync
