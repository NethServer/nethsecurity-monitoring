# Contributing to NethSecurity Monitoring

Thank you for your interest in contributing to NethSecurity Monitoring! This document provides guidelines and instructions for contributing to the project.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Development Setup](#development-setup)
- [Building the Project](#building-the-project)
- [Running Tests](#running-tests)
- [Code Quality](#code-quality)
- [Making Changes](#making-changes)
- [Submitting Changes](#submitting-changes)
- [Dependency Management](#dependency-management)

## Code of Conduct

By participating in this project, you agree to maintain a respectful and collaborative environment for all contributors.

## Development Setup

### Prerequisites

- **Go**: Version 1.23.12 or later
- **golangci-lint**: For code quality checks

### Clone the Repository

```bash
git clone https://github.com/nethserver/nethsecurity-monitoring.git
cd nethsecurity-monitoring
```

### Install Dependencies

```bash
go mod download
```

## Building the Project

### Using Make

The project includes a Makefile for easy building:

```bash
# Build the project
make build

# Run tests and build
make

# Run only tests
make test

# Clean build artifacts
make clean

# Format code
make fmt

# Run linter
make lint
```

## Running Tests

### Using Make

```bash
make test
```

## Code Quality

This project uses [golangci-lint](https://golangci-lint.run/) for maintaining code quality. The configuration is in `.golangci.yml`.

### Running the Linter

Using Make:

```bash
make lint
make format
```

Or directly:

## Making Changes

### Branching Strategy

- Create a feature branch from `main` for new features or bug fixes
- Use descriptive branch names (e.g., `feature/add-metrics-export`, `fix/flow-cleanup-race`)

### Commit Messages

Write clear, descriptive commit messages for your work. Individual commit messages can be informal, as PRs will be squashed upon merge.

### Pull Request Titles

PR titles **must** follow the [Conventional Commits](https://www.conventionalcommits.org/) specification, as they will become the final commit message when the PR is squashed and merged.

**Examples:**
```
feat(flows): add DNS query tracking to flow processor
```

```
fix(scheduler): prevent race condition in flow cleanup
```

```
docs: update installation instructions in README
```

```
chore(deps): update golang to 1.23.13
```

### Code Style

- Follow standard Go conventions and idioms
- Write clear, self-documenting code with meaningful variable names
- Add comments for complex logic or non-obvious behavior
- Keep functions focused and reasonably sized
- Use thread-safe patterns when dealing with concurrent access

### API Changes

Whenever you add or modify an HTTP endpoint or its query parameters, **update [openapi.yaml](openapi.yaml) in the same PR** as the corresponding changes to [api/flows.go](api/flows.go). Keep the following in sync:

- Query parameter names, types, defaults, and validation constraints
- Response body schemas (add or extend `components/schemas` entries as needed)
- `SortBy` enum values — any new value added to `flows/sort.go` and the `validate:"oneof=…"` tag in `api/flows.go` must also appear in the `sort_by` parameter enum in `openapi.yaml`

PRs that change the API surface without updating `openapi.yaml` will be rejected.

### Testing

- Write tests for new features and bug fixes
- Ensure all tests pass before submitting
- Aim for good test coverage (the project tracks coverage)
- Use table-driven tests where appropriate

## Submitting Changes

### Pull Request Process

1. **Update your branch** with the latest changes from `main`:
   ```bash
   git fetch origin
   git rebase origin/main
   ```

2. **Run tests and linting**:
   ```bash
   go test -v --cover ./...
   golangci-lint run
   ```

3. **Push your branch**:
   ```bash
   git push origin your-branch-name
   ```

4. **Create a Pull Request** on GitHub:
   - Use a PR title that follows [the provided standard](#pull-request-titles)
   - Provide a clear description of the changes
   - Reference any related issues
   - Ensure CI checks pass

5. **Address review feedback** promptly and professionally

### Pull Request Guidelines

- Keep PRs focused on a single feature or fix
- Include relevant tests
- Update documentation if needed
- Ensure the build passes in CI
- Be responsive to review comments

## Dependency Management

This project uses [Renovate](https://docs.renovatebot.com/) for automated dependency management (see `renovate.json`).
