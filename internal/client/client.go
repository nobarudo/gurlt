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
	"net/http/httptrace"
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
	Timing   TimingInfo
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
	t0 := time.Now()
	var (
		dnsStart  time.Time
		dnsDone   time.Time
		connStart time.Time
		connDone  time.Time
		tlsStart  time.Time
		tlsDone   time.Time
		reqWrote  time.Time
		firstByte time.Time
	)

	trace := &httptrace.ClientTrace{
		DNSStart: func(_ httptrace.DNSStartInfo) {
			dnsStart = time.Now()
		},
		DNSDone: func(_ httptrace.DNSDoneInfo) {
			dnsDone = time.Now()
		},
		ConnectStart: func(_, _ string) {
			connStart = time.Now()
		},
		ConnectDone: func(_, _ string, _ error) {
			connDone = time.Now()
		},
		TLSHandshakeStart: func() {
			tlsStart = time.Now()
		},
		TLSHandshakeDone: func(_ tls.ConnectionState, _ error) {
			tlsDone = time.Now()
		},
		WroteRequest: func(_ httptrace.WroteRequestInfo) {
			reqWrote = time.Now()
		},
		GotFirstResponseByte: func() {
			firstByte = time.Now()
		},
	}

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
			hasFormKey := false
			for _, line := range strings.Split(opts.Body, "\n") {
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					form.Add(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
					hasFormKey = true
				}
			}
			if hasFormKey {
				reqBody = strings.NewReader(form.Encode())
			} else {
				reqBody = strings.NewReader(opts.Body)
			}
		}
	}

	req, err := http.NewRequest(opts.Method, opts.URL, reqBody)
	if err != nil {
		return Result{Err: err}
	}
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

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
		tEnd := time.Now()
		return Result{
			Err: err,
			Timing: TimingInfo{
				Total: tEnd.Sub(t0),
			},
		}
	}
	defer res.Body.Close()

	bodyBytes, _ := io.ReadAll(res.Body)
	tEnd := time.Now()
	if firstByte.IsZero() || firstByte.After(tEnd) {
		firstByte = tEnd
	}

	timing := TimingInfo{
		Total: tEnd.Sub(t0),
	}
	if !dnsDone.IsZero() && !dnsStart.IsZero() {
		timing.DNSLookup = dnsDone.Sub(dnsStart)
		timing.NameLookup = dnsDone.Sub(t0)
	}
	if !connDone.IsZero() && !connStart.IsZero() {
		timing.TCPConnect = connDone.Sub(connStart)
		timing.Connect = connDone.Sub(t0)
	}
	if !tlsDone.IsZero() && !tlsStart.IsZero() {
		timing.TLSHandshake = tlsDone.Sub(tlsStart)
		timing.AppConnect = tlsDone.Sub(t0)
	}
	if !firstByte.IsZero() {
		timing.StartTransfer = firstByte.Sub(t0)
		startWait := reqWrote
		if startWait.IsZero() {
			if !tlsDone.IsZero() {
				startWait = tlsDone
			} else if !connDone.IsZero() {
				startWait = connDone
			} else {
				startWait = t0
			}
		}
		timing.ServerProcessing = firstByte.Sub(startWait)
		if timing.ServerProcessing < 0 {
			timing.ServerProcessing = 0
		}
		timing.ContentTransfer = tEnd.Sub(firstByte)
		if timing.ContentTransfer < 0 {
			timing.ContentTransfer = 0
		}
	}

	dt.ChainDump.WriteString("\n" + timing.Breakdown())

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
		Timing:   timing,
	}
}
