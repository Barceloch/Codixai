package policies

import (
	"strings"
	"testing"
)

func TestParseYAMLNestedMappings(t *testing.T) {
	document := `# comment
version: 1

project:
  name: demo   # inline comment
`
	node, err := parseYAML([]byte(document))
	if err != nil {
		t.Fatalf("parseYAML returned an error: %v", err)
	}
	if node.kind != yamlMapping {
		t.Fatalf("expected a mapping, got %v", node.kind)
	}
	version := node.get("version")
	if version == nil || version.value != "1" {
		t.Fatalf("version: %+v", version)
	}
	project := node.get("project")
	if project == nil || project.kind != yamlMapping {
		t.Fatalf("project: %+v", project)
	}
	name := project.get("name")
	if name == nil || name.value != "demo" {
		t.Fatalf("name: %+v", name)
	}
}

func TestParseYAMLSequences(t *testing.T) {
	document := `analysis:
  languages:
    - go
    - python
  exclude: [vendor/**, node_modules/**]
`
	node, err := parseYAML([]byte(document))
	if err != nil {
		t.Fatalf("parseYAML returned an error: %v", err)
	}
	analysis := node.get("analysis")
	languages := analysis.get("languages")
	if languages == nil || languages.kind != yamlSequence {
		t.Fatalf("languages: %+v", languages)
	}
	if len(languages.list) != 2 {
		t.Fatalf("expected 2 languages, got %d", len(languages.list))
	}
	if languages.list[0].value != "go" || languages.list[1].value != "python" {
		t.Errorf("languages: %+v", languages.list)
	}
	exclude := analysis.get("exclude")
	if exclude == nil || exclude.kind != yamlSequence {
		t.Fatalf("exclude: %+v", exclude)
	}
	if len(exclude.list) != 2 || exclude.list[0].value != "vendor/**" {
		t.Errorf("exclude: %+v", exclude.list)
	}
}

func TestParseYAMLSequenceOfMappings(t *testing.T) {
	document := `audit:
  suppressions:
    - rule: CDX-IMP-001
      path: vendor/**
      reason: vendored code
    - rule: CDX-SEC-001
      reason: accepted risk
`
	node, err := parseYAML([]byte(document))
	if err != nil {
		t.Fatalf("parseYAML returned an error: %v", err)
	}
	suppressions := node.get("audit").get("suppressions")
	if suppressions == nil || suppressions.kind != yamlSequence {
		t.Fatalf("suppressions: %+v", suppressions)
	}
	if len(suppressions.list) != 2 {
		t.Fatalf("expected 2 suppressions, got %d", len(suppressions.list))
	}
	first := suppressions.list[0]
	if first.kind != yamlMapping {
		t.Fatalf("expected a mapping, got %v", first.kind)
	}
	if first.get("rule").value != "CDX-IMP-001" {
		t.Errorf("rule: %+v", first.get("rule"))
	}
	if first.get("path").value != "vendor/**" {
		t.Errorf("path: %+v", first.get("path"))
	}
	if first.get("reason").value != "vendored code" {
		t.Errorf("reason: %+v", first.get("reason"))
	}
	second := suppressions.list[1]
	if second.get("rule").value != "CDX-SEC-001" {
		t.Errorf("rule: %+v", second.get("rule"))
	}
	if second.get("path") != nil {
		t.Errorf("expected no path, got %+v", second.get("path"))
	}
}

func TestParseYAMLQuotedScalars(t *testing.T) {
	document := `project:
  name: "my project"
  description: 'it is fine'
`
	node, err := parseYAML([]byte(document))
	if err != nil {
		t.Fatalf("parseYAML returned an error: %v", err)
	}
	project := node.get("project")
	if project.get("name").value != "my project" {
		t.Errorf("name: %+v", project.get("name"))
	}
	if project.get("description").value != "it is fine" {
		t.Errorf("description: %+v", project.get("description"))
	}
}

func TestParseYAMLErrors(t *testing.T) {
	cases := []struct {
		name     string
		document string
		want     string
	}{
		{"invalid entry", "not a mapping\n", "invalid mapping entry"},
		{"tab indentation", "a:\n\tb: 1\n", "tabs"},
		{"unterminated quote", "a: \"unterminated\n", "unterminated"},
		{"duplicate key", "a: 1\na: 2\n", "duplicate key"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseYAML([]byte(tc.document))
			if err == nil {
				t.Fatal("expected an error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("the error %q does not contain %q", err.Error(), tc.want)
			}
			if !strings.Contains(err.Error(), "line ") {
				t.Errorf("the error %q does not name the line", err.Error())
			}
		})
	}
}

func TestParseYAMLEmptyDocument(t *testing.T) {
	node, err := parseYAML(nil)
	if err != nil {
		t.Fatalf("parseYAML returned an error: %v", err)
	}
	if node.kind != yamlMapping {
		t.Errorf("expected an empty mapping, got %v", node.kind)
	}
}
