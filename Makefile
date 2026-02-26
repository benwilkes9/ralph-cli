BINARY  := ralph
MODULE  := github.com/benwilkes9/ralph-cli
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: build build-linux test lint clean install install-linux release-snapshot

build:
	go build $(LDFLAGS) -o bin/$(BINARY) ./cmd/ralph

build-linux:
	GOOS=linux GOARCH=$(shell go env GOARCH) CGO_ENABLED=0 go build $(LDFLAGS) -o bin/$(BINARY)-linux ./cmd/ralph

install:
	go install $(LDFLAGS) ./cmd/ralph

install-linux: build-linux
	@mkdir -p $(shell go env GOPATH)/bin
	cp bin/$(BINARY)-linux $(shell go env GOPATH)/bin/$(BINARY)-linux

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
