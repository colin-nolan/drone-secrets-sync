BINARY_NAME = drone-secrets-sync
BINARY_OUTPUT_LOCATION = bin/$(BINARY_NAME)
VERSION = $(shell git describe --tags --exact-match HEAD 2> /dev/null || git rev-parse --short HEAD)
ENTRYPOINT = cmd/cli/*.go
INSTALL_PATH = /usr/local/bin/$(BINARY_NAME)
DOCKER_IMAGE_NAME = colinnolan/$(BINARY_NAME):$(VERSION)

all: build

build: 
	@go build -ldflags "-s -w -X main.version=$(VERSION)" -o $(BINARY_OUTPUT_LOCATION) $(ENTRYPOINT)
	@echo "$(BINARY_OUTPUT_LOCATION)"

build-docker:
	@docker build -t $(DOCKER_IMAGE_NAME) .
	@echo "$(DOCKER_IMAGE_NAME)"

install: build
	cp $(BINARY_OUTPUT_LOCATION) $(INSTALL_PATH)

uninstall:
	rm -f $(INSTALL_PATH)

clean:
	go clean
	rm -f $(BINARY_OUTPUT_LOCATION)

lint: lint-code lint-markdown

lint-code:
	golangci-lint run --timeout 15m0s

lint-markdown:
	mdformat --check *.md

format: format-code format-markdown

format-code:
	go fmt ./...

format-markdown:
	mdformat *.md

test:
	CGO_ENABLED=1 go test -v -race -covermode=atomic -coverprofile=coverage.out ./...

.PHONY: all build clean test build-docker
