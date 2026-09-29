package curl

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mattn/go-shellwords"
)

// ParsedOptions は抽出したcURLのオプションを格納します
type ParsedOptions struct {
	URL            string
	Method         string
	Headers        []string
	Body           string
	User           string
	UserAgent      string
	Location       bool
	Insecure       bool
	Verbose        bool
	Proxy          string
	MaxTime        float64
	ConnectTimeout float64
	IsMultipart    bool
	Forms          []string
	IsDataRaw      bool
	Bearer         string
}

// Parse はcURLコマンドの文字列を安全に分解し、必要な設定だけを抽出します
func Parse(cmdStr string) (*ParsedOptions, error) {
	// OSのターミナルと全く同じルールで、クォーテーションを考慮して分割
	args, err := shellwords.Parse(cmdStr)
	if err != nil {
		return nil, err
	}

	opts := &ParsedOptions{}

	for i := 0; i < len(args); i++ {
		arg := args[i]

		// 先頭の "curl" は無視
		if i == 0 && arg == "curl" {
			continue
		}

		switch arg {
		case "-X", "--request":
			if i+1 < len(args) {
				opts.Method = strings.ToUpper(args[i+1])
				i++
			}
		case "-H", "--header":
			if i+1 < len(args) {
				opts.Headers = append(opts.Headers, args[i+1])
				i++
			}
		case "-d", "--data", "--data-binary", "--data-ascii":
			if i+1 < len(args) {
				opts.Body = args[i+1]
				opts.Method = "POST" // curlの仕様: -dがあるとPOSTになる
				i++
			}
		case "--data-raw":
			if i+1 < len(args) {
				opts.Body = args[i+1]
				opts.IsDataRaw = true
				opts.Method = "POST"
				i++
			}
		case "--json":
			if i+1 < len(args) {
				opts.Body = args[i+1]
				opts.Method = "POST"
				hasAccept := false
				hasContentType := false
				for _, h := range opts.Headers {
					lowerH := strings.ToLower(h)
					if strings.HasPrefix(lowerH, "accept:") {
						hasAccept = true
					}
					if strings.HasPrefix(lowerH, "content-type:") {
						hasContentType = true
					}
				}
				if !hasAccept {
					opts.Headers = append(opts.Headers, "Accept: application/json")
				}
				if !hasContentType {
					opts.Headers = append(opts.Headers, "Content-Type: application/json")
				}
				i++
			}
		case "-u", "--user":
			if i+1 < len(args) {
				opts.User = args[i+1]
				i++
			}
		case "--bearer", "--oauth2-bearer":
			if i+1 < len(args) {
				opts.Bearer = args[i+1]
				opts.Headers = append(opts.Headers, fmt.Sprintf("Authorization: Bearer %s", args[i+1]))
				i++
			}
		case "-A", "--user-agent":
			if i+1 < len(args) {
				opts.UserAgent = args[i+1]
				i++
			}
		case "-L", "--location":
			opts.Location = true
		case "-k", "--insecure":
			opts.Insecure = true
		case "-v", "--verbose":
			opts.Verbose = true
		case "-x", "--proxy":
			if i+1 < len(args) {
				opts.Proxy = args[i+1]
				i++
			}
		case "-m", "--max-time":
			if i+1 < len(args) {
				if val, err := strconv.ParseFloat(args[i+1], 64); err == nil {
					opts.MaxTime = val
				}
				i++
			}
		case "--connect-timeout":
			if i+1 < len(args) {
				if val, err := strconv.ParseFloat(args[i+1], 64); err == nil {
					opts.ConnectTimeout = val
				}
				i++
			}
		case "-F", "--form":
			if i+1 < len(args) {
				opts.Forms = append(opts.Forms, args[i+1])
				opts.Method = "POST"
				opts.IsMultipart = true
				i++
			}
		case "-w", "--write-out":
			if i+1 < len(args) {
				i++
			}
		default:
			// オプションではなく、httpから始まるならURLとして扱う
			if !strings.HasPrefix(arg, "-") && strings.HasPrefix(arg, "http") {
				opts.URL = arg
			}
			// その他未知のオプション（--compressedなど）はすべて無視！
		}
	}

	if opts.IsMultipart && len(opts.Forms) > 0 {
		opts.Body = strings.Join(opts.Forms, "\n")
	}

	return opts, nil
}
