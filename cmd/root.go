// cmd/root.go
package cmd

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

// コマンドラインから受け取るフラグ用変数
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

		// ▼ UIの初期化処理は tui.go 側にある initialModel を呼び出すだけ！
		// 同じ cmd パッケージ内なので、import なしで直接呼び出せます。
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
	// コマンドライン引数 -X の定義
	rootCmd.Flags().StringVarP(&method, "request", "X", "GET", "Specify request command to use")
}
