VERSION := $(shell git describe --always --long --dirty)
GOOS := $(shell go tool dist banner | head -2 | tail -1 | sed -E 's/[^/]* ([a-z0-9]+)\/[A-Za-z0-9 \/]*/\1/')
GOARCH := $(shell go tool dist banner | head -2 | tail -1 | sed -E 's/[^/]*\/([a-z0-9]+)[A-Za-z0-9 \/]*/\1/')

build: prepare compile
.PHONY: build

compile:
	go build -ldflags="-s -w -X main.buildVersion=${VERSION} -X main.buildArch=${GOOS}-${GOARCH}"

prepare:
	go get -d -v ./...
