package jsonpath

import (
	"strings"
	"testing"
)

const sampleJSON = `{
  "status": "success",
  "data": {
    "users": [
      {"id": 1, "name": "Alice", "role": "admin"},
      {"id": 2, "name": "Bob", "role": "user"},
      {"id": 3, "name": "Charlie", "role": "guest"}
    ],
    "count": 3,
    "header-info": {
      "content-type": "application/json"
    }
  }
}`

func TestQuery(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		want    string
		wantErr bool
	}{
		{
			name: "root dot",
			path: ".",
			want: "success", // check contains
		},
		{
			name: "top-level property",
			path: ".status",
			want: "success",
		},
		{
			name: "top-level without leading dot",
			path: "status",
			want: "success",
		},
		{
			name: "nested property",
			path: ".data.count",
			want: "3",
		},
		{
			name: "array index with brackets",
			path: ".data.users[0].name",
			want: "Alice",
		},
		{
			name: "array index with dot notation",
			path: "data.users.1.name",
			want: "Bob",
		},
		{
			name: "negative array index",
			path: ".data.users[-1].name",
			want: "Charlie",
		},
		{
			name: "wildcard array projection",
			path: ".data.users[*].name",
			want: "Alice", // contains Alice, Bob, Charlie
		},
		{
			name: "empty bracket wildcard",
			path: ".data.users[].role",
			want: "admin",
		},
		{
			name: "quoted key with hyphen",
			path: `.data["header-info"]["content-type"]`,
			want: "application/json",
		},
		{
			name: "array length helper",
			path: ".data.users.length",
			want: "3",
		},
		{
			name:    "key not found",
			path:    ".data.nonexistent",
			wantErr: true,
		},
		{
			name:    "index out of bounds",
			path:    ".data.users[99]",
			wantErr: true,
		},
		{
			name:    "invalid property on primitive",
			path:    ".status.something",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := QueryPretty(sampleJSON, tt.path)
			if (err != nil) != tt.wantErr {
				t.Fatalf("QueryPretty() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && !strings.Contains(got, tt.want) {
				t.Errorf("QueryPretty() = %q, want containing %q", got, tt.want)
			}
		})
	}
}

func TestQueryInvalidJSON(t *testing.T) {
	_, err := Query("invalid json", ".name")
	if err == nil {
		t.Errorf("expected error for invalid JSON")
	}

	_, err = Query("", ".name")
	if err == nil {
		t.Errorf("expected error for empty JSON")
	}
}

func TestQueryPrettyPrimitives(t *testing.T) {
	jsonStr := `{"str": "hello", "num": 42, "flag": true, "nullVal": null}`

	s, err := QueryPretty(jsonStr, ".str")
	if err != nil || s != "hello" {
		t.Errorf("expected string 'hello', got %q, err %v", s, err)
	}

	n, err := QueryPretty(jsonStr, ".num")
	if err != nil || n != "42" {
		t.Errorf("expected number '42', got %q, err %v", n, err)
	}

	b, err := QueryPretty(jsonStr, ".flag")
	if err != nil || b != "true" {
		t.Errorf("expected bool 'true', got %q, err %v", b, err)
	}

	nullV, err := QueryPretty(jsonStr, ".nullVal")
	if err != nil || nullV != "null" {
		t.Errorf("expected 'null', got %q, err %v", nullV, err)
	}
}
