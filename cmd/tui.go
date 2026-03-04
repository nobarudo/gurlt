package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"gurlt/internal/client"
	"gurlt/internal/curl"
)

// ==========================================
// Lip Gloss スタイル定義
// ==========================================

var (
	appStyle          = lipgloss.NewStyle().Margin(1, 2).Padding(1, 2).Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#569CD6"))
	titleStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("#4EC9B0")).Bold(true).MarginBottom(1)
	focusedLabelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#CE9178")).Bold(true)
	blurredLabelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#75715E"))
	dividerStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#3E3D32")).Margin(1, 0)
	errorStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("#F44747")).Bold(true)
	successStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#B5CEA8")).Bold(true)
	infoStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("#DCDCAA"))
	responseBoxStyle  = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#808080")).Padding(0, 1)
	curlPreviewStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#C586C0")).Italic(true)
)

// ==========================================
// 型定義とモデル
// ==========================================

type responseMsg struct {
	status     string
	body       string
	rawContent string
	err        error
}

type clearMsg struct{}

type model struct {
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
	location       bool // 追加: リダイレクト追従フラグ
}

func initialModel(reqUrl, method, headerStr, body, format string, location bool) model {
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
	h.SetHeight(3)
	h.SetWidth(60)

	// ▼ ヘッダーの初期値の組み立て（-A や -H で指定されたものを優先する賢い処理）
	finalHeaders := "Accept: */*"
	if !strings.Contains(strings.ToLower(headerStr), "user-agent") {
		finalHeaders = "User-Agent: gurlt/0.1.0\n" + finalHeaders
	}
	if method == "POST" || method == "PUT" || method == "PATCH" {
		if !strings.Contains(strings.ToLower(headerStr), "content-type") {
			if format == "json" {
				finalHeaders += "\nContent-Type: application/json"
			} else {
				finalHeaders += "\nContent-Type: application/x-www-form-urlencoded"
			}
		}
	}
	// root.go から渡された -H などのヘッダーを追記
	if headerStr != "" {
		finalHeaders += "\n" + headerStr
	}
	h.SetValue(strings.TrimSpace(finalHeaders))

	b := textarea.New()
	if format == "json" {
		b.Placeholder = "{\n  \"key\": \"value\"\n}"
	} else {
		b.Placeholder = "key=value"
	}
	b.ShowLineNumbers = true
	b.SetHeight(4)
	b.SetWidth(60)
	// ▼ -d で渡されたボディデータをセット
	if body != "" {
		b.SetValue(body)
	}

	sInput := textinput.New()
	sInput.Placeholder = "output.txt"
	sInput.Prompt = "Save to: "

	return model{
		methodInput: m,
		urlInput:    u,
		headerInput: h,
		bodyInput:   b,
		saveInput:   sInput,
		focusIndex:  1,
		format:      format,
		location:    location, // セット
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, textarea.Blink)
}

// ==========================================
// リクエスト非同期処理
// ==========================================

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

