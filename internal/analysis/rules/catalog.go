// Package rules contains the deterministic rule catalog of Codixai.
//
// Every rule has a stable identifier, a version, a default severity, a
// support matrix by language and a set of known limitations. The support
// matrix is published by the engine and validated against the fixture
// corpus.
//
// The rules are syntactic. Nothing in this package resolves a semantic
// relation, executes code of the analyzed repository or accesses the
// network.
package rules

import (
	"codixai/internal/domain"
)

// Rule identifiers of the catalog.
const (
	// IDImports checks the imports that are not declared in the manifests.
	IDImports = "CDX-IMP-001"
	// IDReferences checks the local symbols that do not exist.
	IDReferences = "CDX-REF-001"
	// IDTests checks the tests that were deleted, disabled or weakened.
	IDTests = "CDX-TST-001"
	// IDSuppressions checks the new analyzer suppressions.
	IDSuppressions = "CDX-SUP-001"
	// IDErrors checks the ignored errors and the empty handlers.
	IDErrors = "CDX-ERR-001"
	// IDPlaceholders checks the placeholders and the stubs.
	IDPlaceholders = "CDX-PLH-001"
)

// RuleVersion is the version of the rules of this milestone.
//
// A rule that changes its behavior increments its own version, so a
// consumer can tell the two behaviors apart.
const RuleVersion = "1"

// RuleVersionReferences is the version of CDX-REF-001.
//
// Version 2 resolves the predeclared identifiers of the language and the
// name of the packages of the module, so a reference to one of them is no
// longer reported as a missing symbol.
const RuleVersionReferences = "2"

// RuleVersionPlaceholders is the version of CDX-PLH-001.
//
// Version 2 scopes the explicit markers to the comments, to the code and
// to the thrown errors, so a description that names a marker is no longer
// reported as a placeholder.
const RuleVersionPlaceholders = "2"

// Definition describes a deterministic rule.
type Definition struct {
	// ID is the stable identifier of the rule.
	ID string
	// Version is the version of the rule. It changes when the behavior of
	// the rule changes.
	Version string
	// Area is the area of the identifier.
	Area string
	// Description explains, in English, what the rule checks.
	Description string
	// Severity is the default severity of the findings.
	Severity domain.Severity
	// EvidenceKind is the kind of the evidence that the rule produces.
	EvidenceKind string
	// Method explains how the rule checks the code.
	Method string
	// Languages is the support matrix of the rule.
	Languages map[string]domain.SupportLevel
	// Limitations holds the known limitations of the rule, by language.
	Limitations map[string][]string
	// Check applies the rule to one changed file.
	Check func(*Context) ([]domain.Finding, error)
}

// Supports reports whether the rule is available for a language.
func (d Definition) Supports(language string) bool {
	return d.Languages[language] != domain.SupportNone
}

