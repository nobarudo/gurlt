package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/nobarudo/gurlt/internal/client"
)

func (m Model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	if m.showOptionsModal {
		return m.optionsModalView()
	}

	if m.showRawView {
		return m.rawView()
	}

	return m.mainView()
}

func (m Model) rawView() string {
	var content string

	content += titleStyle.Render("📡 gurlt - Raw View") + "\n"
	content += responseBoxStyle.Render(m.responseView.View()) + "\n\n"
	if m.isSaving {
		content += m.saveInput.View() + "   [Enter] Confirm   [Esc] Cancel"
	} else if m.isFiltering {
		content += m.filterInput.View() + "   [Enter] Apply   [Esc] Cancel"
	} else if m.isSearching {
		matchInfo := "[0/0]"
		if len(m.searchMatches) > 0 {
			matchInfo = fmt.Sprintf("[%d/%d]", m.searchMatchIndex+1, len(m.searchMatches))
		}
		content += m.searchInput.View() + " " + searchCountStyle.Render(matchInfo) + "   [Enter] Next   [Shift+Tab] Prev   [Esc] Done"
	} else if m.searchQuery != "" {
		matchInfo := "[0/0]"
		if len(m.searchMatches) > 0 {
			matchInfo = fmt.Sprintf("[%d/%d]", m.searchMatchIndex+1, len(m.searchMatches))
		}
		content += searchCountStyle.Render(matchInfo) + " " + infoStyle.Render("[n] Next   [N] Prev   [/] Edit   [Esc] Clear Search   [c] Copy   [ctrl+r] Back") + m.footerMsg + "\n"
	} else if m.jsonPathQuery != "" {
		filterTag := searchCountStyle.Render(fmt.Sprintf("[Filter: %s]", m.jsonPathQuery))
		content += filterTag + " " + infoStyle.Render("[p/f] Edit Filter   [Esc] Clear Filter   [/] Search   [c] Copy   [ctrl+r] Back") + m.footerMsg + "\n"
	} else {
		content += infoStyle.Render("[/] Search   [p/f] JSON Filter   [c/ctrl+a] Copy Raw   [s] Save to File   [ctrl+r] Back") + m.footerMsg + "\n"
	}
	return appStyle.Render(content)
}

func (m Model) mainView() string {
	var content string

	content += titleStyle.Render("🚀 gurlt - TUI HTTP Client") + "\n"
	renderLabel := func(label string, isFocused bool) string {
		if isFocused {
			return focusedLabelStyle.Render("▶ " + label)
		}
		return blurredLabelStyle.Render("  " + label)
	}

	content += renderLabel("Method:", m.focusIndex == 0) + " " + m.methodInput.View() + "\n"
	content += renderLabel("URL:   ", m.focusIndex == 1) + " " + m.urlInput.View() + "\n"
	content += renderLabel("Headers:", m.focusIndex == 2) + "\n" + m.headerInput.View() + "\n"

	bodyLabel := "Params (key=value):"
	if m.format == "json" {
		bodyLabel = "Body (JSON):"
	} else if m.format == "multipart" {
		bodyLabel = "Form Data (key=val or key=@file):"
	}
	content += renderLabel(bodyLabel, m.focusIndex == 3) + "\n" + m.bodyInput.View() + "\n"
	content += dividerStyle.Render(strings.Repeat("─", m.terminalWidth-10)) + "\n"

	if len(m.history) > 0 {
		for _, h := range m.history {
			statusLine := fmt.Sprintf("Status: %s (%s %s)", h.Status, h.Method, h.URL)
			if strings.HasPrefix(h.Status, "2") {
				content += successStyle.Render("✅ "+statusLine) + "\n"
			} else if strings.HasPrefix(h.Status, "3") {
				content += infoStyle.Render("↪️ "+statusLine) + "\n" // リダイレクトは青/黄色系
			} else {
				content += errorStyle.Render("⚠️ "+statusLine) + "\n"
			}
		}
	} else if m.responseStatus != "" {
		content += errorStyle.Render(fmt.Sprintf("⚠️ Status: %s", m.responseStatus)) + "\n"
	} else {
		content += "\n"
	}

	if m.timing.Total > 0 {
		content += m.renderTimingView() + "\n"
	}

	content += dividerStyle.Render(strings.Repeat("─", m.terminalWidth-10)) + "\n"

	contentWidth := m.terminalWidth - 10
	if contentWidth < 1 {
		contentWidth = 1
	}

	curlPreview := m.BuildCurlCmd()

	locStatus := "OFF"
	if m.location {
		locStatus = "ON"
	}
	content += curlPreviewStyle.Copy().Width(contentWidth).Render(fmt.Sprintf("💻 cURL: %s", curlPreview)) + "\n"

	helpText := "[ctrl+j/n] Focus↓  [ctrl+k/p] Focus↑  [ctrl+f] Prettify  [ctrl+s] Send  [ctrl+r] Raw  [ctrl+l] location " + locStatus + "  [ctrl+o] Options  [ctrl+a] cURL Copy" + m.footerMsg
	content += infoStyle.Copy().Width(contentWidth).Render(helpText)

	return appStyle.Render(content)
}

