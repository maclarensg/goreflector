package main

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestNewProxy(t *testing.T) {
	tests := []struct {
		name        string
		config      ProxyConfig
		expectError bool
	}{
		{
			name: "valid configuration",
			config: ProxyConfig{
				ListenAddr: ":8080",
				TargetURL:  mustParseURL("https://example.com"),
				Timeout:    30 * time.Second,
			},
			expectError: false,
		},
		{
			name: "nil target URL",
			config: ProxyConfig{
				ListenAddr: ":8080",
				TargetURL:  nil,
			},
			expectError: true,
		},
		{
			name: "empty listen address",
			config: ProxyConfig{
				ListenAddr: "",
				TargetURL:  mustParseURL("https://example.com"),
			},
			expectError: true,
		},
		{
			name: "zero timeout uses default",
			config: ProxyConfig{
				ListenAddr: ":8080",
				TargetURL:  mustParseURL("https://example.com"),
				Timeout:    0,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := log.New(io.Discard, "", 0)
			proxy, err := NewProxy(tt.config, logger)

			if tt.expectError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.expectError && proxy == nil {
				t.Error("expected proxy but got nil")
			}
		})
	}
}

func TestNewProxyWithNilLogger(t *testing.T) {
	config := ProxyConfig{
		ListenAddr: ":8080",
		TargetURL:  mustParseURL("https://example.com"),
		Timeout:    30 * time.Second,
	}

	proxy, err := NewProxy(config, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if proxy.logger == nil {
		t.Error("expected default logger but got nil")
	}
}

func TestBuildTargetURL(t *testing.T) {
	tests := []struct {
		name      string
		targetURL string
		reqPath   string
		reqQuery  string
		expected  string
	}{
		{
			name:      "simple path",
			targetURL: "https://example.com",
			reqPath:   "/api/v1/users",
			reqQuery:  "",
			expected:  "https://example.com/api/v1/users",
		},
		{
			name:      "with query parameters",
			targetURL: "https://example.com",
			reqPath:   "/api/v1/users",
			reqQuery:  "page=1&limit=10",
			expected:  "https://example.com/api/v1/users?page=1&limit=10",
		},
		{
			name:      "target with base path",
			targetURL: "https://example.com/base",
			reqPath:   "/api/users",
			reqQuery:  "",
			expected:  "https://example.com/base/api/users",
		},
		{
			name:      "target with trailing slash",
			targetURL: "https://example.com/base/",
			reqPath:   "/api/users",
			reqQuery:  "",
			expected:  "https://example.com/base/api/users",
		},
		{
			name:      "root path",
			targetURL: "https://example.com",
			reqPath:   "/",
			reqQuery:  "",
			expected:  "https://example.com/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := ProxyConfig{
				ListenAddr: ":8080",
				TargetURL:  mustParseURL(tt.targetURL),
			}
			logger := log.New(io.Discard, "", 0)
			proxy, _ := NewProxy(config, logger)

			reqURL := &url.URL{
				Path:     tt.reqPath,
				RawQuery: tt.reqQuery,
			}
			req := &http.Request{URL: reqURL}

			result := proxy.buildTargetURL(req)

			if result.String() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.String())
			}
		})
	}
}

func TestCopyHeaders(t *testing.T) {
	targetURL := mustParseURL("https://target.example.com")
	config := ProxyConfig{
		ListenAddr: ":8080",
		TargetURL:  targetURL,
	}
	logger := log.New(io.Discard, "", 0)
	proxy, _ := NewProxy(config, logger)

	srcReq, _ := http.NewRequest("GET", "http://source.example.com/path", nil)
	srcReq.Header.Set("User-Agent", "test-agent")
	srcReq.Header.Set("Accept", "application/json")
	srcReq.Header.Set("Connection", "keep-alive")
	srcReq.Header.Set("Custom-Header", "custom-value")

	dstReq, _ := http.NewRequest("GET", "https://target.example.com/path", nil)

	proxy.copyHeaders(srcReq, dstReq)

	if dstReq.Header.Get("User-Agent") != "test-agent" {
		t.Error("User-Agent header not copied")
	}
	if dstReq.Header.Get("Accept") != "application/json" {
		t.Error("Accept header not copied")
	}
	if dstReq.Header.Get("Custom-Header") != "custom-value" {
		t.Error("Custom-Header not copied")
	}
	if dstReq.Header.Get("Connection") != "" {
		t.Error("Connection header should be skipped")
	}
	if dstReq.Host != "target.example.com" {
		t.Errorf("Host should be set to target host, got %s", dstReq.Host)
	}
}

