package tui

import (
	"fmt"
	"strings"
)

// extractBearerToken extracts the Bearer token value from multiline headers if present.
func extractBearerToken(headers string) string {
	for _, line := range strings.Split(headers, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(trimmed), "authorization:") {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				val := strings.TrimSpace(parts[1])
				if strings.HasPrefix(strings.ToLower(val), "bearer ") {
					return strings.TrimSpace(val[7:])
				}
			}
		}
	}
	return ""
}

// setOrUpdateBearerHeader updates or appends the Bearer token header in the headers string.
// If token is empty, existing Bearer authorization headers are removed.
func setOrUpdateBearerHeader(headers, token string) string {
	var lines []string
	found := false
	for _, line := range strings.Split(headers, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(strings.ToLower(trimmed), "authorization:") {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 && strings.HasPrefix(strings.ToLower(strings.TrimSpace(parts[1])), "bearer ") {
				if token != "" {
					lines = append(lines, fmt.Sprintf("Authorization: Bearer %s", token))
				}
				found = true
				continue
			}
		}
		lines = append(lines, line)
	}
	if !found && token != "" {
		lines = append(lines, fmt.Sprintf("Authorization: Bearer %s", token))
	}
	return strings.Join(lines, "\n")
}
