package rules_test

import (
	"testing"

	"codixai/internal/analysis/rules"
	"codixai/internal/domain"
)

// TestCatalogPublishesTheSupportMatrix checks the matrix of the catalog.
func TestCatalogPublishesTheSupportMatrix(t *testing.T) {
	published := rules.Rules()
	if len(published) != 6 {
		t.Fatalf("the catalog publishes %d rules, want 6", len(published))
	}
	matrix := map[string]map[string]domain.SupportLevel{}
	for _, rule := range published {
		matrix[rule.ID] = rule.Languages
		if len(rule.Languages) != len(rules.SupportedLanguages()) {
			t.Errorf("the rule %s does not publish a level for every language", rule.ID)
		}
		for language, level := range rule.Languages {
			if level == domain.SupportResolved {
				t.Errorf("the rule %s claims a resolved support level for %s", rule.ID, language)
			}
		}
	}
	references := matrix[rules.IDReferences]
	for language, level := range references {
		if language == domain.LangGo {
			if level != domain.SupportSyntactic {
				t.Errorf("the rule %s declares %q for go, want %q", rules.IDReferences, level, domain.SupportSyntactic)
			}
			continue
		}
		if level != domain.SupportNone {
			t.Errorf("the rule %s declares %q for %s, want %q", rules.IDReferences, level, language, domain.SupportNone)
		}
	}
	for _, id := range []string{
		rules.IDImports,
		rules.IDTests,
		rules.IDSuppressions,
		rules.IDErrors,
		rules.IDPlaceholders,
	} {
		for language, level := range matrix[id] {
			if level != domain.SupportSyntactic {
				t.Errorf("the rule %s declares %q for %s, want %q", id, level, language, domain.SupportSyntactic)
			}
		}
	}
}

// TestRuleIdentifiersFollowTheConvention checks the identifiers and the
// areas of the rules.
func TestRuleIdentifiersFollowTheConvention(t *testing.T) {
	for _, rule := range rules.Rules() {
		if len(rule.ID) != len("CDX-XXX-000") {
			t.Errorf("the identifier %q does not follow the convention", rule.ID)
			continue
		}
		if rule.Area != rule.ID[4:7] {
			t.Errorf("the area %q of %s does not match its identifier", rule.Area, rule.ID)
		}
		if rule.Version == "" {
			t.Errorf("the rule %s has no version", rule.ID)
		}
		if rule.Description == "" {
			t.Errorf("the rule %s has no description", rule.ID)
		}
		if rule.EvidenceKind != "diff_hunk" {
			t.Errorf("the evidence kind of %s is %q, want %q", rule.ID, rule.EvidenceKind, "diff_hunk")
		}
	}
}

// TestLimitationsArePublishedPerLanguage checks that the limitations of the
// rules name the language that they affect.
func TestLimitationsArePublishedPerLanguage(t *testing.T) {
	for _, rule := range rules.Rules() {
		if len(rule.Limitations) == 0 {
			t.Errorf("the rule %s does not publish its limitations", rule.ID)
		}
		for _, limitation := range rule.Limitations {
			if len(limitation) < 4 {
				t.Errorf("the limitation %q of %s is too short", limitation, rule.ID)
			}
		}
	}
}
