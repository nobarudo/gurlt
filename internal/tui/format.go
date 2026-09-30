package tui

import (
	"bytes"
	"fmt"
	"strings"
)

// isBinaryContent checks if data appears to be binary by scanning for null bytes in the first 1024 bytes.
func isBinaryContent(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	limit := 1024
	if len(data) < limit {
		limit = len(data)
	}
	return bytes.IndexByte(data[:limit], 0) != -1
}

// formatBytes formats byte size into a human-readable string (e.g., 500 B, 1.5 KB, 2.3 MB).
func formatBytes(b int) string {
	if b < 0 {
		return fmt.Sprintf("%d B", b)
	}
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// sanitizeFullDump replaces binary content in full HTTP dump with a placeholder string.
func sanitizeFullDump(fullDump string, replacement string) string {
	respMarker := "=== Response ==="
	idx := strings.LastIndex(fullDump, respMarker)
	if idx == -1 {
		return fullDump
	}

	headerEnd := strings.Index(fullDump[idx:], "\r\n\r\n")
	sepLen := 4
	if headerEnd == -1 {
		headerEnd = strings.Index(fullDump[idx:], "\n\n")
		sepLen = 2
	}
	if headerEnd == -1 {
		return fullDump
	}

	bodyStart := idx + headerEnd + sepLen
	timingMarker := "\n⏱️  Latency:"
	timingIdx := strings.LastIndex(fullDump, timingMarker)

	var sb strings.Builder
	sb.WriteString(fullDump[:bodyStart])
	sb.WriteString(replacement)
	sb.WriteString("\n")
	if timingIdx != -1 && timingIdx >= bodyStart {
		sb.WriteString(fullDump[timingIdx:])
	}
	return sb.String()
}