func TestAddForwardedHeaders(t *testing.T) {
	targetURL := mustParseURL("https://target.example.com")
	config := ProxyConfig{
		ListenAddr: ":8080",
		TargetURL:  targetURL,
	}
	logger := log.New(io.Discard, "", 0)
	proxy, _ := NewProxy(config, logger)

	srcReq, _ := http.NewRequest("GET", "http://source.example.com/path", nil)
	srcReq.RemoteAddr = "192.168.1.100:12345"
	srcReq.Host = "source.example.com"

	dstReq, _ := http.NewRequest("GET", "https://target.example.com/path", nil)

	proxy.addForwardedHeaders(srcReq, dstReq)

	if xff := dstReq.Header.Get("X-Forwarded-For"); xff != "192.168.1.100" {
		t.Errorf("expected X-Forwarded-For to be 192.168.1.100, got %s", xff)
	}
	if xfh := dstReq.Header.Get("X-Forwarded-Host"); xfh != "source.example.com" {
		t.Errorf("expected X-Forwarded-Host to be source.example.com, got %s", xfh)
	}
	if xfp := dstReq.Header.Get("X-Forwarded-Proto"); xfp != "http" {
		t.Errorf("expected X-Forwarded-Proto to be http, got %s", xfp)
	}
}

func TestAddForwardedHeadersAppendXFF(t *testing.T) {
	targetURL := mustParseURL("https://target.example.com")
	config := ProxyConfig{
		ListenAddr: ":8080",
		TargetURL:  targetURL,
	}
	logger := log.New(io.Discard, "", 0)
	proxy, _ := NewProxy(config, logger)

	srcReq, _ := http.NewRequest("GET", "http://source.example.com/path", nil)
	srcReq.RemoteAddr = "192.168.1.100:12345"

	dstReq, _ := http.NewRequest("GET", "https://target.example.com/path", nil)
	dstReq.Header.Set("X-Forwarded-For", "10.0.0.1")

	proxy.addForwardedHeaders(srcReq, dstReq)

	xff := dstReq.Header.Get("X-Forwarded-For")
	if xff != "10.0.0.1, 192.168.1.100" {
		t.Errorf("expected X-Forwarded-For to be '10.0.0.1, 192.168.1.100', got %s", xff)
	}
}

func TestShouldSkipHeader(t *testing.T) {
	tests := []struct {
		header string
		skip   bool
	}{
		{"Connection", true},
		{"Keep-Alive", true},
		{"Proxy-Authenticate", true},
		{"Proxy-Authorization", true},
		{"Te", true},
		{"Trailers", true},
		{"Transfer-Encoding", true},
		{"Upgrade", true},
		{"Content-Type", false},
		{"User-Agent", false},
		{"Accept", false},
		{"Authorization", false},
	}

	for _, tt := range tests {
		t.Run(tt.header, func(t *testing.T) {
			result := shouldSkipHeader(tt.header)
			if result != tt.skip {
				t.Errorf("shouldSkipHeader(%s) = %v, want %v", tt.header, result, tt.skip)
			}
		})
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		xff        string
		realIP     string
		expected   string
	}{
		{
			name:       "from RemoteAddr",
			remoteAddr: "192.168.1.100:12345",
			xff:        "",
			realIP:     "",
			expected:   "192.168.1.100",
		},
		{
			name:       "from X-Forwarded-For",
			remoteAddr: "192.168.1.100:12345",
			xff:        "10.0.0.1, 10.0.0.2",
			realIP:     "",
			expected:   "10.0.0.1",
		},
		{
			name:       "from X-Real-IP",
			remoteAddr: "192.168.1.100:12345",
			xff:        "",
			realIP:     "10.0.0.5",
			expected:   "10.0.0.5",
		},
		{
			name:       "X-Forwarded-For takes precedence",
			remoteAddr: "192.168.1.100:12345",
			xff:        "10.0.0.1",
			realIP:     "10.0.0.5",
			expected:   "10.0.0.1",
		},
		{
			name:       "RemoteAddr without port",
			remoteAddr: "192.168.1.100",
			xff:        "",
			realIP:     "",
			expected:   "192.168.1.100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "http://example.com/path", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}
			if tt.realIP != "" {
				req.Header.Set("X-Real-IP", tt.realIP)
			}

			result := getClientIP(req)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestServeHTTP(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Backend-Response", "true")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("backend response"))
	}))
	defer backend.Close()

	backendURL := mustParseURL(backend.URL)
	config := ProxyConfig{
		ListenAddr: ":8080",
		TargetURL:  backendURL,
		Timeout:    30 * time.Second,
	}
	logger := log.New(io.Discard, "", 0)
	proxy, _ := NewProxy(config, logger)

	req := httptest.NewRequest("GET", "http://localhost:8080/test", nil)
	req.Header.Set("User-Agent", "test-client")
	w := httptest.NewRecorder()

	proxy.ServeHTTP(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "backend response" {
		t.Errorf("expected 'backend response', got %s", string(body))
	}

	if resp.Header.Get("X-Backend-Response") != "true" {
		t.Error("backend response header not copied")
	}
}

func TestServeHTTPWithDifferentMethods(t *testing.T) {
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			var receivedMethod string
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedMethod = r.Method
				w.WriteHeader(http.StatusOK)
			}))
			defer backend.Close()

			backendURL := mustParseURL(backend.URL)
			config := ProxyConfig{
				ListenAddr: ":8080",
				TargetURL:  backendURL,
			}
			logger := log.New(io.Discard, "", 0)
			proxy, _ := NewProxy(config, logger)

			var body io.Reader
			if method == "POST" || method == "PUT" || method == "PATCH" {
				body = strings.NewReader("test data")
			}

			req := httptest.NewRequest(method, "http://localhost:8080/test", body)
			w := httptest.NewRecorder()

			proxy.ServeHTTP(w, req)

			if receivedMethod != method {
				t.Errorf("expected method %s, got %s", method, receivedMethod)
			}
		})
	}
}

