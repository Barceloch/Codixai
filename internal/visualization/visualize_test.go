package visualization

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/barceloch/codixai/internal/index"
)

func TestVisualize(t *testing.T) {
	// Create a temporary directory with some Go files
	tmpDir := t.TempDir()

	// Write a simple Go file
	goFile := filepath.Join(tmpDir, "main.go")
	err := os.WriteFile(goFile, []byte(`package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("hello")
}
`), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	// Write a Python file
	pyFile := filepath.Join(tmpDir, "script.py")
	err2 := os.WriteFile(pyFile, []byte(`def hello():
    print("hello")

if __name__ == "__main__":
    hello()
`), 0o644)
	if err2 != nil {
		t.Fatal(err2)
	}

	// Build index
	idx := index.New()
	err2 = idx.Build(context.Background(), tmpDir)
	if err2 != nil {
		t.Fatalf("Build failed: %v", err2)
	}

	// Visualize
	result := Visualize(idx)

	if len(result.Symbols) == 0 {
		t.Fatal("expected symbols, got none")
	}

	// Check that we have at least some symbols
	if result.Summary["total_symbols"] == 0 {
		t.Errorf("expected symbols in summary")
	}

	if len(result.Dependencies) == 0 {
		t.Errorf("expected dependencies")
	}

	// Test PrintVisualization doesn't panic
	PrintVisualization(result)
}
