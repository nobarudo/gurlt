package tui

import (
	"bytes"
	"os"
	"path/filepath"
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

	// 4. j を押して Bearer (index 5) に移動
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = res.(Model)
	if m.optionsCursor != 5 {
		t.Errorf("expected cursor at 5 (Bearer), got %d", m.optionsCursor)
	}

	// 5. j を押して Output File (index 6) に移動
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = res.(Model)
	if m.optionsCursor != 6 {
		t.Errorf("expected cursor at 6 (Output File), got %d", m.optionsCursor)
	}

	// 6. さらに j を押せば 0 に循環
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = res.(Model)
	if m.optionsCursor != 0 {
		t.Errorf("expected cursor to cycle back to 0, got %d", m.optionsCursor)
	}

	// 7. k を押して Output File (index 6) に戻る
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = res.(Model)
	if m.optionsCursor != 6 {
		t.Fatalf("expected cursor at 6, got %d", m.optionsCursor)
	}

	// 8. k を押して Bearer (index 5) に戻る
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = res.(Model)
	if m.optionsCursor != 5 {
		t.Fatalf("expected cursor at 5, got %d", m.optionsCursor)
	}

	// 9. k を押して Timeout (index 4) に戻る
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = res.(Model)
	if m.optionsCursor != 4 {
		t.Fatalf("expected cursor at 4, got %d", m.optionsCursor)
	}

	// 10. もう一度 k を押して Proxy (index 3) に戻る
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

func TestOptionsModalBearerEdit(t *testing.T) {
	m := InitialModel("https://api.example.com", "GET", "Content-Type: application/json", "", "json", false, "", "")

	// ctrl+o でモーダルを開く
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlO})
	m = res.(Model)

	// Bearer (index 5) へ移動
	for i := 0; i < 5; i++ {
		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		m = res.(Model)
	}
	if m.optionsCursor != 5 {
		t.Fatalf("expected cursor at 5, got %d", m.optionsCursor)
	}

	// Space で編集開始
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = res.(Model)
	if !m.bearerInput.Focused() {
		t.Fatalf("bearerInput should be focused")
	}

	// "token-abc" と入力
	for _, r := range "token-abc" {
		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = res.(Model)
	}

	// Enter で確定
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(Model)
	if m.bearerInput.Focused() {
		t.Fatalf("bearerInput should blur after Enter")
	}
	if m.bearerInput.Value() != "token-abc" {
		t.Errorf("expected bearerInput value 'token-abc', got %q", m.bearerInput.Value())
	}

	// headerInput に Authorization: Bearer token-abc が反映されていること
	if !strings.Contains(m.headerInput.Value(), "Authorization: Bearer token-abc") {
		t.Errorf("expected Authorization: Bearer token-abc in headers, got %q", m.headerInput.Value())
	}

	// cURLプレビューに -H 'Authorization: Bearer token-abc' が含まれること
	cmd := m.BuildCurlCmd()
	if !strings.Contains(cmd, "-H 'Authorization: Bearer token-abc'") {
		t.Errorf("expected Authorization header in cmd preview, got %s", cmd)
	}

	// SetBearer で直接上書きテスト
	m.SetBearer("new-token-xyz")
	if !strings.Contains(m.headerInput.Value(), "Authorization: Bearer new-token-xyz") {
		t.Errorf("expected new-token-xyz in headers, got %q", m.headerInput.Value())
	}
}

