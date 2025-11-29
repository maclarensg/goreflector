package main

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestIntegrationBasicProxy(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Backend", "true")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, "Hello from backend")
	}))
	defer backend.Close()

	proxyAddr := findFreePort(t)
	backendURL := mustParseURL(backend.URL)

	config := ProxyConfig{
		ListenAddr: proxyAddr,
		TargetURL:  backendURL,
		Timeout:    5 * time.Second,
	}

	proxy, err := NewProxy(config, nil)
	if err != nil {
		t.Fatalf("failed to create proxy: %v", err)
	}

	proxyServer := &http.Server{
		Addr:    proxyAddr,
		Handler: proxy,
	}

	go func() {
		_ = proxyServer.ListenAndServe()
	}()
	defer func() { _ = proxyServer.Close() }()

	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get("http://localhost" + proxyAddr + "/test")
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "Hello from backend" {
		t.Errorf("expected 'Hello from backend', got %s", string(body))
	}

	if resp.Header.Get("X-Backend") != "true" {
		t.Error("expected X-Backend header")
	}
}

func TestIntegrationProxyHeaders(t *testing.T) {
	var receivedHeaders http.Header
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	proxyAddr := findFreePort(t)
	backendURL := mustParseURL(backend.URL)

	config := ProxyConfig{
		ListenAddr: proxyAddr,
		TargetURL:  backendURL,
		Timeout:    5 * time.Second,
	}

	proxy, err := NewProxy(config, nil)
	if err != nil {
		t.Fatalf("failed to create proxy: %v", err)
	}

	proxyServer := &http.Server{
		Addr:    proxyAddr,
		Handler: proxy,
	}

	go func() {
		_ = proxyServer.ListenAndServe()
	}()
	defer func() { _ = proxyServer.Close() }()

	time.Sleep(100 * time.Millisecond)

	client := &http.Client{}
	req, _ := http.NewRequest("GET", "http://localhost"+proxyAddr+"/test", nil)
	req.Header.Set("User-Agent", "test-client")
	req.Header.Set("Custom-Header", "custom-value")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if receivedHeaders.Get("User-Agent") != "test-client" {
		t.Error("User-Agent not forwarded")
	}
	if receivedHeaders.Get("Custom-Header") != "custom-value" {
		t.Error("Custom-Header not forwarded")
	}
	if receivedHeaders.Get("X-Forwarded-For") == "" {
		t.Error("X-Forwarded-For not set")
	}
	if receivedHeaders.Get("X-Forwarded-Host") == "" {
		t.Error("X-Forwarded-Host not set")
	}
	if receivedHeaders.Get("X-Forwarded-Proto") != "http" {
		t.Errorf("expected X-Forwarded-Proto http, got %s", receivedHeaders.Get("X-Forwarded-Proto"))
	}
}

func TestServeHTTPWithMultipleHeaderValues(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Set-Cookie", "session=abc123")
		w.Header().Add("Set-Cookie", "token=xyz789")
		w.WriteHeader(http.StatusOK)
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
	cookies := resp.Header.Values("Set-Cookie")

	if len(cookies) != 2 {
		t.Errorf("expected 2 Set-Cookie headers, got %d", len(cookies))
	}
}

func TestServeHTTPErrorCreatingRequest(t *testing.T) {
	backendURL := mustParseURL("http://example.com")
	config := ProxyConfig{
		ListenAddr: ":8080",
		TargetURL:  backendURL,
	}
	proxy, _ := NewProxy(config, nil)

	req := httptest.NewRequest("GET", "http://localhost:8080/test", nil)
	req.Method = "\n"
	w := httptest.NewRecorder()

	proxy.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", resp.StatusCode)
	}
}

func TestAddForwardedHeadersEmptyHost(t *testing.T) {
	targetURL := mustParseURL("https://target.example.com")
	config := ProxyConfig{
		ListenAddr: ":8080",
		TargetURL:  targetURL,
	}
	proxy, _ := NewProxy(config, nil)

	srcReq, _ := http.NewRequest("GET", "http://source.example.com/path", nil)
	srcReq.RemoteAddr = "192.168.1.100:12345"
	srcReq.Host = ""

	dstReq, _ := http.NewRequest("GET", "https://target.example.com/path", nil)

	proxy.addForwardedHeaders(srcReq, dstReq)

	if xfh := dstReq.Header.Get("X-Forwarded-Host"); xfh != "" {
		t.Errorf("expected empty X-Forwarded-Host, got %s", xfh)
	}
}

func TestServeHTTPWithResponseHeaders(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("X-Custom", "value")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
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

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status 201, got %d", resp.StatusCode)
	}

	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}

	if cc := resp.Header.Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("expected Cache-Control no-cache, got %s", cc)
	}

	if custom := resp.Header.Get("X-Custom"); custom != "value" {
		t.Errorf("expected X-Custom value, got %s", custom)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != `{"status":"ok"}` {
		t.Errorf("unexpected body: %s", string(body))
	}
}

func TestServeHTTPWithLargeBody(t *testing.T) {
	largeData := strings.Repeat("x", 1024*1024)
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(largeData))
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
	body, _ := io.ReadAll(resp.Body)

	if len(body) != len(largeData) {
		t.Errorf("expected body length %d, got %d", len(largeData), len(body))
	}
}

func TestServeHTTPWithRedirect(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			w.Header().Set("Location", "/target")
			w.WriteHeader(http.StatusFound)
		} else {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("reached target"))
		}
	}))
	defer backend.Close()

	backendURL := mustParseURL(backend.URL)
	config := ProxyConfig{
		ListenAddr: ":8080",
		TargetURL:  backendURL,
	}
	proxy, _ := NewProxy(config, nil)

	req := httptest.NewRequest("GET", "http://localhost:8080/redirect", nil)
	w := httptest.NewRecorder()

	proxy.ServeHTTP(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusFound {
		t.Errorf("expected status 302, got %d", resp.StatusCode)
	}

	if location := resp.Header.Get("Location"); location != "/target" {
		t.Errorf("expected Location /target, got %s", location)
	}
}

func findFreePort(t *testing.T) string {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to find free port: %v", err)
	}
	defer func() { _ = listener.Close() }()
	addr := listener.Addr().(*net.TCPAddr)
	return fmt.Sprintf(":%d", addr.Port)
}
