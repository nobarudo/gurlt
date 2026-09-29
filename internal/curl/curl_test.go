package curl

import (
	"testing"
)

func TestBuild(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		reqUrl         string
		headers        string
		body           string
		format         string
		location       bool
		insecure       bool
		verbose        bool
		proxy          string
		maxTime        float64
		connectTimeout float64
		want           string
	}{
		{
			name:           "basic GET request",
			method:         "GET",
			reqUrl:         "https://example.com/api",
			headers:        "",
			body:           "",
			format:         "form",
			location:       false,
			insecure:       false,
			verbose:        false,
			proxy:          "",
			maxTime:        0,
			connectTimeout: 0,
			want:           "curl -X GET 'https://example.com/api'",
		},
		{
			name:           "all options enabled with timeout",
			method:         "POST",
			reqUrl:         "https://example.com/login",
			headers:        "Content-Type: application/json",
			body:           `{"user":"test"}`,
			format:         "json",
			location:       true,
			insecure:       true,
			verbose:        true,
			proxy:          "http://proxy.example.com:8080",
			maxTime:        10.5,
			connectTimeout: 3,
			want:           "curl -X POST 'https://example.com/login' -L -k -v -x 'http://proxy.example.com:8080' -m 10.5 --connect-timeout 3 -H 'Content-Type: application/json' -d '{\"user\":\"test\"}'",
		},
		{
			name:           "multipart form request",
			method:         "POST",
			reqUrl:         "https://example.com/upload",
			headers:        "Authorization: Bearer token",
			body:           "name=alice\nfile=@test.png",
			format:         "multipart",
			location:       false,
			insecure:       false,
			verbose:        false,
			proxy:          "",
			maxTime:        0,
			connectTimeout: 0,
			want:           "curl -X POST 'https://example.com/upload' -H 'Authorization: Bearer token' -F 'name=alice' -F 'file=@test.png'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Build(tt.method, tt.reqUrl, tt.headers, tt.body, tt.format, tt.location, tt.insecure, tt.verbose, tt.proxy, tt.maxTime, tt.connectTimeout)
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

func TestParseWithUserAndUserAgent(t *testing.T) {
	cmdStr := "curl 'https://example.com' -u admin:secret -A 'MyCustomAgent/1.0'"
	opts, err := Parse(cmdStr)
	if err != nil {
		t.Fatalf("Parse() returned error: %v", err)
	}

	if opts.User != "admin:secret" {
		t.Errorf("User = %v, want admin:secret", opts.User)
	}
	if opts.UserAgent != "MyCustomAgent/1.0" {
		t.Errorf("UserAgent = %v, want MyCustomAgent/1.0", opts.UserAgent)
	}
}

func TestParseWithJSON(t *testing.T) {
	cmdStr := `curl 'https://example.com/api' --json '{"name":"gurlt","count":42}'`
	opts, err := Parse(cmdStr)
	if err != nil {
		t.Fatalf("Parse() returned error: %v", err)
	}

	if opts.Method != "POST" {
		t.Errorf("Method = %v, want POST", opts.Method)
	}
	if opts.Body != `{"name":"gurlt","count":42}` {
		t.Errorf("Body = %v, want %v", opts.Body, `{"name":"gurlt","count":42}`)
	}
	hasAccept := false
	hasContentType := false
	for _, h := range opts.Headers {
		if h == "Accept: application/json" {
			hasAccept = true
		}
		if h == "Content-Type: application/json" {
			hasContentType = true
		}
	}
	if !hasAccept {
		t.Errorf("expected Accept: application/json in headers, got %v", opts.Headers)
	}
	if !hasContentType {
		t.Errorf("expected Content-Type: application/json in headers, got %v", opts.Headers)
	}
}

func TestParseWithTimeout(t *testing.T) {
	cmdStr := "curl 'https://example.com' -m 15.5 --connect-timeout 4"
	opts, err := Parse(cmdStr)
	if err != nil {
		t.Fatalf("Parse() returned error: %v", err)
	}

	if opts.MaxTime != 15.5 {
		t.Errorf("MaxTime = %v, want 15.5", opts.MaxTime)
	}
	if opts.ConnectTimeout != 4 {
		t.Errorf("ConnectTimeout = %v, want 4", opts.ConnectTimeout)
	}
}

func TestParseWithMultipart(t *testing.T) {
	cmdStr := "curl 'https://example.com/api' -F 'user=bob' -F 'avatar=@profile.jpg'"
	opts, err := Parse(cmdStr)
	if err != nil {
		t.Fatalf("Parse() returned error: %v", err)
	}

	if opts.Method != "POST" {
		t.Errorf("Method = %v, want POST", opts.Method)
	}
	if !opts.IsMultipart {
		t.Errorf("expected IsMultipart to be true")
	}
	if len(opts.Forms) != 2 {
		t.Fatalf("expected 2 forms, got %d", len(opts.Forms))
	}
	if opts.Forms[0] != "user=bob" {
		t.Errorf("Forms[0] = %v, want user=bob", opts.Forms[0])
	}
	if opts.Forms[1] != "avatar=@profile.jpg" {
		t.Errorf("Forms[1] = %v, want avatar=@profile.jpg", opts.Forms[1])
	}
	expectedBody := "user=bob\navatar=@profile.jpg"
	if opts.Body != expectedBody {
		t.Errorf("Body = %q, want %q", opts.Body, expectedBody)
	}
}

func TestParseWithWriteOut(t *testing.T) {
	cmdStr := "curl 'https://example.com/api' -w '%{time_total}'"
	opts, err := Parse(cmdStr)
	if err != nil {
		t.Fatalf("Parse() returned error: %v", err)
	}

	if opts.URL != "https://example.com/api" {
		t.Errorf("URL = %v, want https://example.com/api", opts.URL)
	}
}

func TestParseWithBearer(t *testing.T) {
	// 1. --bearer token
	cmdStr := "curl 'https://api.example.com' --bearer 'token-123'"
	opts, err := Parse(cmdStr)
	if err != nil {
		t.Fatalf("Parse() returned error: %v", err)
	}
	if opts.Bearer != "token-123" {
		t.Errorf("Bearer = %q, want token-123", opts.Bearer)
	}
	hasBearerHeader := false
	for _, h := range opts.Headers {
		if h == "Authorization: Bearer token-123" {
			hasBearerHeader = true
		}
	}
	if !hasBearerHeader {
		t.Errorf("expected 'Authorization: Bearer token-123' in headers, got %v", opts.Headers)
	}

	// 2. --oauth2-bearer token
	cmdStrOAuth := "curl 'https://api.example.com' --oauth2-bearer 'oauth-456'"
	optsOAuth, err := Parse(cmdStrOAuth)
	if err != nil {
		t.Fatalf("Parse() returned error: %v", err)
	}
	if optsOAuth.Bearer != "oauth-456" {
		t.Errorf("Bearer = %q, want oauth-456", optsOAuth.Bearer)
	}
}
