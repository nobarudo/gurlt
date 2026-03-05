package cmd

import (
	"encoding/base64"
	"fmt"
	"gurlt/internal/curl"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var (
	method    string
	format    string
	headers   []string
	data      string
	user      string
	userAgent string
	location  bool
	logFile   string
)

var rootCmd = &cobra.Command{
	Use:   "gurlt [url]",
	Short: "A TUI-based HTTP client",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		reqUrl := ""
		if len(args) > 0 {
			reqUrl = args[0]
		}

		// ▼ マジックパーサーの介入！
		// 引数が "curl " から始まっていたら、文字列をパースして変数を上書きする
		if strings.HasPrefix(strings.TrimSpace(reqUrl), "curl ") {
			parsedOpts, err := curl.Parse(reqUrl)
			if err == nil {
				reqUrl = parsedOpts.URL
				if parsedOpts.Method != "" {
					method = parsedOpts.Method
				}
				if parsedOpts.Body != "" {
					data = parsedOpts.Body
				}
				if parsedOpts.User != "" {
					user = parsedOpts.User
				}
				if parsedOpts.UserAgent != "" {
					userAgent = parsedOpts.UserAgent
				}
				if parsedOpts.Location {
					location = parsedOpts.Location
				}
				// 抽出したヘッダーを配列に追加
				headers = append(headers, parsedOpts.Headers...)
			}
		}

		// 1. -d (data) が指定されていて、かつ -X がデフォルト(GET)ならPOSTにする
		if data != "" && method == "GET" {
			method = "POST"
		}

		// ▼ 追加: ボディがJSONの形をしていたら、自動的に format を "json" に切り替える！
		if data != "" && format == "form" {
			trimmed := strings.TrimSpace(data)
			if (strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")) ||
				(strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")) {
				format = "json"
			}
		}

		// 2. ヘッダーの組み立て
		var headerLines []string

		// -A (User-Agent) の処理
		if userAgent != "" {
			headerLines = append(headerLines, fmt.Sprintf("User-Agent: %s", userAgent))
		}

		// -u (Basic Auth) の処理
		if user != "" {
			encoded := base64.StdEncoding.EncodeToString([]byte(user))
			headerLines = append(headerLines, fmt.Sprintf("Authorization: Basic %s", encoded))
		}

		// -H (カスタムヘッダー) の処理
		for _, h := range headers {
			headerLines = append(headerLines, h)
		}

		headerStr := strings.Join(headerLines, "\n")

		// tui.go の initialModel にパースした値を全部渡す
		p := tea.NewProgram(initialModel(reqUrl, method, headerStr, data, format, location, logFile), tea.WithAltScreen())
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
	// gurlt 独自のフラグ
	rootCmd.Flags().StringVarP(&format, "format", "f", "form", "Data format (json, form)")

	// curl 互換フラグ
	rootCmd.Flags().StringVarP(&method, "request", "X", "GET", "Specify request command to use")
	rootCmd.Flags().StringArrayVarP(&headers, "header", "H", []string{}, "Pass custom header(s) to server")
	rootCmd.Flags().StringVarP(&data, "data", "d", "", "HTTP POST data")
	rootCmd.Flags().StringVarP(&user, "user", "u", "", "Server user and password")
	rootCmd.Flags().StringVarP(&userAgent, "user-agent", "A", "", "Send User-Agent <name> to server")
	rootCmd.Flags().BoolVarP(&location, "location", "L", false, "Follow redirects")
	rootCmd.Flags().StringVar(&logFile, "log", "", "Append raw request and response to a file (e.g., --log audit.log)")
}
