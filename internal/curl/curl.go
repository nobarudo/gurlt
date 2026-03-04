package curl

import (
	"fmt"
	"net/url"
	"strings"
)

// Build は入力された値からcURLコマンドの文字列を生成します
func Build(method, reqUrl, headers, body, format string, location bool) string {
	cmd := fmt.Sprintf("curl -X %s '%s'", method, reqUrl)
	if location {
		cmd += " -L"
	}
	lines := strings.Split(headers, "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			cmd += fmt.Sprintf(" -H '%s: %s'", strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
		}
	}
	if body != "" {
		if format == "json" {
			singleLine := strings.ReplaceAll(body, "\n", "")
			cmd += fmt.Sprintf(" -d '%s'", singleLine)
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
