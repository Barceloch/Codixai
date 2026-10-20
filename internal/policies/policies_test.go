package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeConfig writes a configuration file in a temporary directory.
func writeConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("cannot write the configuration: %v", err)
	}
	return dir
}

func TestLoadWithoutFileReturnsDefaults(t *testing.T) {
	config, err := Load(t.TempDir())
	if err != nil {
		t.Fatalf("Load returned an error: %v", err)
	}
	if config.Source != "" {
		t.Errorf("expected no source for the defaults, got %q", config.Source)
	}
	if config.Version != 1 {
		t.Errorf("version: got %d, want 1", config.Version)
	}
	if len(config.Analysis.Languages) != len(SupportedLanguages) {
		t.Errorf("languages: got %v, want %v", config.Analysis.Languages, SupportedLanguages)
	}
	if len(config.Analysis.Exclude) != len(DefaultExclude) {
		t.Errorf("exclude: got %v, want %v", config.Analysis.Exclude, DefaultExclude)
	}
	if !config.Analysis.RespectGitignore {
		t.Error("expected respect_gitignore to be true by default")
	}
	if config.Audit.FailOn != FailOnError {
		t.Errorf("fail_on: got %q, want %q", config.Audit.FailOn, FailOnError)
	}
	if config.Audit.Baseline != DefaultBaseline {
		t.Errorf("baseline: got %q, want %q", config.Audit.Baseline, DefaultBaseline)
	}
	if config.Output.Format != FormatText {
		t.Errorf("format: got %q, want %q", config.Output.Format, FormatText)
	}
	if len(config.Audit.Rules) != 0 {
		t.Errorf("expected no rule overrides, got %v", config.Audit.Rules)
	}
}

func TestLoadValidFile(t *testing.T) {
	content := `version: 1

project:
  name: my-project

analysis:
  languages:
    - go
    - python
    - php
    - javascript
    - typescript

  exclude:
    - vendor/**
    - node_modules/**
    - dist/**
    - build/**
    - .git/**
    - coverage/**

audit:
  fail_on: error
  baseline: .codixai/baseline.json   # archivo versionado en el repositorio
  rules:
    CDX-IMP-001:
      enabled: true
      severity: error
  suppressions: []

output:
  format: text
`
	dir := writeConfig(t, content)
	config, err := Load(dir)
	if err != nil {
		t.Fatalf("Load returned an error: %v", err)
	}
	if config.Project.Name != "my-project" {
		t.Errorf("project name: got %q", config.Project.Name)
	}
	if len(config.Analysis.Languages) != 5 {
		t.Errorf("languages: got %v", config.Analysis.Languages)
	}
	if config.Audit.Baseline != ".codixai/baseline.json" {
		t.Errorf("baseline: got %q", config.Audit.Baseline)
	}
	rule, ok := config.Rule("CDX-IMP-001")
	if !ok {
		t.Fatal("the rule CDX-IMP-001 was not configured")
	}
	if !rule.IsEnabled() {
		t.Error("expected the rule to be enabled")
	}
	if rule.Severity != SeverityError {
		t.Errorf("severity: got %q", rule.Severity)
	}
	if len(config.Audit.Suppressions) != 0 {
		t.Errorf("expected no suppressions, got %v", config.Audit.Suppressions)
	}
	if config.Source == "" {
		t.Error("expected the source of the configuration to be recorded")
	}
}

func TestLoadMalformedFile(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    []string
	}{
		{
			name:    "scalar instead of a list",
			content: "version: 1\nanalysis:\n  languages: go\n",
			want:    []string{FileName, "analysis.languages", "line 3"},
		},
		{
			name:    "invalid mapping entry",
			content: "version: 1\nthis is not a mapping\n",
			want:    []string{FileName, "line 2"},
		},
		{
			name:    "tab indentation",
			content: "audit:\n\tfail_on: error\n",
			want:    []string{FileName, "line 2", "tabs"},
		},
		{
			name:    "unsupported version",
			content: "version: 7\n",
			want:    []string{FileName, "version", "line 1"},
		},
		{
			name:    "unsupported language",
			content: "analysis:\n  languages:\n    - ruby\n",
			want:    []string{FileName, "ruby", "line 3"},
		},
		{
			name:    "invalid fail on",
			content: "audit:\n  fail_on: maybe\n",
			want:    []string{FileName, "audit.fail_on", "line 2"},
		},
		{
			name:    "invalid format",
			content: "output:\n  format: xml\n",
			want:    []string{FileName, "output.format", "line 2"},
		},
		{
			name:    "invalid severity",
			content: "audit:\n  rules:\n    CDX-IMP-001:\n      severity: critical\n",
			want:    []string{FileName, "critical", "line 4"},
		},
		{
			name:    "invalid boolean",
			content: "analysis:\n  respect_gitignore: maybe\n",
			want:    []string{FileName, "respect_gitignore", "line 2"},
		},
		{
			name:    "suppression without a reason",
			content: "audit:\n  suppressions:\n    - rule: CDX-IMP-001\n      path: src/**\n",
			want:    []string{FileName, "reason", "line 3"},
		},
		{
			name:    "duplicate key",
			content: "version: 1\nversion: 2\n",
			want:    []string{FileName, "duplicate key", "line 2"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := writeConfig(t, tc.content)
			_, err := Load(dir)
			if err == nil {
				t.Fatal("expected an error")
			}
			message := err.Error()
			for _, needle := range tc.want {
				if !strings.Contains(message, needle) {
					t.Errorf("the error %q does not contain %q", message, needle)
				}
			}
		})
	}
}

