package tui

import (
	"fmt"
	"gurlt/internal/client"
	"gurlt/internal/curl"

	tea "github.com/charmbracelet/bubbletea"
)

type responseMsg struct {
	status     string
	body       string
	rawContent string
	err        error
}

type clearMsg struct{}

func sendRequest(method, reqUrl, headers, body, format string, location bool) tea.Cmd {
	return func() tea.Msg {
		res := client.Send(method, reqUrl, headers, body, format, location)
		if res.Err != nil {
			return responseMsg{err: res.Err}
		}

		curlCmd := curl.Build(method, reqUrl, headers, body, format, location)

		// ▼ 変更：ReqDump, ResDump を個別に結合するのではなく、FullDump を使う
		rawStr := fmt.Sprintf("=== cURL ===\n%s\n\n%s", curlCmd, res.FullDump)

		return responseMsg{
			status:     res.Status,
			body:       res.Body,
			rawContent: rawStr,
		}
	}
}