func (m Model) optionsModalView() string {
	var b strings.Builder
	b.WriteString(modalTitleStyle.Render("⚙️  Options & Settings") + "\n\n")

	renderItem := func(index int, label string, isChecked bool) string {
		cursor := "  "
		if m.optionsCursor == index {
			cursor = "▶ "
		}
		check := "[ ]"
		if isChecked {
			check = "[x]"
		}
		line := fmt.Sprintf("%s%s %s", cursor, check, label)
		if m.optionsCursor == index {
			return modalSelectStyle.Render(line)
		}
		return modalItemStyle.Render(line)
	}

	// 1. -k / --insecure
	b.WriteString(renderItem(0, "-k, --insecure (Ignore SSL certificate errors)", m.insecure) + "\n")

	// 2. -v / --verbose
	b.WriteString(renderItem(1, "-v, --verbose  (Detailed log)", m.verbose) + "\n")

	// 3. -L / --location
	b.WriteString(renderItem(2, "-L, --location (Follow redirects)", m.location) + "\n\n")

	// 4. -x Proxy
	proxyCursor := "  "
	if m.optionsCursor == 3 {
		proxyCursor = "▶ "
	}
	proxyLabel := proxyCursor + "Proxy (-x):"
	if m.optionsCursor == 3 {
		if m.proxyInput.Focused() {
			proxyLabel += " (Editing... [Enter/Esc] Done)"
		} else {
			proxyLabel += " (Press Space to edit)"
		}
		b.WriteString(modalSelectStyle.Render(proxyLabel) + "\n")
	} else {
		b.WriteString(modalItemStyle.Render(proxyLabel) + "\n")
	}
	b.WriteString(m.proxyInput.View() + "\n\n")

	// 5. -m Timeout
	timeoutCursor := "  "
	if m.optionsCursor == 4 {
		timeoutCursor = "▶ "
	}
	timeoutLabel := timeoutCursor + "Timeout (-m, seconds):"
	if m.optionsCursor == 4 {
		if m.timeoutInput.Focused() {
			timeoutLabel += " (Editing... [Enter/Esc] Done)"
		} else {
			timeoutLabel += " (Press Space to edit)"
		}
		b.WriteString(modalSelectStyle.Render(timeoutLabel) + "\n")
	} else {
		b.WriteString(modalItemStyle.Render(timeoutLabel) + "\n")
	}
	b.WriteString(m.timeoutInput.View() + "\n\n")

	// 6. --bearer Bearer Token
	bearerCursor := "  "
	if m.optionsCursor == 5 {
		bearerCursor = "▶ "
	}
	bearerLabel := bearerCursor + "Bearer Token (--bearer):"
	if m.optionsCursor == 5 {
		if m.bearerInput.Focused() {
			bearerLabel += " (Editing... [Enter/Esc] Done)"
		} else {
			bearerLabel += " (Press Space to edit)"
		}
		b.WriteString(modalSelectStyle.Render(bearerLabel) + "\n")
	} else {
		b.WriteString(modalItemStyle.Render(bearerLabel) + "\n")
	}
	b.WriteString(m.bearerInput.View() + "\n\n")

	// 7. -o Output File
	outputCursor := "  "
	if m.optionsCursor == 6 {
		outputCursor = "▶ "
	}
	outputLabel := outputCursor + "Output File (-o):"
	if m.optionsCursor == 6 {
		if m.outputInput.Focused() {
			outputLabel += " (Editing... [Enter/Esc] Done)"
		} else {
			outputLabel += " (Press Space to edit)"
		}
		b.WriteString(modalSelectStyle.Render(outputLabel) + "\n")
	} else {
		b.WriteString(modalItemStyle.Render(outputLabel) + "\n")
	}
	b.WriteString(m.outputInput.View() + "\n\n")

	// Divider
	b.WriteString(dividerStyle.Render(strings.Repeat("─", 54)) + "\n")

	// Read-only info for current flags / environment
	formatVal := m.format
	if formatVal == "" {
		formatVal = "form"
	}
	logVal := m.logFile
	if logVal == "" {
		logVal = "(none)"
	}
	outputVal := strings.TrimSpace(m.outputInput.Value())
	if outputVal == "" {
		outputVal = "(none)"
	}
	extraVal := m.extraArgs
	if extraVal == "" {
		extraVal = "(none)"
	}
	timeoutVal := "(none)"
	if m.maxTime > 0 {
		timeoutVal = fmt.Sprintf("%ss", strconv.FormatFloat(m.maxTime, 'f', -1, 64))
	}
	connTimeoutVal := "(none)"
	if m.connectTimeout > 0 {
		connTimeoutVal = fmt.Sprintf("%ss", strconv.FormatFloat(m.connectTimeout, 'f', -1, 64))
	}

	b.WriteString(modalSectionTitleStyle.Render("Current Configuration:") + "\n")
	b.WriteString(modalItemStyle.Render(fmt.Sprintf("  • Data Format (-f):    %s", formatVal)) + "\n")
	b.WriteString(modalItemStyle.Render(fmt.Sprintf("  • Max Time (-m):       %s", timeoutVal)) + "\n")
	if m.connectTimeout > 0 {
		b.WriteString(modalItemStyle.Render(fmt.Sprintf("  • Connect Timeout:     %s", connTimeoutVal)) + "\n")
	}
	bearerVal := strings.TrimSpace(m.bearerInput.Value())
	if bearerVal != "" {
		masked := bearerVal
		if len(masked) > 16 {
			masked = masked[:6] + "..." + masked[len(masked)-4:]
		}
		b.WriteString(modalItemStyle.Render(fmt.Sprintf("  • Bearer Token:        %s", masked)) + "\n")
	}
	b.WriteString(modalItemStyle.Render(fmt.Sprintf("  • Output File (-o):    %s", outputVal)) + "\n")
	b.WriteString(modalItemStyle.Render(fmt.Sprintf("  • Log File (--log):    %s", logVal)) + "\n")
	if extraVal != "(none)" {
		b.WriteString(modalItemStyle.Render(fmt.Sprintf("  • Extra cURL Args:     %s", extraVal)) + "\n")
	}
	if m.timing.Total > 0 {
		b.WriteString(modalItemStyle.Render(fmt.Sprintf("  • Latency (Last):      %s (TTFB: %s)", client.FormatDuration(m.timing.Total), client.FormatDuration(m.timing.ServerProcessing))) + "\n")
	}
	if m.jsonPathQuery != "" {
		b.WriteString(modalItemStyle.Render(fmt.Sprintf("  • JSON Path Filter:    %s", m.jsonPathQuery)) + "\n")
	}
	b.WriteString("\n")

	// Help text
	if m.proxyInput.Focused() || m.timeoutInput.Focused() || m.bearerInput.Focused() || m.outputInput.Focused() {
		b.WriteString(modalHelpStyle.Render("[Type] Input value   [Enter/Esc] Done Editing"))
	} else {
		b.WriteString(modalHelpStyle.Render("[j/k] Move   [Space] Toggle / Edit   [Esc/ctrl+o] Back"))
	}

	modal := modalBoxStyle.Render(b.String())

	width := m.terminalWidth
	if width < 1 {
		width = 80
	}
	height := m.terminalHeight
	if height < 1 {
		height = 24
	}

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, modal)
}

