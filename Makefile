.PHONY: all build test clean fmt lint help

all: test build

build:
	@mkdir -p dist
	CGO_ENABLED=0 go build -o dist/ns-flows
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
