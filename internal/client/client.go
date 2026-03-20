package client

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

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

func Send(method, reqUrl, headers, body, format string, location bool) Result {
	var reqBody io.Reader
	var history []HistoryEntry

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

	dt := &dumpTransport{
		Transport: http.DefaultTransport,
	}

	httpClient := &http.Client{
		Transport: dt,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if !location {
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
