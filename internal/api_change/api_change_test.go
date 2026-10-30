package api_change

import (
	"context"
	"testing"

	"github.com/barceloch/codixai/internal/domain"
)

func TestDetectGoFunctionRemoved(t *testing.T) {
	d := NewDetector(nil)

	changes := &domain.ChangeSet{
		Files: []domain.FileChange{
			{
				Path:     "pkg/api.go",
				Status:   domain.FileModified,
				Language: domain.LangGo,
				Hunks: []domain.Hunk{
					{
						Lines: []domain.DiffLine{
							{Kind: domain.LineRemoved, Content: "func PublicAPI() string { return \"old\" }"},
							{Kind: domain.LineAdded, Content: "func NewAPI() string { return \"new\" }"},
						},
					},
				},
			},
		},
	}

	result, err := d.Detect(context.Background(), changes)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if len(result.BreakingChanges) == 0 {
		t.Fatal("expected breaking changes, got none")
	}

	found := false
	for _, bc := range result.BreakingChanges {
		if bc.Type == BreakingChangeFunctionRemoved && bc.Symbol == "PublicAPI" {
			found = true
			if bc.Severity != domain.SeverityError {
				t.Errorf("expected SeverityError, got %s", bc.Severity)
			}
			if bc.EpistemicStatus != domain.EpistemicInferred {
				t.Errorf("expected EpistemicInferred, got %s", bc.EpistemicStatus)
			}
		}
	}
	if !found {
		t.Errorf("expected BreakingChangeFunctionRemoved for PublicAPI")
	}
}

func TestDetectGoSignatureChanged(t *testing.T) {
	d := NewDetector(nil)

	changes := &domain.ChangeSet{
		Files: []domain.FileChange{
			{
				Path:     "pkg/api.go",
				Status:   domain.FileModified,
				Language: domain.LangGo,
				Hunks: []domain.Hunk{
					{
						Lines: []domain.DiffLine{
							{Kind: domain.LineRemoved, Content: "func Process(data string) error"},
							{Kind: domain.LineAdded, Content: "func Process(data string, opts Options) error"},
						},
					},
				},
			},
		},
	}

	result, err := d.Detect(context.Background(), changes)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	found := false
	for _, bc := range result.BreakingChanges {
		if bc.Type == BreakingChangeFunctionSignatureChanged && bc.Symbol == "Process" {
			found = true
			if bc.OldSignature == "" || bc.NewSignature == "" {
				t.Errorf("expected old and new signatures")
			}
		}
	}
	if !found {
		t.Errorf("expected BreakingChangeFunctionSignatureChanged for Process")
	}
}

func TestDetectPythonFunctionRemoved(t *testing.T) {
	d := NewDetector(nil)

	changes := &domain.ChangeSet{
		Files: []domain.FileChange{
			{
				Path:     "api.py",
				Status:   domain.FileModified,
				Language: domain.LangPython,
				Hunks: []domain.Hunk{
					{
						Lines: []domain.DiffLine{
							{Kind: domain.LineRemoved, Content: "def public_function():"},
							{Kind: domain.LineAdded, Content: "def new_function():"},
						},
					},
				},
			},
		},
	}

	result, err := d.Detect(context.Background(), changes)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	found := false
	for _, bc := range result.BreakingChanges {
		if bc.Type == BreakingChangeFunctionRemoved && bc.Symbol == "public_function" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected BreakingChangeFunctionRemoved for public_function in Python")
	}
}

func TestDetectJavaScriptFunctionRemoved(t *testing.T) {
	d := NewDetector(nil)

	changes := &domain.ChangeSet{
		Files: []domain.FileChange{
			{
				Path:     "api.js",
				Status:   domain.FileModified,
				Language: domain.LangJavaScript,
				Hunks: []domain.Hunk{
					{
						Lines: []domain.DiffLine{
							{Kind: domain.LineRemoved, Content: "function publicAPI() {}"},
							{Kind: domain.LineAdded, Content: "function newAPI() {}"},
						},
					},
				},
			},
		},
	}

	result, err := d.Detect(context.Background(), changes)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	found := false
	for _, bc := range result.BreakingChanges {
		if bc.Type == BreakingChangeFunctionRemoved && bc.Symbol == "publicAPI" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected BreakingChangeFunctionRemoved for publicAPI in JavaScript")
	}
}

func TestDetectTypeScriptSignatureChanged(t *testing.T) {
	d := NewDetector(nil)

	changes := &domain.ChangeSet{
		Files: []domain.FileChange{
			{
				Path:     "api.ts",
				Status:   domain.FileModified,
				Language: domain.LangTypeScript,
				Hunks: []domain.Hunk{
					{
						Lines: []domain.DiffLine{
							{Kind: domain.LineRemoved, Content: "function process(data: string): void"},
							{Kind: domain.LineAdded, Content: "function process(data: string, opts: Options): void"},
						},
					},
				},
			},
		},
	}

	result, err := d.Detect(context.Background(), changes)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	found := false
	for _, bc := range result.BreakingChanges {
		if bc.Type == BreakingChangeFunctionSignatureChanged && bc.Symbol == "process" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected BreakingChangeFunctionSignatureChanged for process in TypeScript")
	}
}

func TestDetectPHPFunctionRemoved(t *testing.T) {
	d := NewDetector(nil)

	changes := &domain.ChangeSet{
		Files: []domain.FileChange{
			{
				Path:     "api.php",
				Status:   domain.FileModified,
				Language: domain.LangPHP,
				Hunks: []domain.Hunk{
					{
						Lines: []domain.DiffLine{
							{Kind: domain.LineRemoved, Content: "public function publicAPI() {}"},
							{Kind: domain.LineAdded, Content: "public function newAPI() {}"},
						},
					},
				},
			},
		},
	}

	result, err := d.Detect(context.Background(), changes)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	found := false
	for _, bc := range result.BreakingChanges {
		if bc.Type == BreakingChangeFunctionRemoved && bc.Symbol == "publicAPI" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected BreakingChangeFunctionRemoved for publicAPI in PHP")
	}
}

func TestDetectNoBreakingChanges(t *testing.T) {
	d := NewDetector(nil)

	changes := &domain.ChangeSet{
		Files: []domain.FileChange{
			{
				Path:     "pkg/internal.go",
				Status:   domain.FileModified,
				Language: domain.LangGo,
				Hunks: []domain.Hunk{
					{
						Lines: []domain.DiffLine{
							{Kind: domain.LineRemoved, Content: "func helper()"},
							{Kind: domain.LineAdded, Content: "func helper() { return 1 }"},
						},
					},
				},
			},
		},
	}

	result, err := d.Detect(context.Background(), changes)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	// helper() is not public (lowercase), so no breaking change should be detected
	if len(result.BreakingChanges) > 0 {
		t.Errorf("expected no breaking changes for non-public symbols, got %d", len(result.BreakingChanges))
	}
}

func TestDetectEmptyChanges(t *testing.T) {
	d := NewDetector(nil)

	changes := &domain.ChangeSet{
		Files: []domain.FileChange{},
	}

	result, err := d.Detect(context.Background(), changes)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if len(result.BreakingChanges) != 0 {
		t.Errorf("expected 0 breaking changes for empty changes, got %d", len(result.BreakingChanges))
	}
}

func TestDetectNilChanges(t *testing.T) {
	d := NewDetector(nil)

	result, err := d.Detect(context.Background(), nil)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if len(result.BreakingChanges) != 0 {
		t.Errorf("expected 0 breaking changes for nil changes, got %d", len(result.BreakingChanges))
	}
}
