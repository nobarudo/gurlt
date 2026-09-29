package client

import (
	"strings"
	"testing"
	"time"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{0, "0ms"},
		{-1 * time.Second, "0ms"},
		{500 * time.Microsecond, "0.50ms"},
		{4200 * time.Microsecond, "4.2ms"},
		{15 * time.Millisecond, "15ms"},
		{1500 * time.Millisecond, "1.50s"},
	}

	for _, tt := range tests {
		got := FormatDuration(tt.d)
		if got != tt.want {
			t.Errorf("FormatDuration(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestTimingInfoSummaryAndBreakdown(t *testing.T) {
	timing := TimingInfo{
		DNSLookup:        12 * time.Millisecond,
		TCPConnect:       25 * time.Millisecond,
		TLSHandshake:     40 * time.Millisecond,
		ServerProcessing: 110 * time.Millisecond,
		ContentTransfer:  8 * time.Millisecond,
		Total:            195 * time.Millisecond,
		NameLookup:       12 * time.Millisecond,
		Connect:          37 * time.Millisecond,
		AppConnect:       77 * time.Millisecond,
		StartTransfer:    187 * time.Millisecond,
	}

	summary := timing.Summary()
	if !strings.Contains(summary, "Total: 195ms") {
		t.Errorf("expected summary to contain Total: 195ms, got %q", summary)
	}
	if !strings.Contains(summary, "DNS: 12ms") {
		t.Errorf("expected summary to contain DNS: 12ms, got %q", summary)
	}
	if !strings.Contains(summary, "TTFB: 110ms") {
		t.Errorf("expected summary to contain TTFB: 110ms, got %q", summary)
	}

	breakdown := timing.Breakdown()
	if !strings.Contains(breakdown, "=== Latency Breakdown (curl -w) ===") {
		t.Errorf("expected breakdown to contain header, got %q", breakdown)
	}
	if !strings.Contains(breakdown, "time_namelookup:    12ms") {
		t.Errorf("expected breakdown to contain time_namelookup, got %q", breakdown)
	}
	if !strings.Contains(breakdown, "time_total:         195ms") {
		t.Errorf("expected breakdown to contain time_total, got %q", breakdown)
	}
}
