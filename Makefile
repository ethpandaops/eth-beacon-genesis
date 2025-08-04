# eth-beacon-genesis
BUILDTIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
VERSION := $(shell git rev-parse --short HEAD)

GOLDFLAGS += -X 'github.com/ethpandaops/eth-beacon-genesis/buildinfo.BuildVersion="$(VERSION)"'
GOLDFLAGS += -X 'github.com/ethpandaops/eth-beacon-genesis/buildinfo.Buildtime="$(BUILDTIME)"'
GOLDFLAGS += -X 'github.com/ethpandaops/eth-beacon-genesis/buildinfo.BuildRelease="$(RELEASE)"'

.PHONY: all test clean

all: test build

test:
	go test -race -coverprofile=coverage.out -covermode=atomic -vet=off ./...

build:
	@echo version: $(VERSION)
	env CGO_ENABLED=1 go build -v -o bin/ -ldflags="-s -w $(GOLDFLAGS)" ./cmd/*

clean:
	rm -f bin/*
	$(MAKE) -C ui-package clean
