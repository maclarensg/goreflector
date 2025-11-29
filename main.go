package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"time"
)

const version = "1.0.0"

type Options struct {
	Port        int
	TargetURL   string
	Timeout     int
	Verbose     bool
	ShowVersion bool
}

func parseFlags() (*Options, error) {
	opts := &Options{}

	flag.IntVar(&opts.Port, "p", 8080, "Port to listen on")
	flag.IntVar(&opts.Port, "port", 8080, "Port to listen on")
	flag.IntVar(&opts.Timeout, "t", 30, "Request timeout in seconds")
	flag.IntVar(&opts.Timeout, "timeout", 30, "Request timeout in seconds")
	flag.BoolVar(&opts.Verbose, "v", false, "Verbose logging")
	flag.BoolVar(&opts.Verbose, "verbose", false, "Verbose logging")
	flag.BoolVar(&opts.ShowVersion, "version", false, "Show version")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "goreflector v%s - HTTP reverse proxy\n\n", version)
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <target-url>\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  %s -p 8080 https://example.com\n", os.Args[0])
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

	return opts, nil
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

	config := ProxyConfig{
		ListenAddr: fmt.Sprintf(":%d", opts.Port),
		TargetURL:  targetURL,
		Timeout:    time.Duration(opts.Timeout) * time.Second,
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
