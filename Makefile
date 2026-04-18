VERSION ?= dev

# Use rtk wrapper if available, otherwise run commands directly.
# For go commands, plain `rtk go …` works.  For other binaries (golangci-lint),
# use `rtk proxy <bin>` so rtk doesn't misinterpret subcommands as paths.
RTK := $(shell command -v rtk 2>/dev/null && echo rtk)
GOLANGCI := $(if $(RTK),rtk proxy golangci-lint,golangci-lint)

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
	$(GOLANGCI) run ./...
