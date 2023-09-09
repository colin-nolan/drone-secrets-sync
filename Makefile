BINARY_NAME = drone-secrets-sync
BINARY_OUTPUT_LOCATION = bin/$(BINARY_NAME)
VERSION = $(shell git describe --tags --exact-match HEAD 2> /dev/null || git rev-parse --short HEAD)
ENTRYPOINT = cmd/cli/*.go
INSTALL_PATH = /usr/local/bin/$(BINARY_NAME)

all: build

build: 
	go build -ldflags "-s -w -X main.version=$(VERSION)" -o $(BINARY_OUTPUT_LOCATION) $(ENTRYPOINT)

build-docker:
	docker build -t $(BINARY_NAME):$(VERSION) .

install: build
	cp $(BINARY_OUTPUT_LOCATION) $(INSTALL_PATH)

uninstall:
	rm -f $(INSTALL_PATH)

clean:
	go clean
	rm -f $(BINARY_OUTPUT_LOCATION)

lint:
	golangci-lint run --timeout 15m0s

lint-markdown:
	mdformat CHANGELOG.md README.md

test:
	go test -v ./...

.PHONY: all build clean test build-docker
