# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2025-11-28

### Added
- Initial release of goreflector
- HTTP reverse proxy functionality
- Support for HTTP and HTTPS targets
- Automatic X-Forwarded-* header injection
  - X-Forwarded-For: Client IP address
  - X-Forwarded-Host: Original host header
  - X-Forwarded-Proto: Original protocol (http/https)
- Host header modification for proper routing
- Full path and query string preservation
- Support for all HTTP methods (GET, POST, PUT, DELETE, PATCH, etc.)
- Request and response body streaming
- Configurable timeout option (`-t`, `--timeout`)
- Verbose logging mode (`-v`, `--verbose`)
- Custom port selection (`-p`, `--port`)
- Version flag (`--version`)
- Comprehensive test suite with 67.4% coverage
- Integration tests for end-to-end scenarios
- Race condition testing
- Security scanning with gosec
- CI/CD pipeline with Taskfile
- Complete documentation (README, CONTRIBUTING)
- Development environment with devbox
- Code quality tools:
  - golangci-lint for linting
  - gosec for security scanning
  - gofmt for formatting
  - Race detector for concurrency issues

### Security
- TLS 1.2+ minimum for HTTPS targets
- Input validation on all CLI arguments
- Proper error handling throughout
- No sensitive data logging
- Gosec security scan: 0 issues

### Performance
- Efficient HTTP connection pooling
- Streaming request/response bodies (no buffering)
- Keep-alive connection support
- Configurable timeouts
- Minimal memory overhead

### Testing
- 53 comprehensive test cases
- Unit tests for all core functions
- Integration tests for end-to-end scenarios
- Edge case testing (timeouts, errors, large bodies, redirects)
- 100% coverage on core proxy logic functions:
  - NewProxy: 100%
  - buildTargetURL: 100%
  - copyHeaders: 100%
  - shouldSkipHeader: 100%
  - getClientIP: 100%
  - validateOptions: 100%
- High coverage on HTTP handling:
  - ServeHTTP: 95.5%
  - addForwardedHeaders: 90.9%

## [Unreleased]

## [1.1.0] - 2025-12-12

### Added
- **Custom header injection** via `-H` flag for advanced use cases
  - Support for multiple `-H` flags to inject multiple headers
  - Override any header including Host, Authorization, User-Agent, etc.
  - Special handling for Host header (uses `dst.Host` for proper HTTP behavior)
  - Case-insensitive Host header matching
  - Custom headers override original request headers
  - Format validation with clear error messages
- Enhanced CLI help with custom header usage examples
- 13 new comprehensive unit tests for header functionality:
  - `TestParseHeaders` with 8 sub-tests for various header formats
  - `TestParseFlagsWithHeaders` and `TestParseFlagsWithMultipleHeaders`
  - 5 integration tests for custom header injection scenarios
  - Case-insensitive Host header testing
  - Header override behavior verification
- Improved test coverage from 67.4% to 69.2%
- Full backward compatibility maintained

### Changed
- Updated documentation with custom header examples:
  - README.md with usage examples and feature list
  - QUICKSTART.md with real-world scenarios
  - Enhanced CLI help text

### Use Cases
- **Host header override**: Test virtual hosts, SNI bypass, connect to IPs with custom Host
- **Custom authentication**: Inject Authorization, X-API-Key, or Bearer tokens
- **Header spoofing**: Test with different User-Agents or custom headers
- **Load balancer testing**: Hit specific backends by IP with correct Host header
- **Security testing**: Advanced penetration testing and security assessments

### Technical Details
- Added `Headers []string` field to `Options` struct
- Implemented `headerFlags` type for multi-value flag support
- Added `parseHeaders()` function with format validation
- Extended `ProxyConfig` with `CustomHeaders map[string]string`
- Modified `copyHeaders()` to apply custom headers after original headers
- Total changes: 4 files, 379 lines added

### Planned Features
- HTTPS support on listen side (optional)
- Request/response logging to file
- Metrics and health check endpoints
- Rate limiting
- Circuit breaker pattern
- Load balancing to multiple targets
- WebSocket support
- Authentication/authorization options

---

## Version History

### [1.1.0] - 2025-12-12
Feature release adding custom header injection via `-H` flag for advanced proxy scenarios including Host header override, authentication, and security testing.

### [1.0.0] - 2025-11-28
First production-ready release with full feature set, comprehensive testing, and documentation.
