package tui

import (
	"strings"

	"github.com/nobarudo/gurlt/internal/client"
	"github.com/nobarudo/gurlt/internal/curl"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
)

type Model struct {
	methodInput    textinput.Model
	urlInput       textinput.Model
	headerInput    textarea.Model
	bodyInput      textarea.Model
	responseView   viewport.Model
	focusIndex     int
	ready          bool
	showRawView    bool
	terminalWidth  int
	terminalHeight int
	normalContent  string
	rawContent     string
	footerMsg      string
	saveInput      textinput.Model
	isSaving       bool
	responseStatus string
	history        []client.HistoryEntry
	isLoading      bool
	err            error
	format         string
	location       bool //　リダイレクト追従フラグ
	logFile          string
	extraArgs        string
	showOptionsModal bool
	insecure         bool
	verbose          bool
	proxyInput       textinput.Model
	optionsCursor    int
}

// BuildCurlCmd は現在の設定値（URL, Header, Body, 各種オプション）から完全なcURLコマンド文字列を生成します
func (m Model) BuildCurlCmd() string {
	proxy := strings.TrimSpace(m.proxyInput.Value())
	cmd := curl.Build(
		m.methodInput.Value(),
		m.urlInput.Value(),
		m.headerInput.Value(),
		m.bodyInput.Value(),
		m.format,
		m.location,
		m.insecure,
		m.verbose,
		proxy,
	)
	if m.extraArgs != "" {
		cmd += " " + m.extraArgs
	}
	return cmd
}
