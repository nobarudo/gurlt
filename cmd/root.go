package cmd

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/nobarudo/gurlt/internal/curl"
	"github.com/nobarudo/gurlt/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var (
	method         string
	format         string
	headers        []string
	data           string
	dataRaw        string
	forms          []string
	user           string
	userAgent      string
	jsonData       string
	maxTime        float64
	connectTimeout float64
	location       bool
	insecure       bool
	proxy          string
	logFile        string
	writeOut       string
	jqQuery        string
	bearerToken    string
	outputFile     string
)

var rootCmd = &cobra.Command{
	Use:   "gurlt [url]",
	Short: "A TUI-based HTTP client",
	Args:  cobra.MaximumNArgs(1),
	FParseErrWhitelist: cobra.FParseErrWhitelist{
		UnknownFlags: true,
	},
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		urlInput := ""
		if len(args) > 0 {
			urlInput = args[0]
		}

		var parsedOpts *curl.ParsedOptions
		// 引数が "curl " から始まっていたら、文字列をパースして変数を上書きする
		if strings.HasPrefix(strings.TrimSpace(urlInput), "curl ") {
			var err error
			parsedOpts, err = curl.Parse(urlInput)
			if err == nil {
				urlInput = parsedOpts.URL
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
				if parsedOpts.Insecure {
					insecure = parsedOpts.Insecure
				}
				if parsedOpts.Proxy != "" {
					proxy = parsedOpts.Proxy
				}
				if parsedOpts.MaxTime > 0 {
					maxTime = parsedOpts.MaxTime
				}
				if parsedOpts.ConnectTimeout > 0 {
					connectTimeout = parsedOpts.ConnectTimeout
				}
				if parsedOpts.Bearer != "" && bearerToken == "" {
					bearerToken = parsedOpts.Bearer
				}
				if parsedOpts.OutputFile != "" && outputFile == "" {
					outputFile = parsedOpts.OutputFile
				}
				headers = append(headers, parsedOpts.Headers...)
			}
		}

		// --data-raw または parsedOpts.IsDataRaw の場合、@ によるファイル読み込みは行わない
		isRaw := cmd.Flags().Changed("data-raw") || (parsedOpts != nil && parsedOpts.IsDataRaw)
		if dataRaw != "" {
			data = dataRaw
		}

		// --json オプションが指定された場合
		if jsonData != "" {
			loaded, err := curl.LoadPayload(jsonData)
			if err != nil {
				return err
			}
			data = loaded
			format = "json"
			if method == "GET" {
				method = "POST"
			}
			hasAccept := false
			hasContentType := false
			for _, h := range headers {
				lowerH := strings.ToLower(h)
				if strings.HasPrefix(lowerH, "accept:") {
					hasAccept = true
				}
				if strings.HasPrefix(lowerH, "content-type:") {
					hasContentType = true
				}
			}
			if !hasAccept {
				headers = append(headers, "Accept: application/json")
			}
			if !hasContentType {
				headers = append(headers, "Content-Type: application/json")
			}
		}

		// -F, --form オプションが指定された場合
		if len(forms) > 0 {
			data = strings.Join(forms, "\n")
			format = "multipart"
			if method == "GET" {
				method = "POST"
			}
		} else if parsedOpts != nil && parsedOpts.IsMultipart {
			format = "multipart"
			if method == "GET" {
				method = "POST"
			}
		} else if !isRaw && data != "" && strings.HasPrefix(data, "@") {
			loaded, err := curl.LoadPayload(data)
			if err != nil {
				return err
			}
			data = loaded
		}

		// -d (data) が指定されていて、かつ -X がデフォルト(GET)ならPOSTにする
		if data != "" && method == "GET" {
			method = "POST"
		}

		// ボディがJSONの形をしていたら、自動的に format を "json" に切り替える
		if data != "" && format == "form" {
			trimmed := strings.TrimSpace(data)
			if (strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")) ||
				(strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")) {
				format = "json"
			}
		}

		// ヘッダーの組み立て
		var headerLines []string
		if userAgent != "" {
			headerLines = append(headerLines, fmt.Sprintf("User-Agent: %s", userAgent))
		}
		if user != "" {
			encoded := base64.StdEncoding.EncodeToString([]byte(user))
			headerLines = append(headerLines, fmt.Sprintf("Authorization: Basic %s", encoded))
		} else if bearerToken != "" {
			hasAuth := false
			for _, h := range headers {
				if strings.HasPrefix(strings.ToLower(strings.TrimSpace(h)), "authorization:") {
					hasAuth = true
					break
				}
			}
			if !hasAuth {
				headerLines = append(headerLines, fmt.Sprintf("Authorization: Bearer %s", bearerToken))
			}
		}
		for _, h := range headers {
			headerLines = append(headerLines, h)
		}
		headerList := strings.Join(headerLines, "\n")

		extraArgs := getExtraArgs(os.Args[1:])

		m := tui.InitialModel(urlInput, method, headerList, data, format, location, logFile, extraArgs)
		if maxTime > 0 {
			m.SetMaxTime(maxTime)
		}
		if connectTimeout > 0 {
			m.SetConnectTimeout(connectTimeout)
		}
		if insecure {
			m.SetInsecure(true)
		}
		if proxy != "" {
			m.SetProxy(proxy)
		}
		if parsedOpts != nil {
			if parsedOpts.Verbose {
				m.SetVerbose(true)
			}
		}
		if jqQuery != "" {
			m.SetJSONPathQuery(jqQuery)
		}
		if bearerToken != "" {
			m.SetBearer(bearerToken)
		}
		if outputFile != "" {
			m.SetOutputFile(outputFile)
		}

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
		"-d": true, "--data": true, "--data-raw": true, "--data-binary": true, "--data-ascii": true,
		"-F": true, "--form": true,
		"--json":            true,
		"--bearer":          true, "--oauth2-bearer": true,
		"-u":                true, "--user": true,
		"-A":                true, "--user-agent": true,
		"-f":                true, "--format": true,
		"-m":                true, "--max-time": true,
		"--connect-timeout": true,
		"-x":                true, "--proxy": true,
		"-w":                true, "--write-out": true,
		"-q":                true, "--query": true, "--jq": true,
		"-o":                true, "--output": true,
		"--log":             true,
	}
	knownBoolFlags := map[string]bool{
		"-L": true, "--location": true,
		"-k": true, "--insecure": true,
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]

		// --flag=value 形式の場合のチェック
		flagName := arg
		if eqIdx := strings.Index(arg, "="); eqIdx != -1 {
			flagName = arg[:eqIdx]
			if knownValueFlags[flagName] {
				continue
			}
		}

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
	rootCmd.Flags().StringVarP(&format, "format", "f", "form", "Data format (json, form, multipart)")

	// curl 互換フラグ
	rootCmd.Flags().StringVarP(&method, "request", "X", "GET", "Specify request command to use")
	rootCmd.Flags().StringArrayVarP(&headers, "header", "H", []string{}, "Pass custom header(s) to server")
	rootCmd.Flags().StringVarP(&data, "data", "d", "", "HTTP POST data")
	rootCmd.Flags().StringVar(&dataRaw, "data-raw", "", "HTTP POST data (same as --data, but @ is not treated as a file)")
	rootCmd.Flags().StringVar(&data, "data-binary", "", "HTTP POST binary data")
	rootCmd.Flags().StringVar(&data, "data-ascii", "", "HTTP POST ASCII data")
	rootCmd.Flags().StringArrayVarP(&forms, "form", "F", []string{}, "Specify multipart MIME data")
	rootCmd.Flags().StringVar(&jsonData, "json", "", "HTTP POST data with JSON content-type and accept headers")
	rootCmd.Flags().StringVar(&bearerToken, "bearer", "", "OAuth 2.0 / Bearer token for Authorization header")
	rootCmd.Flags().StringVar(&bearerToken, "oauth2-bearer", "", "OAuth 2.0 Bearer token (curl compatible)")
	rootCmd.Flags().StringVarP(&user, "user", "u", "", "Server user and password")
	rootCmd.Flags().StringVarP(&userAgent, "user-agent", "A", "", "Send User-Agent <name> to server")
	rootCmd.Flags().Float64VarP(&maxTime, "max-time", "m", 0, "Maximum time allowed for the transfer (in seconds)")
	rootCmd.Flags().Float64Var(&connectTimeout, "connect-timeout", 0, "Maximum time allowed for connection (in seconds)")
	rootCmd.Flags().BoolVarP(&insecure, "insecure", "k", false, "Allow insecure server connections when using SSL")
	rootCmd.Flags().StringVarP(&proxy, "proxy", "x", "", "[protocol://]host[:port] Use this proxy")
	rootCmd.Flags().BoolVarP(&location, "location", "L", false, "Follow redirects")
	rootCmd.Flags().StringVarP(&writeOut, "write-out", "w", "", "Output format after completion (curl compatible)")
	rootCmd.Flags().StringVarP(&jqQuery, "query", "q", "", "Filter JSON response using JSON path (e.g. .data.users[0])")
	rootCmd.Flags().StringVar(&jqQuery, "jq", "", "Filter JSON response using JSON path (same as -q)")
	rootCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Write to file instead of stdout/display")
	rootCmd.Flags().StringVar(&logFile, "log", "", "Append raw request and response to a file (e.g., --log audit.log)")
}
