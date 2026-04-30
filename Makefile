.PHONY: all build test clean format lint help $(CMDS) $(addprefix build-,$(CMDS))

CMDS := ns-flows ns-stats

all: test build

# Build all commands
build: $(addprefix build-,$(CMDS))
	@echo "Build complete, binaries are in dist/"

# Build individual commands
$(addprefix build-,$(CMDS)): build-%:
	@mkdir -p dist
	go build -o dist/$* ./cmd/$*

# Convenience targets for individual commands
$(CMDS): %: build-%

test:
	go test -v --cover ./...

clean:
	rm -rf dist
	@echo "Removed build artifacts"

format:
	golangci-lint fmt

lint:
	golangci-lint run

help:
	@echo "Available targets:"
	@echo "  make all              - Run tests and build all commands"
	@echo "  make build            - Build all commands (ns-flows, ns-stats)"
	@echo "  make build-ns-flows   - Build ns-flows only"
	@echo "  make build-ns-stats   - Build ns-stats only"
	@echo "  make ns-flows         - Build ns-flows (shorthand)"
	@echo "  make ns-stats         - Build ns-stats (shorthand)"
	@echo "  make test             - Run all tests"
	@echo "  make clean            - Remove build artifacts"
	@echo "  make format           - Format code with golangci-lint"
	@echo "  make lint             - Run golangci-lint checks"
	@echo "  make help             - Show this help message"
