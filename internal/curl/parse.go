package curl

import (
	"strings"

	"github.com/mattn/go-shellwords"
)

// ParsedOptions は抽出したcURLのオプションを格納します
type ParsedOptions struct {
	URL       string
	Method    string
	Headers   []string
	Body      string
	User      string
	UserAgent string
	Location  bool
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
		case "-d", "--data", "--data-raw", "--data-binary":
			if i+1 < len(args) {
				opts.Body = args[i+1]
				opts.Method = "POST" // curlの仕様: -dがあるとPOSTになる
				i++
			}
		case "-u", "--user":
			if i+1 < len(args) {
				opts.User = args[i+1]
				i++
			}
		case "-A", "--user-agent":
			if i+1 < len(args) {
				opts.UserAgent = args[i+1]
				i++
			}
		case "-L", "--location":
			opts.Location = true
		default:
			// オプションではなく、httpから始まるならURLとして扱う
			if !strings.HasPrefix(arg, "-") && strings.HasPrefix(arg, "http") {
				opts.URL = arg
			}
			// その他未知のオプション（--compressedなど）はすべて無視！
		}
	}

	return opts, nil
}
