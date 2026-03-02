package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

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
		url := ""
		if len(args) > 0 {
			url = args[0]
		}
		p := tea.NewProgram(initialModel(url, method), tea.WithAltScreen())
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
	status string
	body   string
	err    error
}

type model struct {
	methodInput  textinput.Model
	urlInput     textinput.Model
	headerInput  textarea.Model
	responseView viewport.Model // ▼ 追加: レスポンス表示用のビューポート
	focusIndex   int            // 0: Method, 1: URL, 2: Header, 3: Viewport(Scroll)
	ready        bool           // 画面サイズが取得できて初期化が完了したか

	responseStatus string
	isLoading      bool
	err            error
}

func initialModel(url, method string) model {
	m := textinput.New()
	m.Placeholder = "GET, POST, PUT, DELETE..."
	m.SetValue(method)

	u := textinput.New()
	u.Placeholder = "https://api.example.com"
	u.SetValue(url)
	u.Focus()

	h := textarea.New()
	h.Placeholder = "Key: Value\nAuthorization: Bearer token..."
	h.ShowLineNumbers = false
	h.SetHeight(4)
	h.SetWidth(50)

	defaultHeaders := "User-Agent: gurlt/0.1.0\nAccept: */*"
	if method == "POST" || method == "PUT" || method == "PATCH" {
		defaultHeaders += "\nContent-Type: application/json"
	}
	h.SetValue(defaultHeaders)

	return model{
		methodInput: m,
		urlInput:    u,
		headerInput: h,
		focusIndex:  1,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, textarea.Blink)
}

func sendRequest(method, url, headers string) tea.Cmd {
	return func() tea.Msg {
		req, err := http.NewRequest(method, url, nil)
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

		client := &http.Client{Timeout: 10 * time.Second}
		res, err := client.Do(req)
		if err != nil {
			return responseMsg{err: err}
		}
		defer res.Body.Close()

		bodyBytes, _ := io.ReadAll(res.Body)

		return responseMsg{
			status: res.Status,
			body:   string(bodyBytes),
		}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	// ▼ 追加: ターミナルのサイズ変更イベント（起動時にも必ず1回呼ばれます）
	case tea.WindowSizeMsg:
		// 上部の入力欄などが占有する高さ（約17行分）を引いて、残りをスクロール領域にする
		headerHeight := 17
		viewportHeight := msg.Height - headerHeight
		if viewportHeight < 0 {
			viewportHeight = 0
		}

		if !m.ready {
			// 初回起動時のビューポート初期化
			m.responseView = viewport.New(msg.Width, viewportHeight)
			m.responseView.SetContent("Ready to send request.\nPress Ctrl+S to fetch.")
			m.ready = true
		} else {
			// ターミナルがリサイズされた時に幅と高さを追従させる
			m.responseView.Width = msg.Width
			m.responseView.Height = viewportHeight
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit

		case "tab":
			m.focusIndex++
			if m.focusIndex > 3 { // ▼ 項目が4つ（Viewport含む）になったので > 3 に変更
				m.focusIndex = 0
			}
		case "shift+tab":
			m.focusIndex--
			if m.focusIndex < 0 {
				m.focusIndex = 3
			}

		case "ctrl+s":
			m.isLoading = true
			m.err = nil
			m.responseStatus = ""
			m.responseView.SetContent("⏳ Loading...") // ロード中の表示
			return m, sendRequest(m.methodInput.Value(), m.urlInput.Value(), m.headerInput.Value())
		}

	case responseMsg:
		m.isLoading = false
		m.err = msg.err
		m.responseStatus = msg.status
		if msg.err == nil {
			m.responseView.SetContent(msg.body) // ▼ 取得したHTMLやJSONをそのままViewportに流し込む！
			m.responseView.GotoTop()            // 新しいレスポンスが来たら一番上までスクロールを戻す
		}
		return m, nil
	}

	// Focus/Blur の切り替え
	m.methodInput.Blur()
	m.urlInput.Blur()
	m.headerInput.Blur()

	if m.focusIndex == 0 {
		m.methodInput.Focus()
	} else if m.focusIndex == 1 {
		m.urlInput.Focus()
	} else if m.focusIndex == 2 {
		m.headerInput.Focus()
	}

	// 選択中の入力欄にキーボード操作を流す
	m.methodInput, cmd = m.methodInput.Update(msg)
	cmds = append(cmds, cmd)

	m.urlInput, cmd = m.urlInput.Update(msg)
	cmds = append(cmds, cmd)

	m.headerInput, cmd = m.headerInput.Update(msg)
	cmds = append(cmds, cmd)

	// ▼ フォーカスが Viewport (3) に合っている時だけ、上下キーでスクロールできるようにする
	if m.focusIndex == 3 {
		m.responseView, cmd = m.responseView.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	s := "Welcome to gurlt!\nHow to use gurlt : gurlt -h\n\n"

	// 入力中かどうかの目印（>）をつける親切設計
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
	s += m.headerInput.View() + "\n"

	s += "----------------------------------------\n"

	// ステータスとレスポンスの表示
	if m.err != nil {
		s += fmt.Sprintf("❌ Error: %v\n", m.err)
	} else if m.responseStatus != "" {
		s += fmt.Sprintf("✅ Status: %s\n", m.responseStatus)
	} else {
		s += "\n" // 空白合わせ
	}

	s += "----------------------------------------\n"

	// ▼ Viewport（レスポンス本文）の描画。フォーカスされている時は目立たせる
	if m.focusIndex == 3 {
		s += "[ Response Body (Use ↑/↓/PgUp/PgDn to scroll) ]\n"
	} else {
		s += "[ Response Body ]\n"
	}
	s += m.responseView.View() + "\n"

	// フッター
	s += "----------------------------------------\n"
	s += "[Tab] Focus   [Ctrl+S] Send   [Esc] Quit\n"

	return s
}
