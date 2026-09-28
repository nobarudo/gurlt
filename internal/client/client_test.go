package client

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestSendWithMaxTime(t *testing.T) {
	// 500ms スリープするテストサーバー
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	// 1. タイムアウト 100ms (0.1s) で実行 -> エラーになるべき
	optsTimeout := RequestOptions{
		Method:  "GET",
		URL:     server.URL,
		MaxTime: 0.1,
	}
	resTimeout := Send(optsTimeout)
	if resTimeout.Err == nil {
		t.Errorf("expected timeout error, but got nil")
	}

	// 2. タイムアウト 2s で実行 -> 成功するべき
	optsSuccess := RequestOptions{
		Method:  "GET",
		URL:     server.URL,
		MaxTime: 2.0,
	}
	resSuccess := Send(optsSuccess)
	if resSuccess.Err != nil {
		t.Errorf("expected success, got error: %v", resSuccess.Err)
	}
	if resSuccess.Body != "OK" {
		t.Errorf("expected body 'OK', got %q", resSuccess.Body)
	}
}

func TestSendMultipart(t *testing.T) {
	// 一時ファイル作成
	tmpFile, err := os.CreateTemp("", "gurlt-test-*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString("file content test")
	tmpFile.Close()

	var receivedField string
	var receivedFileContent string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseMultipartForm(10 << 20)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		receivedField = r.FormValue("user")

		file, _, err := r.FormFile("upload")
		if err == nil {
			defer file.Close()
			buf := new(strings.Builder)
			io.Copy(buf, file)
			receivedFileContent = buf.String()
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("UPLOADED"))
	}))
	defer server.Close()

	opts := RequestOptions{
		Method:  "POST",
		URL:     server.URL,
		Format:  "multipart",
		Body:    fmt.Sprintf("user=alice\nupload=@%s", tmpFile.Name()),
		MaxTime: 2.0,
	}

	res := Send(opts)
	if res.Err != nil {
		t.Fatalf("Send returned error: %v", res.Err)
	}
	if res.Body != "UPLOADED" {
		t.Errorf("expected body 'UPLOADED', got %s", res.Body)
	}
	if receivedField != "alice" {
		t.Errorf("expected form field 'alice', got %s", receivedField)
	}
	if receivedFileContent != "file content test" {
		t.Errorf("expected file content 'file content test', got %s", receivedFileContent)
	}
}

