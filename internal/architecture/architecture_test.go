package architecture

import (
	"context"
	"testing"

	"github.com/barceloch/codixai/internal/domain"
	"github.com/barceloch/codixai/internal/index"
)

func TestNamingStyleCamelCase(t *testing.T) {
	valid := []string{"camelCase", "fooBar", "fooBarBaz", "a", "foo"}
	invalid := []string{"CamelCase", "snake_case", "foo_bar", "FOOBAR", ""}

	for _, s := range valid {
		if !isCamelCase(s) {
			t.Errorf("expected %q to be valid camelCase", s)
		}
	}
	for _, s := range invalid {
		if isCamelCase(s) {
			t.Errorf("expected %q to be invalid camelCase", s)
		}
	}
}

func TestNamingStylePascalCase(t *testing.T) {
	valid := []string{"PascalCase", "FooBar", "FooBarBaz", "A", "Foo"}
	invalid := []string{"camelCase", "snake_case", "foo_bar", "foobar", ""}

	for _, s := range valid {
		if !isPascalCase(s) {
			t.Errorf("expected %q to be valid PascalCase", s)
		}
	}
	for _, s := range invalid {
		if isPascalCase(s) {
			t.Errorf("expected %q to be invalid PascalCase", s)
		}
	}
}

func TestNamingStyleSnakeCase(t *testing.T) {
	valid := []string{"snake_case", "foo_bar", "foo_bar_baz", "a", "foo"}
	invalid := []string{"camelCase", "PascalCase", "FOOBAR", "foo__bar", "foo_bar_"}

	for _, s := range valid {
		if !isSnakeCase(s) {
			t.Errorf("expected %q to be valid snake_case", s)
		}
	}
	for _, s := range invalid {
		if isSnakeCase(s) {
			t.Errorf("expected %q to be invalid snake_case", s)
		}
	}
}

func TestNamingStyleUpperCase(t *testing.T) {
	valid := []string{"UPPER_CASE", "FOO_BAR", "FOO", "A"}
	invalid := []string{"camelCase", "PascalCase", "snake_case", "foo", "foo_bar"}

	for _, s := range valid {
		if !isUpperCase(s) {
			t.Errorf("expected %q to be valid UPPER_CASE", s)
		}
	}
	for _, s := range invalid {
		if isUpperCase(s) {
			t.Errorf("expected %q to be invalid UPPER_CASE", s)
		}
	}
}

func TestCheckNameStyle(t *testing.T) {
	a := NewAnalyzer(nil).WithNamingStyle(domain.LangGo, "camelCase")

	violation := a.checkNameStyle(domain.LangGo, "camelCase", "camelCaseName", "file.go", 10, "function")
	if violation != nil {
		t.Errorf("expected no violation for valid camelCase, got: %s", violation.Message)
	}

	violation = a.checkNameStyle(domain.LangGo, "camelCase", "PascalCaseName", "file.go", 10, "function")
	if violation == nil {
		t.Errorf("expected violation for invalid camelCase")
	}
	if violation != nil && violation.Type != ConventionNamingStyle {
		t.Errorf("expected ConventionNamingStyle violation")
	}
	if violation != nil && violation.Severity != domain.SeverityWarning {
		t.Errorf("expected SeverityWarning")
	}
	if violation != nil && violation.EpistemicStatus != domain.EpistemicInferred {
		t.Errorf("expected EpistemicInferred")
	}
}

func TestImportOrderAlphabetical(t *testing.T) {
	a := NewAnalyzer(nil).WithImportOrder(domain.LangGo, "alphabetical")

	deps := []index.Dependency{
		{Path: "fmt", File: "main.go", Line: 1, Language: domain.LangGo},
		{Path: "github.com/foo/bar", File: "main.go", Line: 2, Language: domain.LangGo},
		{Path: "os", File: "main.go", Line: 3, Language: domain.LangGo},
	}

	violations := a.checkImportOrder("main.go", deps, domain.LangGo, "alphabetical")

	if len(violations) > 0 {
		t.Errorf("expected no violations for correct alphabetical order, got %d", len(violations))
	}
}

