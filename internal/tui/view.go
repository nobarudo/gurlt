package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
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
	} else {
		content += infoStyle.Render("[c/ctrl+a] Copy Raw   [s] Save to File   [ctrl+r] Back") + m.footerMsg + "\n"
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
	extraVal := m.extraArgs
	if extraVal == "" {
		extraVal = "(none)"
	}

	b.WriteString(modalSectionTitleStyle.Render("Current Configuration:") + "\n")
	b.WriteString(modalItemStyle.Render(fmt.Sprintf("  • Data Format (-f): %s", formatVal)) + "\n")
	b.WriteString(modalItemStyle.Render(fmt.Sprintf("  • Log File (--log): %s", logVal)) + "\n")
	if extraVal != "(none)" {
		b.WriteString(modalItemStyle.Render(fmt.Sprintf("  • Extra cURL Args:  %s", extraVal)) + "\n")
	}
	b.WriteString("\n")

	// Help text
	if m.proxyInput.Focused() {
		b.WriteString(modalHelpStyle.Render("[Type] Input URL   [Enter/Esc] Done Editing"))
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

