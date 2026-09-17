package curl

import (
	"testing"
)

func TestBuild(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		reqUrl   string
		headers  string
		body     string
		format   string
		location bool
		insecure bool
		verbose  bool
		proxy    string
		want     string
	}{
		{
			name:     "basic GET request",
			method:   "GET",
			reqUrl:   "https://example.com/api",
			headers:  "",
			body:     "",
			format:   "form",
			location: false,
			insecure: false,
			verbose:  false,
			proxy:    "",
			want:     "curl -X GET 'https://example.com/api'",
		},
		{
			name:     "all options enabled",
			method:   "POST",
			reqUrl:   "https://example.com/login",
			headers:  "Content-Type: application/json",
			body:     `{"user":"test"}`,
			format:   "json",
			location: true,
			insecure: true,
			verbose:  true,
			proxy:    "http://proxy.example.com:8080",
			want:     "curl -X POST 'https://example.com/login' -L -k -v -x 'http://proxy.example.com:8080' -H 'Content-Type: application/json' -d '{\"user\":\"test\"}'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Build(tt.method, tt.reqUrl, tt.headers, tt.body, tt.format, tt.location, tt.insecure, tt.verbose, tt.proxy)
			if got != tt.want {
				t.Errorf("Build() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParse(t *testing.T) {
	cmdStr := "curl -X POST 'https://example.com/data' -L -k -v -x 'http://proxy.internal:3128' -H 'Accept: application/json' -d '{\"foo\":\"bar\"}'"
	opts, err := Parse(cmdStr)
	if err != nil {
		t.Fatalf("Parse() returned error: %v", err)
	}

	if opts.Method != "POST" {
		t.Errorf("Method = %v, want POST", opts.Method)
	}
	if opts.URL != "https://example.com/data" {
		t.Errorf("URL = %v, want https://example.com/data", opts.URL)
	}
	if !opts.Location {
		t.Errorf("Location = false, want true")
	}
	if !opts.Insecure {
		t.Errorf("Insecure = false, want true")
	}
	if !opts.Verbose {
		t.Errorf("Verbose = false, want true")
	}
	if opts.Proxy != "http://proxy.internal:3128" {
		t.Errorf("Proxy = %v, want http://proxy.internal:3128", opts.Proxy)
	}
}
