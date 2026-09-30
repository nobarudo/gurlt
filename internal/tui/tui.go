package tui

import (
	"bytes"
	"encoding/json"
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

	// ユーザーが Accept を指定していなければデフォルトを付ける
	if !strings.Contains(lowerHeaderStr, "accept:") {
		finalHeaderLines = append(finalHeaderLines, "Accept: */*")
	}

	// メソッドがPOST等で、ユーザーが Content-Type を指定していなければ自動付与
	if method == "POST" || method == "PUT" || method == "PATCH" {
		if !strings.Contains(lowerHeaderStr, "content-type:") {
			if format == "json" {
				finalHeaderLines = append(finalHeaderLines, "Content-Type: application/json")
			} else if format == "multipart" {
				// multipart の場合は送信時に boundary 付きで自動生成されるためヘッダー指定不要
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
	} else if format == "multipart" {
		b.Placeholder = "field=value\nfile=@/path/to/file"
	} else {
		b.Placeholder = "key=value"
	}
	b.ShowLineNumbers = true
	b.SetHeight(5)
	b.SetWidth(60)
	if body != "" {
		if format == "json" {
			var pretty bytes.Buffer
			if err := json.Indent(&pretty, []byte(body), "", "  "); err == nil {
				b.SetValue(pretty.String())
			} else {
				b.SetValue(body)
			}
		} else {
			b.SetValue(body)
		}
	}

	sInput := textinput.New()
	sInput.Placeholder = "output.txt"
	sInput.Prompt = "Save to: "

	pInput := textinput.New()
	pInput.Placeholder = "http://proxy.example.com:8080"
	pInput.Prompt = "  URL: "
	pInput.CharLimit = 128

	tInput := textinput.New()
	tInput.Placeholder = "10 (0 for none)"
	tInput.Prompt = "  Seconds: "
	tInput.CharLimit = 16

	srcInput := textinput.New()
	srcInput.Placeholder = "Search query..."
	srcInput.Prompt = "/ "
	srcInput.CharLimit = 128

	fInput := textinput.New()
	fInput.Placeholder = ".data.users[0].name"
	fInput.Prompt = "JSON Path: "
	fInput.CharLimit = 128

	bBearerInput := textinput.New()
	bBearerInput.Placeholder = "eyJhbGciOiJIUzI1NiIs..."
	bBearerInput.Prompt = "  Token: "
	bBearerInput.CharLimit = 1024
	if initialBearer := extractBearerToken(headerStr); initialBearer != "" {
		bBearerInput.SetValue(initialBearer)
	}

	oInput := textinput.New()
	oInput.Placeholder = "output.json (empty for none)"
	oInput.Prompt = "  File: "
	oInput.CharLimit = 256

	return Model{
		methodInput:  m,
		urlInput:     u,
		headerInput:  h,
		bodyInput:    b,
		saveInput:    sInput,
		proxyInput:   pInput,
		timeoutInput: tInput,
		searchInput:  srcInput,
		filterInput:  fInput,
		bearerInput:  bBearerInput,
		outputInput:  oInput,
		focusIndex:   1,
		format:       format,
		location:     location,
		logFile:      logFile,
		extraArgs:    extraArgs,
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
