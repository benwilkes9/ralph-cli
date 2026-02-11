BINARY  := ralph
MODULE  := github.com/benmyles/ralph-cli
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: build test lint clean install release-snapshot

build:
	go build $(LDFLAGS) -o bin/$(BINARY) ./cmd/ralph

install:
	go install $(LDFLAGS) ./cmd/ralph

test:
	go test -race ./...

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/ dist/

release-snapshot:
	goreleaser release --snapshot --clean
