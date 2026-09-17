package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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
	// まだ編集モードに入っていない（Focused() == false）ので、さらに j を押せば 0 に循環するはず
	if m.proxyInput.Focused() {
		t.Fatalf("proxyInput should NOT be focused merely by moving cursor to it")
	}

	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = res.(Model)
	if m.optionsCursor != 0 {
		t.Errorf("expected cursor to cycle back to 0, got %d", m.optionsCursor)
	}

	// 3. k を押して Proxy (index 3) に戻る
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = res.(Model)
	if m.optionsCursor != 3 {
		t.Fatalf("expected cursor at 3, got %d", m.optionsCursor)
	}

	// 4. Space を押して編集モードに入る
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = res.(Model)
	if !m.proxyInput.Focused() {
		t.Fatalf("proxyInput SHOULD be focused after pressing Space on Proxy item")
	}

	// 5. 編集モード中に入力
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = res.(Model)
	if m.proxyInput.Value() != "a" {
		t.Errorf("expected proxy value 'a', got '%s'", m.proxyInput.Value())
	}

	// 6. Enter で編集モードを抜ける
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(Model)
	if m.proxyInput.Focused() {
		t.Fatalf("proxyInput should Blur after Enter")
	}
	if !m.showOptionsModal {
		t.Fatalf("modal should still be open after exiting proxy edit mode")
	}

	// 7. 再び j/k で移動できることを確認
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = res.(Model)
	if m.optionsCursor != 2 {
		t.Errorf("expected cursor at 2 after pressing k, got %d", m.optionsCursor)
	}
}
