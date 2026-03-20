package tui

import (
	"fmt"
	"gurlt/internal/curl"
	"strings"
)

func (m Model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	if m.showRawView {
		return m.rawView()
	}

	return m.mainView()
}

func (m Model) rawView() string {
	var content string

	content += titleStyle.Render("📡 gurlt - Raw View") + "\n"
	content += responseBoxStyle.Render(m.responseView.View()) + "\n"
	if m.isSaving {
		content += m.saveInput.View() + "   [Enter] Confirm   [Esc] Cancel"
	} else {
		content += infoStyle.Render("[c/ctrl+a] Copy Raw   [s] Save to File   [ctrl+r] Back") + m.footerMsg
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

	curlPreview := curl.Build(m.methodInput.Value(), m.urlInput.Value(), m.headerInput.Value(), m.bodyInput.Value(), m.format, m.location)
	content += curlPreviewStyle.Copy().Width(contentWidth).Render(fmt.Sprintf("💻 cURL: %s", curlPreview)) + "\n\n"

	helpText := "[ctrl+j/n] Focus↓  [ctrl+k/p] Focus↑  [ctrl+f] Prettify  [ctrl+s] Send  [ctrl+r] Raw  [ctrl+a] cURL Copy" + m.footerMsg
	content += infoStyle.Copy().Width(contentWidth).Render(helpText) + "\n"

	return appStyle.Render(content)
}
