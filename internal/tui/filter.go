package tui

import (
	"fmt"
	"strings"

	"github.com/nobarudo/gurlt/internal/jsonpath"
)

func (m Model) activeContent() string {
	if m.filteredContent != "" {
		return m.filteredContent
	}
	return m.rawContent
}

func (m *Model) applyJSONFilter() error {
	q := strings.TrimSpace(m.jsonPathQuery)
	if q == "" {
		m.filteredContent = ""
		m.updateSearch(false)
		return nil
	}

	if strings.TrimSpace(m.normalContent) == "" {
		return fmt.Errorf("no response body to filter")
	}

	filtered, err := jsonpath.QueryPretty(m.normalContent, q)
	if err != nil {
		return err
	}

	m.filteredContent = filtered
	m.responseView.GotoTop()
	m.updateSearch(false)
	return nil
}
