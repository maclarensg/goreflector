package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestRunFunctionEnd2End(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Test", "true")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("test response"))
	}))
	defer backend.Close()

	backendURL, _ := url.Parse(backend.URL)

	port := getFreePort(t)

	opts := &Options{
		Port:      port,
		TargetURL: backendURL.String(),
		Timeout:   10,
		Verbose:   false,
	}

	targetURL, err := url.Parse(opts.TargetURL)
	if err != nil {
		t.Fatalf("failed to parse target URL: %v", err)
	}

	if targetURL.Scheme != "http" && targetURL.Scheme != "https" {
		t.Fatal("target URL must use http or https scheme")
	}

	logger := log.New(io.Discard, "", log.LstdFlags)
	if opts.Verbose {
		logger.SetOutput(io.Writer(io.Discard))
	}

	config := ProxyConfig{
		ListenAddr: fmt.Sprintf(":%d", opts.Port),
		TargetURL:  targetURL,
		Timeout:    time.Duration(opts.Timeout) * time.Second,
	}

	proxy, err := NewProxy(config, logger)
	if err != nil {
		t.Fatalf("error creating proxy: %v", err)
	}

	server := &http.Server{
		Addr:         config.ListenAddr,
		Handler:      proxy,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		_ = server.ListenAndServe()
	}()
	defer func() { _ = server.Close() }()

	time.Sleep(200 * time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/test", port))
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "test response" {
		t.Errorf("expected 'test response', got %s", string(body))
	}

	if resp.Header.Get("X-Test") != "true" {
		t.Error("expected X-Test header")
	}
}

func TestValidateOptionsInvalidScheme(t *testing.T) {
	opts := &Options{
		Port:      8080,
		TargetURL: "ftp://example.com",
		Timeout:   30,
	}

	err := validateOptions(opts)
	if err != nil {
		return
	}

	targetURL, err := url.Parse(opts.TargetURL)
	if err != nil {
		t.Errorf("failed to parse URL: %v", err)
		return
	}

	if targetURL.Scheme != "http" && targetURL.Scheme != "https" {
		return
	}

	t.Error("expected validation to fail for non-http/https scheme")
}

func getFreePort(t *testing.T) int {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to find free port: %v", err)
	}
	defer func() { _ = listener.Close() }()
	addr := listener.Addr().(*net.TCPAddr)
	return addr.Port
}
