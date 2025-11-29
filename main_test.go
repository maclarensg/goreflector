package main

import (
	"flag"
	"os"
	"testing"
)

func TestValidateOptions(t *testing.T) {
	tests := []struct {
		name          string
		opts          *Options
		expectError   bool
		errorContains string
	}{
		{
			name: "valid options",
			opts: &Options{
				Port:      8080,
				TargetURL: "https://example.com",
				Timeout:   30,
			},
			expectError: false,
		},
		{
			name: "port too low",
			opts: &Options{
				Port:      0,
				TargetURL: "https://example.com",
				Timeout:   30,
			},
			expectError:   true,
			errorContains: "invalid port",
		},
		{
			name: "port too high",
			opts: &Options{
				Port:      65536,
				TargetURL: "https://example.com",
				Timeout:   30,
			},
			expectError:   true,
			errorContains: "invalid port",
		},
		{
			name: "negative timeout",
			opts: &Options{
				Port:      8080,
				TargetURL: "https://example.com",
				Timeout:   -1,
			},
			expectError:   true,
			errorContains: "invalid timeout",
		},
		{
			name: "zero timeout",
			opts: &Options{
				Port:      8080,
				TargetURL: "https://example.com",
				Timeout:   0,
			},
			expectError:   true,
			errorContains: "invalid timeout",
		},
		{
			name: "empty target URL",
			opts: &Options{
				Port:      8080,
				TargetURL: "",
				Timeout:   30,
			},
			expectError:   true,
			errorContains: "target URL cannot be empty",
		},
		{
			name: "invalid URL",
			opts: &Options{
				Port:      8080,
				TargetURL: "://invalid-url",
				Timeout:   30,
			},
			expectError:   true,
			errorContains: "invalid target URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateOptions(tt.opts)

			if tt.expectError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.expectError && err != nil && tt.errorContains != "" {
				if !contains(err.Error(), tt.errorContains) {
					t.Errorf("expected error to contain '%s', got '%s'", tt.errorContains, err.Error())
				}
			}
		})
	}
}

func TestParseFlagsNoArgs(t *testing.T) {
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	}()

	os.Args = []string{"goreflector"}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	_, err := parseFlags()
	if err == nil {
		t.Error("expected error when no target URL provided")
	}
	if !contains(err.Error(), "target URL is required") {
		t.Errorf("expected 'target URL is required' error, got %v", err)
	}
}

func TestParseFlagsBasic(t *testing.T) {
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	}()

	os.Args = []string{"goreflector", "https://example.com"}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	opts, err := parseFlags()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if opts.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", opts.Port)
	}
	if opts.TargetURL != "https://example.com" {
		t.Errorf("expected target URL 'https://example.com', got %s", opts.TargetURL)
	}
	if opts.Timeout != 30 {
		t.Errorf("expected default timeout 30, got %d", opts.Timeout)
	}
	if opts.Verbose {
		t.Error("expected verbose to be false")
	}
}

func TestParseFlagsWithPort(t *testing.T) {
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	}()

	os.Args = []string{"goreflector", "-p", "9090", "https://example.com"}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	opts, err := parseFlags()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if opts.Port != 9090 {
		t.Errorf("expected port 9090, got %d", opts.Port)
	}
}

func TestParseFlagsWithLongPort(t *testing.T) {
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	}()

	os.Args = []string{"goreflector", "--port", "9090", "https://example.com"}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	opts, err := parseFlags()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if opts.Port != 9090 {
		t.Errorf("expected port 9090, got %d", opts.Port)
	}
}

func TestParseFlagsWithTimeout(t *testing.T) {
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	}()

	os.Args = []string{"goreflector", "-t", "60", "https://example.com"}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	opts, err := parseFlags()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if opts.Timeout != 60 {
		t.Errorf("expected timeout 60, got %d", opts.Timeout)
	}
}

func TestParseFlagsWithVerbose(t *testing.T) {
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	}()

	os.Args = []string{"goreflector", "-v", "https://example.com"}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	opts, err := parseFlags()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !opts.Verbose {
		t.Error("expected verbose to be true")
	}
}

func TestParseFlagsWithAllOptions(t *testing.T) {
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	}()

	os.Args = []string{"goreflector", "-p", "9090", "-t", "60", "-v", "https://example.com"}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	opts, err := parseFlags()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if opts.Port != 9090 {
		t.Errorf("expected port 9090, got %d", opts.Port)
	}
	if opts.Timeout != 60 {
		t.Errorf("expected timeout 60, got %d", opts.Timeout)
	}
	if !opts.Verbose {
		t.Error("expected verbose to be true")
	}
	if opts.TargetURL != "https://example.com" {
		t.Errorf("expected target URL 'https://example.com', got %s", opts.TargetURL)
	}
}

func TestValidOptionsEndToEnd(t *testing.T) {
	tests := []struct {
		name    string
		opts    *Options
		wantErr bool
	}{
		{
			name: "http target",
			opts: &Options{
				Port:      8080,
				TargetURL: "http://example.com",
				Timeout:   30,
			},
			wantErr: false,
		},
		{
			name: "https target",
			opts: &Options{
				Port:      8080,
				TargetURL: "https://example.com",
				Timeout:   30,
			},
			wantErr: false,
		},
		{
			name: "target with path",
			opts: &Options{
				Port:      8080,
				TargetURL: "https://example.com/api/v1",
				Timeout:   30,
			},
			wantErr: false,
		},
		{
			name: "target with port",
			opts: &Options{
				Port:      8080,
				TargetURL: "https://example.com:8443",
				Timeout:   30,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateOptions(tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateOptions() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
