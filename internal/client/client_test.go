package client

import (
	"net/http"
	"net/http/httptest"
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
