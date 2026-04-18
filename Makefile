VERSION ?= dev

# Use rtk wrapper if available, otherwise run commands directly.
RTK := $(shell command -v rtk 2>/dev/null && echo rtk)

.PHONY: build install data test lint

build:
	$(RTK) go build -ldflags="-X main.AppVersion=$(VERSION)" -o bin/ongc ./cmd/ongc

install:
	$(RTK) go install -ldflags="-X main.AppVersion=$(VERSION)" ./cmd/ongc

data:
	$(RTK) scripts/download_data.sh

test:
	$(RTK) go test ./...

lint:
	$(RTK) golangci-lint run ./...
