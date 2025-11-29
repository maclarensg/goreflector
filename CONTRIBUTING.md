# Contributing to goreflector

Thank you for your interest in contributing to goreflector! This document provides guidelines and instructions for contributing.

## Getting Started

### Prerequisites

- Go 1.21 or higher
- [devbox](https://www.jetify.com/devbox) (recommended)
- Git

### Setting Up Development Environment

1. Fork and clone the repository:
```bash
git clone https://github.com/YOUR_USERNAME/goreflector.git
cd goreflector
```

2. Set up devbox environment:
```bash
devbox shell
```

3. Install dependencies:
```bash
task deps
```

4. Verify your setup:
```bash
task ci
```

## Development Workflow

### 1. Create a Branch

Create a feature branch from `main`:
```bash
git checkout -b feature/your-feature-name
```

Use prefixes:
- `feature/` - New features
- `fix/` - Bug fixes
- `docs/` - Documentation updates
- `refactor/` - Code refactoring
- `test/` - Test additions/updates

### 2. Make Your Changes

- Write clean, idiomatic Go code
- Follow the existing code style
- Add tests for new functionality
- Update documentation as needed

### 3. Test Your Changes

Run the full test suite:
```bash
task test
```

Check code coverage:
```bash
task test:coverage
```

Run linters:
```bash
task lint
```

Run security scan:
```bash
task gosec
```

Or run everything at once:
```bash
task ci
```

### 4. Commit Your Changes

We use conventional commits. Format your commit messages like:

```
<type>: <description>

[optional body]

[optional footer]
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

Examples:
```bash
git commit -m "feat: add support for custom headers"
git commit -m "fix: handle timeout errors gracefully"
git commit -m "docs: update README with new examples"
```

### 5. Push and Create Pull Request

```bash
git push origin feature/your-feature-name
```

Then create a pull request on GitHub.

## Code Quality Standards

### Testing Requirements

- All new features must include tests
- Aim for 90% code coverage on new code
- Tests must pass with race detector (`-race`)
- Integration tests for end-to-end scenarios

### Code Style

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` for formatting (run `task format`)
- Pass `golangci-lint` checks
- Pass `gosec` security checks
- No compiler warnings

### Documentation

- Update README.md for user-facing changes
- Add GoDoc comments for exported functions
- Include examples in documentation
- Update CHANGELOG.md

## Testing Guidelines

### Unit Tests

Test individual functions in isolation:

```go
func TestFunctionName(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"case 1", "input", "output"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := FunctionName(tt.input)
            if result != tt.expected {
                t.Errorf("expected %s, got %s", tt.expected, result)
            }
        })
    }
}
```

### Integration Tests

Test end-to-end scenarios:

```go
func TestIntegrationScenario(t *testing.T) {
    // Set up test backend
    backend := httptest.NewServer(...)
    defer backend.Close()

    // Create proxy
    proxy, _ := NewProxy(config, logger)

    // Make requests and verify
    ...
}
```

### Running Specific Tests

```bash
# Run specific test
go test -run TestFunctionName

# Run tests in specific file
go test -run TestProxy proxy_test.go

# Run with verbose output
go test -v ./...

# Run with coverage
go test -coverprofile=coverage.out ./...
```

## Pull Request Guidelines

### Before Submitting

- [ ] All tests pass (`task test`)
- [ ] Linting passes (`task lint`)
- [ ] Security scan passes (`task gosec`)
- [ ] Code coverage maintained or improved
- [ ] Documentation updated
- [ ] CHANGELOG.md updated
- [ ] Commits follow conventional commit format

### PR Description Template

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
Describe testing performed

## Checklist
- [ ] Tests added/updated
- [ ] Documentation updated
- [ ] CI passes
- [ ] No breaking changes (or documented)
```

### Review Process

1. Automated checks must pass (CI)
2. Code review by maintainer
3. Address review feedback
4. Approval and merge

## Project Structure

```
goreflector/
├── main.go              # CLI entry point and argument parsing
├── proxy.go             # Core proxy implementation
├── *_test.go            # Test files
├── Taskfile.yml         # Task automation
├── devbox.json          # Development environment
├── go.mod               # Go module definition
├── go.sum               # Dependency checksums
├── README.md            # User documentation
├── CONTRIBUTING.md      # This file
├── CHANGELOG.md         # Version history
└── LICENSE              # License file
```

## Common Tasks

### Adding a New Feature

1. Design the feature and discuss in an issue
2. Write tests first (TDD)
3. Implement the feature
4. Update documentation
5. Submit PR

### Fixing a Bug

1. Create a test that reproduces the bug
2. Fix the bug
3. Verify test passes
4. Add regression test
5. Submit PR

### Improving Performance

1. Add benchmarks
2. Profile the code
3. Make improvements
4. Verify benchmarks show improvement
5. Submit PR with benchmark results

## Need Help?

- Open an issue for bugs or feature requests
- Join discussions in existing issues
- Ask questions in pull requests
- Check the README for usage examples

## Code of Conduct

- Be respectful and inclusive
- Welcome newcomers
- Provide constructive feedback
- Focus on the code, not the person

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
