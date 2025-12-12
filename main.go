package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"strings"
	"time"
)

const version = "1.0.0"

type Options struct {
	Port        int
	TargetURL   string
	Timeout     int
	Verbose     bool
	ShowVersion bool
	Headers     []string
}

// headerFlags implements flag.Value to support multiple -H flags
type headerFlags []string

func (h *headerFlags) String() string {
	return fmt.Sprint(*h)
}

func (h *headerFlags) Set(value string) error {
	*h = append(*h, value)
	return nil
}

func parseFlags() (*Options, error) {
	opts := &Options{}
	var headers headerFlags

	flag.IntVar(&opts.Port, "p", 8080, "Port to listen on")
	flag.IntVar(&opts.Port, "port", 8080, "Port to listen on")
	flag.IntVar(&opts.Timeout, "t", 30, "Request timeout in seconds")
	flag.IntVar(&opts.Timeout, "timeout", 30, "Request timeout in seconds")
	flag.BoolVar(&opts.Verbose, "v", false, "Verbose logging")
	flag.BoolVar(&opts.Verbose, "verbose", false, "Verbose logging")
	flag.BoolVar(&opts.ShowVersion, "version", false, "Show version")
	flag.Var(&headers, "H", "Custom header (can be used multiple times, format: 'Name: Value')")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "goreflector v%s - HTTP reverse proxy\n\n", version)
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <target-url>\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s -p 8080 https://example.com\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -H \"Host: example.com\" https://1.2.3.4/\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -H \"Authorization: Bearer token\" -H \"X-API-Key: key123\" https://api.example.com\n", os.Args[0])
	}

	flag.Parse()

	if opts.ShowVersion {
		fmt.Printf("goreflector version %s\n", version)
		os.Exit(0)
	}

	if flag.NArg() < 1 {
		return nil, fmt.Errorf("target URL is required")
	}

	opts.TargetURL = flag.Arg(0)
	opts.Headers = headers

	return opts, nil
}

func parseHeaders(headers []string) (map[string]string, error) {
	result := make(map[string]string)
	for _, header := range headers {
		parts := strings.SplitN(header, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid header format: %q (expected 'Name: Value')", header)
		}
		name := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if name == "" {
			return nil, fmt.Errorf("invalid header format: %q (header name cannot be empty)", header)
		}
		result[name] = value
	}
	return result, nil
}

func validateOptions(opts *Options) error {
	if opts.Port < 1 || opts.Port > 65535 {
		return fmt.Errorf("invalid port: %d (must be between 1 and 65535)", opts.Port)
	}

	if opts.Timeout < 1 {
		return fmt.Errorf("invalid timeout: %d (must be positive)", opts.Timeout)
	}

	if opts.TargetURL == "" {
		return fmt.Errorf("target URL cannot be empty")
	}

	_, err := url.Parse(opts.TargetURL)
	if err != nil {
		return fmt.Errorf("invalid target URL: %w", err)
	}

	return nil
}

func main() {
	opts, err := parseFlags()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		flag.Usage()
		os.Exit(1)
	}

	if err := validateOptions(opts); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		flag.Usage()
		os.Exit(1)
	}

	targetURL, err := url.Parse(opts.TargetURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing target URL: %v\n", err)
		os.Exit(1)
	}

	if targetURL.Scheme != "http" && targetURL.Scheme != "https" {
		fmt.Fprintf(os.Stderr, "Error: target URL must use http or https scheme\n")
		os.Exit(1)
	}

	logger := log.New(os.Stdout, "", log.LstdFlags)
	if !opts.Verbose {
		logger.SetOutput(io.Discard)
	}

	customHeaders, err := parseHeaders(opts.Headers)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing headers: %v\n", err)
		os.Exit(1)
	}

	config := ProxyConfig{
		ListenAddr:    fmt.Sprintf(":%d", opts.Port),
		TargetURL:     targetURL,
		Timeout:       time.Duration(opts.Timeout) * time.Second,
		CustomHeaders: customHeaders,
	}

	proxy, err := NewProxy(config, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating proxy: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Starting goreflector v%s\n", version)
	fmt.Printf("Listening on: http://0.0.0.0:%d\n", opts.Port)
	fmt.Printf("Proxying to:  %s\n", targetURL.String())

	if err := proxy.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting proxy: %v\n", err)
		os.Exit(1)
	}
}
