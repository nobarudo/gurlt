package tui

import (
	"strings"
	"testing"
)

func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name    string
		headers string
		want    string
	}{
		{
			name:    "empty headers",
			headers: "",
			want:    "",
		},
		{
			name:    "no auth header",
			headers: "Content-Type: application/json\nAccept: */*",
			want:    "",
		},
		{
			name:    "basic auth header",
			headers: "Authorization: Basic YWRtaW46c2VjcmV0",
			want:    "",
		},
		{
			name:    "standard bearer token",
			headers: "Content-Type: application/json\nAuthorization: Bearer my-secret-token\nAccept: */*",
			want:    "my-secret-token",
		},
		{
			name:    "lowercase bearer header with spaces",
			headers: "authorization: bearer   token-xyz  ",
			want:    "token-xyz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractBearerToken(tt.headers)
			if got != tt.want {
				t.Errorf("extractBearerToken() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSetOrUpdateBearerHeader(t *testing.T) {
	// 1. Add bearer token to empty headers
	got := setOrUpdateBearerHeader("", "token123")
	if got != "Authorization: Bearer token123" {
		t.Errorf("expected 'Authorization: Bearer token123', got %q", got)
	}

	// 2. Add bearer token to existing headers
	initial := "Content-Type: application/json\nAccept: application/json"
	got = setOrUpdateBearerHeader(initial, "token123")
	expected := "Content-Type: application/json\nAccept: application/json\nAuthorization: Bearer token123"
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}

	// 3. Update existing bearer token
	updated := setOrUpdateBearerHeader(got, "new-token-456")
	expectedUpdated := "Content-Type: application/json\nAccept: application/json\nAuthorization: Bearer new-token-456"
	if updated != expectedUpdated {
		t.Errorf("got %q, want %q", updated, expectedUpdated)
	}

	// 4. Remove bearer token by setting empty
	cleared := setOrUpdateBearerHeader(updated, "")
	if strings.Contains(cleared, "Authorization") {
		t.Errorf("expected Authorization header to be removed, got %q", cleared)
	}
	if cleared != initial {
		t.Errorf("got %q, want %q", cleared, initial)
	}

	// 5. Basic auth should not be touched
	basic := "Authorization: Basic YWRtaW46c2VjcmV0\nContent-Type: text/plain"
	clearedBasic := setOrUpdateBearerHeader(basic, "")
	if !strings.Contains(clearedBasic, "Authorization: Basic") {
		t.Errorf("expected Basic auth to remain untouched, got %q", clearedBasic)
	}
}
