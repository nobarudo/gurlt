package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nobarudo/gurlt/internal/client"
)

func TestBuildCurlCmd(t *testing.T) {
	m := InitialModel("https://api.example.com/users", "GET", "", "", "form", false, "", "")

	// 初期状態
	cmd := m.BuildCurlCmd()
	if !strings.Contains(cmd, "https://api.example.com/users") {
		t.Errorf("expected URL in cmd, got %s", cmd)
	}
	if strings.Contains(cmd, "-k") || strings.Contains(cmd, "-v") || strings.Contains(cmd, "-x") {
		t.Errorf("expected no flags, got %s", cmd)
	}

	// オプションを有効化
	m.insecure = true
	m.verbose = true
	m.proxyInput.SetValue("http://127.0.0.1:8888")
	m.SetMaxTime(10)
	m.SetConnectTimeout(3.5)

	cmdWithOpts := m.BuildCurlCmd()
	if !strings.Contains(cmdWithOpts, "-k") {
		t.Errorf("expected -k flag in cmd, got %s", cmdWithOpts)
	}
	if !strings.Contains(cmdWithOpts, "-v") {
		t.Errorf("expected -v flag in cmd, got %s", cmdWithOpts)
	}
	if !strings.Contains(cmdWithOpts, "-x 'http://127.0.0.1:8888'") {
		t.Errorf("expected -x proxy flag in cmd, got %s", cmdWithOpts)
	}
	if !strings.Contains(cmdWithOpts, "-m 10") {
		t.Errorf("expected -m 10 flag in cmd, got %s", cmdWithOpts)
	}
	if !strings.Contains(cmdWithOpts, "--connect-timeout 3.5") {
		t.Errorf("expected --connect-timeout 3.5 flag in cmd, got %s", cmdWithOpts)
	}
}

func TestOptionsModalNavigationAndProxyEdit(t *testing.T) {
	m := InitialModel("https://api.example.com", "GET", "", "", "form", false, "", "")

	// 1. ctrl+o でモーダルを開く
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlO})
	m = res.(Model)
	if !m.showOptionsModal {
		t.Fatalf("expected showOptionsModal to be true")
	}
	if m.optionsCursor != 0 {
		t.Errorf("expected cursor at 0, got %d", m.optionsCursor)
	}

	// 2. j を3回押して Proxy (index 3) に移動
	for i := 0; i < 3; i++ {
		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		m = res.(Model)
	}
	if m.optionsCursor != 3 {
		t.Fatalf("expected cursor at 3 (Proxy), got %d", m.optionsCursor)
	}
	if m.proxyInput.Focused() {
		t.Fatalf("proxyInput should NOT be focused merely by moving cursor to it")
	}

	// 3. j を押して Timeout (index 4) に移動
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = res.(Model)
	if m.optionsCursor != 4 {
		t.Errorf("expected cursor at 4 (Timeout), got %d", m.optionsCursor)
	}

	// 4. さらに j を押せば 0 に循環
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = res.(Model)
	if m.optionsCursor != 0 {
		t.Errorf("expected cursor to cycle back to 0, got %d", m.optionsCursor)
	}

	// 5. k を押して Timeout (index 4) に戻る
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = res.(Model)
	if m.optionsCursor != 4 {
		t.Fatalf("expected cursor at 4, got %d", m.optionsCursor)
	}

	// 6. もう一度 k を押して Proxy (index 3) に戻る
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = res.(Model)
	if m.optionsCursor != 3 {
		t.Fatalf("expected cursor at 3, got %d", m.optionsCursor)
	}

	// 7. Space を押して Proxy 編集モードに入る
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = res.(Model)
	if !m.proxyInput.Focused() {
		t.Fatalf("proxyInput SHOULD be focused after pressing Space on Proxy item")
	}

	// 8. 編集モード中に入力
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = res.(Model)
	if m.proxyInput.Value() != "a" {
		t.Errorf("expected proxy value 'a', got '%s'", m.proxyInput.Value())
	}

	// 9. Enter で編集モードを抜ける
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(Model)
	if m.proxyInput.Focused() {
		t.Fatalf("proxyInput should Blur after Enter")
	}
	if !m.showOptionsModal {
		t.Fatalf("modal should still be open after exiting proxy edit mode")
	}
}

