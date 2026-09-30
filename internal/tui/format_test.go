package tui

import (
	"testing"
)

func TestIsBinaryContent(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want bool
	}{
		{
			name: "empty data",
			data: []byte{},
			want: false,
		},
		{
			name: "plain text",
			data: []byte("Hello, world!\nThis is plain text with no null bytes."),
			want: false,
		},
		{
			name: "json text",
			data: []byte(`{"status":"ok","items":[1,2,3]}`),
			want: false,
		},
		{
			name: "binary data with null byte early",
			data: []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00},
			want: true,
		},
		{
			name: "binary data with null byte at limit",
			data: append(make([]byte, 100), 0x00),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isBinaryContent(tt.data)
			if got != tt.want {
				t.Errorf("isBinaryContent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes int
		want  string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{104857600, "100.0 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tt := range tests {
		got := formatBytes(tt.bytes)
		if got != tt.want {
			t.Errorf("formatBytes(%d) = %q, want %q", tt.bytes, got, tt.want)
		}
	}
}

func TestSanitizeFullDump(t *testing.T) {
	inputDump := "=== Request ===\nGET /img HTTP/1.1\n\n=== Response ===\nHTTP/1.1 200 OK\r\nContent-Type: image/png\r\n\r\n\x89PNG\x00\x00\x00\n⏱️  Latency: 15ms\n  DNS: 5ms"
	got := sanitizeFullDump(inputDump, "[Binary data: 1.0 KB (1024 bytes)]")
	expected := "=== Request ===\nGET /img HTTP/1.1\n\n=== Response ===\nHTTP/1.1 200 OK\r\nContent-Type: image/png\r\n\r\n[Binary data: 1.0 KB (1024 bytes)]\n\n⏱️  Latency: 15ms\n  DNS: 5ms"

	if got != expected {
		t.Errorf("sanitizeFullDump() =\n%q\nwant:\n%q", got, expected)
	}
}

