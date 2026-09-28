package curl

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// Build は入力された値からcURLコマンドの文字列を生成します
func Build(method, reqUrl, headers, body, format string, location, insecure, verbose bool, proxy string, maxTime, connectTimeout float64) string {
	cmd := fmt.Sprintf("curl -X %s '%s'", method, reqUrl)
	if location {
		cmd += " -L"
	}
	if insecure {
		cmd += " -k"
	}
	if verbose {
		cmd += " -v"
	}
	if proxy != "" {
		cmd += fmt.Sprintf(" -x '%s'", proxy)
	}
	if maxTime > 0 {
		cmd += fmt.Sprintf(" -m %s", strconv.FormatFloat(maxTime, 'f', -1, 64))
	}
	if connectTimeout > 0 {
		cmd += fmt.Sprintf(" --connect-timeout %s", strconv.FormatFloat(connectTimeout, 'f', -1, 64))
	}
	lines := strings.Split(headers, "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			k := strings.TrimSpace(parts[0])
			v := strings.TrimSpace(parts[1])
			if format == "multipart" && strings.EqualFold(k, "content-type") {
				continue
			}
			cmd += fmt.Sprintf(" -H '%s: %s'", k, v)
		}
	}
	if body != "" {
		if format == "json" {
			singleLine := strings.ReplaceAll(body, "\n", "")
			cmd += fmt.Sprintf(" -d '%s'", singleLine)
		} else if format == "multipart" {
			for _, line := range strings.Split(body, "\n") {
				line = strings.TrimSpace(line)
				if line != "" {
					cmd += fmt.Sprintf(" -F '%s'", line)
				}
			}
		} else {
			form := url.Values{}
			for _, line := range strings.Split(body, "\n") {
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					form.Add(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
				}
			}
			if len(form) > 0 {
				cmd += fmt.Sprintf(" -d '%s'", form.Encode())
			}
		}
	}
	return cmd
}