func (m Model) renderTimingView() string {
	if m.timing.Total <= 0 {
		return ""
	}

	barWidth := m.terminalWidth - 30
	if barWidth > 40 {
		barWidth = 40
	}
	if barWidth < 15 {
		barWidth = 15
	}

	dns := m.timing.DNSLookup
	tcp := m.timing.TCPConnect
	tls := m.timing.TLSHandshake
	ttfb := m.timing.ServerProcessing
	transfer := m.timing.ContentTransfer
	total := m.timing.Total

	calcWidth := func(d time.Duration) int {
		if d <= 0 || total <= 0 {
			return 0
		}
		w := int(float64(d) / float64(total) * float64(barWidth))
		if w == 0 && d > 0 {
			w = 1
		}
		return w
	}

	wDNS := calcWidth(dns)
	wTCP := calcWidth(tcp)
	wTLS := calcWidth(tls)
	wTTFB := calcWidth(ttfb)
	wTransfer := calcWidth(transfer)

	sum := wDNS + wTCP + wTLS + wTTFB + wTransfer
	if sum > barWidth {
		diff := sum - barWidth
		if wTTFB > diff {
			wTTFB -= diff
		} else if wTransfer > diff {
			wTransfer -= diff
		}
	} else if sum < barWidth {
		wTTFB += (barWidth - sum)
	}

	bar := "[" +
		timingDNSStyle.Render(strings.Repeat("█", wDNS)) +
		timingTCPStyle.Render(strings.Repeat("█", wTCP)) +
		timingTLSStyle.Render(strings.Repeat("█", wTLS)) +
		timingTTFBStyle.Render(strings.Repeat("█", wTTFB)) +
		timingTransferStyle.Render(strings.Repeat("█", wTransfer)) +
		"]"

	var legendParts []string
	if dns > 0 {
		legendParts = append(legendParts, timingDNSStyle.Render(fmt.Sprintf("DNS: %s", client.FormatDuration(dns))))
	}
	if tcp > 0 {
		legendParts = append(legendParts, timingTCPStyle.Render(fmt.Sprintf("TCP: %s", client.FormatDuration(tcp))))
	}
	if tls > 0 {
		legendParts = append(legendParts, timingTLSStyle.Render(fmt.Sprintf("TLS: %s", client.FormatDuration(tls))))
	}
	if ttfb > 0 {
		legendParts = append(legendParts, timingTTFBStyle.Render(fmt.Sprintf("TTFB: %s", client.FormatDuration(ttfb))))
	}
	if transfer > 0 {
		legendParts = append(legendParts, timingTransferStyle.Render(fmt.Sprintf("Transfer: %s", client.FormatDuration(transfer))))
	}

	legend := strings.Join(legendParts, " │ ")

	title := timingTotalStyle.Render(fmt.Sprintf("⏱️  Latency: %s", client.FormatDuration(total)))

	return fmt.Sprintf("%s  %s\n%s", title, bar, legend)
}

