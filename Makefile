BINARY  := ralph
MODULE  := github.com/benwilkes9/ralph-cli
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: build test lint clean install release-snapshot

build:
	go build $(LDFLAGS) -o bin/$(BINARY) ./cmd/ralph

install:
	go install $(LDFLAGS) ./cmd/ralph

test:
	go test -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -func=coverage.out | tail -1

cover: test
	go tool cover -html=coverage.out

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/ dist/

release-snapshot:
	goreleaser release --snapshot --clean
