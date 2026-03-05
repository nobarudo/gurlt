# gurlt

A simple, TUI-based HTTP client written in Go.
Designed for security engineers and developers who want the power of `curl` with the comfort of a Text User Interface.

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

## Installation

```bash
go install https://github.com/nobarudo/gurlt@latest
```

## Usage

`gurlt` supports standard `curl` flags. You can simply replace `curl` with `gurlt` in your snippets.

**1. Basic Request**

```bash
gurlt https://example.com/
```

**2. With Flags (-X, -H, -d, -u, -A, -L)**

```bash
gurlt -X POST -H "Authorization: Bearer token" -d '{"test":123}' https://httpbin.org/post

```

**3. Magic cURL Parse**
Paste a raw cURL command (e.g., copied from Chrome DevTools) inside quotes. `gurlt` will automatically parse the necessary data and ignore the noise.

```bash
gurlt "curl 'https://api.example.com' -H 'Accept: */*' --compressed --insecure"

```

**4. Audit Logging**
Automatically save request and response dumps to a file.

```bash
gurlt --log audit.log https://example.com

```

## ⌨️ Keybindings

| Key | Action |
| --- | --- |
| `Ctrl+J` / `Ctrl+K` | Move focus (Down / Up) |
| `Ctrl+S` | Send request |
| `Ctrl+R` | Toggle Raw View |
| `Ctrl+F` | Prettify JSON body |
| `Ctrl+A` | Copy cURL command (or Raw Dump in Raw View) |
| `C` | Copy Raw Dump (in Raw View) |
| `S` | Save Raw Dump to file (in Raw View) |
| `Esc` / `Ctrl+C` | Quit |

## 📄 License

MIT License
