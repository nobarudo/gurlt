package tui

import (
	"strings"
	"testing"
)

func TestPerformSearch(t *testing.T) {
	content := "Hello World\nhello again\nHELLO Gophers"

	// 1. 空のクエリ
	emptyRes := performSearch(content, "", 0)
	if emptyRes.totalMatches != 0 {
		t.Errorf("expected 0 matches for empty query, got %d", emptyRes.totalMatches)
	}
	if emptyRes.highlighted != content {
		t.Errorf("expected original content, got %s", emptyRes.highlighted)
	}

	// 2. マッチなし
	noneRes := performSearch(content, "nonexistent", 0)
	if noneRes.totalMatches != 0 {
		t.Errorf("expected 0 matches, got %d", noneRes.totalMatches)
	}

	// 3. 大文字小文字を区別しない検索
	res := performSearch(content, "hello", 0)
	if res.totalMatches != 3 {
		t.Errorf("expected 3 matches for 'hello', got %d", res.totalMatches)
	}
	if len(res.matchLines) != 3 {
		t.Errorf("expected 3 match lines, got %d", len(res.matchLines))
	}
	if res.matchLines[0] != 0 || res.matchLines[1] != 1 || res.matchLines[2] != 2 {
		t.Errorf("expected match lines [0, 1, 2], got %v", res.matchLines)
	}

	// 4. ハイライトが含まれているか確認
	if !strings.Contains(res.highlighted, "Hello") || !strings.Contains(res.highlighted, "hello") {
		t.Errorf("expected original text preserved in highlighted output")
	}

	// 5. 複数マッチが同一行にある場合
	multiContent := "foo bar foo baz foo"
	multiRes := performSearch(multiContent, "foo", 1)
	if multiRes.totalMatches != 3 {
		t.Errorf("expected 3 matches, got %d", multiRes.totalMatches)
	}
	if len(multiRes.matchLines) != 3 {
		t.Fatalf("expected 3 match lines, got %d", len(multiRes.matchLines))
	}
	if multiRes.matchLines[0] != 0 || multiRes.matchLines[1] != 0 || multiRes.matchLines[2] != 0 {
		t.Errorf("expected match lines [0, 0, 0], got %v", multiRes.matchLines)
	}
}
