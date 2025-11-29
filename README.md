# goreflector

A lightweight, production-ready HTTP reverse proxy written in Go. Similar to `kubectl proxy` but works with any HTTP/HTTPS target.

## Documentation

- **[Quick Start Guide](docs/QUICKSTART.md)** - Get started in 5 minutes
- **[Architecture](docs/ARCHITECTURE.md)** - Detailed technical architecture
- **[Deployment Guide](docs/DEPLOYMENT.md)** - Production deployment instructions
- **[Contributing](CONTRIBUTING.md)** - How to contribute to the project
- **[Changelog](CHANGELOG.md)** - Version history and changes

## Features

- ✅ HTTP reverse proxy for HTTP and HTTPS targets
- ✅ Automatic X-Forwarded-* header injection
- ✅ Host header modification for proper routing
- ✅ Full path and query string preservation
- ✅ All HTTP methods supported (GET, POST, PUT, DELETE, PATCH, etc.)
- ✅ Request/response body streaming
- ✅ Configurable timeouts
- ✅ Verbose logging mode
- ✅ Production-ready with comprehensive tests (67.4% coverage)
- ✅ Security scanned with gosec (0 issues)
- ✅ Race-condition tested

## Installation

### Using devbox (recommended)

```bash
devbox shell
task build
```

### Using go install

```bash
go install github.com/gavinyap/goreflector@latest
```

### Build from source

```bash
git clone https://github.com/gavinyap/goreflector
cd goreflector
go build -o goreflector .
```

## Usage

### Basic usage

```bash
./goreflector -p 8080 https://api.example.com
```

This starts a proxy server on `http://localhost:8080` that forwards all requests to `https://api.example.com`.

### With verbose logging

```bash
./goreflector -p 8080 -v https://api.example.com
```

### Custom timeout

```bash
./goreflector -p 8080 -t 60 https://api.example.com
```

### All options

```
Usage: goreflector [options] <target-url>

Options:
  -p, --port int       Port to listen on (default: 8080)
  -t, --timeout int    Request timeout in seconds (default: 30)
  -v, --verbose        Verbose logging
  --version            Show version

Example:
  goreflector -p 8080 https://example.com
```

## Examples

### Proxy to an API server

```bash
# Start the proxy
./goreflector -p 8080 https://api.github.com

# Make requests through the proxy
curl http://localhost:8080/users/octocat
```

### Proxy with base path

```bash
# Proxy to https://api.example.com/v1/
./goreflector -p 8080 https://api.example.com/v1

# Request to http://localhost:8080/users maps to https://api.example.com/v1/users
curl http://localhost:8080/users
```

### POST requests with body

```bash
./goreflector -p 8080 https://httpbin.org

curl -X POST http://localhost:8080/post \
  -H "Content-Type: application/json" \
  -d '{"key":"value"}'
```

## Header Handling

goreflector automatically:

1. **Preserves** all client headers (except hop-by-hop headers like Connection, Keep-Alive, etc.)
2. **Adds** X-Forwarded-* headers:
   - `X-Forwarded-For`: Client IP address
   - `X-Forwarded-Host`: Original Host header
   - `X-Forwarded-Proto`: Original protocol (http/https)
3. **Modifies** the `Host` header to match the target URL for proper routing

## Development

### Prerequisites

- Go 1.21+
- [devbox](https://www.jetify.com/devbox) (optional, for isolated environment)
- [Task](https://taskfile.dev/) (included in devbox)

### Setup

```bash
# Using devbox
devbox shell

# Or install dependencies manually
go mod download
```

### Available Tasks

```bash
task --list
```

Common tasks:

```bash
task build          # Build the binary
task test           # Run all tests
task test:coverage  # Run tests with coverage report
task lint           # Run golangci-lint
task gosec          # Run security scanner
task ci             # Run full CI pipeline
task clean          # Clean build artifacts
```

### Running Tests

```bash
# Run all tests
task test

# Run tests with coverage
task test:coverage

# Run only unit tests
task test:unit

# Run only integration tests
task test:integration

# Run CI pipeline (format, vet, lint, gosec, test)
task ci
```

### Code Quality

This project maintains high code quality standards:

- **Test Coverage**: 67.4% (all core logic covered)
- **Linting**: golangci-lint with strict rules
- **Security**: gosec scanning (0 issues)
- **Race Detection**: All tests pass with `-race`
- **Code Format**: gofmt + goimports

## Architecture

```
Client Request
     |
     v
goreflector (localhost:8080)
     |
     |- Parse & validate request
     |- Add X-Forwarded-* headers
     |- Modify Host header
     |- Forward to target
     v
Target Server (https://example.com)
     |
     |- Process request
     |- Return response
     v
goreflector
     |
     |- Stream response back
     v
Client
```

## Testing

The project includes comprehensive tests:

- **Unit tests**: Test individual functions and components
- **Integration tests**: Test end-to-end proxy behavior
- **Edge case tests**: Test error handling, timeouts, large bodies, etc.

Test coverage by component:
- `NewProxy`: 100%
- `ServeHTTP`: 95.5%
- `buildTargetURL`: 100%
- `copyHeaders`: 100%
- `addForwardedHeaders`: 90.9%
- `shouldSkipHeader`: 100%
- `getClientIP`: 100%
- `validateOptions`: 100%

## Performance

goreflector is designed for high performance:

- Efficient HTTP connection pooling
- Streaming request/response bodies (no buffering)
- Configurable timeouts
- Keep-alive connections
- Minimal memory overhead

## Security

Security best practices:

- TLS 1.2+ for HTTPS targets
- No sensitive data logging (even in verbose mode)
- Input validation on all CLI arguments
- Gosec security scanning (0 issues)
- Proper error handling

## Limitations

- Listen side is HTTP only (not HTTPS)
- No authentication/authorization built-in
- No request/response modification beyond headers
- Single target per instance

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Write tests for your changes
4. Ensure `task ci` passes
5. Submit a pull request

## License

MIT License - see LICENSE file for details

## Credits

Built with:
- [Go](https://golang.org/)
- [Task](https://taskfile.dev/)
- [devbox](https://www.jetify.com/devbox)
- [golangci-lint](https://golangci-lint.run/)
- [gosec](https://github.com/securego/gosec)