func TestOptionsModalOutputEdit(t *testing.T) {
	m := InitialModel("https://api.example.com", "GET", "", "", "form", false, "", "")

	// 1. ctrl+o でモーダルを開く
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlO})
	m = res.(Model)

	// 2. index 6 (Output File) まで移動
	for i := 0; i < 6; i++ {
		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		m = res.(Model)
	}
	if m.optionsCursor != 6 {
		t.Fatalf("expected cursor at 6, got %d", m.optionsCursor)
	}

	// Space で編集開始
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = res.(Model)
	if !m.outputInput.Focused() {
		t.Fatalf("outputInput should be focused")
	}

	// "out.json" と入力
	for _, r := range "out.json" {
		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = res.(Model)
	}

	// Enter で確定
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(Model)
	if m.outputInput.Focused() {
		t.Fatalf("outputInput should blur after Enter")
	}
	if m.outputInput.Value() != "out.json" {
		t.Errorf("expected outputInput value 'out.json', got %q", m.outputInput.Value())
	}

	// cURLプレビューに -o 'out.json' が含まれること
	cmd := m.BuildCurlCmd()
	if !strings.Contains(cmd, "-o 'out.json'") {
		t.Errorf("expected -o 'out.json' in cmd preview, got %s", cmd)
	}

	// SetOutputFile で直接上書きテスト
	m.SetOutputFile("new.png")
	if m.outputInput.Value() != "new.png" {
		t.Errorf("expected outputInput value 'new.png', got %q", m.outputInput.Value())
	}
	cmd = m.BuildCurlCmd()
	if !strings.Contains(cmd, "-o 'new.png'") {
		t.Errorf("expected -o 'new.png' in cmd preview, got %s", cmd)
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

func TestRawViewSearch(t *testing.T) {
	m := InitialModel("https://api.example.com", "GET", "", "", "form", false, "", "")
	m.terminalWidth = 80
	m.terminalHeight = 24
	m.ready = true

	// レスポンス受信
	res, _ := m.Update(responseMsg{
		status:     "200 OK",
		body:       "Hello World\nSecond line with Hello\nThird line",
		rawContent: "Hello World\nSecond line with Hello\nThird line",
	})
	m = res.(Model)

	// Ctrl+R で Raw View に入る
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	m = res.(Model)
	if !m.showRawView {
		t.Fatalf("expected showRawView to be true")
	}

	// '/' を押して検索モードに入る
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = res.(Model)
	if !m.isSearching {
		t.Fatalf("expected isSearching to be true")
	}

	// "hello" と入力
	for _, ch := range "hello" {
		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		m = res.(Model)
	}

	if m.searchQuery != "hello" {
		t.Errorf("expected searchQuery 'hello', got %q", m.searchQuery)
	}
	if len(m.searchMatches) != 2 {
		t.Fatalf("expected 2 search matches, got %d", len(m.searchMatches))
	}
	if m.searchMatchIndex != 0 {
		t.Errorf("expected searchMatchIndex 0, got %d", m.searchMatchIndex)
	}

	// Enter で次のマッチへ
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(Model)
	if m.searchMatchIndex != 1 {
		t.Errorf("expected searchMatchIndex 1 after Enter, got %d", m.searchMatchIndex)
	}

	// Esc で検索入力モードを終了（検索自体はアクティブ）
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = res.(Model)
	if m.isSearching {
		t.Errorf("expected isSearching to be false after Esc")
	}
	if m.searchQuery != "hello" {
		t.Errorf("expected searchQuery 'hello' to persist, got %q", m.searchQuery)
	}

	// 'n' で次のマッチへ（ループして 0 に戻る）
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = res.(Model)
	if m.searchMatchIndex != 0 {
		t.Errorf("expected searchMatchIndex 0 after wrapping, got %d", m.searchMatchIndex)
	}

	// 'N' で前のマッチへ（1 に戻る）
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'N'}})
	m = res.(Model)
	if m.searchMatchIndex != 1 {
		t.Errorf("expected searchMatchIndex 1 after 'N', got %d", m.searchMatchIndex)
	}

	// Esc で検索クリア
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = res.(Model)
	if m.searchQuery != "" {
		t.Errorf("expected searchQuery to be cleared after Esc, got %q", m.searchQuery)
	}
}

