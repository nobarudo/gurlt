# gurlt

gurlt is a CLI command designed to be a comfortable text-based user interface version of `curl`.

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

## Installation

```bash
go install github.com/nobarudo/gurlt@latest
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

**3.cURL Parse**

Paste a raw cURL command (e.g., copied from Chrome DevTools) inside quotes. `gurlt` will automatically parse the necessary data and ignore the noise.

```bash
gurlt "curl 'https://api.example.com' -H 'Accept: */*' --compressed --insecure"

```

**4. Audit Logging**

Automatically save request and response dumps to a file.

```bash
gurlt --log audit.log https://example.com
```

**5. Options Modal (`Ctrl+O`)**

Press `Ctrl+O` from the main view to open the options modal and configure advanced cURL settings:
- `-k / --insecure`: Ignore SSL certificate verification errors
- `-v / --verbose`: Detailed logging
- `-L / --location`: Follow HTTP redirects
- `-x`: Specify HTTP/HTTPS proxy URL
- View current configuration (`--format`, `--log`, and extra CLI arguments)

Changes made in the modal are immediately reflected in the live `💻 cURL:` preview and copied with `Ctrl+A`.

## ⌨️ Keybindings

### Main View

| Key | Action |
| --- | --- |
| `Ctrl+J` / `Ctrl+N` | Move focus down |
| `Ctrl+K` / `Ctrl+P` | Move focus up |
| `Ctrl+S` | Send request |
| `Ctrl+R` | Toggle Raw View |
| `Ctrl+F` | Prettify JSON body |
| `Ctrl+L` | Toggle redirect follow (`-L / --location`) |
| `Ctrl+O` | Open Options Modal |
| `Ctrl+A` | Copy cURL command |
| `Esc` / `Ctrl+C` | Quit |

### Options Modal (`Ctrl+O`)

| Key | Action |
| --- | --- |
| `j` / `k` (or `↓` / `↑`, `Tab`) | Move item |
| `Space` | Toggle checkbox / Edit Proxy URL |
| `Enter` / `Esc` | Finish editing Proxy URL |
| `Esc` / `Ctrl+O` | Close Options Modal |

### Raw View (`Ctrl+R`)

| Key | Action |
| --- | --- |
| `Ctrl+A` / `C` | Copy Raw Dump |
| `S` | Save Raw Dump to file |
| `Ctrl+R` | Back to Main View |

## 📄 License

MIT License
