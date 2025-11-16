# Contributing to Controller Framework (ctrlfwk)

Thank you for your interest in contributing to the Controller Framework! This document provides guidelines and information for contributors.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Contributing Process](#contributing-process)
- [Development Guidelines](#development-guidelines)
- [Testing](#testing)
- [Documentation](#documentation)
- [Release Process](#release-process)
- [Getting Help](#getting-help)

## Code of Conduct

This project adheres to a code of conduct that we expect all contributors to follow. Please be respectful and inclusive in all interactions.

## Getting Started

### Prerequisites

Before contributing, ensure you have the following installed:

- **Go 1.25+**: The project uses Go 1.25 or later
- **Git**: For version control
- **Make**: For running build tasks
- **Docker**: For container builds and testing (optional)
- **kubectl**: For Kubernetes testing (optional)
- **Kind/k3d**: For local Kubernetes testing (optional)

### Setting Up Your Development Environment

1. **Fork the repository** on GitHub
2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/YOUR_USERNAME/controller-fwk.git
   cd controller-fwk
   ```

3. **Add the original repository as upstream**:
   ```bash
   git remote add upstream https://github.com/u-ctf/controller-fwk.git
   ```

4. **Install dependencies**:
   ```bash
   go mod download
   ```

5. **Generate mocks and code**:
   ```bash
   go generate ./...
   ```

6. **Run tests to verify setup**:
   ```bash
   go test ./...
   ```

## Development Setup

### Project Structure

```
controller-fwk/
├── *.go                    # Core framework files
├── instrument/             # Instrumentation and observability
│   ├── *.go
│   └── *_test.go
├── mocks/                  # Generated mocks for testing
├── tests/                  # Integration tests and examples
│   └── operator/           # Test operator implementation
└── go.mod                  # Go module definition
```

### Building the Project

```bash
# Build all packages
go build ./...

# Run tests
go test ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Code Generation

The project uses code generation for mocks and other artifacts:

```bash
# Generate all code
go generate ./...

# Generate only mocks
go generate ./generate.go
```

## Contributing Process

### 1. Create an Issue

Before starting work, please:
- Check if an issue already exists for your proposed change
- If not, create a new issue describing:
  - The problem you're solving
  - Your proposed solution
  - Any breaking changes

### 2. Create a Branch

Create a feature branch from the main branch:

```bash
git checkout main
git pull upstream main
git checkout -b feature/your-feature-name
```

Branch naming conventions:
- `feature/description` - for new features
- `fix/description` - for bug fixes
- `docs/description` - for documentation changes
- `refactor/description` - for refactoring
- `test/description` - for test improvements

### 3. Make Changes

Follow the [Development Guidelines](#development-guidelines) when making changes.

### 4. Test Your Changes

```bash
# Run unit tests
go test ./...

# Run tests with race detection
go test -race ./...

# Run integration tests (if applicable)
cd tests/operator
make test
```

### 5. Commit Your Changes

Write clear, descriptive commit messages:

```bash
git add .
git commit -m "feat: add support for custom resource lifecycle hooks

- Add BeforeReconcile and AfterReconcile hooks to Resource
- Update Resource interface to include new hook methods
- Add comprehensive tests for lifecycle hooks
- Update documentation with hook examples

Closes #123"
```

Commit message format:
- `feat:` - new features
- `fix:` - bug fixes
- `docs:` - documentation changes
- `style:` - formatting changes
- `refactor:` - code refactoring
- `test:` - test additions/changes
- `chore:` - maintenance tasks

### 6. Push and Create Pull Request

```bash
git push origin feature/your-feature-name
```

Create a pull request through GitHub with:
- Clear title and description
- Reference to related issues
- Description of changes made
- Testing instructions
- Breaking change notes (if any)

## Development Guidelines

### Go Code Style

- Follow standard Go formatting (`gofmt`, `goimports`)
- Use meaningful variable and function names
- Add comments for exported functions and types
- Follow Go best practices and idioms

### API Design

- **Backwards Compatibility**: Avoid breaking changes in public APIs
- **Generics**: Use type-safe generics where appropriate
- **Interfaces**: Prefer small, focused interfaces
- **Error Handling**: Return descriptive errors with context

### Framework Patterns

- **Step-based Architecture**: Follow the stepper pattern for reconciliation
- **Resource Lifecycle**: Implement proper lifecycle hooks
- **Type Safety**: Leverage Go generics for type-safe operations
- **Observability**: Include logging and metrics where appropriate

### Documentation

- Document all exported functions and types
- Include examples in documentation
- Update README.md for significant changes
- Add inline comments for complex logic

## Testing

### Unit Tests

- Write tests for all new functionality
- Maintain or improve test coverage
- Use table-driven tests where appropriate
- Mock external dependencies using the provided mocks

Example test structure:
```go
func TestNewResource(t *testing.T) {
    tests := []struct {
        name     string
        input    any
        expected any
        wantErr  bool
    }{
        // Test cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### Integration Tests

- Add integration tests for significant features
- Use the test operator in `tests/operator/` as a reference
- Ensure tests can run in CI environment

### Mocking

- Generate mocks using `go generate`
- Use mocks for external dependencies
- Keep mocks up to date with interface changes

```bash
# Regenerate mocks
go generate ./...
```

## Documentation

### Code Documentation

- All exported types, functions, and methods must have doc comments
- Doc comments should start with the name of the item
- Include examples for complex functionality

### README Updates

- Update README.md for new features
- Add examples for new functionality
- Keep the quick start guide current

### API Documentation

- Use `go doc` friendly documentation
- Include examples in doc comments where helpful
- Document any breaking changes

## Release Process

### Versioning

We follow [Semantic Versioning](https://semver.org/):
- **MAJOR**: Breaking changes
- **MINOR**: New features (backwards compatible)
- **PATCH**: Bug fixes (backwards compatible)

### Release Checklist

1. Update version in relevant files
2. Update CHANGELOG.md (if exists)
3. Ensure all tests pass
4. Update documentation
5. Create release notes
6. Tag the release

## Getting Help

### Community

- **GitHub Issues**: For bugs and feature requests
- **GitHub Discussions**: For questions and general discussion
- **Documentation**: Check pkg.go.dev for API documentation

### Development Questions

If you have questions about:
- **Architecture decisions**: Open a discussion or issue
- **Implementation details**: Check existing code and tests
- **Best practices**: Refer to this guide and existing patterns

### Reporting Issues

When reporting issues, please include:
- Go version
- Operating system
- Steps to reproduce
- Expected vs actual behavior
- Relevant logs or error messages

## Code Review Process

### Review Criteria

Pull requests are reviewed for:
- **Functionality**: Does it work as intended?
- **Code Quality**: Is it well-written and maintainable?
- **Testing**: Are there adequate tests?
- **Documentation**: Is it properly documented?
- **Compatibility**: Does it maintain backwards compatibility?

### Review Timeline

- Initial review: Within 1-2 business days
- Follow-up reviews: Within 1 business day
- Merge: After approval and CI passes

## License

By contributing to this project, you agree that your contributions will be licensed under the [BSD 3-Clause License](LICENSE).

---

Thank you for contributing to the Controller Framework! Your contributions help make Kubernetes controller development easier and more efficient for everyone.
