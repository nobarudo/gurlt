package curl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadPayload(t *testing.T) {
	// 1. Regular inline string
	inline := `{"key": "value"}`
	got, err := LoadPayload(inline)
	if err != nil {
		t.Fatalf("unexpected error for inline string: %v", err)
	}
	if got != inline {
		t.Errorf("got %q, want %q", got, inline)
	}

	// 2. Empty string
	got, err = LoadPayload("")
	if err != nil {
		t.Fatalf("unexpected error for empty string: %v", err)
	}
	if got != "" {
		t.Errorf("got %q, want empty string", got)
	}

	// 3. Valid file with @path
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "payload.json")
	expectedContent := `{"user": "alice", "active": true}`
	if err := os.WriteFile(testFile, []byte(expectedContent), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	got, err = LoadPayload("@" + testFile)
	if err != nil {
		t.Fatalf("unexpected error loading file: %v", err)
	}
	if got != expectedContent {
		t.Errorf("got %q, want %q", got, expectedContent)
	}

	// 4. File path with quotes (double quotes and single quotes)
	got, err = LoadPayload(`@"` + testFile + `"`)
	if err != nil {
		t.Fatalf("unexpected error with double quotes: %v", err)
	}
	if got != expectedContent {
		t.Errorf("got %q, want %q", got, expectedContent)
	}

	got, err = LoadPayload(`@'` + testFile + `'`)
	if err != nil {
		t.Fatalf("unexpected error with single quotes: %v", err)
	}
	if got != expectedContent {
		t.Errorf("got %q, want %q", got, expectedContent)
	}

	// 5. File with spaces in filename
	spacedFile := filepath.Join(tmpDir, "my payload file.json")
	if err := os.WriteFile(spacedFile, []byte(`{"spaced": true}`), 0644); err != nil {
		t.Fatalf("failed to create spaced file: %v", err)
	}
	got, err = LoadPayload(`@"` + spacedFile + `"`)
	if err != nil {
		t.Fatalf("unexpected error with spaced file: %v", err)
	}
	if got != `{"spaced": true}` {
		t.Errorf("got %q, want %q", got, `{"spaced": true}`)
	}

	// 6. Non-existent file
	nonExistent := filepath.Join(tmpDir, "does-not-exist.json")
	_, err = LoadPayload("@" + nonExistent)
	if err == nil {
		t.Fatalf("expected error for non-existent file, got nil")
	}
	if !strings.Contains(err.Error(), "failed to read data file") {
		t.Errorf("expected error message to contain 'failed to read data file', got: %v", err)
	}

	// 7. Empty path after @
	_, err = LoadPayload("@")
	if err == nil {
		t.Fatalf("expected error for '@', got nil")
	}
	if !strings.Contains(err.Error(), "empty file path") {
		t.Errorf("expected 'empty file path', got %v", err)
	}

	// 8. Empty path with quotes (@"")
	_, err = LoadPayload(`@""`)
	if err == nil {
		t.Fatalf("expected error for '@\"\"', got nil")
	}
}
