package tui

import (
	"fmt"

	"github.com/nobarudo/gurlt/internal/client"

	tea "github.com/charmbracelet/bubbletea"
)

type responseMsg struct {
	status     string
	body       string
	bodyBytes  []byte
	outputFile string
	savedBytes int
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
			return responseMsg{
				err:        res.Err,
				timing:     res.Timing,
				outputFile: res.OutputFile,
			}
		}

		var rawStr string
		if isBinaryContent(res.BodyBytes) {
			sizeStr := formatBytes(len(res.BodyBytes))
			binDesc := fmt.Sprintf("[Binary data: %s (%d bytes)]", sizeStr, len(res.BodyBytes))
			if res.OutputFile != "" {
				binDesc += fmt.Sprintf("\nSaved to: %s", res.OutputFile)
			}
			rawStr = fmt.Sprintf("=== cURL ===\n%s\n\n%s", curlCmd, sanitizeFullDump(res.FullDump, binDesc))
		} else {
			rawStr = fmt.Sprintf("=== cURL ===\n%s\n\n%s", curlCmd, res.FullDump)
		}

		return responseMsg{
			status:     res.Status,
			body:       res.Body,
			bodyBytes:  res.BodyBytes,
			outputFile: res.OutputFile,
			savedBytes: res.SavedBytes,
			rawContent: rawStr,
			history:    res.History,
			timing:     res.Timing,
		}
	}
}