func TestOptionsModalTimeoutEdit(t *testing.T) {
	m := InitialModel("https://api.example.com", "GET", "", "", "form", false, "", "")

	// ctrl+o でモーダルを開く
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlO})
	m = res.(Model)

	// Timeout (index 4) へ移動
	for i := 0; i < 4; i++ {
		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		m = res.(Model)
	}
	if m.optionsCursor != 4 {
		t.Fatalf("expected cursor at 4, got %d", m.optionsCursor)
	}

	// Space で編集開始
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = res.(Model)
	if !m.timeoutInput.Focused() {
		t.Fatalf("timeoutInput should be focused")
	}

	// "15" と入力
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	m = res.(Model)
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'5'}})
	m = res.(Model)

	// Enter で確定
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(Model)
	if m.timeoutInput.Focused() {
		t.Fatalf("timeoutInput should blur after Enter")
	}
	if m.maxTime != 15 {
		t.Errorf("expected maxTime 15, got %v", m.maxTime)
	}

	// cURLプレビューに -m 15 が含まれること
	cmd := m.BuildCurlCmd()
	if !strings.Contains(cmd, "-m 15") {
		t.Errorf("expected -m 15 in cmd, got %s", cmd)
	}
}

func TestInitialModelJSONPrettify(t *testing.T) {
	rawJSON := `{"foo":"bar","num":123}`
	m := InitialModel("https://api.example.com", "POST", "", rawJSON, "json", false, "", "")

	expectedIndent := "{\n  \"foo\": \"bar\",\n  \"num\": 123\n}"
	if m.bodyInput.Value() != expectedIndent {
		t.Errorf("expected pretty JSON body:\n%s\ngot:\n%s", expectedIndent, m.bodyInput.Value())
	}

	// 不正なJSONはそのまま入ること
	invalidJSON := `{"foo":`
	m2 := InitialModel("https://api.example.com", "POST", "", invalidJSON, "json", false, "", "")
	if m2.bodyInput.Value() != invalidJSON {
		t.Errorf("expected raw invalid JSON body, got: %s", m2.bodyInput.Value())
	}
}

func TestLatencyBreakdownRendering(t *testing.T) {
	m := InitialModel("https://api.example.com", "GET", "", "", "form", false, "", "")
	m.terminalWidth = 80
	m.terminalHeight = 24
	m.ready = true

	timing := client.TimingInfo{
		DNSLookup:        15 * time.Millisecond,
		TCPConnect:       25 * time.Millisecond,
		TLSHandshake:     35 * time.Millisecond,
		ServerProcessing: 80 * time.Millisecond,
		ContentTransfer:  5 * time.Millisecond,
		Total:            160 * time.Millisecond,
	}

	res, _ := m.Update(responseMsg{
		status:     "200 OK",
		body:       `{"ok":true}`,
		rawContent: "raw data",
		timing:     timing,
	})
	m = res.(Model)

	if m.timing.Total != 160*time.Millisecond {
		t.Fatalf("expected timing total 160ms, got %v", m.timing.Total)
	}

	viewStr := m.View()
	if !strings.Contains(viewStr, "Latency:") {
		t.Errorf("expected View() to contain 'Latency:', got:\n%s", viewStr)
	}
	if !strings.Contains(viewStr, "160ms") {
		t.Errorf("expected View() to contain '160ms', got:\n%s", viewStr)
	}
	if !strings.Contains(viewStr, "TTFB:") {
		t.Errorf("expected View() to contain 'TTFB:', got:\n%s", viewStr)
	}

	// Modal check
	m.showOptionsModal = true
	modalStr := m.View()
	if !strings.Contains(modalStr, "Latency (Last):") {
		t.Errorf("expected modal to contain 'Latency (Last):', got:\n%s", modalStr)
	}
}


