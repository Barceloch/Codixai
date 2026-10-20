package rules

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Barceloch/Codixai/internal/domain"
	"github.com/Barceloch/Codixai/internal/languages/go"
	"github.com/Barceloch/Codixai/internal/languages/javascript"
	"github.com/Barceloch/Codixai/internal/languages/lexical"
	"github.com/Barceloch/Codixai/internal/languages/php"
	"github.com/Barceloch/Codixai/internal/languages/python"
	"github.com/Barceloch/Codixai/internal/languages/typescript"
)

// importDecision is the result of the classification of one import.
type importDecision struct {
	// Skip states whether the import must not be reported.
	Skip bool
	// Declared states whether a manifest of the repository declares the
	// dependency of the import.
	Declared bool
	// ManifestFound states whether a manifest of the language was found in
	// the repository.
	ManifestFound bool
	// Shape states whether the import is unambiguously third party.
	Shape bool
	// Local states whether the import points to the local project.
	Local bool
}

// checkImports implements the rule CDX-IMP-001.
func checkImports(ctx *Context) ([]domain.Finding, error) {
	var findings []domain.Finding
	for _, spec := range importSpecs(ctx) {
		if !ctx.View.IsAdded(spec.Line) {
			continue
		}
		decision := classifyImport(ctx, spec)
		if decision.Skip || decision.Declared {
			continue
		}
		status := domain.EpistemicInferred
		if decision.ManifestFound && decision.Shape {
			status = domain.EpistemicObserved
		}
		message := fmt.Sprintf("The import %q is not declared in %s.", spec.Module, manifestName(ctx.Language))
		recommendation := fmt.Sprintf("Declare %q in %s or remove the import.", spec.Module, manifestName(ctx.Language))
		findings = append(findings, ctx.finding(ctx.Definition, status, message, spec.Line, recommendation))
	}
	return findings, nil
}

// importSpecs returns the imports declared by the changed file.
func importSpecs(ctx *Context) []lexical.ImportSpec {
	switch ctx.Language {
	case domain.LangGo:
		return goImportSpecs(ctx)
	case domain.LangPython:
		return python.ExtractImports(ctx.View.NewLines, ctx.View.NewSegments)
	case domain.LangPHP:
		return php.ExtractImports(ctx.View.NewLines, ctx.View.NewSegments)
	case domain.LangJavaScript:
		return javascript.ExtractImports(ctx.View.NewLines, ctx.View.NewSegments)
	case domain.LangTypeScript:
		return typescript.ExtractImports(ctx.View.NewLines, ctx.View.NewSegments)
	}
	return nil
}

// goImportSpecs returns the imports of a Go file.
//
// The extraction of go/parser is used when the content of the file is
// available. Otherwise the textual extraction of the changed lines is used.
func goImportSpecs(ctx *Context) []lexical.ImportSpec {
	if file, ok := goSourceFile(ctx); ok {
		specs := make([]lexical.ImportSpec, 0, len(file.Imports))
		for _, spec := range file.Imports {
			specs = append(specs, lexical.ImportSpec{Module: spec.Path, Line: spec.Line})
		}
		return specs
	}
	return golang.ExtractImports(ctx.View.NewLines)
}

// goSourceFile reads and parses the changed Go file of the repository.
func goSourceFile(ctx *Context) (*golang.File, bool) {
	if ctx.Root == "" {
		return nil, false
	}
	path := filepath.Join(ctx.Root, filepath.FromSlash(ctx.View.Path))
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	file, err := golang.Parse(ctx.View.Path, data)
	if err != nil {
		ctx.Diagnostic(fmt.Sprintf("the Go file cannot be parsed: %v", err))
		return nil, false
	}
	return file, true
}

// classifyImport decides whether an import has to be reported.
func classifyImport(ctx *Context, spec lexical.ImportSpec) importDecision {
	switch ctx.Language {
	case domain.LangGo:
		return classifyGoImport(ctx, spec)
	case domain.LangPython:
		return classifyPythonImport(ctx, spec)
	case domain.LangPHP:
		return classifyPHPImport(ctx, spec)
	case domain.LangJavaScript:
		return classifyJavaScriptImport(ctx, spec)
	case domain.LangTypeScript:
		return classifyTypeScriptImport(ctx, spec)
	}
	return importDecision{Skip: true}
}