func TestServeHTTPWithBody(t *testing.T) {
	var receivedBody []byte
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	backendURL := mustParseURL(backend.URL)
	config := ProxyConfig{
		ListenAddr: ":8080",
		TargetURL:  backendURL,
	}
	logger := log.New(io.Discard, "", 0)
	proxy, _ := NewProxy(config, logger)

	testData := "test request body"
	req := httptest.NewRequest("POST", "http://localhost:8080/test", strings.NewReader(testData))
	w := httptest.NewRecorder()

	proxy.ServeHTTP(w, req)

	if string(receivedBody) != testData {
		t.Errorf("expected body %s, got %s", testData, string(receivedBody))
	}
}

func TestServeHTTPBackendError(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("backend error"))
	}))
	defer backend.Close()

	backendURL := mustParseURL(backend.URL)
	config := ProxyConfig{
		ListenAddr: ":8080",
		TargetURL:  backendURL,
	}
	logger := log.New(io.Discard, "", 0)
	proxy, _ := NewProxy(config, logger)

	req := httptest.NewRequest("GET", "http://localhost:8080/test", nil)
	w := httptest.NewRecorder()

	proxy.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "backend error" {
		t.Errorf("expected 'backend error', got %s", string(body))
	}
}

func TestServeHTTPInvalidBackend(t *testing.T) {
	config := ProxyConfig{
		ListenAddr: ":8080",
		TargetURL:  mustParseURL("http://invalid-backend-that-does-not-exist.local:9999"),
		Timeout:    1 * time.Second,
	}
	logger := log.New(io.Discard, "", 0)
	proxy, _ := NewProxy(config, logger)

	req := httptest.NewRequest("GET", "http://localhost:8080/test", nil)
	w := httptest.NewRecorder()

	proxy.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadGateway {
		t.Errorf("expected status 502, got %d", resp.StatusCode)
	}
}

func TestServeHTTPPreservesQueryParams(t *testing.T) {
	var receivedQuery string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	backendURL := mustParseURL(backend.URL)
	config := ProxyConfig{
		ListenAddr: ":8080",
		TargetURL:  backendURL,
	}
	logger := log.New(io.Discard, "", 0)
	proxy, _ := NewProxy(config, logger)

	req := httptest.NewRequest("GET", "http://localhost:8080/test?foo=bar&baz=qux", nil)
	w := httptest.NewRecorder()

	proxy.ServeHTTP(w, req)

	if receivedQuery != "foo=bar&baz=qux" {
		t.Errorf("expected query 'foo=bar&baz=qux', got %s", receivedQuery)
	}
}

func TestServeHTTPLogging(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	backendURL := mustParseURL(backend.URL)
	config := ProxyConfig{
		ListenAddr: ":8080",
		TargetURL:  backendURL,
	}

	var logBuf bytes.Buffer
	logger := log.New(&logBuf, "", 0)
	proxy, _ := NewProxy(config, logger)

	req := httptest.NewRequest("GET", "http://localhost:8080/test", nil)
	w := httptest.NewRecorder()

	proxy.ServeHTTP(w, req)

	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "GET") {
		t.Error("log should contain HTTP method")
	}
	if !strings.Contains(logOutput, "/test") {
		t.Error("log should contain request path")
	}
}

func mustParseURL(rawURL string) *url.URL {
	u, err := url.Parse(rawURL)
	if err != nil {
		panic(err)
	}
	return u
}
