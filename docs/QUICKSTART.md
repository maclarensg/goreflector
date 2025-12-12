# Quick Start Guide

Get goreflector up and running in 5 minutes.

## Installation

### Option 1: Build from Source

```bash
git clone https://github.com/gavinyap/goreflector
cd goreflector
go build -o goreflector .
```

### Option 2: Using devbox (Recommended for Development)

```bash
git clone https://github.com/gavinyap/goreflector
cd goreflector
devbox shell
task build
```

## Basic Usage

### Start a Simple Proxy

```bash
# Proxy localhost:8080 to httpbin.org
./goreflector -p 8080 https://httpbin.org

# In another terminal, test it
curl http://localhost:8080/get
```

### Proxy to Your API

```bash
# Proxy to your backend
./goreflector -p 8080 https://api.yoursite.com

# Now your API is available at localhost:8080
curl http://localhost:8080/api/endpoint
```

## Common Scenarios

### Development Proxy

Proxy your local development traffic to a staging API:

```bash
./goreflector -p 8080 -v https://staging-api.example.com
```

Then configure your app to use `http://localhost:8080` instead of the staging URL.

### CORS Workaround

When developing frontend apps, CORS can be problematic. Use goreflector:

```bash
# Start proxy
./goreflector -p 8080 https://api-with-cors-issues.com

# Your frontend can now call
fetch('http://localhost:8080/api/data')
```

### API Testing

Test an API without modifying your code:

```bash
# Original API
curl https://production-api.com/users

# Through proxy (to test/debug)
./goreflector -p 8080 -v https://production-api.com
curl http://localhost:8080/users
```

### Local Development with Remote Backend

```bash
# Start your local frontend on 3000
npm run dev

# Proxy API calls to remote backend
./goreflector -p 8080 https://remote-api.example.com

# Configure frontend to use http://localhost:8080
```

### Custom Headers (Host Override, Authentication)

Override or inject custom headers for advanced scenarios:

```bash
# Override Host header (useful for virtual host testing, SNI bypass)
./goreflector -p 8080 -H "Host: example.com" https://1.2.3.4/

# Add authentication headers
./goreflector -p 8080 -H "Authorization: Bearer secret123" https://api.example.com

# Multiple custom headers
./goreflector -p 8080 \
  -H "Host: api.example.com" \
  -H "Authorization: Bearer token123" \
  -H "X-API-Key: key456" \
  https://192.168.1.100/

# Override User-Agent for testing
./goreflector -p 8080 -H "User-Agent: CustomBot/1.0" https://api.example.com
```

**Use Cases:**
- **Host header override**: Test virtual hosts by connecting to an IP but sending a different Host header
- **SNI bypass**: Connect to IP addresses while presenting the correct hostname
- **Custom authentication**: Add API keys, Bearer tokens, or other auth headers
- **Header spoofing**: Test how backends handle different User-Agents or custom headers
- **Load balancer testing**: Hit specific backend servers by IP with correct Host header

## Command Line Options

```bash
# Basic
./goreflector -p 8080 https://example.com

# With custom headers
./goreflector -p 8080 -H "Host: example.com" https://1.2.3.4/

# With custom timeout (60 seconds)
./goreflector -p 8080 -t 60 https://slow-api.com

# With verbose logging
./goreflector -p 8080 -v https://example.com

# All options combined
./goreflector -p 9000 -t 45 -v \
  -H "Host: api.example.com" \
  -H "Authorization: Bearer token" \
  https://192.168.1.100/
```

## Real-World Examples

### Example 1: GitHub API Proxy

```bash
./goreflector -p 8080 https://api.github.com

# Test it
curl http://localhost:8080/users/octocat
```

### Example 2: Proxy with Base Path

```bash
# Proxy to API v2 endpoint
./goreflector -p 8080 https://api.example.com/v2

# All requests will be prefixed with /v2
curl http://localhost:8080/users
# → Proxies to https://api.example.com/v2/users
```

### Example 3: Multiple Proxies

Run multiple proxies for different services:

