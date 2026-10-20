package rules

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"codixai/internal/domain"
	"codixai/internal/languages/go"
)

// checkReferences implements the rule CDX-REF-001.
//
// The rule is implemented for Go only. The other languages declare the
// support level "none" and never produce findings.
func checkReferences(ctx *Context) ([]domain.Finding, error) {
	if ctx.Language != domain.LangGo {
		return nil, nil
	}
	if ctx.View.Status == domain.FileDeleted {
		return nil, nil
	}
	source, ok := goSource(ctx)
	if !ok {
		ctx.Diagnostic("the content of the file is not available in the repository root, so its references cannot be resolved")
		return nil, nil
	}
	siblings, ok := goSiblings(ctx)
	if !ok {
		ctx.Diagnostic("the package directory cannot be read, so the references cannot be resolved")
		return nil, nil
	}
	references, complete, err := golang.FindUndefinedReferences(ctx.View.Path, source, siblings)
	if err != nil {
		ctx.Diagnostic(fmt.Sprintf("the Go file cannot be parsed: %v", err))
		return nil, nil
	}
	if !complete {
		ctx.Diagnostic("a file of the package directory cannot be parsed, so the references are not resolved")
		return nil, nil
	}
	var findings []domain.Finding
	for _, reference := range references {
		if !ctx.View.IsAdded(reference.Line) {
			continue
		}
		message := fmt.Sprintf("The symbol %q is referenced but it is not declared in the package directory.", reference.Name)
		recommendation := "Declare the symbol in the package or fix the name of the reference."
		findings = append(findings, ctx.finding(ctx.Definition, domain.EpistemicInferred, message, reference.Line, recommendation))
	}
	return findings, nil
}

// goSource reads the content of the changed Go file from the repository
// root.
func goSource(ctx *Context) ([]byte, bool) {
	if ctx.Root == "" {
		return nil, false
	}
	path := filepath.Join(ctx.Root, filepath.FromSlash(ctx.View.Path))
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	return data, true
}

// goSiblings reads the Go files of the package directory of the changed
// file.
func goSiblings(ctx *Context) ([]golang.SiblingFile, bool) {
	if ctx.Root == "" {
		return nil, false
	}
	directory := filepath.Join(ctx.Root, filepath.FromSlash(ctx.directory()))
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, false
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		if entry.Name() == filepath.Base(ctx.View.Path) {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	siblings := make([]golang.SiblingFile, 0, len(names))
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(directory, name))
		if err != nil {
			return nil, false
		}
		siblings = append(siblings, golang.SiblingFile{Name: name, Src: data})
	}
	return siblings, true
}
