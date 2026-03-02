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
	"github.com/spf13/cobra"
)

var method string

var rootCmd = &cobra.Command{
	Use:   "gurlt [url]",
	Short: "A TUI-based HTTP client",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		reqUrl := ""
		if len(args) > 0 {
			reqUrl = args[0]
		}
		p := tea.NewProgram(initialModel(reqUrl, method), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return err
		}
		return nil
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVarP(&method, "request", "X", "GET", "Specify request command to use")
}

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

	// ▼ 変更: footerMsg に名前を変え、保存用の状態を追加
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

	// ▼ 追加: 保存ファイル名入力欄のセットアップ
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

		rawStr := fmt.Sprintf("%s\n\n%s\n\n%s", reqUrl, string(reqBytes), string(resBytes))

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
			h := m.terminalHeight - 24
			if h < 0 {
				h = 0
			}
			m.responseView.Height = h
		}
		m.responseView.Width = m.terminalWidth

	case tea.KeyMsg:
		// ▼ 追加: 保存モード中のキー操作を優先して処理
		if m.isSaving {
			switch msg.String() {
			case "esc", "ctrl+c":
				m.isSaving = false // キャンセルして元の画面に戻る
				m.saveInput.Blur()
				return m, nil
			case "enter":
				filename := strings.TrimSpace(m.saveInput.Value())
				if filename != "" {
					// os.WriteFile を使うと、現在のカレントディレクトリを基準に保存されます
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
				// その他のキー入力（文字入力）は textinput に流す
				m.saveInput, cmd = m.saveInput.Update(msg)
				return m, cmd
			}
		}

		// ▼ 通常時のキー操作
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

		// ▼ 追加: 's' キーで保存モードに入る
		case "s":
			if m.showRawView && m.rawContent != "" && m.rawContent != "No request sent yet." {
				m.isSaving = true
				m.saveInput.Focus()
				m.saveInput.SetValue("") // 以前の入力をクリア
				m.footerMsg = ""         // メッセージもクリア
				return m, textinput.Blink
			}

		case "ctrl+r":
			m.showRawView = !m.showRawView
			m.footerMsg = ""
			m.isSaving = false // 画面を切り替えたら保存状態もリセット
			if m.showRawView {
				m.responseView.Height = m.terminalHeight - 4
				m.responseView.SetContent(m.rawContent)
			} else {
				h := m.terminalHeight - 24
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

		// ▼ 追加: 保存モード中かそうでないかでフッターを切り替える
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
	s += "[Tab] Focus  [Ctrl+S] Send  [Ctrl+R] Raw View  [Esc] Quit\n"

	return s
}