// catalog holds the rules of the milestone, in a stable order.
var catalog = []Definition{
	{
		ID:           IDImports,
		Version:      RuleVersion,
		Area:         "IMP",
		Description:  "An import added by the change refers to a dependency that is not declared in the manifests of the repository and does not belong to the standard library of the language.",
		Severity:     domain.SeverityError,
		EvidenceKind: "diff_hunk",
		Method:       "manifest reading plus language specific import extraction",
		Languages: map[string]domain.SupportLevel{
			domain.LangGo:         domain.SupportSyntactic,
			domain.LangPython:     domain.SupportSyntactic,
			domain.LangPHP:        domain.SupportSyntactic,
			domain.LangJavaScript: domain.SupportSyntactic,
			domain.LangTypeScript: domain.SupportSyntactic,
		},
		Limitations: map[string][]string{
			domain.LangGo: {
				"Only the go.mod file of the repository root is read; the workspace modules and the vendored dependencies are not resolved.",
				"Only the import declarations that the change adds are checked.",
			},
			domain.LangPython: {
				"requirements.txt and pyproject.toml are read with a reduced parser; unusual TOML layouts are ignored.",
				"A module of the project that lives next to the changed file is only detected when the file system is available.",
			},
			domain.LangPHP: {
				"The composer package of a namespace is inferred from its first two segments, which may be wrong for unusual package names.",
				"Only the use statements that the change adds are checked.",
			},
			domain.LangJavaScript: {
				"Only the dependencies, the devDependencies and the optionalDependencies of package.json are read.",
				"Only the import statements that the change adds are checked.",
			},
			domain.LangTypeScript: {
				"Only the dependencies, the devDependencies and the optionalDependencies of package.json are read.",
				"Only the import statements that the change adds are checked.",
			},
		},
		Check: checkImports,
	},
	{
		ID:           IDReferences,
		Version:      RuleVersionReferences,
		Area:         "REF",
		Description:  "A local symbol referenced by a changed Go file does not exist in the package directory of the file.",
		Severity:     domain.SeverityError,
		EvidenceKind: "diff_hunk",
		Method:       "go/parser and go/ast over the package directory",
		Languages: map[string]domain.SupportLevel{
			domain.LangGo:         domain.SupportSyntactic,
			domain.LangPython:     domain.SupportNone,
			domain.LangPHP:        domain.SupportNone,
			domain.LangJavaScript: domain.SupportNone,
			domain.LangTypeScript: domain.SupportNone,
		},
		Limitations: map[string][]string{
			domain.LangGo: {
				"The scope of the analysis is the package directory of the changed file, not the whole module.",
				"Only the references that the change introduces are reported.",
				"A symbol declared in another package of the module is not resolved, so it is reported as a missing symbol.",
				"The predeclared identifiers of the language and the packages of the module are resolved.",
			},
		},
		Check: checkReferences,
	},
	{
		ID:           IDTests,
		Version:      RuleVersion,
		Area:         "TST",
		Description:  "The change deletes, disables or weakens a test.",
		Severity:     domain.SeverityWarning,
		EvidenceKind: "diff_hunk",
		Method:       "diff analysis with the language specific test markers",
		Languages: map[string]domain.SupportLevel{
			domain.LangGo:         domain.SupportSyntactic,
			domain.LangPython:     domain.SupportSyntactic,
			domain.LangPHP:        domain.SupportSyntactic,
			domain.LangJavaScript: domain.SupportSyntactic,
			domain.LangTypeScript: domain.SupportSyntactic,
		},
		Limitations: map[string][]string{
			domain.LangGo: {
				"A removed assertion is reported when the file does not add an equivalent one; another assertion may still cover the same behavior.",
				"Only the files that match the test patterns of the language are analyzed.",
			},
			domain.LangPython: {
				"A removed assertion is reported when the file does not add an equivalent one; another assertion may still cover the same behavior.",
				"Only the files that match the test patterns of the language are analyzed.",
			},
			domain.LangPHP: {
				"A removed assertion is reported when the file does not add an equivalent one; another assertion may still cover the same behavior.",
				"Only the files that match the test patterns of the language are analyzed.",
			},
			domain.LangJavaScript: {
				"A removed assertion is reported when the file does not add an equivalent one; another assertion may still cover the same behavior.",
				"Only the files that match the test patterns of the language are analyzed.",
			},
			domain.LangTypeScript: {
				"A removed assertion is reported when the file does not add an equivalent one; another assertion may still cover the same behavior.",
				"Only the files that match the test patterns of the language are analyzed.",
			},
		},
		Check: checkTests,
	},
	{
		ID:           IDSuppressions,
		Version:      RuleVersion,
		Area:         "SUP",
		Description:  "The change adds a suppression directive that silences an analyzer.",
		Severity:     domain.SeverityWarning,
		EvidenceKind: "diff_hunk",
		Method:       "lexical scan of the changed lines",
		Languages: map[string]domain.SupportLevel{
			domain.LangGo:         domain.SupportSyntactic,
			domain.LangPython:     domain.SupportSyntactic,
			domain.LangPHP:        domain.SupportSyntactic,
			domain.LangJavaScript: domain.SupportSyntactic,
			domain.LangTypeScript: domain.SupportSyntactic,
		},
		Limitations: map[string][]string{
			domain.LangGo: {
				"Only the nolint directives of the Go comments are detected.",
				"The SuppressWarnings marker belongs to Java and Java is not one of the supported languages, so it is never reported.",
			},
			domain.LangPython: {
				"Only the noqa, type: ignore, pylint and mypy directives are detected.",
			},
			domain.LangPHP: {
				"Only the phpcs, phpstan and IDE inspection directives are detected.",
				"The SuppressWarnings marker belongs to Java and Java is not one of the supported languages, so it is never reported.",
			},
			domain.LangJavaScript: {
				"Only the eslint, prettier and nolint directives are detected.",
			},
			domain.LangTypeScript: {
				"Only the eslint, prettier, nolint and TypeScript compiler directives are detected.",
			},
		},
		Check: checkSuppressions,
	},
	{
		ID:           IDErrors,
		Version:      RuleVersion,
		Area:         "ERR",
		Description:  "The change adds a pattern that discards an error or an empty handler.",
		Severity:     domain.SeverityWarning,
		EvidenceKind: "diff_hunk",
		Method:       "lexical scan of the changed lines",
		Languages: map[string]domain.SupportLevel{
			domain.LangGo:         domain.SupportSyntactic,
			domain.LangPython:     domain.SupportSyntactic,
			domain.LangPHP:        domain.SupportSyntactic,
			domain.LangJavaScript: domain.SupportSyntactic,
			domain.LangTypeScript: domain.SupportSyntactic,
		},
		Limitations: map[string][]string{
			domain.LangGo: {
				"An empty error check is only reported when the closing brace is visible in the changed lines.",
				"A swallowed error is not necessarily a defect; the finding is a signal to review the intent of the change.",
			},
			domain.LangPython: {
				"An empty handler is only reported when its body is visible in the changed lines.",
				"A swallowed error is not necessarily a defect; the finding is a signal to review the intent of the change.",
			},
			domain.LangPHP: {
				"An empty catch block is only reported when its closing brace is visible in the changed lines.",
				"A swallowed error is not necessarily a defect; the finding is a signal to review the intent of the change.",
			},
			domain.LangJavaScript: {
				"An empty catch block is only reported when its closing brace is visible in the changed lines.",
				"A swallowed error is not necessarily a defect; the finding is a signal to review the intent of the change.",
			},
			domain.LangTypeScript: {
				"An empty catch block is only reported when its closing brace is visible in the changed lines.",
				"A swallowed error is not necessarily a defect; the finding is a signal to review the intent of the change.",
			},
		},
		Check: checkErrors,
	},
	{
		ID:           IDPlaceholders,
		Version:      RuleVersionPlaceholders,
		Area:         "PLH",
		Description:  "The change adds an obvious placeholder or stub.",
		Severity:     domain.SeverityWarning,
		EvidenceKind: "diff_hunk",
		Method:       "lexical scan of the changed lines",
		Languages: map[string]domain.SupportLevel{
			domain.LangGo:         domain.SupportSyntactic,
			domain.LangPython:     domain.SupportSyntactic,
			domain.LangPHP:        domain.SupportSyntactic,
			domain.LangJavaScript: domain.SupportSyntactic,
			domain.LangTypeScript: domain.SupportSyntactic,
		},
		Limitations: map[string][]string{
			domain.LangGo: {
				"A placeholder marker is a signal and not a defect; the marker may describe a legitimate pending task.",
				"The explicit markers are matched in the comments, in the code and in the thrown errors; a plain string that names a marker is not a placeholder.",
			},
			domain.LangPython: {
				"A placeholder marker is a signal and not a defect; the marker may describe a legitimate pending task.",
				"The explicit markers are matched in the comments, in the code and in the thrown errors; a plain string that names a marker is not a placeholder.",
			},
			domain.LangPHP: {
				"A placeholder marker is a signal and not a defect; the marker may describe a legitimate pending task.",
				"The explicit markers are matched in the comments, in the code and in the thrown errors; a plain string that names a marker is not a placeholder.",
			},
			domain.LangJavaScript: {
				"A placeholder marker is a signal and not a defect; the marker may describe a legitimate pending task.",
				"The explicit markers are matched in the comments, in the code and in the thrown errors; a plain string that names a marker is not a placeholder.",
			},
			domain.LangTypeScript: {
				"A placeholder marker is a signal and not a defect; the marker may describe a legitimate pending task.",
				"The explicit markers are matched in the comments, in the code and in the thrown errors; a plain string that names a marker is not a placeholder.",
			},
		},
		Check: checkPlaceholders,
	},
}