```bash
# Terminal 1: Users service
./goreflector -p 8081 https://users-api.example.com

# Terminal 2: Products service
./goreflector -p 8082 https://products-api.example.com

# Terminal 3: Orders service
./goreflector -p 8083 https://orders-api.example.com
```

## Verification

### Check Headers

```bash
curl -v http://localhost:8080/test 2>&1 | grep X-Forwarded
```

You should see:
```
< X-Forwarded-For: ::1
< X-Forwarded-Host: localhost:8080
< X-Forwarded-Proto: http
```

### Test POST Request

```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{"key":"value"}' \
  http://localhost:8080/api/data
```

### Test with Authentication

```bash
curl -H "Authorization: Bearer YOUR_TOKEN" \
  http://localhost:8080/api/protected
```

## Troubleshooting

### Port Already in Use

```bash
# Error: listen tcp :8080: bind: address already in use

# Solution 1: Use different port
./goreflector -p 8081 https://example.com

# Solution 2: Find and kill process
lsof -ti:8080 | xargs kill -9
```

### Connection Refused

```bash
# Error: dial tcp: connection refused

# Check if target is reachable
curl https://your-target-url.com

# Check DNS
nslookup your-target-url.com

# Try with verbose logging
./goreflector -p 8080 -v https://your-target-url.com
```

### Timeout Errors

```bash
# Increase timeout to 60 seconds
./goreflector -p 8080 -t 60 https://slow-backend.com
```

## Next Steps

- Read the [README](../README.md) for detailed features
- Check [ARCHITECTURE.md](ARCHITECTURE.md) to understand how it works
- See [DEPLOYMENT.md](DEPLOYMENT.md) for production deployment
- Review [CONTRIBUTING.md](../CONTRIBUTING.md) to contribute

## Testing with Real APIs

### httpbin.org (Great for Testing)

```bash
./goreflector -p 8080 https://httpbin.org

# Test various endpoints
curl http://localhost:8080/get
curl http://localhost:8080/post -d "test=data"
curl http://localhost:8080/status/404
curl http://localhost:8080/delay/2
```

### JSONPlaceholder (Fake API)

```bash
./goreflector -p 8080 https://jsonplaceholder.typicode.com

# Test endpoints
curl http://localhost:8080/posts
curl http://localhost:8080/users/1
curl -X POST http://localhost:8080/posts \
  -d '{"title":"test","body":"content","userId":1}'
```

## Performance Tips

### For High Traffic

```bash
# Use longer timeout for slow backends
./goreflector -p 8080 -t 60 https://slow-api.com
```

### For Development

```bash
# Enable verbose logging to debug
./goreflector -p 8080 -v https://api.example.com
```

### For Production

See the [DEPLOYMENT.md](DEPLOYMENT.md) guide for:
- Systemd service setup
- Docker deployment
- Kubernetes deployment
- Monitoring and logging

## Common Patterns

### Pattern 1: Local + Remote Services

```bash
# Backend API (remote)
./goreflector -p 8080 https://api.production.com

# Local frontend development
cd frontend && npm run dev
# Configure to use http://localhost:8080
```

### Pattern 2: API Gateway Simulation

```bash
# Run multiple proxies
./goreflector -p 8001 https://service1.com &
./goreflector -p 8002 https://service2.com &
./goreflector -p 8003 https://service3.com &

# Access different services on different ports
```

### Pattern 3: Testing/Debugging

```bash
# Verbose mode to see all requests
./goreflector -p 8080 -v https://api.example.com

# Watch the logs while making requests
curl http://localhost:8080/test
```

## Getting Help

- Check the logs with `-v` flag
- Review error messages
- Test target URL directly first
- Check firewall/network settings
- See [Troubleshooting](DEPLOYMENT.md#troubleshooting) guide

## Summary

goreflector is simple:

1. **Install**: `go build`
2. **Run**: `./goreflector -p 8080 https://target.com`
3. **Use**: `curl http://localhost:8080/endpoint`

That's it! Your proxy is running.
