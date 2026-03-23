package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func InitialModel(reqUrl, method, headerStr, body, format string, location bool, logFile string, extraArgs string) Model {
	m := textinput.New()
	m.SetValue(method)
	m.Prompt = ""

	u := textinput.New()
	u.Placeholder = "https://api.example.com"
	u.SetValue(reqUrl)
	u.Focus()
	u.Prompt = ""

	h := textarea.New()
	h.Placeholder = "Key: Value..."
	h.SetHeight(5)
	h.SetWidth(60)

	var finalHeaderLines []string
	lowerHeaderStr := strings.ToLower(headerStr)

	// ユーザーが User-Agent を指定していなければデフォルトを付ける
	if !strings.Contains(lowerHeaderStr, "user-agent:") {
		finalHeaderLines = append(finalHeaderLines, "User-Agent: gurlt/0.1.0")
	}
	// ユーザーが Accept を指定していなければデフォルトを付ける
	if !strings.Contains(lowerHeaderStr, "accept:") {
		finalHeaderLines = append(finalHeaderLines, "Accept: */*")
	}

	// メソッドがPOST等で、ユーザーが Content-Type を指定していなければ自動付与
	if method == "POST" || method == "PUT" || method == "PATCH" {
		if !strings.Contains(lowerHeaderStr, "content-type:") {
			if format == "json" {
				finalHeaderLines = append(finalHeaderLines, "Content-Type: application/json")
			} else {
				finalHeaderLines = append(finalHeaderLines, "Content-Type: application/x-www-form-urlencoded")
			}
		}
	}

	finalHeaders := strings.Join(finalHeaderLines, "\n")
	if headerStr != "" {
		if finalHeaders != "" {
			finalHeaders += "\n" + headerStr
		} else {
			finalHeaders = headerStr
		}
	}
	h.SetValue(strings.TrimSpace(finalHeaders))

	b := textarea.New()

	if format == "json" {
		b.Placeholder = "{\n  \"key\": \"value\"\n}"
	} else {
		b.Placeholder = "key=value"
	}
	b.ShowLineNumbers = true
	b.SetHeight(5)
	b.SetWidth(60)
	if body != "" {
		b.SetValue(body)
	}

	sInput := textinput.New()
	sInput.Placeholder = "output.txt"
	sInput.Prompt = "Save to: "

	return Model{
		methodInput: m,
		urlInput:    u,
		headerInput: h,
		bodyInput:   b,
		saveInput:   sInput,
		focusIndex:  1,
		format:      format,
		location:    location,
		logFile:     logFile,
		extraArgs:   extraArgs,
	}
}

// ▼ 各入力欄のフォーカス状態を正しく更新するヘルパー関数
func updateFocus(m *Model) tea.Cmd {
	var cmds []tea.Cmd

	// 一旦すべての入力欄のフォーカスを外す
	m.methodInput.Blur()
	m.urlInput.Blur()
	m.headerInput.Blur()
	m.bodyInput.Blur()

	// 現在の focusIndex に応じて、該当する入力欄だけにフォーカスを当てる
	switch m.focusIndex {
	case 0:
		cmds = append(cmds, m.methodInput.Focus())
	case 1:
		cmds = append(cmds, m.urlInput.Focus())
	case 2:
		cmds = append(cmds, m.headerInput.Focus())
	case 3:
		cmds = append(cmds, m.bodyInput.Focus())
	}

	return tea.Batch(cmds...)
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, textarea.Blink)
}
