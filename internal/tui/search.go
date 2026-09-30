package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type searchResult struct {
	highlighted  string
	matchLines   []int
	totalMatches int
}

func countMatches(content, query string) int {
	if query == "" || content == "" {
		return 0
	}
	return strings.Count(strings.ToLower(content), strings.ToLower(query))
}

func performSearch(content, query string, currentMatchIdx int) searchResult {
	totalMatches := countMatches(content, query)
	if totalMatches == 0 {
		return searchResult{
			highlighted:  content,
			matchLines:   nil,
			totalMatches: 0,
		}
	}

	currentMatchIdx = (currentMatchIdx%totalMatches + totalMatches) % totalMatches

	lines := strings.Split(content, "\n")
	lowerQ := strings.ToLower(query)
	qLen := len(query)

	var matchLines []int
	var newLines []string
	matchCount := 0

	for lineIdx, line := range lines {
		lowerLine := strings.ToLower(line)
		if !strings.Contains(lowerLine, lowerQ) {
			newLines = append(newLines, line)
			continue
		}

		var b strings.Builder
		lastIdx := 0
		for {
			idx := strings.Index(lowerLine[lastIdx:], lowerQ)
			if idx == -1 {
				b.WriteString(line[lastIdx:])
				break
			}

			start := lastIdx + idx
			end := start + qLen

			b.WriteString(line[lastIdx:start])

			isCurrent := (matchCount == currentMatchIdx)
			matchText := line[start:end]
			if isCurrent {
				b.WriteString(searchCurrentHighlightStyle.Render(matchText))
			} else {
				b.WriteString(searchHighlightStyle.Render(matchText))
			}

			matchLines = append(matchLines, lineIdx)
			matchCount++
			lastIdx = end
		}
		newLines = append(newLines, b.String())
	}

	return searchResult{
		highlighted:  strings.Join(newLines, "\n"),
		matchLines:   matchLines,
		totalMatches: totalMatches,
	}
}

func (m *Model) updateSearch(jump bool) {
	content := m.activeContent()
	if m.searchQuery == "" {
		m.searchMatches = nil
		m.searchMatchIndex = 0
		if m.responseView.Width > 0 {
			wrappedRaw := lipgloss.NewStyle().Width(m.responseView.Width).Render(content)
			m.responseView.SetContent(wrappedRaw)
		} else {
			m.responseView.SetContent(content)
		}
		return
	}

	res := performSearch(content, m.searchQuery, m.searchMatchIndex)
	m.searchMatches = res.matchLines
	if res.totalMatches > 0 {
		m.searchMatchIndex = (m.searchMatchIndex%res.totalMatches + res.totalMatches) % res.totalMatches
	} else {
		m.searchMatchIndex = 0
	}

	if m.responseView.Width > 0 {
		wrappedRaw := lipgloss.NewStyle().Width(m.responseView.Width).Render(res.highlighted)
		m.responseView.SetContent(wrappedRaw)
	} else {
		m.responseView.SetContent(res.highlighted)
	}

	if jump && len(res.matchLines) > 0 {
		targetLine := res.matchLines[m.searchMatchIndex]
		offset := targetLine - m.responseView.Height/2
		if offset < 0 {
			offset = 0
		}
		m.responseView.SetYOffset(offset)
	}
}
