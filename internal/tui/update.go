// internal/tui/update.go
package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/nobarudo/gurlt/internal/curl"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case clearMsg:
		m.footerMsg = ""
		return m, nil
	case tea.WindowSizeMsg:
		return m.handleWindowSize(msg)
	case responseMsg:
		return m.handleResponse(msg)
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	}

	return m.updateInputs(msg)
}

func (m Model) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.terminalWidth, m.terminalHeight = msg.Width, msg.Height
	contentWidth := m.terminalWidth - 12
	if contentWidth < 1 {
		contentWidth = 1
	}
	fixedVerticalLines := 15

	// 入力欄に使える余りスペースを計算し、2つのテキストエリア（Headers/Params）で割る
	availableHeight := m.terminalHeight - fixedVerticalLines
	textAreaHeight := availableHeight / 2

	// 最低5行は必ず確保する（ターミナルが極端に狭い場合）
	if textAreaHeight < 5 {
		textAreaHeight = 5
	}

	// 計算した高さをテキストエリアにセット
	m.headerInput.SetHeight(textAreaHeight)
	m.bodyInput.SetHeight(textAreaHeight)

	rawFixedLines := 5
	rawHeight := m.terminalHeight - rawFixedLines
	if rawHeight < 1 {
		rawHeight = 1
	}

	if !m.ready {
		// 初回起動時
		m.responseView = viewport.New(contentWidth, rawHeight)
		m.normalContent = "Ready to send request.\nPress Ctrl+S to fetch."
		m.ready = true
	} else {
		// 画面リサイズ時
		m.responseView.Width = contentWidth
		m.responseView.Height = rawHeight // ▼ ここで計算した高さをセット！
	}

	if !m.ready {
		m.responseView = viewport.New(contentWidth, 1)
		m.normalContent = "Ready to send request.\nPress Ctrl+S to fetch."
		m.responseView.SetContent(m.normalContent)
		m.ready = true
	}
	m.responseView.Width = contentWidth
	m.responseView.Height = m.terminalHeight - 12
	if m.rawContent != "" {
		wrappedRaw := lipgloss.NewStyle().Width(contentWidth).Render(m.rawContent)
		m.responseView.SetContent(wrappedRaw)
	} else {
		m.responseView.SetContent(m.normalContent)
	}
	return m, nil
}

func (m Model) handleResponse(msg responseMsg) (tea.Model, tea.Cmd) {
	m.isLoading = false
	m.responseStatus = msg.status
	if msg.err == nil {
		m.normalContent, m.rawContent = msg.body, msg.rawContent
		m.history = msg.history

		// ログ保存
		if m.logFile != "" {
			f, err := os.OpenFile(m.logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err == nil {
				defer f.Close()
				timestamp := time.Now().Format("2006-01-02 15:04:05")
				logEntry := fmt.Sprintf("========== [ %s ] ==========\n%s\n\n", timestamp, m.rawContent)
				f.WriteString(logEntry)
				m.footerMsg = successStyle.Render(fmt.Sprintf(" [📝 Logged to %s]", m.logFile))
			}
		}
	} else {
		m.normalContent = errorStyle.Render(fmt.Sprintf("Error: %v", msg.err))
	}

	if m.showRawView {
		m.responseView.SetContent(m.rawContent)
	} else {
		m.responseView.SetContent(m.normalContent)
	}
	wrappedRaw := lipgloss.NewStyle().Width(m.responseView.Width).Render(m.rawContent)
	m.responseView.SetContent(wrappedRaw)
	m.responseView.GotoTop()

	return m, nil
}

func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if len(msg.String()) == 1 {
		m.footerMsg = ""
	}

	// 1. 保存モード中のキーボード操作
	if m.isSaving {
		var cmd tea.Cmd
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

	// 2. グローバルショートカットの処理
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
			if m.focusIndex > 3 {
				m.focusIndex = 0
			}
			return m, updateFocus(&m)
		}
		return m, nil
	case "ctrl+k", "ctrl+p":
		if !m.showRawView {
			m.focusIndex--
			if m.focusIndex < 0 {
				m.focusIndex = 3
			}
			return m, updateFocus(&m)
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
			wrappedRaw := lipgloss.NewStyle().Width(m.responseView.Width).Render(m.rawContent)
			m.responseView.SetContent(wrappedRaw)
		}
		m.responseView.GotoTop()
		return m, nil
	case "ctrl+s":
		if !m.showRawView {
			m.isLoading = true
			m.footerMsg = ""
			m.responseView.SetContent(infoStyle.Render("⏳ Loading..."))
			return m, sendRequest(m.methodInput.Value(), m.urlInput.Value(), m.headerInput.Value(), m.bodyInput.Value(), m.format, m.location)
		}
	case "ctrl+a":
		if !m.showRawView {
			fullCurl := curl.Build(m.methodInput.Value(), m.urlInput.Value(), m.headerInput.Value(), m.bodyInput.Value(), m.format, m.location)
			clipboard.WriteAll(fullCurl)
			m.footerMsg = successStyle.Render(" [✅ Copied!]")
			return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg { return clearMsg{} })
		} else {
			clipboard.WriteAll(m.rawContent)
			m.footerMsg = successStyle.Render(" [✅ Raw Copied!]")
			return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg { return clearMsg{} })
		}
	}

	// 3. どのショートカットにも該当しない場合は、入力欄の文字入力として処理
	return m.updateInputs(msg)
}

func (m Model) updateInputs(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	if !m.showRawView {
		m.methodInput, cmd = m.methodInput.Update(msg)
		cmds = append(cmds, cmd)
		m.urlInput, cmd = m.urlInput.Update(msg)
		cmds = append(cmds, cmd)
		m.headerInput, cmd = m.headerInput.Update(msg)
		cmds = append(cmds, cmd)
		m.bodyInput, cmd = m.bodyInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	if m.showRawView {
		m.responseView, cmd = m.responseView.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}
