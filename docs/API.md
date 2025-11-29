# API Documentation

This document describes the programmatic API of goreflector for developers who want to use it as a library or understand its internals.

## Package: main

### Types

#### ProxyConfig

Configuration for creating a proxy instance.

```go
type ProxyConfig struct {
    ListenAddr string        // Address to listen on (e.g., ":8080")
    TargetURL  *url.URL      // Target backend URL
    Timeout    time.Duration // Request timeout
}
```

**Example:**
```go
config := ProxyConfig{
    ListenAddr: ":8080",
    TargetURL:  mustParseURL("https://api.example.com"),
    Timeout:    30 * time.Second,
}
```

#### Proxy

The main proxy handler that implements `http.Handler`.

```go
type Proxy struct {
    config     ProxyConfig
    httpClient *http.Client
    logger     *log.Logger
}
```

**Fields:**
- `config`: Proxy configuration
- `httpClient`: HTTP client for making requests to backend
- `logger`: Logger for request/error logging

### Functions

#### NewProxy

Creates a new proxy instance with the given configuration and logger.

```go
func NewProxy(config ProxyConfig, logger *log.Logger) (*Proxy, error)
```

**Parameters:**
- `config`: ProxyConfig - Configuration for the proxy
- `logger`: *log.Logger - Logger instance (can be nil for default logger)

**Returns:**
- `*Proxy`: Configured proxy instance
- `error`: Error if configuration is invalid

**Errors:**
- Returns error if TargetURL is nil
- Returns error if ListenAddr is empty

**Example:**
```go
logger := log.New(os.Stdout, "[PROXY] ", log.LstdFlags)
config := ProxyConfig{
    ListenAddr: ":8080",
    TargetURL:  mustParseURL("https://api.example.com"),
    Timeout:    30 * time.Second,
}

proxy, err := NewProxy(config, logger)
if err != nil {
    log.Fatal(err)
}
```

**Default Behavior:**
- If timeout is 0, defaults to 30 seconds
- If logger is nil, uses default logger
- Sets up HTTP client with connection pooling
- Configures TLS with minimum version 1.2

#### ServeHTTP

Handles incoming HTTP requests and forwards them to the target backend.

```go
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request)
```

**Parameters:**
- `w`: http.ResponseWriter - Response writer
- `r`: *http.Request - Incoming request

**Behavior:**
1. Builds target URL from request path
2. Creates new request to backend
3. Copies and modifies headers
4. Adds X-Forwarded-* headers
5. Forwards request to backend
6. Streams response back to client

**Error Responses:**
- 500 Internal Server Error: Failed to create proxy request
- 502 Bad Gateway: Failed to proxy request (backend error)

**Example:**
```go
http.Handle("/", proxy)
log.Fatal(http.ListenAndServe(":8080", nil))
```

#### Start

Starts the HTTP server with the proxy handler.

```go
func (p *Proxy) Start() error
```

**Returns:**
- `error`: Error if server fails to start

**Example:**
```go
if err := proxy.Start(); err != nil {
    log.Fatal(err)
}
```

**Server Configuration:**
- Read timeout: 15 seconds
- Write timeout: 15 seconds
- Idle timeout: 60 seconds

### Helper Functions

#### buildTargetURL

Constructs the target URL from the proxy configuration and incoming request.

```go
func (p *Proxy) buildTargetURL(r *http.Request) *url.URL
```

**Parameters:**
- `r`: *http.Request - Incoming request

**Returns:**
- `*url.URL`: Complete target URL with path and query

**Behavior:**
- Uses scheme and host from target URL
- Appends request path
- Preserves query parameters
- Handles base paths in target URL

**Example:**
```go
// Target: https://api.example.com/v1
// Request: /users?page=1
// Result: https://api.example.com/v1/users?page=1
```

#### copyHeaders

Copies request headers from source to destination, filtering hop-by-hop headers.

