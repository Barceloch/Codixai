package similarity

import (
	"context"
	"strings"
	"testing"

	"github.com/barceloch/codixai/internal/domain"
)

func TestSimilarityThreshold(t *testing.T) {
	d := NewDetector(nil).WithThreshold(80.0)

	if d.threshold != 80.0 {
		t.Errorf("expected threshold 80.0, got %f", d.threshold)
	}
}

func TestSimilarityMinLines(t *testing.T) {
	d := NewDetector(nil).WithMinLines(10)

	if d.minLines != 10 {
		t.Errorf("expected minLines 10, got %d", d.minLines)
	}
}

func TestIgnoreTests(t *testing.T) {
	d := NewDetector(nil).WithIgnoreTests(false)

	if d.ignoreTests {
		t.Errorf("expected ignoreTests false")
	}
}

func TestNormalizeFunction(t *testing.T) {
	d := NewDetector(nil)

	lines := []string{
		"func CalculateSum(a, b int) int {",
		"    return a + b",
		"    // comment",
		"",
		"}",
	}

	normalized := d.normalizeFunction(lines, domain.LangGo)

	if strings.Contains(normalized, "comment") {
		t.Errorf("normalized should not contain comments")
	}
	if !strings.Contains(normalized, "return") {
		t.Errorf("normalized should contain return statement")
	}
}

func TestIsTestFile(t *testing.T) {
	tests := []struct {
		path   string
		expect bool
	}{
		{"pkg/foo_test.go", true},
		{"pkg/foo.go", false},
		{"test/foo.go", false},
		{"pkg/foo_test.py", true},
		{"pkg/test_foo.py", true},
		{"pkg/foo.test.js", true},
		{"pkg/foo.test.ts", true},
		{"pkg/foo_test.php", true},
	}

	for _, tc := range tests {
		result := isTestFile(tc.path)
		if result != tc.expect {
			t.Errorf("isTestFile(%q): got %v, want %v", tc.path, result, tc.expect)
		}
	}
}

func TestCompareFunctionsEmpty(t *testing.T) {
	d := NewDetector(nil)

	a := FunctionInfo{Normalized: ""}
	b := FunctionInfo{Normalized: "func foo() {}"}

	sim := d.compareFunctions(a, b)
	if sim != 0 {
		t.Errorf("expected 0 similarity for empty vs non-empty, got %f", sim)
	}
}

func TestCompareFunctionsIdentical(t *testing.T) {
	d := NewDetector(nil)

	a := FunctionInfo{Normalized: "func foo() {\nreturn 1\n}", TotalLines: 2}
	b := FunctionInfo{Normalized: "func foo() {\nreturn 1\n}", TotalLines: 2}

	sim := d.compareFunctions(a, b)
	if sim < 90.0 {
		t.Errorf("expected high similarity for identical functions, got %f", sim)
	}
}

func TestExtractFunctionNamesGo(t *testing.T) {
	d := NewDetector(nil)

	lines := []string{
		"func PublicAPI() string {",
		"    return \"hello\"",
		"}",
		"func privateHelper() {",
		"}",
	}

	funcs := d.extractFunctionNamesFromLines("pkg/api.go", domain.LangGo, lines)

	if len(funcs) != 1 {
		t.Errorf("expected 1 public function, got %d", len(funcs))
	}
	if funcs[0].Name != "PublicAPI" {
		t.Errorf("expected PublicAPI, got %s", funcs[0].Name)
	}
}

func TestExtractFunctionNamesPython(t *testing.T) {
	d := NewDetector(nil)

	lines := []string{
		"def public_function():",
		"    return 42",
		"def _private_helper():",
		"    pass",
	}

	funcs := d.extractFunctionNamesFromLines("api.py", domain.LangPython, lines)

	if len(funcs) != 1 {
		t.Errorf("expected 1 public function, got %d", len(funcs))
	}
	if funcs[0].Name != "public_function" {
		t.Errorf("expected public_function, got %s", funcs[0].Name)
	}
}

func TestExtractFunctionNamesJavaScript(t *testing.T) {
	d := NewDetector(nil)

	lines := []string{
		"function publicAPI() {",
		"    return 'hello';",
		"}",
		"function _privateHelper() {",
		"}",
	}

	funcs := d.extractFunctionNamesFromLines("api.js", domain.LangJavaScript, lines)

	if len(funcs) != 1 {
		t.Errorf("expected 1 public function, got %d", len(funcs))
	}
	if funcs[0].Name != "publicAPI" {
		t.Errorf("expected publicAPI, got %s", funcs[0].Name)
	}
}

func TestExtractFunctionNamesPHP(t *testing.T) {
	d := NewDetector(nil)

	lines := []string{
		"function publicAPI() {",
		"    return 'hello';",
		"}",
		"function _privateHelper() {",
		"}",
	}

	funcs := d.extractFunctionNamesFromLines("api.php", domain.LangPHP, lines)

	if len(funcs) != 1 {
		t.Errorf("expected 1 public function, got %d", len(funcs))
	}
	if funcs[0].Name != "publicAPI" {
		t.Errorf("expected publicAPI, got %s", funcs[0].Name)
	}
}

func TestDetectSimilarityFromChanges(t *testing.T) {
	d := NewDetector(nil)

	changes := &domain.ChangeSet{
		Files: []domain.FileChange{
			{
				Path:     "pkg/newutil.go",
				Status:   domain.FileAdded,
				Language: domain.LangGo,
				Hunks: []domain.Hunk{
					{
						Lines: []domain.DiffLine{
							{Kind: domain.LineAdded, Content: "func CalculateSum(a, b int) int {"},
							{Kind: domain.LineAdded, Content: "    return a + b"},
							{Kind: domain.LineAdded, Content: "}"},
						},
					},
				},
			},
		},
	}

	result, err := d.DetectFromChanges(context.Background(), changes)
	if err != nil {
		t.Fatalf("DetectFromChanges failed: %v", err)
	}

	if len(result.Matches) != 0 {
		t.Errorf("expected 0 matches without index, got %d", len(result.Matches))
	}
}