// ==========================================
// メインループ (Update & View)
// ==========================================

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case clearMsg:
		m.footerMsg = ""
		return m, nil

	case tea.WindowSizeMsg:
		m.terminalWidth, m.terminalHeight = msg.Width, msg.Height
		contentWidth := m.terminalWidth - 8
		if !m.ready {
			m.responseView = viewport.New(contentWidth, 1)
			m.normalContent = "Ready to send request.\nPress Ctrl+S to fetch."
			m.responseView.SetContent(m.normalContent)
			m.ready = true
		}
		m.responseView.Width = contentWidth
		if m.showRawView {
			m.responseView.Height = m.terminalHeight - 12
		} else {
			h := m.terminalHeight - 31
			if h < 0 {
				h = 0
			}
			m.responseView.Height = h
		}

	case tea.KeyMsg:
		if len(msg.String()) == 1 {
			m.footerMsg = ""
		}

		if m.isSaving {
			switch msg.String() {
			case "esc", "ctrl+c":
				m.isSaving = false
				m.saveInput.Blur()
				return m, nil
			case "enter":
				filename := strings.TrimSpace(m.saveInput.Value())
				if filename != "" {
					os.WriteFile(filename, []byte(m.rawContent), 0644)
					m.footerMsg = successStyle.Render(" [✅ Saved!]")
					m.isSaving = false
					m.saveInput.Blur()
					return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg { return clearMsg{} })
				}
				m.isSaving = false
				m.saveInput.Blur()
				return m, nil
			default:
				m.saveInput, cmd = m.saveInput.Update(msg)
				return m, cmd
			}
		}

		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit

		case "tab":
			if m.focusIndex == 2 {
				m.headerInput.InsertString("  ")
				return m, nil
			}
			if m.focusIndex == 3 {
				m.bodyInput.InsertString("  ")
				return m, nil
			}

		case "ctrl+j", "ctrl+n":
			if !m.showRawView {
				m.focusIndex++
				if m.focusIndex > 4 {
					m.focusIndex = 0
				}
			}
			return m, nil
		case "ctrl+k", "ctrl+p":
			if !m.showRawView {
				m.focusIndex--
				if m.focusIndex < 0 {
					m.focusIndex = 4
				}
			}
			return m, nil

		case "c":
			if m.showRawView && m.rawContent != "" {
				clipboard.WriteAll(m.rawContent)
				m.footerMsg = successStyle.Render(" [✅ Copied!]")
				return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg { return clearMsg{} })
			}
		case "s":
			if m.showRawView {
				m.isSaving = true
				m.saveInput.Focus()
				return m, textinput.Blink
			}

		case "ctrl+f":
			if m.format == "json" && m.focusIndex == 3 {
				input := m.bodyInput.Value()
				if input == "" {
					return m, nil
				}
				var obj interface{}
				if err := json.Unmarshal([]byte(input), &obj); err != nil {
					m.footerMsg = errorStyle.Render(" [❌ Invalid JSON]")
					return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg { return clearMsg{} })
				}
				pretty, _ := json.MarshalIndent(obj, "", "  ")
				m.bodyInput.SetValue(string(pretty))
				m.footerMsg = successStyle.Render(" [✨ Formatted!]")
				return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg { return clearMsg{} })
			}

		case "ctrl+r":
			m.showRawView = !m.showRawView
			m.footerMsg = ""
			m.isSaving = false
			if m.showRawView {
				m.responseView.Height = m.terminalHeight - 12
				m.responseView.SetContent(m.rawContent)
			} else {
				h := m.terminalHeight - 31
				if h < 0 {
					h = 0
				}
				m.responseView.Height = h
				m.responseView.SetContent(m.normalContent)
			}
			m.responseView.GotoTop()
			return m, nil
		case "ctrl+s":
			if !m.showRawView {
				m.isLoading = true
				m.footerMsg = ""
				m.responseView.SetContent(infoStyle.Render("⏳ Loading..."))
				// ▼ 修正: 一番最後に m.location を追加します
				return m, sendRequest(m.methodInput.Value(), m.urlInput.Value(), m.headerInput.Value(), m.bodyInput.Value(), m.format, m.location)
			}
		case "ctrl+a":
			if !m.showRawView {
				fullCurl := curl.Build(m.methodInput.Value(), m.urlInput.Value(), m.headerInput.Value(), m.bodyInput.Value(), m.format, m.location)
				clipboard.WriteAll(fullCurl)
				m.footerMsg = successStyle.Render(" [✅ Copied!]")
				return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg { return clearMsg{} })
			} else {
				// ▼ 追加：Raw画面で押されたら、Rawデータ全体をコピーする！
				clipboard.WriteAll(m.rawContent)
				m.footerMsg = successStyle.Render(" [✅ Raw Copied!]")
				return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg { return clearMsg{} })
			}
		}

	case responseMsg:
		m.isLoading = false
		m.responseStatus = msg.status
		if msg.err == nil {
			m.normalContent, m.rawContent = msg.body, msg.rawContent
		} else {
			m.normalContent = errorStyle.Render(fmt.Sprintf("Error: %v", msg.err))
		}
		if m.showRawView {
			m.responseView.SetContent(m.rawContent)
		} else {
			m.responseView.SetContent(m.normalContent)
		}
		m.responseView.GotoTop()
		return m, nil
	}

	if !m.showRawView {
		m.methodInput.Blur()
		m.urlInput.Blur()
		m.headerInput.Blur()
		m.bodyInput.Blur()
		if m.focusIndex == 0 {
			m.methodInput.Focus()
		}
		if m.focusIndex == 1 {
			m.urlInput.Focus()
		}
		if m.focusIndex == 2 {
			m.headerInput.Focus()
		}
		if m.focusIndex == 3 {
			m.bodyInput.Focus()
		}
		m.methodInput, cmd = m.methodInput.Update(msg)
		cmds = append(cmds, cmd)
		m.urlInput, cmd = m.urlInput.Update(msg)
		cmds = append(cmds, cmd)
		m.headerInput, cmd = m.headerInput.Update(msg)
		cmds = append(cmds, cmd)
		m.bodyInput, cmd = m.bodyInput.Update(msg)
		cmds = append(cmds, cmd)
	}
	if m.focusIndex == 4 || m.showRawView {
		m.responseView, cmd = m.responseView.Update(msg)
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}
	var content string

	if m.showRawView {
		content += titleStyle.Render("📡 gurlt - Raw View") + "\n\n"
		content += responseBoxStyle.Render(m.responseView.View()) + "\n\n"
		if m.isSaving {
			content += m.saveInput.View() + "   [Enter] Confirm   [Esc] Cancel"
		} else {
			content += infoStyle.Render("[c/ctrl+a] Copy Raw   [s] Save to File   [ctrl+r] Back") + m.footerMsg
		}
		return appStyle.Render(content)
	}

	content += titleStyle.Render("🚀 gurlt - TUI HTTP Client") + "\n\n"
	renderLabel := func(label string, isFocused bool) string {
		if isFocused {
			return focusedLabelStyle.Render("▶ " + label)
		}
		return blurredLabelStyle.Render("  " + label)
	}
	content += renderLabel("Method:", m.focusIndex == 0) + " " + m.methodInput.View() + "\n"
	content += renderLabel("URL:   ", m.focusIndex == 1) + " " + m.urlInput.View() + "\n\n"
	content += renderLabel("Headers:", m.focusIndex == 2) + "\n" + m.headerInput.View() + "\n\n"

	bodyLabel := "Params (key=value):"
	if m.format == "json" {
		bodyLabel = "Body (JSON):"
	}
	content += renderLabel(bodyLabel, m.focusIndex == 3) + "\n" + m.bodyInput.View() + "\n"
	content += dividerStyle.Render(strings.Repeat("─", m.terminalWidth-10)) + "\n"

	if m.responseStatus != "" {
		if strings.HasPrefix(m.responseStatus, "2") {
			content += successStyle.Render(fmt.Sprintf("✅ Status: %s", m.responseStatus)) + "\n"
		} else {
			content += errorStyle.Render(fmt.Sprintf("⚠️ Status: %s", m.responseStatus)) + "\n"
		}
	} else {
		content += "\n"
	}

	content += renderLabel("Response Body:", m.focusIndex == 4) + "\n"
	content += responseBoxStyle.Render(m.responseView.View()) + "\n"
	content += dividerStyle.Render(strings.Repeat("─", m.terminalWidth-10)) + "\n"

	curlPreview := curl.Build(m.methodInput.Value(), m.urlInput.Value(), m.headerInput.Value(), m.bodyInput.Value(), m.format, m.location)
	if len(curlPreview) > m.terminalWidth-20 {
		curlPreview = curlPreview[:m.terminalWidth-25] + "..."
	}
	content += curlPreviewStyle.Render(fmt.Sprintf("💻 cURL: %s", curlPreview)) + "\n\n"

	content += infoStyle.Render("[ctrl+j/n] Focus↓  [ctrl+k/p] Focus↑  [ctrl+f] Prettify  [ctrl+s] Send  [ctrl+r] Raw  [ctrl+a] cURL Copy") + m.footerMsg + "\n"

	return appStyle.Render(content)
}
