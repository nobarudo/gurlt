package client

import (
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

// Result はHTTPリクエストの結果を格納する構造体です
type Result struct {
	Status  string
	Body    string
	ReqDump string
	ResDump string
	Err     error
}

// Send は実際にHTTPリクエストを送信し、結果とダンプデータを返します
func Send(method, reqUrl, headers, body, format string) Result {
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

	reqBytes, _ := httputil.DumpRequestOut(req, true)

	httpClient := &http.Client{Timeout: 10 * time.Second}
	res, err := httpClient.Do(req)
	if err != nil {
		return Result{Err: err}
	}
	defer res.Body.Close()

	resBytes, _ := httputil.DumpResponse(res, true)
	bodyBytes, _ := io.ReadAll(res.Body)

	return Result{
		Status:  res.Status,
		Body:    string(bodyBytes),
		ReqDump: string(reqBytes),
		ResDump: string(resBytes),
	}
}
