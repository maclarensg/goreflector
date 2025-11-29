# Architecture Documentation

## Overview

goreflector is a lightweight HTTP reverse proxy built in Go. It acts as an intermediary between clients and backend servers, forwarding requests while adding proxy-specific headers and modifying the Host header for proper routing.

## System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                          Client                                  │
│                    (curl, browser, etc.)                         │
└────────────────┬────────────────────────────────────────────────┘
                 │
                 │ HTTP Request
                 │ GET /api/users
                 │ Host: localhost:8080
                 │
                 ▼
┌─────────────────────────────────────────────────────────────────┐
│                       goreflector                                │
│                    (localhost:8080)                              │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ 1. Request Reception                                      │  │
│  │    - Accept incoming HTTP request                        │  │
│  │    - Parse method, path, headers, body                   │  │
│  └──────────────────────────────────────────────────────────┘  │
│                           │                                      │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ 2. URL Construction                                       │  │
│  │    - Build target URL from config + request path         │  │
│  │    - Preserve query strings                              │  │
│  └──────────────────────────────────────────────────────────┘  │
│                           │                                      │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ 3. Header Processing                                      │  │
│  │    - Copy original headers (preserve all)                │  │
│  │    - Skip hop-by-hop headers                             │  │
│  │    - Modify Host header to target                        │  │
│  │    - Add X-Forwarded-For (client IP)                     │  │
│  │    - Add X-Forwarded-Host (original host)                │  │
│  │    - Add X-Forwarded-Proto (http/https)                  │  │
│  └──────────────────────────────────────────────────────────┘  │
│                           │                                      │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ 4. Request Forwarding                                     │  │
│  │    - Create new HTTP request to target                   │  │
│  │    - Stream request body (no buffering)                  │  │
│  │    - Apply timeout settings                              │  │
│  └──────────────────────────────────────────────────────────┘  │
└────────────────┬────────────────────────────────────────────────┘
                 │
                 │ HTTP Request
                 │ GET /api/users
                 │ Host: target.example.com
                 │ X-Forwarded-For: 192.168.1.100
                 │ X-Forwarded-Host: localhost:8080
                 │ X-Forwarded-Proto: http
                 │
                 ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Target Server                               │
