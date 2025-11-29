# Deployment Guide

This guide covers deploying goreflector in various environments.

## Table of Contents

- [Quick Start](#quick-start)
- [Production Deployment](#production-deployment)
- [Container Deployment](#container-deployment)
- [Systemd Service](#systemd-service)
- [Monitoring](#monitoring)
- [Security Hardening](#security-hardening)
- [Troubleshooting](#troubleshooting)

## Quick Start

### Development/Testing

```bash
# Build
go build -o goreflector .

# Run
./goreflector -p 8080 -v https://api.example.com
```

### Using Pre-built Binary

```bash
# Download latest release
wget https://github.com/gavinyap/goreflector/releases/latest/download/goreflector-linux-amd64

# Make executable
chmod +x goreflector-linux-amd64

# Run
./goreflector-linux-amd64 -p 8080 https://api.example.com
```

## Production Deployment

### Prerequisites

- Linux server (Ubuntu 20.04+ recommended)
- Systemd (for service management)
- Network access to target backend
- Firewall rules configured

### Installation Steps

1. **Create dedicated user:**

```bash
sudo useradd -r -s /bin/false goreflector
```

2. **Install binary:**

```bash
sudo cp goreflector /usr/local/bin/
sudo chown root:root /usr/local/bin/goreflector
sudo chmod 755 /usr/local/bin/goreflector
```

3. **Create configuration directory:**

```bash
sudo mkdir -p /etc/goreflector
sudo chown goreflector:goreflector /etc/goreflector
```

4. **Create log directory:**

```bash
sudo mkdir -p /var/log/goreflector
sudo chown goreflector:goreflector /var/log/goreflector
```

### Environment Configuration

Create `/etc/goreflector/env`:

```bash
# Target backend URL
TARGET_URL=https://api.example.com

# Listen port
PORT=8080

# Request timeout (seconds)
TIMEOUT=30

# Enable verbose logging
VERBOSE=true
```

## Systemd Service

### Service File

Create `/etc/systemd/system/goreflector.service`:

```ini
[Unit]
Description=goreflector HTTP Reverse Proxy
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=goreflector
Group=goreflector

# Load environment variables
EnvironmentFile=/etc/goreflector/env

# Start command
ExecStart=/usr/local/bin/goreflector \
    -p ${PORT} \
    -t ${TIMEOUT} \
    -v \
    ${TARGET_URL}

# Restart policy
Restart=on-failure
RestartSec=5s

# Resource limits
LimitNOFILE=65536

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/log/goreflector

# Logging
StandardOutput=append:/var/log/goreflector/access.log
StandardError=append:/var/log/goreflector/error.log

[Install]
WantedBy=multi-user.target
```

### Service Management

```bash
# Reload systemd
sudo systemctl daemon-reload

# Enable service
sudo systemctl enable goreflector

# Start service
sudo systemctl start goreflector

# Check status
sudo systemctl status goreflector

# View logs
sudo journalctl -u goreflector -f

# Restart service
sudo systemctl restart goreflector

# Stop service
sudo systemctl stop goreflector
```

## Container Deployment

### Dockerfile

Create `Dockerfile`:

```dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY *.go ./

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o goreflector .

# Runtime image
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /build/goreflector .

# Create non-root user
RUN addgroup -g 1000 goreflector && \
    adduser -D -u 1000 -G goreflector goreflector

USER goreflector

EXPOSE 8080

ENTRYPOINT ["./goreflector"]
```

### Build and Run

```bash
# Build image
docker build -t goreflector:latest .

# Run container
docker run -d \
  --name goreflector \
  -p 8080:8080 \
  goreflector:latest \
  -p 8080 -v https://api.example.com

# View logs
docker logs -f goreflector

# Stop container
docker stop goreflector
```

### Docker Compose

Create `docker-compose.yml`:

```yaml
version: '3.8'

services:
  goreflector:
    build: .
    image: goreflector:latest
    container_name: goreflector
    restart: unless-stopped
    ports:
      - "8080:8080"
    environment:
      - TARGET_URL=https://api.example.com
    command: ["-p", "8080", "-t", "30", "-v", "${TARGET_URL}"]
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
```

Run with:

```bash
docker-compose up -d
```

## Kubernetes Deployment

### Deployment YAML

Create `k8s/deployment.yaml`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: goreflector
  labels:
    app: goreflector
spec:
  replicas: 3
  selector:
    matchLabels:
      app: goreflector
  template:
    metadata:
      labels:
        app: goreflector
    spec:
      containers:
      - name: goreflector
        image: goreflector:latest
        args:
          - "-p"
          - "8080"
          - "-t"
          - "30"
          - "-v"
          - "https://api.example.com"
        ports:
        - containerPort: 8080
          name: http
        resources:
          requests:
            memory: "64Mi"
            cpu: "100m"
          limits:
            memory: "128Mi"
            cpu: "200m"
        livenessProbe:
          httpGet:
            path: /
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: goreflector
spec:
  selector:
    app: goreflector
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8080
  type: LoadBalancer
```

Deploy:

```bash
kubectl apply -f k8s/deployment.yaml
kubectl get pods -l app=goreflector
kubectl logs -f -l app=goreflector
```

## Reverse Proxy (Nginx)

### Nginx Configuration

```nginx
upstream goreflector {
    server localhost:8080;
    keepalive 32;
}

server {
    listen 80;
    server_name proxy.example.com;

    location / {
        proxy_pass http://goreflector;
        proxy_http_version 1.1;

        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;

        proxy_buffering off;
    }
}
```

## Monitoring

### Logging

**Access Logs:**
```bash
tail -f /var/log/goreflector/access.log
```

**Error Logs:**
```bash
tail -f /var/log/goreflector/error.log
```

### Log Rotation

Create `/etc/logrotate.d/goreflector`:

```
/var/log/goreflector/*.log {
    daily
    rotate 14
    compress
    delaycompress
    notifempty
    create 0640 goreflector goreflector
    sharedscripts
    postrotate
        systemctl reload goreflector > /dev/null 2>&1 || true
    endscript
}
```

### Health Checks

```bash
# Simple health check
curl -f http://localhost:8080/ || exit 1

# With timeout
timeout 5 curl -f http://localhost:8080/ || exit 1
```

### Monitoring Scripts

Create `/usr/local/bin/check-goreflector.sh`:

```bash
#!/bin/bash

# Check if process is running
if ! systemctl is-active --quiet goreflector; then
    echo "CRITICAL: goreflector is not running"
    exit 2
fi

# Check if port is listening
if ! netstat -tuln | grep -q ":8080 "; then
    echo "CRITICAL: Port 8080 not listening"
    exit 2
fi

# Check if responding
if ! curl -sf http://localhost:8080/ > /dev/null; then
    echo "WARNING: goreflector not responding"
    exit 1
fi

echo "OK: goreflector is running"
exit 0
```

## Security Hardening

### Firewall Rules

```bash
# Allow only necessary ports
sudo ufw allow 8080/tcp
sudo ufw enable

# Or with iptables
sudo iptables -A INPUT -p tcp --dport 8080 -j ACCEPT
sudo iptables-save > /etc/iptables/rules.v4
```

### SELinux/AppArmor

**SELinux Policy:**
```bash
# Allow network binding
sudo setsebool -P httpd_can_network_connect 1
```

### TLS/SSL

For production, run behind a TLS terminator:

- Nginx with Let's Encrypt
- HAProxy with TLS
- Cloud load balancer (AWS ALB, GCP LB)

### Rate Limiting

Use nginx or cloud provider rate limiting:

```nginx
limit_req_zone $binary_remote_addr zone=proxy_limit:10m rate=10r/s;

server {
    location / {
        limit_req zone=proxy_limit burst=20 nodelay;
        proxy_pass http://goreflector;
    }
}
```

## Performance Tuning

### System Limits

Edit `/etc/security/limits.conf`:

```
goreflector soft nofile 65536
goreflector hard nofile 65536
```

### Kernel Parameters

Edit `/etc/sysctl.conf`:

```
net.core.somaxconn = 1024
net.ipv4.tcp_max_syn_backlog = 2048
net.ipv4.ip_local_port_range = 10000 65000
net.ipv4.tcp_tw_reuse = 1
net.ipv4.tcp_fin_timeout = 15
```

Apply:
```bash
sudo sysctl -p
```

## Troubleshooting

### Common Issues

**1. Port already in use:**
```bash
# Find process using port
sudo lsof -i :8080
sudo netstat -tlnp | grep 8080

# Kill process or choose different port
```

**2. Permission denied:**
```bash
# Ports < 1024 require root or CAP_NET_BIND_SERVICE
# Use port >= 1024 or grant capability:
sudo setcap 'cap_net_bind_service=+ep' /usr/local/bin/goreflector
```

**3. Connection refused:**
```bash
# Check firewall
sudo ufw status
sudo iptables -L -n

# Check if service is running
sudo systemctl status goreflector
```

**4. Timeout errors:**
```bash
# Increase timeout
./goreflector -p 8080 -t 60 https://slow-backend.com
```

### Debug Mode

Enable verbose logging:
```bash
./goreflector -p 8080 -v https://api.example.com
```

### Testing

```bash
# Test basic connectivity
curl -v http://localhost:8080/

# Test with headers
curl -H "X-Test: true" http://localhost:8080/api/test

# Test POST
curl -X POST -d '{"test":"data"}' http://localhost:8080/api/data

# Load testing
ab -n 1000 -c 10 http://localhost:8080/
```

## Backup and Recovery

### Configuration Backup

```bash
# Backup configuration
sudo tar -czf goreflector-config-$(date +%Y%m%d).tar.gz \
  /etc/goreflector/ \
  /etc/systemd/system/goreflector.service

# Restore
sudo tar -xzf goreflector-config-*.tar.gz -C /
sudo systemctl daemon-reload
```

### Disaster Recovery

1. Stop service: `sudo systemctl stop goreflector`
2. Restore configuration from backup
3. Verify configuration
4. Start service: `sudo systemctl start goreflector`
5. Verify functionality

## Updates and Maintenance

### Zero-Downtime Updates

```bash
# Build new version
go build -o goreflector-new .

# Test new version
./goreflector-new -p 8081 https://api.example.com &
curl http://localhost:8081/

# If successful, swap
sudo systemctl stop goreflector
sudo cp goreflector-new /usr/local/bin/goreflector
sudo systemctl start goreflector
```

### Rolling Updates (Kubernetes)

```bash
# Update image
kubectl set image deployment/goreflector goreflector=goreflector:v1.1.0

# Check rollout status
kubectl rollout status deployment/goreflector

# Rollback if needed
kubectl rollout undo deployment/goreflector
```
