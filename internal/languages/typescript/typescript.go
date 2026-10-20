// Package typescript implements the TypeScript specific extraction of
// Codixai.
//
// TypeScript shares the module syntax, the manifest and the test layout of
// JavaScript. It adds the TypeScript specific suppression directives.
package typescript

import (
	"regexp"

	"codixai/internal/domain"
	"codixai/internal/languages/javascript"
	"codixai/internal/languages/lexical"
)

// Profile returns the lexical profile and the syntactic markers of
// TypeScript.
func Profile() lexical.Profile {
	profile := javascript.Profile()
	profile.ID = domain.LangTypeScript
	profile.Name = "TypeScript"
	profile.Extensions = []string{".ts", ".tsx", ".mts", ".cts"}
	profile.Suppressions = append(profile.Suppressions,
		lexical.Marker{
			Pattern:     regexp.MustCompile(`@ts-(ignore|expect-error|nocheck)\b`),
			Description: "TypeScript compiler suppression directive",
			Scope:       lexical.ScopeComment,
		},
	)
	profile.TestPaths = []string{
		"**/*.test.ts", "**/*.spec.ts", "**/*.test.tsx", "**/*.spec.tsx",
		"**/*.test.mts", "**/*.spec.mts", "**/__tests__/**",
	}
	return profile
}

// ExtractImports returns the module specifiers imported by the given source
// lines. TypeScript shares the syntax of JavaScript.
func ExtractImports(lines []lexical.Line, segments map[int][]lexical.Segment) []lexical.ImportSpec {
	return javascript.ExtractImports(lines, segments)
}

// EmptyHandlers returns the numbers of the lines that open an empty catch
// block. TypeScript shares the syntax of JavaScript.
func EmptyHandlers(lines []lexical.Line, segments map[int][]lexical.Segment) []int {
	return javascript.EmptyHandlers(lines, segments)
}

// IsBuiltin reports whether the given module specifier refers to a module of
// the Node.js runtime.
func IsBuiltin(specifier string) bool {
	return javascript.IsBuiltin(specifier)
}

// IsRelative reports whether the module specifier points to a local file.
func IsRelative(specifier string) bool {
	return javascript.IsRelative(specifier)
}

// PackageOf returns the package name of a module specifier.
func PackageOf(specifier string) string {
	return javascript.PackageOf(specifier)
}

// Manifest holds the dependencies declared by the package.json file of a
// repository.
type Manifest = javascript.Manifest

// ReadManifest reads the package.json file of the given repository root.
func ReadManifest(root string) Manifest {
	return javascript.ReadManifest(root)
}
