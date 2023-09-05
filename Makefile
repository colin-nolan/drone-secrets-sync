BINARY_NAME = drone-secrets-manager
BINARY_OUTPUT_LOCATION = bin/$(BINARY_NAME)
VERSION ?= unset
ENTRYPOINT = cmd/cli/main.go

all: build

build: 
	go build -ldflags "-s -w -X main.version=$(VERSION)" -o $(BINARY_OUTPUT_LOCATION) $(ENTRYPOINT)

build-docker:
	docker build -t $(BINARY_NAME):$(VERSION) .

clean:
	go clean
	rm -f $(BINARY_OUTPUT_LOCATION)

lint:
	golangci-lint run --timeout 5m0s

test:
	go test -v ./...

.PHONY: all build clean test build-docker
