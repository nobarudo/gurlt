package cmd

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// ==========================================
// Bubble Tea のモデル構成
// ==========================================

type responseMsg struct {
	status     string
	body       string
	rawContent string
	err        error
}

type model struct {
	methodInput  textinput.Model
	urlInput     textinput.Model
	headerInput  textarea.Model
	bodyInput    textarea.Model
	responseView viewport.Model
	focusIndex   int
	ready        bool

	showRawView    bool
	terminalWidth  int
	terminalHeight int
	normalContent  string
	rawContent     string

	footerMsg string
	saveInput textinput.Model
	isSaving  bool

	responseStatus string
	isLoading      bool
	err            error
}

func initialModel(reqUrl, method string) model {
	m := textinput.New()
	m.Placeholder = "GET, POST, PUT, DELETE..."
	m.SetValue(method)

	u := textinput.New()
	u.Placeholder = "https://api.example.com"
	u.SetValue(reqUrl)
	u.Focus()

	h := textarea.New()
	h.Placeholder = "Key: Value\nAuthorization: Bearer token..."
	h.ShowLineNumbers = false
	h.SetHeight(4)
	h.SetWidth(50)

	defaultHeaders := "User-Agent: gurlt/0.1.0\nAccept: */*"
	if method == "POST" || method == "PUT" || method == "PATCH" {
		defaultHeaders += "\nContent-Type: application/x-www-form-urlencoded"
	}
	h.SetValue(defaultHeaders)

	b := textarea.New()
	b.Placeholder = "name=taro\nage=20"
	b.ShowLineNumbers = true
	b.SetHeight(5)
	b.SetWidth(50)

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
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, textarea.Blink)
}

func buildCurlCmd(method, reqUrl, headers, body string) string {
	cmd := fmt.Sprintf("curl -X %s '%s'", method, reqUrl)

	lines := strings.Split(headers, "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			cmd += fmt.Sprintf(" -H '%s: %s'", strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
		}
	}

	if body != "" {
		form := url.Values{}
		lines := strings.Split(body, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				form.Add(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
			} else {
				form.Add(strings.TrimSpace(parts[0]), "")
			}
		}
		if len(form) > 0 {
			cmd += fmt.Sprintf(" -d '%s'", form.Encode())
		}
	}

	return cmd
}

