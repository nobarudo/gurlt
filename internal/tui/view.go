package tui

import (
	"fmt"
	"gurlt/internal/curl"
	"strings"
)

func (m Model) requestView() string {
	// リクエストパネルの描画ロジック
	return "Request Panel Content"
}

func (m Model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	if m.showRawView {
		m.rawView()

	}
	return m.mainView()

}

func (m Model) rawView() string {
	var content string

	content += titleStyle.Render("📡 gurlt - Raw View") + "\n\n"
	content += responseBoxStyle.Render(m.responseView.View()) + "\n\n"
	if m.isSaving {
		content += m.saveInput.View() + "   [Enter] Confirm   [Esc] Cancel"
	} else {
		content += infoStyle.Render("[c/ctrl+a] Copy Raw   [s] Save to File   [ctrl+r] Back") + m.footerMsg
	}
	return appStyle.Render(content)
}

func (m Model) mainView() string {
	var content string

	content += titleStyle.Render("🚀 gurlt - TUI HTTP Client") + "\n\n"
	renderLabel := func(label string, isFocused bool) string {
		if isFocused {
			return focusedLabelStyle.Render("▶ " + label)
		}
		return blurredLabelStyle.Render("  " + label)
	}
	content += renderLabel("Method:", m.focusIndex == 0) + " " + m.methodInput.View() + "\n"
	content += renderLabel("URL:   ", m.focusIndex == 1) + " " + m.urlInput.View() + "\n\n"
	content += renderLabel("Headers:", m.focusIndex == 2) + "\n" + m.headerInput.View() + "\n\n"

	bodyLabel := "Params (key=value):"
	if m.format == "json" {
		bodyLabel = "Body (JSON):"
	}
	content += renderLabel(bodyLabel, m.focusIndex == 3) + "\n" + m.bodyInput.View() + "\n"
	content += dividerStyle.Render(strings.Repeat("─", m.terminalWidth-10)) + "\n"

	if m.responseStatus != "" {
		if strings.HasPrefix(m.responseStatus, "2") {
			content += successStyle.Render(fmt.Sprintf("✅ Status: %s", m.responseStatus)) + "\n"
		} else {
			content += errorStyle.Render(fmt.Sprintf("⚠️ Status: %s", m.responseStatus)) + "\n"
		}
	} else {
		content += "\n"
	}

	content += renderLabel("Response Body:", m.focusIndex == 4) + "\n"
	content += responseBoxStyle.Render(m.responseView.View()) + "\n"
	content += dividerStyle.Render(strings.Repeat("─", m.terminalWidth-10)) + "\n"

	curlPreview := curl.Build(m.methodInput.Value(), m.urlInput.Value(), m.headerInput.Value(), m.bodyInput.Value(), m.format, m.location)
	if len(curlPreview) > m.terminalWidth-20 {
		curlPreview = curlPreview[:m.terminalWidth-25] + "..."
	}
	content += curlPreviewStyle.Render(fmt.Sprintf("💻 cURL: %s", curlPreview)) + "\n\n"

	content += infoStyle.Render("[ctrl+j/n] Focus↓  [ctrl+k/p] Focus↑  [ctrl+f] Prettify  [ctrl+s] Send  [ctrl+r] Raw  [ctrl+a] cURL Copy") + m.footerMsg + "\n"

	return appStyle.Render(content)
}
