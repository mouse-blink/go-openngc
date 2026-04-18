# SPDX-FileCopyrightText: 2026 Adam Stevko
# SPDX-License-Identifier: MIT

VERSION ?= dev

# Use rtk wrapper if available, otherwise run commands directly.
# For go commands, plain `rtk go …` works.  For other binaries (golangci-lint,
# govulncheck, reuse), use `rtk proxy <bin>` so rtk doesn't misinterpret
# subcommands as paths.
RTK      := $(shell command -v rtk 2>/dev/null)
GOLANGCI := $(if $(RTK),$(RTK) proxy golangci-lint,golangci-lint)
GOVULN   := $(if $(RTK),$(RTK) proxy govulncheck,govulncheck)
REUSE    := $(if $(RTK),$(RTK) proxy reuse,reuse)

.PHONY: build install data gen gen-check test lint vuln reuse check

build:
	$(RTK) go build -ldflags="-X main.AppVersion=$(VERSION)" -o bin/ongc ./cmd/ongc

# gen regenerates ongc/catalog_generated.go from ongc/data/*.csv.
# Run after updating the catalog data files.
gen:
	$(RTK) go generate ./...

# gen-check fails if the committed generated file is out of sync with
# ongc/data/*.csv. Intended for CI to catch missing `make gen` runs.
gen-check:
	$(RTK) go generate ./...
	@if [ -n "$$(git status --porcelain ongc/catalog_generated.go)" ]; then \
		echo "ERROR: ongc/catalog_generated.go is out of sync with ongc/data/*.csv"; \
		echo "       run 'make gen' and commit the result"; \
		git diff --stat ongc/catalog_generated.go; \
		exit 1; \
	fi

install:
	$(RTK) go install -ldflags="-X main.AppVersion=$(VERSION)" ./cmd/ongc

data:
	$(RTK) scripts/download_data.sh

test:
	$(RTK) go mod verify
	$(RTK) go test -race ./...

lint:
	$(GOLANGCI) run ./...

vuln:
	$(GOVULN) ./...

reuse:
	$(REUSE) lint

check: lint vuln test reuse
