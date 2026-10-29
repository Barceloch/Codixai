package index

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/barceloch/codixai/internal/languages/registry"
)

func TestIndexFiveLanguages(t *testing.T) {
	// Use the testdata/index directory with fixtures for all 5 languages
	testDir := filepath.Join("..", "..", "testdata", "index")

	idx := New()
	err := idx.Build(context.Background(), testDir)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	symbols := idx.Symbols()
	deps := idx.Dependencies()

	// We should have at least 1 symbol per language
	byLang := make(map[string]int)
	for _, sym := range symbols {
		byLang[sym.Language]++
	}

	expectedLangs := []string{"go", "python", "javascript", "typescript", "php"}
	for _, lang := range expectedLangs {
		count := byLang[lang]
		if count == 0 {
			t.Errorf("expected symbols for language %q, got 0", lang)
		}
	}

	// Should have dependencies for Go (fmt, github.com/foo/bar)
	foundFmt := false
	foundBar := false
	for _, dep := range deps {
		if dep.Path == "fmt" {
			foundFmt = true
		}
		if dep.Path == "github.com/foo/bar" {
			foundBar = true
		}
	}
	if !foundFmt {
		t.Errorf("expected Go dependency 'fmt'")
	}
	if !foundBar {
		t.Errorf("expected Go dependency 'github.com/foo/bar'")
	}

	// Verify registry detects all 5 languages
	for _, lang := range expectedLangs {
		// Create a temp file with that extension
		tmpFile := filepath.Join(t.TempDir(), "test."+getExtension(lang))
		os.WriteFile(tmpFile, []byte(""), 0o644)
		
		detected, ok := registry.Detect(tmpFile)
		if !ok {
			t.Errorf("registry.Detect failed for %q", lang)
		}
		if detected != lang {
			t.Errorf("registry.Detect returned %q for %q, want %q", detected, lang, lang)
		}
	}
}

func getExtension(lang string) string {
	switch lang {
	case "go":
		return "go"
	case "python":
		return "py"
	case "javascript":
		return "js"
	case "typescript":
		return "ts"
	case "php":
		return "php"
	default:
		return "txt"
	}
}