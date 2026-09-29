package tui

import (
	"fmt"

	"github.com/nobarudo/gurlt/internal/client"

	tea "github.com/charmbracelet/bubbletea"
)

type responseMsg struct {
	status     string
	body       string
	rawContent string
	err        error
	history    []client.HistoryEntry
	timing     client.TimingInfo
}

type clearMsg struct{}

func sendRequest(opts client.RequestOptions, curlCmd string) tea.Cmd {
	return func() tea.Msg {
		res := client.Send(opts)
		if res.Err != nil {
			return responseMsg{err: res.Err, timing: res.Timing}
		}

		rawStr := fmt.Sprintf("=== cURL ===\n%s\n\n%s", curlCmd, res.FullDump)

		return responseMsg{
			status:     res.Status,
			body:       res.Body,
			rawContent: rawStr,
			history:    res.History,
			timing:     res.Timing,
		}
	}
}
