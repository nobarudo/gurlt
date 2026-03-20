package tui

import (
	"fmt"

	"github.com/nobarudo/gurlt/internal/client"
	"github.com/nobarudo/gurlt/internal/curl"

	tea "github.com/charmbracelet/bubbletea"
)

type responseMsg struct {
	status     string
	body       string
	rawContent string
	err        error
	history    []client.HistoryEntry
}

type clearMsg struct{}

func sendRequest(method, reqUrl, headers, body, format string, location bool) tea.Cmd {
	return func() tea.Msg {
		res := client.Send(method, reqUrl, headers, body, format, location)
		if res.Err != nil {
			return responseMsg{err: res.Err}
		}

		curlCmd := curl.Build(method, reqUrl, headers, body, format, location)
		rawStr := fmt.Sprintf("=== cURL ===\n%s\n\n%s", curlCmd, res.FullDump)

		return responseMsg{
			status:     res.Status,
			body:       res.Body,
			rawContent: rawStr,
			history:    res.History,
		}
	}
}
