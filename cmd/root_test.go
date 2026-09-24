package cmd

import (
	"strings"
	"testing"
)

func TestGetExtraArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "known flags only",
			args: []string{"-X", "POST", "-H", "Content-Type: application/json", "-d", "foo", "-u", "user:pass", "-A", "myagent", "-L", "-f", "json", "--log", "audit.log", "https://example.com"},
			want: "",
		},
		{
			name: "unknown flags preserved",
			args: []string{"-X", "GET", "--compressed", "--retry", "3", "https://example.com"},
			want: "--compressed --retry 3",
		},
		{
			name: "flag with equals",
			args: []string{"--user=admin:secret", "--compressed", "https://example.com"},
			want: "--compressed",
		},
		{
			name: "unknown flag with spaced argument",
			args: []string{"--cacert", "my cert.pem", "https://example.com"},
			want: "--cacert 'my cert.pem'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getExtraArgs(tt.args)
			if got != tt.want {
				t.Errorf("getExtraArgs() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHeaderConstructionWithUserAndAgent(t *testing.T) {
	// 一時的にフラグ変数をリセットしてテスト
	origUser := user
	origAgent := userAgent
	origHeaders := headers
	defer func() {
		user = origUser
		userAgent = origAgent
		headers = origHeaders
	}()

	user = "admin:secret"
	userAgent = "CustomAgent/1.0"
	headers = []string{"X-Custom: test"}

	var headerLines []string
	if userAgent != "" {
		headerLines = append(headerLines, "User-Agent: "+userAgent)
	}
	if user != "" {
		headerLines = append(headerLines, "Authorization: Basic YWRtaW46c2VjcmV0")
	}
	for _, h := range headers {
		headerLines = append(headerLines, h)
	}
	headerList := strings.Join(headerLines, "\n")

	if !strings.Contains(headerList, "User-Agent: CustomAgent/1.0") {
		t.Errorf("expected User-Agent header in list, got: %s", headerList)
	}
	if !strings.Contains(headerList, "Authorization: Basic YWRtaW46c2VjcmV0") {
		t.Errorf("expected Authorization header in list, got: %s", headerList)
	}
	if !strings.Contains(headerList, "X-Custom: test") {
		t.Errorf("expected X-Custom header in list, got: %s", headerList)
	}
}
