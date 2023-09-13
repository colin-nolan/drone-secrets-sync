VERSION = $(shell git describe --tags --exact-match HEAD 2> /dev/null || git rev-parse --short HEAD)

BUILD_DIRECTORY := build
RELEASE_DIRECTORY:= $(BUILD_DIRECTORY)/release/$(VERSION)

GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
BINARY_NAME := drone-secrets-sync
BINARY_OUTPUT_LOCATION := $(RELEASE_DIRECTORY)/$(BINARY_NAME)_$(GOOS)-$(GOARCH)
ENTRYPOINT := cmd/cli/*.go

TARGET_ARCH := amd64 arm64 arm
TARGET_OS := linux 

GO_FILES := $(shell find . -type f -name '*.go' ! -name '*_test.go' ! -path '*/build/*')
MARKDOWN_FILES := $(shell find . -type f -name '*.md' ! -path '*/site-packages/*' ! -path '*/build/*')
JSONNET_FILES := $(shell find . -type f -name '*.jsonnet' ! -path '*/build/*')

INSTALL_PATH := /usr/local/bin/$(BINARY_NAME)

KANIKO_EXECUTOR ?= docker run --rm -v ${PWD}:${PWD} -w ${PWD} gcr.io/kaniko-project/executor:latest
DOCKER_IMAGE_NAME := colin-nolan/$(BINARY_NAME):$(VERSION)
CONTAINER_OUTPUT_LOCATION := $(RELEASE_DIRECTORY)/container_$(GOOS)-$(GOARCH).tar

all: build

build: $(BINARY_OUTPUT_LOCATION)
$(BINARY_OUTPUT_LOCATION): $(GO_FILES)
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags "-s -w -X main.version=$(VERSION)" -o $(BINARY_OUTPUT_LOCATION) $(ENTRYPOINT)

build-container: $(CONTAINER_OUTPUT_LOCATION) 
$(CONTAINER_OUTPUT_LOCATION): $(GO_FILES) Dockerfile .dockerignore
	mkdir -p $$(dirname $(CONTAINER_OUTPUT_LOCATION))
	# Must work both containerised and not
	$(KANIKO_EXECUTOR) \
		--custom-platform=$(GOOS)/$(GOARCH) \
		--no-push \
		--dockerfile Dockerfile \
		--build-arg VERSION=$(VERSION) \
		--tar-path $(CONTAINER_OUTPUT_LOCATION) \
		--destination $(DOCKER_IMAGE_NAME) \
		--context ${PWD} \
		>&2

install: build
	cp $(BINARY_OUTPUT_LOCATION) $(INSTALL_PATH)

uninstall:
	rm -f $(INSTALL_PATH)

clean:
	go clean
	rm -rf $(BUILD_DIRECTORY)
	rm -f coverage.out output.log

lint: lint-code lint-markdown lint-jsonnet

lint-code:
	golangci-lint run --timeout 15m0s

lint-markdown:
	mdformat --check $(MARKDOWN_FILES)

lint-jsonnet:
	for file in $(JSONNET_FILES); do \
		jsonnetfmt --test $${file}; \
	done			

format: format-code format-markdown format-jsonnet
fmt: format

format-code: $(GO_FILES)
	go fmt ./...

format-markdown:
	mdformat $(MARKDOWN_FILES)

format-jsonnet:
	for file in $(JSONNET_FILES); do \
		jsonnetfmt -i $${file}; \
	done			

test:
	CGO_ENABLED=1 go test -v -race -covermode=atomic -coverprofile=coverage.out ./...

version:
	@echo $(VERSION)

.PHONY: all build build-container install uninstall clean lint lint-code lint-markdown lint-jsonnet format fmt format-code format-markdown format-jsonnet test
