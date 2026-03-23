package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/nobarudo/gurlt/internal/tui"

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
	FParseErrWhitelist: cobra.FParseErrWhitelist{
		UnknownFlags: true,
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		urlInput := ""
		if len(args) > 0 {
			urlInput = args[0]
		}

		headerList := strings.Join(headers, "\n")

		extraArgs := getExtraArgs(os.Args[1:])

		m := tui.InitialModel(urlInput, method, headerList, data, format, location, logFile, extraArgs)

		p := tea.NewProgram(m, tea.WithAltScreen())
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

func getExtraArgs(args []string) string {
	var extras []string

	// gurltが既に知っているフラグ
	knownValueFlags := map[string]bool{
		"-X": true, "--request": true,
		"-H": true, "--header": true,
		"-d": true, "--data": true, "--data-raw": true,
	}
	knownBoolFlags := map[string]bool{
		"-L": true, "--location": true,
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]

		// 知っているフラグ（値をとるもの）なら、フラグと次の値をスキップ
		if knownValueFlags[arg] {
			i++
			continue
		}
		// 知っているフラグ（真偽値）なら、フラグだけスキップ
		if knownBoolFlags[arg] {
			continue
		}

		// '-' から始まる知らないフラグを見つけた場合
		if strings.HasPrefix(arg, "-") {
			extras = append(extras, arg)
			// 次の引数が '-' から始まらず、URL（http）でもない場合、それはこのフラグの値とみなす
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") && !strings.HasPrefix(args[i+1], "http") {
				val := args[i+1]
				// 値にスペースが含まれていたらクォーテーションで囲む
				if strings.Contains(val, " ") {
					extras = append(extras, fmt.Sprintf("'%s'", val))
				} else {
					extras = append(extras, val)
				}
				i++
			}
		}
	}
	return strings.Join(extras, " ")
}

func init() {
	// gurlt 独自のフラグ
	rootCmd.Flags().StringVarP(&format, "format", "f", "form", "Data format (json, form)")

	// curl 互換フラグ
	rootCmd.Flags().StringVarP(&method, "request", "X", "GET", "Specify request command to use")
	rootCmd.Flags().StringArrayVarP(&headers, "header", "H", []string{}, "Pass custom header(s) to server")
	rootCmd.Flags().StringVarP(&data, "data", "d", "", "HTTP POST data")
	rootCmd.Flags().StringVar(&data, "data-raw", "", "HTTP POST data (same as --data)")
	rootCmd.Flags().StringVarP(&user, "user", "u", "", "Server user and password")
	rootCmd.Flags().StringVarP(&userAgent, "user-agent", "A", "", "Send User-Agent <name> to server")
	rootCmd.Flags().BoolVarP(&location, "location", "L", false, "Follow redirects")
	rootCmd.Flags().StringVar(&logFile, "log", "", "Append raw request and response to a file (e.g., --log audit.log)")
}
