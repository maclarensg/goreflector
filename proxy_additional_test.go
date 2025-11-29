package main

import (
	"bytes"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("read error")
}

func TestServeHTTPWithRequestBodyError(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	backendURL := mustParseURL(backend.URL)
	config := ProxyConfig{
		ListenAddr: ":8080",
		TargetURL:  backendURL,
		Timeout:    1 * time.Second,
	}
	proxy, _ := NewProxy(config, nil)

	req := httptest.NewRequest("POST", "http://localhost:8080/test", &errorReader{})
	w := httptest.NewRecorder()

	proxy.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadGateway {
		t.Errorf("expected status 502, got %d", resp.StatusCode)
	}
}

func TestCopyHeadersWithMultipleValues(t *testing.T) {
	targetURL := mustParseURL("https://target.example.com")
	config := ProxyConfig{
		ListenAddr: ":8080",
		TargetURL:  targetURL,
	}
	proxy, _ := NewProxy(config, nil)

	srcReq, _ := http.NewRequest("GET", "http://source.example.com/path", nil)
	srcReq.Header.Add("Accept", "text/html")
	srcReq.Header.Add("Accept", "application/json")
	srcReq.Header.Add("Custom", "value1")
	srcReq.Header.Add("Custom", "value2")

	dstReq, _ := http.NewRequest("GET", "https://target.example.com/path", nil)

	proxy.copyHeaders(srcReq, dstReq)

	acceptValues := dstReq.Header.Values("Accept")
	if len(acceptValues) != 2 {
		t.Errorf("expected 2 Accept values, got %d", len(acceptValues))
	}

	customValues := dstReq.Header.Values("Custom")
	if len(customValues) != 2 {
		t.Errorf("expected 2 Custom values, got %d", len(customValues))
	}
}

func TestShouldSkipHeaderCaseInsensitive(t *testing.T) {
	tests := []struct {
		header string
		skip   bool
	}{
		{"connection", true},
		{"CONNECTION", true},
		{"CoNnEcTiOn", true},
		{"keep-alive", true},
		{"KEEP-ALIVE", true},
		{"transfer-encoding", true},
		{"TRANSFER-ENCODING", true},
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

func TestAddForwardedHeadersWithExistingXFFSpaces(t *testing.T) {
	targetURL := mustParseURL("https://target.example.com")
	config := ProxyConfig{
		ListenAddr: ":8080",
		TargetURL:  targetURL,
	}
	proxy, _ := NewProxy(config, nil)

	srcReq, _ := http.NewRequest("GET", "http://source.example.com/path", nil)
	srcReq.RemoteAddr = "192.168.1.100:12345"

	dstReq, _ := http.NewRequest("GET", "https://target.example.com/path", nil)
	dstReq.Header.Set("X-Forwarded-For", "10.0.0.1,10.0.0.2")

	proxy.addForwardedHeaders(srcReq, dstReq)

	xff := dstReq.Header.Get("X-Forwarded-For")
	expected := "10.0.0.1,10.0.0.2, 192.168.1.100"
	if xff != expected {
		t.Errorf("expected X-Forwarded-For to be '%s', got '%s'", expected, xff)
	}
}

func TestBuildTargetURLWithComplexPath(t *testing.T) {
	tests := []struct {
		name      string
		targetURL string
		reqPath   string
		expected  string
	}{
		{
			name:      "nested path with target base",
			targetURL: "https://example.com/api/v1",
			reqPath:   "/users/123/posts",
			expected:  "https://example.com/api/v1/users/123/posts",
		},
		{
			name:      "empty path",
			targetURL: "https://example.com/base",
			reqPath:   "",
			expected:  "https://example.com/base",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := ProxyConfig{
				ListenAddr: ":8080",
				TargetURL:  mustParseURL(tt.targetURL),
			}
			proxy, _ := NewProxy(config, nil)

			reqURL := &url.URL{Path: tt.reqPath}
			req := &http.Request{URL: reqURL}

			result := proxy.buildTargetURL(req)

			if result.String() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.String())
			}
		})
	}
}

func TestServeHTTPBackendTimeout(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(3 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	backendURL := mustParseURL(backend.URL)
	config := ProxyConfig{
		ListenAddr: ":8080",
		TargetURL:  backendURL,
		Timeout:    500 * time.Millisecond,
	}
	proxy, _ := NewProxy(config, nil)

	req := httptest.NewRequest("GET", "http://localhost:8080/test", nil)
	w := httptest.NewRecorder()

	proxy.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadGateway {
		t.Errorf("expected status 502, got %d", resp.StatusCode)
	}
}

func TestServeHTTPLoggingError(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	backendURL := mustParseURL("http://invalid-backend:9999")
	config := ProxyConfig{
		ListenAddr: ":8080",
		TargetURL:  backendURL,
		Timeout:    1 * time.Second,
	}

	var logBuf bytes.Buffer
	logger := log.New(&logBuf, "", 0)
	proxy, _ := NewProxy(config, logger)

	req := httptest.NewRequest("GET", "http://localhost:8080/test", nil)
	w := httptest.NewRecorder()

	proxy.ServeHTTP(w, req)

	logOutput := logBuf.String()
	if !contains(logOutput, "Error proxying request") {
		t.Error("log should contain error message")
	}
}

func TestNewProxyDefaultTimeout(t *testing.T) {
	config := ProxyConfig{
		ListenAddr: ":8080",
		TargetURL:  mustParseURL("https://example.com"),
		Timeout:    0,
	}

	proxy, err := NewProxy(config, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if proxy.config.Timeout != 30*time.Second {
		t.Errorf("expected default timeout 30s, got %v", proxy.config.Timeout)
	}
}

func TestServeHTTPWithEmptyResponse(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer backend.Close()

	backendURL := mustParseURL(backend.URL)
	config := ProxyConfig{
		ListenAddr: ":8080",
		TargetURL:  backendURL,
	}
	proxy, _ := NewProxy(config, nil)

	req := httptest.NewRequest("GET", "http://localhost:8080/test", nil)
	w := httptest.NewRecorder()

	proxy.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if len(body) != 0 {
		t.Errorf("expected empty body, got %d bytes", len(body))
	}
}

func TestGetClientIPWithSpacesInXFF(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://example.com/path", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	req.Header.Set("X-Forwarded-For", "  10.0.0.1  , 10.0.0.2")

	result := getClientIP(req)
	if result != "10.0.0.1" {
		t.Errorf("expected 10.0.0.1, got %s", result)
	}
}