```go
func (p *Proxy) copyHeaders(src *http.Request, dst *http.Request)
```

**Parameters:**
- `src`: *http.Request - Source request (from client)
- `dst`: *http.Request - Destination request (to backend)

**Behavior:**
- Copies all headers except hop-by-hop headers
- Modifies Host header to target hostname
- Preserves header values and ordering

**Filtered Headers:**
- Connection
- Keep-Alive
- Proxy-Authenticate
- Proxy-Authorization
- Te
- Trailers
- Transfer-Encoding
- Upgrade

#### addForwardedHeaders

Adds X-Forwarded-* headers to the proxied request.

```go
func (p *Proxy) addForwardedHeaders(src *http.Request, dst *http.Request)
```

**Parameters:**
- `src`: *http.Request - Original request
- `dst`: *http.Request - Proxied request

**Headers Added:**
- `X-Forwarded-For`: Client IP address (appends to existing chain)
- `X-Forwarded-Host`: Original Host header
- `X-Forwarded-Proto`: Original protocol (http/https)

**Example:**
```go
// Incoming headers:
// Host: localhost:8080
// X-Forwarded-For: 10.0.0.1

// Added headers:
// X-Forwarded-For: 10.0.0.1, 192.168.1.100
// X-Forwarded-Host: localhost:8080
// X-Forwarded-Proto: http
```

#### shouldSkipHeader

Determines if a header should be skipped when copying.

```go
func shouldSkipHeader(header string) bool
```

**Parameters:**
- `header`: string - Header name (case-insensitive)

**Returns:**
- `bool`: true if header should be skipped

**Skipped Headers:**
- Hop-by-hop headers per HTTP/1.1 spec
- Connection-related headers

#### getClientIP

Extracts the client IP address from the request.

```go
func getClientIP(r *http.Request) string
```

**Parameters:**
- `r`: *http.Request - Incoming request

**Returns:**
- `string`: Client IP address

**Logic:**
1. Check X-Forwarded-For header (use first IP)
2. Check X-Real-IP header
3. Fall back to RemoteAddr
4. Strip port if present

**Example:**
```go
ip := getClientIP(req)
// Returns: "192.168.1.100"
```

## CLI Functions

### parseFlags

Parses command-line flags and arguments.

```go
func parseFlags() (*Options, error)
```

**Returns:**
- `*Options`: Parsed options
- `error`: Error if required arguments missing

**Flags:**
- `-p, --port`: Listen port (default: 8080)
- `-t, --timeout`: Timeout in seconds (default: 30)
- `-v, --verbose`: Enable verbose logging
- `--version`: Show version

**Arguments:**
- `target-url`: Required target backend URL

### validateOptions

Validates parsed options.

```go
func validateOptions(opts *Options) error
```

**Parameters:**
- `opts`: *Options - Options to validate

**Returns:**
- `error`: Validation error or nil

**Validations:**
- Port: 1-65535
- Timeout: > 0
- Target URL: valid HTTP/HTTPS URL

## HTTP Client Configuration

The proxy uses a customized HTTP client with optimal settings:

```go
transport := &http.Transport{
    DialContext: (&net.Dialer{
        Timeout:   10 * time.Second,  // Connection timeout
        KeepAlive: 30 * time.Second,  // Keep-alive probe interval
    }).DialContext,
    TLSClientConfig: &tls.Config{
        MinVersion: tls.VersionTLS12,  // Minimum TLS 1.2
    },
    MaxIdleConns:          100,            // Connection pool size
    IdleConnTimeout:       90 * time.Second,
    TLSHandshakeTimeout:   10 * time.Second,
    ExpectContinueTimeout: 1 * time.Second,
}

httpClient := &http.Client{
    Transport: transport,
    Timeout:   30 * time.Second,  // Overall request timeout
    CheckRedirect: func(req *http.Request, via []*http.Request) error {
        return http.ErrUseLastResponse  // Don't follow redirects
    },
}
```