func TestRawViewJSONFiltering(t *testing.T) {
	m := InitialModel("https://api.example.com", "GET", "", "", "form", false, "", "")
	m.terminalWidth = 80
	m.terminalHeight = 24
	m.ready = true

	jsonBody := `{"users":[{"name":"Alice","role":"admin"},{"name":"Bob","role":"user"}]}`
	rawDump := "=== Response ===\n" + jsonBody

	res, _ := m.Update(responseMsg{
		status:     "200 OK",
		body:       jsonBody,
		rawContent: rawDump,
	})
	m = res.(Model)

	// Ctrl+R で Raw View に入る
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	m = res.(Model)

	// 'p' で JSON フィルタモードに入る
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	m = res.(Model)
	if !m.isFiltering {
		t.Fatalf("expected isFiltering to be true")
	}

	// ".users[0].name" を入力
	for _, ch := range ".users[0].name" {
		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		m = res.(Model)
	}

	// Enter でフィルタ適用
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(Model)
	if m.isFiltering {
		t.Fatalf("expected isFiltering to be false after Enter")
	}
	if m.jsonPathQuery != ".users[0].name" {
		t.Errorf("expected jsonPathQuery '.users[0].name', got %q", m.jsonPathQuery)
	}
	if m.filteredContent != "Alice" {
		t.Errorf("expected filteredContent 'Alice', got %q", m.filteredContent)
	}

	// activeContent が "Alice" になっていること
	if m.activeContent() != "Alice" {
		t.Errorf("expected activeContent 'Alice', got %q", m.activeContent())
	}

	// View() に [Filter: .users[0].name] が表示されること
	viewStr := m.View()
	if !strings.Contains(viewStr, "[Filter: .users[0].name]") {
		t.Errorf("expected View() to contain filter tag, got:\n%s", viewStr)
	}

	// Esc でフィルタクリア
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = res.(Model)
	if m.jsonPathQuery != "" {
		t.Errorf("expected jsonPathQuery to be cleared, got %q", m.jsonPathQuery)
	}
	if m.filteredContent != "" {
		t.Errorf("expected filteredContent to be cleared, got %q", m.filteredContent)
	}
	if m.activeContent() != rawDump {
		t.Errorf("expected activeContent to be rawDump after clearing filter")
	}
}

func TestBinaryResponseAndSaving(t *testing.T) {
	m := InitialModel("https://api.example.com", "GET", "", "", "form", false, "", "")

	binaryPayload := []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D}
	resMsg := responseMsg{
		status:     "200 OK",
		body:       string(binaryPayload),
		bodyBytes:  binaryPayload,
		outputFile: "download.png",
		savedBytes: len(binaryPayload),
		rawContent: "=== cURL ===\ncurl https://api.example.com\n\n=== Response ===\nHTTP/1.1 200 OK\r\n\r\n[Binary data: 12 B (12 bytes)]\nSaved to: download.png\n\n⏱️  Latency: 5ms",
	}

	res, _ := m.Update(resMsg)
	m = res.(Model)

	if !m.isBinaryResponse {
		t.Fatalf("expected isBinaryResponse to be true")
	}
	if !strings.Contains(m.normalContent, "[Binary data: 12 B (12 bytes)]") {
		t.Errorf("expected binary descriptor in normalContent, got %q", m.normalContent)
	}
	if !strings.Contains(m.normalContent, "Saved to: download.png") {
		t.Errorf("expected Saved to: download.png in normalContent, got %q", m.normalContent)
	}
	if !strings.Contains(m.footerMsg, "Saved to download.png") {
		t.Errorf("expected footerMsg to contain save confirmation, got %q", m.footerMsg)
	}

	// Test saving binary content from raw view
	tmpFile := filepath.Join(t.TempDir(), "test_saved.png")
	m.showRawView = true
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = res.(Model)
	m.saveInput.SetValue(tmpFile)
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(Model)

	savedData, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read saved file: %v", err)
	}
	if !bytes.Equal(savedData, binaryPayload) {
		t.Fatalf("saved data does not match original binary payload")
	}
}





