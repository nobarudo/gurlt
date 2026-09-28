package client

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// RequestOptions はリクエスト実行時のオプションを保持します
type RequestOptions struct {
	Method         string
	URL            string
	Headers        string
	Body           string
	Format         string
	Location       bool
	Insecure       bool
	Proxy          string
	MaxTime        float64
	ConnectTimeout float64
}

// Result はHTTPリクエストの結果を格納する構造体です
type Result struct {
	Status   string
	Body     string
	FullDump string
	Err      error
	History  []HistoryEntry
}

type HistoryEntry struct {
	Method string
	URL    string
	Status string
}

type dumpTransport struct {
	Transport http.RoundTripper
	ChainDump strings.Builder
	History   []HistoryEntry
}

func (d *dumpTransport) RoundTrip(req *http.Request) (*http.Response, error) {

	reqBytes, _ := httputil.DumpRequestOut(req, true)
	d.ChainDump.WriteString(fmt.Sprintf("=== Request ===\n%s\n%s\n", req.URL.String(), string(reqBytes)))

	res, err := d.Transport.RoundTrip(req)
	if err != nil {
		return res, err
	}

	d.History = append(d.History, HistoryEntry{
		Method: req.Method,
		URL:    req.URL.String(),
		Status: res.Status,
	})

	resBytes, _ := httputil.DumpResponse(res, true)
	d.ChainDump.WriteString(fmt.Sprintf("=== Response ===\n%s\n", string(resBytes)))

	return res, nil
}

func Send(opts RequestOptions) Result {
	var reqBody io.Reader
	var history []HistoryEntry
	var multipartContentType string

	if opts.Body != "" {
		if opts.Format == "json" {
			reqBody = strings.NewReader(opts.Body)
		} else if opts.Format == "multipart" {
			var b bytes.Buffer
			w := multipart.NewWriter(&b)

			for _, line := range strings.Split(opts.Body, "\n") {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				parts := strings.SplitN(line, "=", 2)
				if len(parts) != 2 {
					continue
				}
				key := strings.TrimSpace(parts[0])
				val := strings.TrimSpace(parts[1])

				if strings.HasPrefix(val, "@") {
					filePath := strings.TrimPrefix(val, "@")
					file, err := os.Open(filePath)
					if err != nil {
						return Result{Err: fmt.Errorf("failed to open file %s: %w", filePath, err)}
					}
					part, err := w.CreateFormFile(key, filepath.Base(filePath))
					if err != nil {
						file.Close()
						return Result{Err: fmt.Errorf("failed to create form file: %w", err)}
					}
					if _, err := io.Copy(part, file); err != nil {
						file.Close()
						return Result{Err: fmt.Errorf("failed to copy file content: %w", err)}
					}
					file.Close()
				} else {
					if err := w.WriteField(key, val); err != nil {
						return Result{Err: fmt.Errorf("failed to write form field: %w", err)}
					}
				}
			}

			if err := w.Close(); err != nil {
				return Result{Err: fmt.Errorf("failed to close multipart writer: %w", err)}
			}

			reqBody = &b
			multipartContentType = w.FormDataContentType()
		} else {
			form := url.Values{}
			for _, line := range strings.Split(opts.Body, "\n") {
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					form.Add(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
				}
			}
			reqBody = strings.NewReader(form.Encode())
		}
	}

	req, err := http.NewRequest(opts.Method, opts.URL, reqBody)
	if err != nil {
		return Result{Err: err}
	}

	for _, line := range strings.Split(opts.Headers, "\n") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			k := strings.TrimSpace(parts[0])
			v := strings.TrimSpace(parts[1])
			// multipart の場合は自動生成された Content-Type (boundary付き) を優先
			if opts.Format == "multipart" && strings.EqualFold(k, "content-type") {
				continue
			}
			req.Header.Add(k, v)
		}
	}

	if multipartContentType != "" {
		req.Header.Set("Content-Type", multipartContentType)
	}

	dialer := &net.Dialer{
		KeepAlive: 30 * time.Second,
	}
	if opts.ConnectTimeout > 0 {
		dialer.Timeout = time.Duration(opts.ConnectTimeout * float64(time.Second))
	}

	baseTransport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	if opts.ConnectTimeout > 0 {
		baseTransport.TLSHandshakeTimeout = time.Duration(opts.ConnectTimeout * float64(time.Second))
	}
	if opts.Insecure {
		baseTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	if opts.Proxy != "" {
		if proxyURL, err := url.Parse(opts.Proxy); err == nil {
			baseTransport.Proxy = http.ProxyURL(proxyURL)
		}
	}

	dt := &dumpTransport{
		Transport: baseTransport,
	}

	httpClient := &http.Client{
		Transport: dt,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if !opts.Location {
				return http.ErrUseLastResponse
			}

			lastReq := via[len(via)-1]
			if lastReq.Response != nil {
				history = append(history, HistoryEntry{
					Method: lastReq.Method,
					URL:    lastReq.URL.String(),
					Status: lastReq.Response.Status,
				})
			}

			if len(via) >= 10 {
				return errors.New("stopped after 10 redirects")
			}
			return nil
		},
	}

	if opts.MaxTime > 0 {
		httpClient.Timeout = time.Duration(opts.MaxTime * float64(time.Second))
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return Result{Err: err}
	}
	defer res.Body.Close()

	bodyBytes, _ := io.ReadAll(res.Body)

	history = append(history, HistoryEntry{
		Method: res.Request.Method,
		URL:    res.Request.URL.String(),
		Status: res.Status,
	})

	return Result{
		Status:   res.Status,
		Body:     string(bodyBytes),
		FullDump: dt.ChainDump.String(),
		History:  dt.History,
	}
}
