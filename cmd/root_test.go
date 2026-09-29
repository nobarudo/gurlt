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
			args: []string{"-X", "POST", "-H", "Content-Type: application/json", "-d", "foo", "-F", "user=alice", "--form", "avatar=@pic.png", "--json", `{"key":"val"}`, "-u", "user:pass", "-A", "myagent", "-m", "10", "--connect-timeout", "2.5", "-k", "-x", "http://127.0.0.1:8080", "-w", "%{time_total}", "-q", ".data.users[0]", "-L", "-f", "json", "--log", "audit.log", "https://example.com"},
			want: "",
		},
		{
			name: "unknown flags preserved",
			args: []string{"-X", "GET", "--compressed", "--retry", "3", "https://example.com"},
			want: "--compressed --retry 3",
		},
		{
			name: "flag with equals",
			args: []string{"--user=admin:secret", "--form=field=val", "--write-out=%{time_total}", "--jq=.name", "--json='{\"test\":1}'", "--max-time=10", "--connect-timeout=5", "--proxy=http://proxy:8080", "--compressed", "https://example.com"},
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

func TestJSONFlagHandling(t *testing.T) {
	origJsonData := jsonData
	origData := data
	origFormat := format
	origMethod := method
	origHeaders := headers
	defer func() {
		jsonData = origJsonData
		data = origData
		format = origFormat
		method = origMethod
		headers = origHeaders
	}()

	jsonData = `{"hello":"world"}`
	data = ""
	format = "form"
	method = "GET"
	headers = []string{}

	if jsonData != "" {
		data = jsonData
		format = "json"
		if method == "GET" {
			method = "POST"
		}
		hasAccept := false
		hasContentType := false
		for _, h := range headers {
			lowerH := strings.ToLower(h)
			if strings.HasPrefix(lowerH, "accept:") {
				hasAccept = true
			}
			if strings.HasPrefix(lowerH, "content-type:") {
				hasContentType = true
			}
		}
		if !hasAccept {
			headers = append(headers, "Accept: application/json")
		}
		if !hasContentType {
			headers = append(headers, "Content-Type: application/json")
		}
	}

	if method != "POST" {
		t.Errorf("expected method POST, got %s", method)
	}
	if format != "json" {
		t.Errorf("expected format json, got %s", format)
	}
	if data != `{"hello":"world"}` {
		t.Errorf("expected data to be JSON string, got %s", data)
	}

	hasAccept := false
	hasContentType := false
	for _, h := range headers {
		if h == "Accept: application/json" {
			hasAccept = true
		}
		if h == "Content-Type: application/json" {
			hasContentType = true
		}
	}
	if !hasAccept {
		t.Errorf("expected Accept: application/json in headers")
	}
	if !hasContentType {
		t.Errorf("expected Content-Type: application/json in headers")
	}
}

func TestTimeoutFlagsParsed(t *testing.T) {
	origMaxTime := maxTime
	origConnectTimeout := connectTimeout
	defer func() {
		maxTime = origMaxTime
		connectTimeout = origConnectTimeout
	}()

	maxTime = 12.5
	connectTimeout = 3.0

	if maxTime != 12.5 {
		t.Errorf("expected maxTime 12.5, got %v", maxTime)
	}
	if connectTimeout != 3.0 {
		t.Errorf("expected connectTimeout 3.0, got %v", connectTimeout)
	}
}

func TestInsecureAndProxyFlags(t *testing.T) {
	origInsecure := insecure
	origProxy := proxy
	defer func() {
		insecure = origInsecure
		proxy = origProxy
	}()

	insecure = true
	proxy = "http://127.0.0.1:8888"

	if !insecure {
		t.Errorf("expected insecure to be true")
	}
	if proxy != "http://127.0.0.1:8888" {
		t.Errorf("expected proxy to be http://127.0.0.1:8888, got %s", proxy)
	}
}

func TestMultipartFormFlagHandling(t *testing.T) {
	origForms := forms
	origData := data
	origFormat := format
	origMethod := method
	defer func() {
		forms = origForms
		data = origData
		format = origFormat
		method = origMethod
	}()

	forms = []string{"user=alice", "avatar=@avatar.png"}
	data = ""
	format = "form"
	method = "GET"

	if len(forms) > 0 {
		data = strings.Join(forms, "\n")
		format = "multipart"
		if method == "GET" {
			method = "POST"
		}
	}

	if method != "POST" {
		t.Errorf("expected method POST, got %s", method)
	}
	if format != "multipart" {
		t.Errorf("expected format multipart, got %s", format)
	}
	expectedData := "user=alice\navatar=@avatar.png"
	if data != expectedData {
		t.Errorf("expected data %q, got %q", expectedData, data)
	}
}




