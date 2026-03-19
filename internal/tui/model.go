package tui

import (
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
	isLoading      bool
	err            error
	format         string
	location       bool //　リダイレクト追従フラグ
	logFile        string
}