func TestImportOrderViolation(t *testing.T) {
	a := NewAnalyzer(nil).WithImportOrder(domain.LangGo, "alphabetical")

	deps := []index.Dependency{
		{Path: "os", File: "main.go", Line: 1, Language: domain.LangGo},
		{Path: "fmt", File: "main.go", Line: 2, Language: domain.LangGo},
	}

	violations := a.checkImportOrder("main.go", deps, domain.LangGo, "alphabetical")

	if len(violations) != 1 {
		t.Errorf("expected 1 violation for wrong alphabetical order, got %d", len(violations))
	}
	if len(violations) > 0 && violations[0].Type != ConventionImportOrder {
		t.Errorf("expected ConventionImportOrder violation")
	}
}

func TestCalculateCoupling(t *testing.T) {
	a := NewAnalyzer(nil)

	deps := []index.Dependency{
		{Path: "github.com/foo/bar", File: "pkg/a.go", Line: 1, Language: domain.LangGo},
		{Path: "github.com/foo/baz", File: "pkg/a.go", Line: 2, Language: domain.LangGo},
		{Path: "github.com/foo/bar", File: "pkg/b.go", Line: 1, Language: domain.LangGo},
	}

	coupling := a.calculateCoupling(deps)

	if coupling.EfferentCoupling["pkg/a.go"] != 2 {
		t.Errorf("expected 2 efferent couplings for pkg/a.go, got %d", coupling.EfferentCoupling["pkg/a.go"])
	}
	if coupling.EfferentCoupling["pkg/b.go"] != 1 {
		t.Errorf("expected 1 efferent coupling for pkg/b.go, got %d", coupling.EfferentCoupling["pkg/b.go"])
	}
}

func TestAnalyzeEmpty(t *testing.T) {
	a := NewAnalyzer(nil)

	result, err := a.Analyze(context.Background())
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if len(result.Violations) != 0 {
		t.Errorf("expected 0 violations for empty index, got %d", len(result.Violations))
	}
	if len(result.CyclicDependencies) != 0 {
		t.Errorf("expected 0 cycles for empty index, got %d", len(result.CyclicDependencies))
	}
}

func TestAnalyzeRepository(t *testing.T) {
	a := NewAnalyzer(nil)

	result, err := a.AnalyzeRepository(context.Background(), "")
	if err != nil {
		t.Fatalf("AnalyzeRepository failed: %v", err)
	}

	if len(result.Violations) != 0 {
		t.Errorf("expected 0 violations for empty repository, got %d", len(result.Violations))
	}
}

func TestCheckCyclesEmpty(t *testing.T) {
	a := NewAnalyzer(nil)

	cycles := a.detectCycles([]index.Dependency{})

	if len(cycles) != 0 {
		t.Errorf("expected 0 cycles for empty dependencies, got %d", len(cycles))
	}
}

func TestCheckNamingConventions(t *testing.T) {
	a := NewAnalyzer(nil)

	a = a.WithNamingStyle(domain.LangGo, "camelCase")
	a = a.WithNamingStyle(domain.LangPython, "snake_case")
	a = a.WithNamingStyle(domain.LangJavaScript, "camelCase")
	a = a.WithNamingStyle(domain.LangTypeScript, "camelCase")
	a = a.WithNamingStyle(domain.LangPHP, "snake_case")

	if a.namingStyle[domain.LangGo] != "camelCase" {
		t.Errorf("naming style not set correctly")
	}
}

func TestCheckImportConventions(t *testing.T) {
	a := NewAnalyzer(nil)

	a = a.WithImportOrder(domain.LangGo, "alphabetical")
	a = a.WithImportOrder(domain.LangPython, "alphabetical")
	a = a.WithImportOrder(domain.LangJavaScript, "grouped")
	a = a.WithImportOrder(domain.LangTypeScript, "grouped")
	a = a.WithImportOrder(domain.LangPHP, "alphabetical")

	if a.importOrder[domain.LangGo] != "alphabetical" {
		t.Errorf("import order not set correctly")
	}
}