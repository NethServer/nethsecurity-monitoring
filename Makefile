.PHONY: all build test clean fmt lint help

all: test build

build:
	@mkdir -p dist
	go build -o dist/ns-flows ./cmd/ns-flows
	@echo "Build complete, binaries are in dist/"

test:
	go test -v --cover ./...

clean:
	rm -rf dist
	@echo "Removed build artifacts"

format:
	golangci-lint fmt

lint:
	golangci-lint run
