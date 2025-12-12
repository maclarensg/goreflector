package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type ProxyConfig struct {
	ListenAddr    string
	TargetURL     *url.URL
	Timeout       time.Duration
	CustomHeaders map[string]string
}

type Proxy struct {
	config     ProxyConfig
	httpClient *http.Client
	logger     *log.Logger
}

func NewProxy(config ProxyConfig, logger *log.Logger) (*Proxy, error) {
	if config.TargetURL == nil {
		return nil, fmt.Errorf("target URL cannot be nil")
	}

	if config.ListenAddr == "" {
		return nil, fmt.Errorf("listen address cannot be empty")
	}

	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	if logger == nil {
		logger = log.Default()
	}

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

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   config.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	return &Proxy{
		config:     config,
		httpClient: httpClient,
		logger:     logger,
	}, nil
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	targetURL := p.buildTargetURL(r)

	proxyReq, err := http.NewRequest(r.Method, targetURL.String(), r.Body)
	if err != nil {
		p.logger.Printf("Error creating proxy request: %v", err)
		http.Error(w, "Failed to create proxy request", http.StatusInternalServerError)
		return
	}

	p.copyHeaders(r, proxyReq)
	p.addForwardedHeaders(r, proxyReq)

	p.logger.Printf("%s %s -> %s", r.Method, r.URL.Path, targetURL.String())

	resp, err := p.httpClient.Do(proxyReq)
	if err != nil {
		p.logger.Printf("Error proxying request: %v", err)
		http.Error(w, "Failed to proxy request", http.StatusBadGateway)
		return
	}
	defer func() { _ = resp.Body.Close() }()

	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)

	if _, err := io.Copy(w, resp.Body); err != nil {
		p.logger.Printf("Error copying response body: %v", err)
	}
}

func (p *Proxy) buildTargetURL(r *http.Request) *url.URL {
	targetURL := &url.URL{
		Scheme:   p.config.TargetURL.Scheme,
		Host:     p.config.TargetURL.Host,
		Path:     r.URL.Path,
		RawQuery: r.URL.RawQuery,
	}

	if p.config.TargetURL.Path != "" && p.config.TargetURL.Path != "/" {
		targetURL.Path = strings.TrimSuffix(p.config.TargetURL.Path, "/") + r.URL.Path
	}

	return targetURL
}

func (p *Proxy) copyHeaders(src *http.Request, dst *http.Request) {
	// Copy original request headers (except hop-by-hop headers)
	for key, values := range src.Header {
		if shouldSkipHeader(key) {
			continue
		}
		for _, value := range values {
			dst.Header.Add(key, value)
		}
	}

	// Set default Host header to target URL's host
	dst.Host = p.config.TargetURL.Host

	// Apply custom headers (these override any existing headers)
	for name, value := range p.config.CustomHeaders {
		// Special handling for Host header - must be set via dst.Host
		if http.CanonicalHeaderKey(name) == "Host" {
			dst.Host = value
		} else {
			dst.Header.Set(name, value)
		}
	}
}

func (p *Proxy) addForwardedHeaders(src *http.Request, dst *http.Request) {
	clientIP := getClientIP(src)
	if clientIP != "" {
		if prior := dst.Header.Get("X-Forwarded-For"); prior != "" {
			clientIP = prior + ", " + clientIP
		}
		dst.Header.Set("X-Forwarded-For", clientIP)
	}

	if src.Host != "" {
		dst.Header.Set("X-Forwarded-Host", src.Host)
	}

	scheme := "http"
	if src.TLS != nil {
		scheme = "https"
	}
	dst.Header.Set("X-Forwarded-Proto", scheme)
}

func (p *Proxy) Start() error {
	p.logger.Printf("Starting proxy server on %s, forwarding to %s", p.config.ListenAddr, p.config.TargetURL.String())

	server := &http.Server{
		Addr:         p.config.ListenAddr,
		Handler:      p,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return server.ListenAndServe()
}

func shouldSkipHeader(header string) bool {
	skipHeaders := map[string]bool{
		"Connection":          true,
		"Keep-Alive":          true,
		"Proxy-Authenticate":  true,
		"Proxy-Authorization": true,
		"Te":                  true,
		"Trailers":            true,
		"Transfer-Encoding":   true,
		"Upgrade":             true,
	}

	return skipHeaders[http.CanonicalHeaderKey(header)]
}

func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}

	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
