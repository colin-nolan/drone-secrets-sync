ifeq ($(origin DRONE_TAG), environment)
    VERSION := $(DRONE_TAG)
else
    VERSION ?= $(shell git describe --tags --exact-match HEAD 2> /dev/null || git rev-parse --short HEAD)
endif

BUILD_DIRECTORY := build
RELEASE_DIRECTORY := $(BUILD_DIRECTORY)/release/$(VERSION)

GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)
TARGET_BUILD ?= $(GOOS)/$(GOARCH)
TARGET_BUILDS ?= $(TARGET_BUILD)
BINARY_NAME := drone-secrets-sync
BINARY_OUTPUT_LOCATION ?= $(RELEASE_DIRECTORY)/$(BINARY_NAME)_$(subst /,-,$(TARGET_BUILD))
BINARY_OUTPUT_BIN_COPY_LOCATION := bin/$(BINARY_NAME)
ENTRYPOINT := $(wildcard cmd/cli/*.go)

GO_FILES := $(shell find . -type f -name '*.go' ! -name '*_test.go' ! -path '*/build/*')
MARKDOWN_FILES := $(shell find . -type f -name '*.md' ! -path '*/site-packages/*' ! -path '*build/*' ! -path './test/bats/*')
JSONNET_FILES := $(shell find . -type f -name '*.jsonnet' ! -path '*/build/*')

INSTALL_PATH = /usr/local/bin/$(BINARY_NAME)

KANIKO_EXECUTOR ?= docker run --rm -v ${PWD}:${PWD} -w ${PWD} gcr.io/kaniko-project/executor:latest
DOCKER_IMAGE_NAME := colin-nolan/$(BINARY_NAME):$(VERSION)
TARGET_PLATFORM ?= $(if $(filter darwin,$(GOOS)),linux/$(GOARCH),$(GOOS)/$(GOARCH))
TARGET_PLATFORMS ?= $(TARGET_PLATFORM)
IMAGE_OUTPUT_LOCATION ?= $(RELEASE_DIRECTORY)/$(BINARY_NAME)-image_$(TARGET_PLATFORM).tar
IMAGE_OUTPUT_LOCATIONS := $(foreach target_platform,$(TARGET_PLATFORMS),$(RELEASE_DIRECTORY)/$(BINARY_NAME)-image_$(subst /,-,$(target_platform)).tar)
MULTIARCH_OUTPUT_LOCATION := $(RELEASE_DIRECTORY)/multiarch

SHELL := /bin/bash

all: build

build: 
	iterations=0; \
	for target_build in $(TARGET_BUILDS); do \
		iterations=$$(($${iterations} + 1)); \
		IFS="/" read -r os arch <<< "$${target_build}"; \
		target="$(RELEASE_DIRECTORY)/$(BINARY_NAME)_$${os}-$${arch}"; \
		make "$${target}" GOOS="$${os}" GOARCH="$${arch}" BINARY_OUTPUT_LOCATION="$${target}"; \
	done; \
	if [ "$${iterations}" -eq 1 ]; then \
		mkdir -p "$(dir $(BINARY_OUTPUT_BIN_COPY_LOCATION))"; \
		cp "$${target}" "$(BINARY_OUTPUT_BIN_COPY_LOCATION)"; \
	fi

$(BINARY_OUTPUT_LOCATION): $(GO_FILES)
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build \
		-ldflags "-s -w -X main.version=$(VERSION)" \
		-o "$(BINARY_OUTPUT_LOCATION)" $(ENTRYPOINT)

build-image:
	for target_platform in $(TARGET_PLATFORMS); do \
		target="$(RELEASE_DIRECTORY)/$(BINARY_NAME)-image_$${target_platform//\//-}.tar"; \
		make "$${target}" TARGET_PLATFORM="$${target_platform}" IMAGE_OUTPUT_LOCATION="$${target}"; \
	done

$(IMAGE_OUTPUT_LOCATION): $(GO_FILES) Dockerfile .dockerignore
	mkdir -p $$(dirname $(IMAGE_OUTPUT_LOCATION))
	@# Must work both containerised and not
	$(KANIKO_EXECUTOR) \
		--custom-platform=$(TARGET_PLATFORM) \
		--no-push \
		--dockerfile Dockerfile \
		--build-arg VERSION=$(VERSION) \
		--tar-path $(IMAGE_OUTPUT_LOCATION) \
		--destination $(DOCKER_IMAGE_NAME) \
		--context $${PWD} \
		>&2

build-image-and-load: build-image
	docker load -i $(IMAGE_OUTPUT_LOCATION)

# XXX: this rule does not align with `build-image`, which defines how to build only one image. There is no
#	   multi-image build rule, which will lead to `make` complaining of a target issue if one of the images
#	   does not exist. To get around this, all `build` and `build-image` need to be changed to have multi-os/arch support.
build-image-multiarch: $(MULTIARCH_OUTPUT_LOCATION)
$(MULTIARCH_OUTPUT_LOCATION): $(IMAGE_OUTPUT_LOCATIONS)
	scripts/create-multiarch-image.sh $(MULTIARCH_OUTPUT_LOCATION) $(IMAGE_OUTPUT_LOCATIONS)

install: build
	cp $(BINARY_OUTPUT_LOCATION) $(INSTALL_PATH)

uninstall:
	rm -f $(INSTALL_PATH)

clean:
	go clean
	rm -rf $(RELEASE_DIRECTORY) bin
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
	rm -rf .coverage
	make test-unit
	make test-system

test-unit:
	rm -rf .coverage/unit
	mkdir -p .coverage/unit
	CGO_ENABLED=1 go test -v -cover -race ./... -args -test.gocoverdir="$${PWD}/.coverage/unit"

test-system:
	CGO_ENABLED=1 go build -cover -race -o build/drone-secrets-sync-coveraged $(ENTRYPOINT)

	rm -rf .coverage/system
	mkdir -p .coverage/system

	GOCOVERDIR=.coverage/system SUT=build/drone-secrets-sync-coveraged test/bats/bin/bats -j 4 test/system/tests.bats

test-coverage-report:
	@# TODO: The system test paths are absolute file paths opposed to package paths. It's not clear
	@#       how to correct these. However, codecov.io merges them correctly so not spending any longer
	@#       now trying to fix this so it works locally
	go tool covdata textfmt -i=.coverage/unit,.coverage/system -o .coverage/coverage.out
	go tool cover -html .coverage/coverage.out -o .coverage/coverage.html

version:
	@echo $(VERSION)

.PHONY: all build build-image build-image-multiarch install uninstall clean lint lint-code lint-markdown lint-jsonnet format fmt format-code format-markdown format-jsonnet test test-unit test-system test-coverage-report
