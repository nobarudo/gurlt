package client

import (
	"fmt"
	"strings"
	"time"
)

// TimingInfo holds network latency breakdown metrics similar to curl -w
type TimingInfo struct {
	DNSLookup        time.Duration // Time spent on DNS resolution
	TCPConnect       time.Duration // Time spent establishing TCP connection
	TLSHandshake     time.Duration // Time spent on TLS/SSL handshake
	ServerProcessing time.Duration // Time from request sent to first byte received (TTFB)
	ContentTransfer  time.Duration // Time spent transferring response body
	Total            time.Duration // Total elapsed time from request start to body read completion

	// curl -w equivalent cumulative durations from start
	NameLookup    time.Duration // time_namelookup
	Connect       time.Duration // time_connect
	AppConnect    time.Duration // time_appconnect
	StartTransfer time.Duration // time_starttransfer
}

// FormatDuration formats a time.Duration into a human-readable string (e.g., "12ms", "1.50s")
func FormatDuration(d time.Duration) string {
	if d <= 0 {
		return "0ms"
	}
	if d < time.Millisecond {
		return fmt.Sprintf("%.2fms", float64(d.Microseconds())/1000.0)
	}
	if d < time.Second {
		ms := float64(d.Microseconds()) / 1000.0
		if ms < 10 {
			return fmt.Sprintf("%.1fms", ms)
		}
		return fmt.Sprintf("%.0fms", ms)
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

// Summary returns a single-line summary of the latency breakdown
func (t TimingInfo) Summary() string {
	parts := []string{}
	if t.DNSLookup > 0 {
		parts = append(parts, fmt.Sprintf("DNS: %s", FormatDuration(t.DNSLookup)))
	}
	if t.TCPConnect > 0 {
		parts = append(parts, fmt.Sprintf("TCP: %s", FormatDuration(t.TCPConnect)))
	}
	if t.TLSHandshake > 0 {
		parts = append(parts, fmt.Sprintf("TLS: %s", FormatDuration(t.TLSHandshake)))
	}
	if t.ServerProcessing > 0 {
		parts = append(parts, fmt.Sprintf("TTFB: %s", FormatDuration(t.ServerProcessing)))
	}
	if t.ContentTransfer > 0 {
		parts = append(parts, fmt.Sprintf("Transfer: %s", FormatDuration(t.ContentTransfer)))
	}
	if len(parts) == 0 {
		return fmt.Sprintf("Total: %s", FormatDuration(t.Total))
	}
	return fmt.Sprintf("Total: %s [%s]", FormatDuration(t.Total), strings.Join(parts, " | "))
}

// Breakdown returns a multi-line formatted string compatible with curl -w format
func (t TimingInfo) Breakdown() string {
	var b strings.Builder
	b.WriteString("=== Latency Breakdown (curl -w) ===\n")
	b.WriteString(fmt.Sprintf("%-18s %8s  (time_namelookup:    %s)\n", "DNS Lookup:", FormatDuration(t.DNSLookup), FormatDuration(t.NameLookup)))
	b.WriteString(fmt.Sprintf("%-18s %8s  (time_connect:       %s)\n", "TCP Connect:", FormatDuration(t.TCPConnect), FormatDuration(t.Connect)))
	if t.TLSHandshake > 0 || t.AppConnect > 0 {
		b.WriteString(fmt.Sprintf("%-18s %8s  (time_appconnect:    %s)\n", "TLS Handshake:", FormatDuration(t.TLSHandshake), FormatDuration(t.AppConnect)))
	} else {
		b.WriteString(fmt.Sprintf("%-18s %8s  (time_appconnect:    -)\n", "TLS Handshake:", "-"))
	}
	b.WriteString(fmt.Sprintf("%-18s %8s  (time_starttransfer: %s)\n", "Server Processing:", FormatDuration(t.ServerProcessing), FormatDuration(t.StartTransfer)))
	b.WriteString(fmt.Sprintf("%-18s %8s  (time_total:         %s)\n", "Content Transfer:", FormatDuration(t.ContentTransfer), FormatDuration(t.Total)))
	return b.String()
}