│                 (https://target.example.com)                     │
│                                                                  │
│  - Process request                                               │
│  - Generate response                                             │
│  - Send back to proxy                                            │
└────────────────┬────────────────────────────────────────────────┘
                 │
                 │ HTTP Response
                 │ Status: 200 OK
                 │ Headers + Body
                 │
                 ▼
┌─────────────────────────────────────────────────────────────────┐
│                       goreflector                                │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ 5. Response Processing                                    │  │
│  │    - Receive response from target                        │  │
│  │    - Copy all response headers                           │  │
│  │    - Preserve status code                                │  │
│  └──────────────────────────────────────────────────────────┘  │
│                           │                                      │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ 6. Response Streaming                                     │  │
│  │    - Stream response body to client (no buffering)       │  │
│  │    - Handle errors gracefully                            │  │
│  └──────────────────────────────────────────────────────────┘  │
└────────────────┬────────────────────────────────────────────────┘
                 │
                 │ HTTP Response
                 │ Status: 200 OK
                 │ Headers + Body
                 │
                 ▼
┌─────────────────────────────────────────────────────────────────┐
│                          Client                                  │
│                   (receives response)                            │
└─────────────────────────────────────────────────────────────────┘
```

## Component Architecture

### Main Components

#### 1. CLI Layer (`main.go`)

**Responsibilities:**
- Parse command-line arguments
- Validate configuration
- Initialize proxy
- Start HTTP server

**Key Functions:**
- `parseFlags()`: Parse CLI flags and arguments
- `validateOptions()`: Validate port, timeout, URL format
- `main()`: Entry point, orchestrates startup

**Configuration:**
```go
type Options struct {
    Port       int    // Listen port (1-65535)
    TargetURL  string // Target backend URL
    Timeout    int    // Request timeout in seconds
    Verbose    bool   // Enable verbose logging
}
```

#### 2. Proxy Layer (`proxy.go`)

**Responsibilities:**
- HTTP request/response handling
- Header manipulation
- URL construction
- Connection management

**Key Structures:**
```go
type Proxy struct {
    config     ProxyConfig
    httpClient *http.Client
    logger     *log.Logger
}

type ProxyConfig struct {
    ListenAddr string
    TargetURL  *url.URL
    Timeout    time.Duration
}
```

**Key Functions:**
- `NewProxy()`: Create and configure proxy instance
- `ServeHTTP()`: Handle incoming HTTP requests (implements http.Handler)
- `buildTargetURL()`: Construct target URL from request
- `copyHeaders()`: Copy and filter request headers
- `addForwardedHeaders()`: Add X-Forwarded-* headers
- `Start()`: Start HTTP server

## Data Flow

### Request Flow

```
1. Client Request
   ↓
2. HTTP Server (net/http)
   ↓
3. ServeHTTP (proxy.go:71)
   ├─→ buildTargetURL() - Construct destination
   ├─→ http.NewRequest() - Create backend request
   ├─→ copyHeaders() - Copy client headers
   ├─→ addForwardedHeaders() - Add proxy headers
   ├─→ httpClient.Do() - Forward to backend
   ↓
4. Backend Server
   ↓
5. Response Processing
   ├─→ Copy response headers
   ├─→ Set response status
   ├─→ Stream response body
   ↓
6. Client Response
```

### Header Processing Pipeline

```
Original Request Headers
   ↓
┌──────────────────────────────┐
│ Filter hop-by-hop headers    │
│ (Connection, Keep-Alive, etc)│
└──────────────────────────────┘
   ↓
┌──────────────────────────────┐
│ Copy application headers     │
│ (User-Agent, Accept, etc)    │
└──────────────────────────────┘
   ↓
┌──────────────────────────────┐
│ Modify Host header           │
│ → target.example.com         │
└──────────────────────────────┘
   ↓
┌──────────────────────────────┐
│ Add X-Forwarded-For          │
│ → client IP or append to     │
│   existing XFF chain         │
└──────────────────────────────┘
   ↓
┌──────────────────────────────┐
│ Add X-Forwarded-Host         │
│ → original Host header       │
└──────────────────────────────┘
   ↓
┌──────────────────────────────┐
│ Add X-Forwarded-Proto        │
│ → http or https              │
└──────────────────────────────┘
   ↓
Final Request Headers
```

## Connection Management

### HTTP Client Configuration

```go
transport := &http.Transport{
    DialContext: (&net.Dialer{
        Timeout:   10 * time.Second,
        KeepAlive: 30 * time.Second,
    }).DialContext,
    TLSClientConfig:       &tls.Config{MinVersion: tls.VersionTLS12},
    MaxIdleConns:          100,
    IdleConnTimeout:       90 * time.Second,
    TLSHandshakeTimeout:   10 * time.Second,
    ExpectContinueTimeout: 1 * time.Second,
}
```

**Features:**
- Connection pooling (up to 100 idle connections)
- Keep-alive for performance
- TLS 1.2+ minimum security
- Configurable timeouts
- No automatic redirect following

### HTTP Server Configuration

```go
server := &http.Server{
    Addr:         ":8080",
    Handler:      proxy,
    ReadTimeout:  15 * time.Second,
    WriteTimeout: 15 * time.Second,
    IdleTimeout:  60 * time.Second,
}
```

## Error Handling

### Error Types and Responses

1. **Request Creation Error**
   - Status: 500 Internal Server Error
   - Logged: "Error creating proxy request"
   - Cause: Invalid request method or malformed URL

2. **Backend Connection Error**
   - Status: 502 Bad Gateway
   - Logged: "Error proxying request"
   - Cause: Backend unreachable, timeout, DNS failure

3. **Response Streaming Error**
   - Logged: "Error copying response body"
   - Action: Log only (connection may be broken)

### Error Recovery

- Errors are logged with context
- HTTP error responses sent to client
- Resources properly cleaned up (deferred closes)
- No partial responses sent

## Security Considerations

### Input Validation

- Port range: 1-65535
- Timeout: Must be positive
- URL: Must be valid HTTP/HTTPS URL
- Scheme: Only http:// and https:// allowed

### TLS Configuration

```go
TLSClientConfig: &tls.Config{
    MinVersion: tls.VersionTLS12,  // No TLS 1.0/1.1
}
```

### Header Security

- Hop-by-hop headers stripped (prevents header injection)
- No modification of security headers (CSP, CORS, etc.)
- Proper X-Forwarded-* chain handling

### Resource Limits

- Connection pooling limits
- Timeouts on all operations
- Automatic cleanup (defer statements)

## Performance Characteristics

### Memory

- **Streaming**: No request/response buffering
- **Connection Pooling**: Reuse connections
- **Minimal Allocations**: Headers copied, not duplicated

### Latency

- **Overhead**: ~1-2ms per request
- **Connection Reuse**: Near-zero for keep-alive
- **DNS Caching**: Via Go runtime

### Throughput

- **Concurrent Requests**: Limited by Go scheduler
- **Bottlenecks**: Backend server, network bandwidth
- **Scaling**: Horizontal (multiple instances)

## Testing Architecture

### Test Layers

1. **Unit Tests** (`*_test.go`)
   - Test individual functions
   - Mock dependencies
   - Fast execution

2. **Integration Tests** (`integration_test.go`)
   - End-to-end scenarios
   - Real HTTP servers (httptest)
   - Verify complete flow

3. **Edge Case Tests** (`proxy_additional_test.go`)
   - Timeouts, errors
   - Large bodies, redirects
   - Concurrent requests

### Test Coverage Strategy

```
Core Logic Functions: 100%
├── buildTargetURL
├── copyHeaders
├── shouldSkipHeader
├── getClientIP
└── validateOptions

HTTP Handling: 90%+
├── ServeHTTP: 95.5%
└── addForwardedHeaders: 90.9%

Integration: Full coverage
└── End-to-end scenarios
```

## Future Enhancements

### Planned Features

1. **HTTPS Listen Support**
   - TLS termination
   - Certificate management
   - SNI support

2. **Advanced Features**
   - WebSocket proxying
   - Load balancing
   - Circuit breaker
   - Rate limiting

3. **Observability**
   - Metrics (Prometheus)
   - Distributed tracing
   - Access logs
   - Health checks

## Configuration Examples

### Basic Proxy

```bash
./goreflector -p 8080 https://api.example.com
```

### With Timeout

```bash
./goreflector -p 8080 -t 60 https://slow-api.example.com
```

### Verbose Logging

```bash
./goreflector -p 8080 -v https://api.example.com
# Logs: GET /path -> https://api.example.com/path
```

### Production Deployment

```bash
# Use systemd or similar process manager
# Set appropriate timeouts
# Enable verbose logging for debugging
# Monitor logs for errors

./goreflector \
  -p 8080 \
  -t 30 \
  -v \
  https://production-api.example.com
```