## Usage as Library

### Basic Example

```go
package main

import (
    "log"
    "net/http"
    "net/url"
    "time"
)

func main() {
    // Parse target URL
    targetURL, err := url.Parse("https://api.example.com")
    if err != nil {
        log.Fatal(err)
    }

    // Create logger
    logger := log.New(os.Stdout, "[PROXY] ", log.LstdFlags)

    // Configure proxy
    config := ProxyConfig{
        ListenAddr: ":8080",
        TargetURL:  targetURL,
        Timeout:    30 * time.Second,
    }

    // Create proxy
    proxy, err := NewProxy(config, logger)
    if err != nil {
        log.Fatal(err)
    }

    // Start server
    log.Println("Starting proxy on :8080")
    if err := proxy.Start(); err != nil {
        log.Fatal(err)
    }
}
```

### Custom HTTP Handler

```go
// Use proxy as http.Handler
mux := http.NewServeMux()
mux.Handle("/api/", proxy)
mux.HandleFunc("/health", healthCheckHandler)

server := &http.Server{
    Addr:    ":8080",
    Handler: mux,
}
server.ListenAndServe()
```

### With Middleware

```go
// Add logging middleware
loggingProxy := loggingMiddleware(proxy)

http.Handle("/", loggingProxy)
http.ListenAndServe(":8080", nil)

func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        log.Printf("%s %s", r.Method, r.URL.Path)
        next.ServeHTTP(w, r)
    })
}
```

### Multiple Targets

```go
// Different proxies for different paths
mux := http.NewServeMux()

usersProxy, _ := NewProxy(ProxyConfig{
    ListenAddr: ":8080",
    TargetURL:  mustParseURL("https://users-api.example.com"),
}, logger)

productsProxy, _ := NewProxy(ProxyConfig{
    ListenAddr: ":8080",
    TargetURL:  mustParseURL("https://products-api.example.com"),
}, logger)

mux.Handle("/users/", usersProxy)
mux.Handle("/products/", productsProxy)

http.ListenAndServe(":8080", mux)
```

## Error Handling

### Error Types

1. **Configuration Errors**
   ```go
   proxy, err := NewProxy(config, logger)
   if err != nil {
       // Handle configuration error
       log.Fatal(err)
   }
   ```

2. **Runtime Errors**
   ```go
   // Logged, not returned
   // Check logs for:
   // - "Error creating proxy request"
   // - "Error proxying request"
   // - "Error copying response body"
   ```

## Performance Considerations

### Connection Pooling

The proxy maintains a connection pool to the backend:
- Max idle connections: 100
- Idle timeout: 90 seconds
- Keep-alive enabled

### Streaming

Requests and responses are streamed without buffering:
- Memory efficient for large payloads
- Lower latency
- Suitable for file uploads/downloads

### Timeouts

Multiple timeout layers:
- Connection timeout: 10s
- TLS handshake: 10s
- Overall request: Configurable (default 30s)
- Server read/write: 15s each

## Thread Safety

All public functions are thread-safe:
- `NewProxy`: Safe to call from multiple goroutines
- `ServeHTTP`: Handles concurrent requests
- `Start`: Safe to call once per instance

The HTTP client is shared and thread-safe.

## Testing Utilities

### Mock Backend

```go
backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("test response"))
}))
defer backend.Close()

proxy, _ := NewProxy(ProxyConfig{
    TargetURL: mustParseURL(backend.URL),
}, nil)
```

### Request Testing

```go
req := httptest.NewRequest("GET", "http://localhost/test", nil)
w := httptest.NewRecorder()

proxy.ServeHTTP(w, req)

resp := w.Result()
body, _ := io.ReadAll(resp.Body)
```

## See Also

- [Architecture Documentation](ARCHITECTURE.md)
- [Quick Start Guide](QUICKSTART.md)
- [Deployment Guide](DEPLOYMENT.md)
