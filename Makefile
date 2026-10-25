.PHONY: all web build test run clean

all: build

# The interface, built into internal/webui/dist.
web:
	cd web && npm ci && npm run build

# The version the binary reports: the latest tag, without the "v" (or "dev" outside a git checkout).
# Override it with: make build VERSION=1.2.0
VERSION ?= $(or $(shell git describe --tags --always --dirty 2>/dev/null | sed 's/^v//'),dev)

# One binary with the interface inside.
build: web
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version=$(VERSION)" -o bin/letspush ./cmd/letspush

test:
	go vet ./...
	go test ./...

# Needs LETSPUSH_SECRET (and LETSPUSH_ADMIN_EMAIL / LETSPUSH_ADMIN_PASSWORD the first time).
run: build
	./bin/letspush

clean:
	rm -rf bin