// Catalog returns the definitions of the rules, in a stable order.
func Catalog() []Definition {
	return catalog
}

// definitionOf returns the definition of a rule of the catalog.
func definitionOf(id string) Definition {
	for _, definition := range catalog {
		if definition.ID == id {
			return definition
		}
	}
	return Definition{}
}

// Rule converts a definition into the domain rule published by the engine.
//
// The known limitations of the rule are published with the language that
// they affect, because the domain model keeps a single list of limitations
// for the rule.
func Rule(definition Definition) domain.Rule {
	rule := domain.Rule{
		ID:           definition.ID,
		Version:      definition.Version,
		Area:         definition.Area,
		Description:  definition.Description,
		Severity:     definition.Severity,
		Languages:    map[string]domain.SupportLevel{},
		Method:       definition.Method,
		EvidenceKind: definition.EvidenceKind,
	}
	for _, language := range SupportedLanguages() {
		level, ok := definition.Languages[language]
		if !ok {
			level = domain.SupportNone
		}
		rule.Languages[language] = level
	}
	for _, language := range SupportedLanguages() {
		for _, limitation := range definition.Limitations[language] {
			rule.Limitations = append(rule.Limitations, language+": "+limitation)
		}
	}
	return rule
}

// Rules converts the catalog into the domain rules published by the engine.
func Rules() []domain.Rule {
	definitions := Catalog()
	rules := make([]domain.Rule, 0, len(definitions))
	for _, definition := range definitions {
		rules = append(rules, Rule(definition))
	}
	return rules
}

// SupportedLanguages returns the identifiers of the supported languages, in a
// stable order.
func SupportedLanguages() []string {
	return []string{
		domain.LangGo,
		domain.LangJavaScript,
		domain.LangPHP,
		domain.LangPython,
		domain.LangTypeScript,
	}
}
