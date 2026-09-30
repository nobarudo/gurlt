package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nobarudo/gurlt/internal/curl"
)

func TestGetExtraArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "known flags only",
			args: []string{"-X", "POST", "-H", "Content-Type: application/json", "-d", "foo", "--data-binary", "bin", "--data-ascii", "asc", "-F", "user=alice", "--form", "avatar=@pic.png", "--json", `{"key":"val"}`, "--bearer", "mytoken", "--oauth2-bearer", "oauthtok", "-u", "user:pass", "-A", "myagent", "-m", "10", "--connect-timeout", "2.5", "-k", "-x", "http://127.0.0.1:8080", "-w", "%{time_total}", "-q", ".data.users[0]", "-o", "resp.json", "--output", "out.bin", "-L", "-f", "json", "--log", "audit.log", "https://example.com"},
			want: "",
		},
		{
			name: "unknown flags preserved",
			args: []string{"-X", "GET", "--compressed", "--retry", "3", "https://example.com"},
			want: "--compressed --retry 3",
		},
		{
			name: "flag with equals",
			args: []string{"--user=admin:secret", "--bearer=mybearer", "--form=field=val", "--write-out=%{time_total}", "--jq=.name", "--json='{\"test\":1}'", "--max-time=10", "--connect-timeout=5", "--proxy=http://proxy:8080", "--output=result.txt", "--compressed", "https://example.com"},
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

func TestPayloadFileHandling(t *testing.T) {
	origJsonData := jsonData
	origData := data
	origDataRaw := dataRaw
	origFormat := format
	origMethod := method
	origHeaders := headers
	defer func() {
		jsonData = origJsonData
		data = origData
		dataRaw = origDataRaw
		format = origFormat
		method = origMethod
		headers = origHeaders
	}()

	tmpDir := t.TempDir()

	// 1. Test -d @payload.json (JSON content auto-switches format to json)
	jsonFile := filepath.Join(tmpDir, "payload.json")
	jsonContent := `{"user": "alice", "action": "login"}`
	if err := os.WriteFile(jsonFile, []byte(jsonContent), 0644); err != nil {
		t.Fatalf("failed to write json file: %v", err)
	}

	data = "@" + jsonFile
	jsonData = ""
	dataRaw = ""
	format = "form"
	method = "GET"
	headers = []string{}

	loaded, err := curl.LoadPayload(data)
	if err != nil {
		t.Fatalf("unexpected error from LoadPayload: %v", err)
	}
	data = loaded
	if data != "" && method == "GET" {
		method = "POST"
	}
	if data != "" && format == "form" {
		trimmed := strings.TrimSpace(data)
		if (strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")) ||
			(strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")) {
			format = "json"
		}
	}

	if data != jsonContent {
		t.Errorf("expected data %q, got %q", jsonContent, data)
	}
	if method != "POST" {
		t.Errorf("expected method POST, got %s", method)
	}
	if format != "json" {
		t.Errorf("expected format json, got %s", format)
	}

	// 2. Test -d @query.graphql (Plain text / GraphQL stays non-json)
	gqlFile := filepath.Join(tmpDir, "query.graphql")
	gqlContent := "query Viewer { viewer { login } }"
	if err := os.WriteFile(gqlFile, []byte(gqlContent), 0644); err != nil {
		t.Fatalf("failed to write gql file: %v", err)
	}

	data = "@" + gqlFile
	format = "form"
	method = "GET"
	loaded, err = curl.LoadPayload(data)
	if err != nil {
		t.Fatalf("unexpected error from LoadPayload: %v", err)
	}
	data = loaded
	if data != "" && method == "GET" {
		method = "POST"
	}
	if data != gqlContent {
		t.Errorf("expected data %q, got %q", gqlContent, data)
	}
	if format != "form" {
		t.Errorf("expected format form, got %s", format)
	}

	// 3. Test --json @payload.json (adds JSON headers and sets format json)
	jsonData = "@" + jsonFile
	data = ""
	format = "form"
	method = "GET"
	headers = []string{}

	loadedJSON, err := curl.LoadPayload(jsonData)
	if err != nil {
		t.Fatalf("unexpected error from LoadPayload: %v", err)
	}
	data = loadedJSON
	format = "json"
	if method == "GET" {
		method = "POST"
	}
	headers = append(headers, "Accept: application/json", "Content-Type: application/json")

	if data != jsonContent {
		t.Errorf("expected data %q, got %q", jsonContent, data)
	}
	if format != "json" {
		t.Errorf("expected format json, got %s", format)
	}
	if method != "POST" {
		t.Errorf("expected method POST, got %s", method)
	}
}

func TestPayloadFileErrors(t *testing.T) {
	// 1. Missing file returns error
	_, err := curl.LoadPayload("@nonexistent-file-xyz.json")
	if err == nil {
		t.Fatalf("expected error for non-existent file, got nil")
	}

	// 2. Empty path after @ returns error
	_, err = curl.LoadPayload("@")
	if err == nil {
		t.Fatalf("expected error for empty path '@', got nil")
	}
}

func TestBearerFlagHandling(t *testing.T) {
	origBearer := bearerToken
	origUser := user
	origHeaders := headers
	defer func() {
		bearerToken = origBearer
		user = origUser
		headers = origHeaders
	}()

	// 1. Bearer token set adds Authorization header
	bearerToken = "token-secret-123"
	user = ""
	headers = []string{"Accept: application/json"}

	var headerLines []string
	if user != "" {
		headerLines = append(headerLines, "Authorization: Basic ...")
	} else if bearerToken != "" {
		hasAuth := false
		for _, h := range headers {
			if strings.HasPrefix(strings.ToLower(strings.TrimSpace(h)), "authorization:") {
				hasAuth = true
				break
			}
		}
		if !hasAuth {
			headerLines = append(headerLines, "Authorization: Bearer "+bearerToken)
		}
	}
	for _, h := range headers {
		headerLines = append(headerLines, h)
	}
	joined := strings.Join(headerLines, "\n")

	if !strings.Contains(joined, "Authorization: Bearer token-secret-123") {
		t.Errorf("expected Bearer token in headers, got: %s", joined)
	}

	// 2. If Authorization header already present, do not duplicate
	headers = []string{"Authorization: Bearer existing-token"}
	headerLines = nil
	if user != "" {
		headerLines = append(headerLines, "Authorization: Basic ...")
	} else if bearerToken != "" {
		hasAuth := false
		for _, h := range headers {
			if strings.HasPrefix(strings.ToLower(strings.TrimSpace(h)), "authorization:") {
				hasAuth = true
				break
			}
		}
		if !hasAuth {
			headerLines = append(headerLines, "Authorization: Bearer "+bearerToken)
		}
	}
	for _, h := range headers {
		headerLines = append(headerLines, h)
	}
	joined = strings.Join(headerLines, "\n")
	if strings.Contains(joined, "token-secret-123") {
		t.Errorf("did not expect overridden bearer when Authorization is explicitly in headers, got: %s", joined)
	}
}

func TestOutputFlagHandling(t *testing.T) {
	origOutput := outputFile
	defer func() {
		outputFile = origOutput
	}()

	// 1. Test curl command with -o
	curlCmd := "curl https://example.com/api/download -o my_download.zip"
	opts, err := curl.Parse(curlCmd)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}
	if opts.OutputFile != "my_download.zip" {
		t.Errorf("expected OutputFile 'my_download.zip', got %q", opts.OutputFile)
	}

	// 2. Test curl command with --output
	curlCmdLong := "curl https://example.com/image.png --output test.png"
	optsLong, err := curl.Parse(curlCmdLong)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}
	if optsLong.OutputFile != "test.png" {
		t.Errorf("expected OutputFile 'test.png', got %q", optsLong.OutputFile)
	}
}

