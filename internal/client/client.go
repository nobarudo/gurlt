package client

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

// Result はHTTPリクエストの結果を格納する構造体です
type Result struct {
	Status   string
	Body     string
	FullDump string // ▼ 追加：すべての通信履歴をまとめた文字列
	Err      error
}

// ▼ 追加：すべての通信をフックしてダンプを記録するスパイ（Transport）
type dumpTransport struct {
	Transport http.RoundTripper
	ChainDump strings.Builder
}

func (d *dumpTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// 1. リクエストをダンプして記録
	reqBytes, _ := httputil.DumpRequestOut(req, true)
	d.ChainDump.WriteString(fmt.Sprintf("=== Request ===\n%s\n%s\n", req.URL.String(), string(reqBytes)))

	// 2. 実際の通信を実行
	res, err := d.Transport.RoundTrip(req)
	if err != nil {
		return res, err
	}

	// 3. レスポンスをダンプして記録
	resBytes, _ := httputil.DumpResponse(res, true)
	d.ChainDump.WriteString(fmt.Sprintf("=== Response ===\n%s\n", string(resBytes)))

	return res, nil
}

// Send は実際にHTTPリクエストを送信します
func Send(method, reqUrl, headers, body, format string, location bool) Result {
	var reqBody io.Reader
	if body != "" {
		if format == "json" {
			reqBody = strings.NewReader(body)
		} else {
			form := url.Values{}
			for _, line := range strings.Split(body, "\n") {
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					form.Add(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
				}
			}
			reqBody = strings.NewReader(form.Encode())
		}
	}

	req, err := http.NewRequest(method, reqUrl, reqBody)
	if err != nil {
		return Result{Err: err}
	}

	for _, line := range strings.Split(headers, "\n") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			req.Header.Add(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
		}
	}

	// ▼ 変更：カスタムTransportをクライアントにセットする
	dt := &dumpTransport{
		Transport: http.DefaultTransport,
	}
	httpClient := &http.Client{
		Timeout:   10 * time.Second,
		Transport: dt, // ここでスパイを仕掛ける
	}

	if !location {
		// リダイレクトを追従しない設定
		httpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return Result{Err: err}
	}
	defer res.Body.Close()

	bodyBytes, _ := io.ReadAll(res.Body)

	return Result{
		Status:   res.Status,
		Body:     string(bodyBytes),
		FullDump: dt.ChainDump.String(), // 記録したすべての履歴を返す
	}
}