// classifyGoImport decides whether a Go import has to be reported.
func classifyGoImport(ctx *Context, spec lexical.ImportSpec) importDecision {
	manifest := golang.ReadManifest(ctx.Root)
	decision := importDecision{ManifestFound: manifest.Found}
	if golang.IsStdlib(spec.Module) {
		decision.Skip = true
		return decision
	}
	if manifest.IsLocal(spec.Module) {
		decision.Skip = true
		decision.Local = true
		return decision
	}
	if manifest.Declares(spec.Module) {
		decision.Declared = true
		return decision
	}
	decision.Shape = golang.IsThirdPartyShape(spec.Module)
	return decision
}

// classifyPythonImport decides whether a Python import has to be reported.
func classifyPythonImport(ctx *Context, spec lexical.ImportSpec) importDecision {
	manifest := python.ReadManifest(ctx.Root)
	decision := importDecision{ManifestFound: manifest.Found}
	if spec.Relative {
		decision.Skip = true
		decision.Local = true
		return decision
	}
	if python.IsStdlib(spec.Module) {
		decision.Skip = true
		return decision
	}
	root := spec.Module
	if idx := strings.Index(root, "."); idx > 0 {
		root = root[:idx]
	}
	if pythonLocalModule(ctx, root) {
		decision.Skip = true
		decision.Local = true
		return decision
	}
	if manifest.Declares(root) {
		decision.Declared = true
		return decision
	}
	decision.Shape = true
	return decision
}

// pythonLocalModule reports whether a module of the project lives next to
// the changed file.
func pythonLocalModule(ctx *Context, name string) bool {
	if ctx.Root == "" || name == "" {
		return false
	}
	directory := filepath.Join(ctx.Root, filepath.FromSlash(ctx.directory()))
	if info, err := os.Stat(filepath.Join(directory, name+".py")); err == nil && !info.IsDir() {
		return true
	}
	if info, err := os.Stat(filepath.Join(directory, name)); err == nil && info.IsDir() {
		return true
	}
	return false
}

// classifyPHPImport decides whether a PHP import has to be reported.
func classifyPHPImport(ctx *Context, spec lexical.ImportSpec) importDecision {
	manifest := php.ReadManifest(ctx.Root)
	decision := importDecision{ManifestFound: manifest.Found}
	if manifest.IsLocal(spec.Module) {
		decision.Skip = true
		decision.Local = true
		return decision
	}
	if !strings.Contains(spec.Module, "\\") {
		// A single segment names a class of the global namespace. It is
		// either part of the standard library of PHP or a class of the
		// project, so it is never a third party dependency.
		decision.Skip = true
		decision.Local = !php.IsGlobal(spec.Module)
		return decision
	}
	packageName, ok := php.PackageOf(spec.Module)
	if !ok {
		decision.Skip = true
		return decision
	}
	if manifest.Declares(packageName) {
		decision.Declared = true
		return decision
	}
	decision.Shape = true
	return decision
}

// classifyJavaScriptImport decides whether a JavaScript import has to be
// reported.
func classifyJavaScriptImport(ctx *Context, spec lexical.ImportSpec) importDecision {
	manifest := javascript.ReadManifest(ctx.Root)
	decision := importDecision{ManifestFound: manifest.Found}
	if spec.Relative {
		decision.Skip = true
		decision.Local = true
		return decision
	}
	if javascript.IsBuiltin(spec.Module) {
		decision.Skip = true
		return decision
	}
	packageName := javascript.PackageOf(spec.Module)
	if manifest.Declares(packageName) {
		decision.Declared = true
		return decision
	}
	decision.Shape = true
	return decision
}

// classifyTypeScriptImport decides whether a TypeScript import has to be
// reported.
func classifyTypeScriptImport(ctx *Context, spec lexical.ImportSpec) importDecision {
	manifest := typescript.ReadManifest(ctx.Root)
	decision := importDecision{ManifestFound: manifest.Found}
	if spec.Relative {
		decision.Skip = true
		decision.Local = true
		return decision
	}
	if typescript.IsBuiltin(spec.Module) {
		decision.Skip = true
		return decision
	}
	packageName := typescript.PackageOf(spec.Module)
	if manifest.Declares(packageName) {
		decision.Declared = true
		return decision
	}
	decision.Shape = true
	return decision
}

// manifestName returns the name of the manifest of a language, for the
// messages of the rule.
func manifestName(language string) string {
	switch language {
	case domain.LangGo:
		return "go.mod"
	case domain.LangPython:
		return "requirements.txt or pyproject.toml"
	case domain.LangPHP:
		return "composer.json"
	case domain.LangJavaScript, domain.LangTypeScript:
		return "package.json"
	}
	return "the manifests of the repository"
}