func TestRuleOverrides(t *testing.T) {
	content := `audit:
  rules:
    CDX-IMP-001:
      enabled: false
    CDX-REF-001:
      severity: warning
    CDX-SEC-001:
      enabled: true
      severity: info
`
	dir := writeConfig(t, content)
	config, err := Load(dir)
	if err != nil {
		t.Fatalf("Load returned an error: %v", err)
	}
	imp, ok := config.Rule("CDX-IMP-001")
	if !ok {
		t.Fatal("CDX-IMP-001 was not configured")
	}
	if imp.IsEnabled() {
		t.Error("expected CDX-IMP-001 to be disabled")
	}
	if imp.Severity != "" {
		t.Errorf("expected no severity override, got %q", imp.Severity)
	}

	ref, ok := config.Rule("CDX-REF-001")
	if !ok {
		t.Fatal("CDX-REF-001 was not configured")
	}
	if !ref.IsEnabled() {
		t.Error("expected a rule without an explicit flag to stay enabled")
	}
	if ref.Severity != SeverityWarning {
		t.Errorf("severity: got %q, want %q", ref.Severity, SeverityWarning)
	}

	sec, ok := config.Rule("CDX-SEC-001")
	if !ok {
		t.Fatal("CDX-SEC-001 was not configured")
	}
	if !sec.IsEnabled() {
		t.Error("expected CDX-SEC-001 to be enabled")
	}
	if sec.Severity != SeverityInfo {
		t.Errorf("severity: got %q, want %q", sec.Severity, SeverityInfo)
	}
}

func TestSuppressions(t *testing.T) {
	content := `audit:
  suppressions:
    - rule: CDX-IMP-001
      path: vendor/**
      reason: vendored dependencies are not audited
    - rule: CDX-SEC-001
      reason: accepted risk, reviewed on 2026-01-01
`
	dir := writeConfig(t, content)
	config, err := Load(dir)
	if err != nil {
		t.Fatalf("Load returned an error: %v", err)
	}
	if len(config.Audit.Suppressions) != 2 {
		t.Fatalf("expected 2 suppressions, got %d", len(config.Audit.Suppressions))
	}
	first := config.Audit.Suppressions[0]
	if first.Rule != "CDX-IMP-001" || first.Path != "vendor/**" || first.Reason == "" {
		t.Errorf("first suppression: %+v", first)
	}
	second := config.Audit.Suppressions[1]
	if second.Rule != "CDX-SEC-001" || second.Path != "" || second.Reason == "" {
		t.Errorf("second suppression: %+v", second)
	}
	if !config.IsSuppressed("CDX-IMP-001", "vendor/github.com/pkg/file.go") {
		t.Error("expected the vendor path to be suppressed")
	}
	if config.IsSuppressed("CDX-IMP-001", "src/main.go") {
		t.Error("expected the src path to not be suppressed")
	}
	if !config.IsSuppressed("CDX-SEC-001", "any/path.go") {
		t.Error("expected a suppression without a path to match every path")
	}
	if config.IsSuppressed("CDX-REF-001", "any/path.go") {
		t.Error("expected another rule to not be suppressed")
	}
}

func TestExcludedUsesTheConfiguredPatterns(t *testing.T) {
	content := `analysis:
  exclude:
    - vendor/**
    - "**/*.generated.go"
`
	dir := writeConfig(t, content)
	config, err := Load(dir)
	if err != nil {
		t.Fatalf("Load returned an error: %v", err)
	}
	if !config.Excluded("vendor/github.com/pkg/file.go") {
		t.Error("expected the vendor path to be excluded")
	}
	if !config.Excluded("internal/api/model.generated.go") {
		t.Error("expected the generated file to be excluded")
	}
	if config.Excluded("internal/app.go") {
		t.Error("expected the application file to not be excluded")
	}
}

func TestInlineListAndBooleanValues(t *testing.T) {
	content := `analysis:
  languages: [go, python]
  respect_gitignore: false
output:
  format: json
`
	dir := writeConfig(t, content)
	config, err := Load(dir)
	if err != nil {
		t.Fatalf("Load returned an error: %v", err)
	}
	if len(config.Analysis.Languages) != 2 {
		t.Errorf("languages: got %v", config.Analysis.Languages)
	}
	if config.Analysis.RespectGitignore {
		t.Error("expected respect_gitignore to be false")
	}
	if config.Output.Format != FormatJSON {
		t.Errorf("format: got %q", config.Output.Format)
	}
}

func TestLoadAlternativeFileName(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, AltFileName), []byte("project:\n  name: alt\n"), 0o644); err != nil {
		t.Fatalf("cannot write the configuration: %v", err)
	}
	config, err := Load(dir)
	if err != nil {
		t.Fatalf("Load returned an error: %v", err)
	}
	if config.Project.Name != "alt" {
		t.Errorf("project name: got %q", config.Project.Name)
	}
}