func sendRequest(method, reqUrl, headers, body string) tea.Cmd {
	return func() tea.Msg {
		var reqBody io.Reader

		if body != "" {
			form := url.Values{}
			lines := strings.Split(body, "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					form.Add(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
				} else {
					form.Add(strings.TrimSpace(parts[0]), "")
				}
			}
			reqBody = strings.NewReader(form.Encode())
		}

		req, err := http.NewRequest(method, reqUrl, reqBody)
		if err != nil {
			return responseMsg{err: err}
		}

		lines := strings.Split(headers, "\n")
		for _, line := range lines {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				req.Header.Add(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
			}
		}

		reqBytes, _ := httputil.DumpRequestOut(req, true)

		client := &http.Client{Timeout: 10 * time.Second}
		res, err := client.Do(req)
		if err != nil {
			return responseMsg{err: err}
		}
		defer res.Body.Close()

		resBytes, _ := httputil.DumpResponse(res, true)
		bodyBytes, _ := io.ReadAll(res.Body)

		curlCmd := buildCurlCmd(method, reqUrl, headers, body)
		rawStr := fmt.Sprintf("=== cURL ===\n%s\n\n=== Request ===\n%s\n%s\n\n=== Response ===\n%s", curlCmd, reqUrl, string(reqBytes), string(resBytes))

		return responseMsg{
			status:     res.Status,
			body:       string(bodyBytes),
			rawContent: rawStr,
		}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.terminalWidth = msg.Width
		m.terminalHeight = msg.Height

		if !m.ready {
			m.responseView = viewport.New(msg.Width, 1)
			m.normalContent = "Ready to send request.\nPress Ctrl+S to fetch."
			m.rawContent = "No request sent yet."
			m.responseView.SetContent(m.normalContent)
			m.ready = true
		}

		if m.showRawView {
			m.responseView.Height = m.terminalHeight - 4
		} else {
			h := m.terminalHeight - 26
			if h < 0 {
				h = 0
			}
			m.responseView.Height = h
		}
		m.responseView.Width = m.terminalWidth

	case tea.KeyMsg:
		// 文字が入力されたらフッターのメッセージを消す
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
					err := os.WriteFile(filename, []byte(m.rawContent), 0644)
					if err != nil {
						m.footerMsg = fmt.Sprintf(" [❌ Save failed: %v]", err)
					} else {
						m.footerMsg = fmt.Sprintf(" [✅ Saved to %s]", filename)
					}
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
			if !m.showRawView {
				m.focusIndex++
				if m.focusIndex > 4 {
					m.focusIndex = 0
				}
			}
		case "shift+tab":
			if !m.showRawView {
				m.focusIndex--
				if m.focusIndex < 0 {
					m.focusIndex = 4
				}
			}

		case "c":
			if m.showRawView && m.rawContent != "" && m.rawContent != "No request sent yet." {
				err := clipboard.WriteAll(m.rawContent)
				if err != nil {
					m.footerMsg = " [❌ Copy failed (Requires xclip/xsel on Linux)]"
				} else {
					m.footerMsg = " [✅ Copied to clipboard!]"
				}
			}

		case "s":
			if m.showRawView && m.rawContent != "" && m.rawContent != "No request sent yet." {
				m.isSaving = true
				m.saveInput.Focus()
				m.saveInput.SetValue("")
				m.footerMsg = ""
				return m, textinput.Blink
			}

		case "ctrl+r":
			m.showRawView = !m.showRawView
			m.footerMsg = ""
			m.isSaving = false
			if m.showRawView {
				m.responseView.Height = m.terminalHeight - 4
				m.responseView.SetContent(m.rawContent)
			} else {
				h := m.terminalHeight - 26
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
				m.err = nil
				m.responseStatus = ""
				m.footerMsg = ""
				m.responseView.SetContent("⏳ Loading...")
				return m, sendRequest(m.methodInput.Value(), m.urlInput.Value(), m.headerInput.Value(), m.bodyInput.Value())
			}

		// Ctrl+A が押された時の処理
		case "ctrl+a":
			if !m.showRawView {
				fullCurl := buildCurlCmd(m.methodInput.Value(), m.urlInput.Value(), m.headerInput.Value(), m.bodyInput.Value())
				err := clipboard.WriteAll(fullCurl)

				if err != nil {
					// Linux等でクリップボード操作が失敗した場合
					m.footerMsg = " [❌ Copy failed. Please select the text above!]"
				} else {
					// 成功した場合
					m.footerMsg = " [✅ Copied cURL!]"
				}
				return m, nil
			}
		}

	case responseMsg:
		m.isLoading = false
		m.err = msg.err
		m.responseStatus = msg.status
		if msg.err == nil {
			m.normalContent = msg.body
			m.rawContent = msg.rawContent
		} else {
			m.normalContent = fmt.Sprintf("Error: %v", msg.err)
			m.rawContent = fmt.Sprintf("Error: %v", msg.err)
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
		} else if m.focusIndex == 1 {
			m.urlInput.Focus()
		} else if m.focusIndex == 2 {
			m.headerInput.Focus()
		} else if m.focusIndex == 3 {
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

	if m.showRawView {
		s := "[ Raw View (Use ↑/↓/PgUp/PgDn to scroll) ]\n"
		s += "----------------------------------------\n"
		s += m.responseView.View() + "\n"
		s += "----------------------------------------\n"

		if m.isSaving {
			s += m.saveInput.View() + "   [Enter] Confirm   [Esc] Cancel\n"
		} else {
			s += "[c] Copy   [s] Save to File" + m.footerMsg + "   [Ctrl+R] Back to Form   [Esc] Quit\n"
		}
		return s
	}

	s := "Welcome to gurlt!\nHow to use gurlt : gurlt -h\n\n"

	if m.focusIndex == 0 {
		s += fmt.Sprintf("> Method: %s\n", m.methodInput.View())
	} else {
		s += fmt.Sprintf("  Method: %s\n", m.methodInput.View())
	}

	if m.focusIndex == 1 {
		s += fmt.Sprintf("> URL:    %s\n\n", m.urlInput.View())
	} else {
		s += fmt.Sprintf("  URL:    %s\n\n", m.urlInput.View())
	}

	if m.focusIndex == 2 {
		s += "> Headers:\n"
	} else {
		s += "  Headers:\n"
	}
	s += m.headerInput.View() + "\n\n"

	if m.focusIndex == 3 {
		s += "> Params (key=value):\n"
	} else {
		s += "  Params (key=value):\n"
	}
	s += m.bodyInput.View() + "\n"

	s += "----------------------------------------\n"

	if m.err != nil {
		s += fmt.Sprintf("❌ Error: %v\n", m.err)
	} else if m.responseStatus != "" {
		s += fmt.Sprintf("✅ Status: %s\n", m.responseStatus)
	} else {
		s += "\n"
	}

	s += "----------------------------------------\n"

	if m.focusIndex == 4 {
		s += "[ Response Body (Use ↑/↓/PgUp/PgDn to scroll) ]\n"
	} else {
		s += "[ Response Body ]\n"
	}
	s += m.responseView.View() + "\n"

	s += "----------------------------------------\n"

	curlPreview := buildCurlCmd(m.methodInput.Value(), m.urlInput.Value(), m.headerInput.Value(), m.bodyInput.Value())
	if m.terminalWidth > 15 && len(curlPreview) > m.terminalWidth-15 {
		curlPreview = curlPreview[:m.terminalWidth-15] + "..."
	}
	s += fmt.Sprintf("💻 cURL: %s\n", curlPreview)

	s += "----------------------------------------\n"

	// ▼ 修正: 通常画面のフッターにも m.footerMsg を表示するように変更
	s += "[Tab] Focus  [Ctrl+S] Send  [Ctrl+R] Raw  [Ctrl+A] Copy cURL" + m.footerMsg + "  [Esc] Quit\n"

	return s
}
