package analysis_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"codixai/internal/analysis"
	"codixai/internal/domain"
	"codixai/internal/integrations/git"
)

// expectedCase is the expected result of a fixture of the corpus.
type expectedCase struct {
	// ExpectedRuleIDs are the identifiers of the rules that the case must
	// produce, in any order.
	ExpectedRuleIDs []string `json:"expected_rule_ids"`
	// ExpectedCount is the number of findings that the case must produce.
	ExpectedCount int `json:"expected_count"`
}

// fixtureCase is one case of the fixture corpus.
type fixtureCase struct {
	// Rule is the identifier of the rule that the case belongs to.
	Rule string
	// Language is the language of the case.
	Language string
	// Kind is "positive" or "negative".
	Kind string
	// Name is the name of the case.
	CaseName string
	// Directory is the directory of the case.
	Directory string
	// DiffPath is the path of the diff of the case.
	DiffPath string
	// ExpectedPath is the path of the expected result of the case.
	ExpectedPath string
}

// Name returns the identifier of the case for the test output.
func (c fixtureCase) Name() string {
	return c.Rule + "/" + c.Language + "/" + c.Kind + "/" + c.CaseName
}

// fixedClock returns a clock that always reports the same instant.
func fixedClock() func() time.Time {
	moment := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	return func() time.Time { return moment }
}

// fixtureRoot returns the path of the fixture corpus.
func fixtureRoot() string {
	return filepath.Join("..", "..", "testdata", "rules")
}

// TestFixtureCorpus runs the engine over every case of the fixture corpus
// and compares the findings with the expected result of the case.
func TestFixtureCorpus(t *testing.T) {
	cases := collectCases(t, fixtureRoot())
	if len(cases) == 0 {
		t.Fatal("the fixture corpus is empty")
	}
	for _, testCase := range cases {
		t.Run(testCase.Name(), func(t *testing.T) {
			runFixture(t, testCase)
		})
	}
}

// runFixture runs one case of the corpus.
func runFixture(t *testing.T, testCase fixtureCase) {
	t.Helper()
	expected := readExpected(t, testCase.ExpectedPath)
	file, err := os.Open(testCase.DiffPath)
	if err != nil {
		t.Fatalf("the diff cannot be opened: %v", err)
	}
	defer file.Close()
	changes, err := git.ParseUnifiedDiff(file)
	if err != nil {
		t.Fatalf("the diff cannot be parsed: %v", err)
	}
	root, err := filepath.Abs(testCase.Directory)
	if err != nil {
		t.Fatalf("the directory of the case cannot be resolved: %v", err)
	}
	// A case may bring its own repository, named after the case, or share
	// the repository of the directory that holds it.
	for _, candidate := range []string{testCase.CaseName + ".repo", "repo"} {
		info, err := os.Stat(filepath.Join(root, candidate))
		if err == nil && info.IsDir() {
			root = filepath.Join(root, candidate)
			break
		}
	}
	changes.Root = root
	changes.Origin = domain.OriginWorktree
	changes.Base = "base"
	changes.Head = "head"

	engine := analysis.New(analysis.WithClock(fixedClock()))
	run, err := engine.Analyze(context.Background(), changes)
	if err != nil {
		t.Fatalf("the analysis returned an error: %v", err)
	}

	produced := make([]string, 0, len(run.Findings))
	for _, finding := range run.Findings {
		produced = append(produced, finding.RuleID)
	}
	want := append([]string{}, expected.ExpectedRuleIDs...)
	sort.Strings(produced)
	sort.Strings(want)
	if strings.Join(produced, ",") != strings.Join(want, ",") {
		t.Errorf("the rules of the findings are %v, want %v", produced, want)
	}
	if len(run.Findings) != expected.ExpectedCount {
		t.Errorf("the number of findings is %d, want %d: %v", len(run.Findings), expected.ExpectedCount, describe(run.Findings))
	}
	for _, finding := range run.Findings {
		if finding.RuleVersion == "" {
			t.Errorf("the finding %q has no rule version", finding.Message)
		}
		if finding.EpistemicStatus == "" {
			t.Errorf("the finding %q has no epistemic status", finding.Message)
		}
		if finding.Support.Level == "" {
			t.Errorf("the finding %q has no support level", finding.Message)
		}
		if finding.Support.Level == domain.SupportResolved {
			t.Errorf("the finding %q claims a resolved support level", finding.Message)
		}
		if len(finding.Evidence) == 0 {
			t.Errorf("the finding %q has no evidence", finding.Message)
		}
		if finding.Location.Path == "" {
			t.Errorf("the finding %q has no path", finding.Message)
		}
		if finding.Recommendation == "" {
			t.Errorf("the finding %q has no recommendation", finding.Message)
		}
	}
}

// describe returns a readable description of a list of findings.
func describe(findings []domain.Finding) []string {
	var described []string
	for _, finding := range findings {
		described = append(described, finding.RuleID+"@"+finding.Location.Path+":"+
			itoa(finding.Location.StartLine)+" "+finding.Message)
	}
	return described
}

// itoa converts an integer into a string without importing strconv in every
// call site.
func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	digits := ""
	negative := value < 0
	if negative {
		value = -value
	}
	for value > 0 {
		digits = string(rune('0'+value%10)) + digits
		value /= 10
	}
	if negative {
		return "-" + digits
	}
	return digits
}

// readExpected reads the expected result of a case.
func readExpected(t *testing.T, path string) expectedCase {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("the expected result cannot be read: %v", err)
	}
	var expected expectedCase
	if err := json.Unmarshal(data, &expected); err != nil {
		t.Fatalf("the expected result is not valid JSON: %v", err)
	}
	return expected
}

// collectCases walks the fixture corpus and returns every case.
func collectCases(t *testing.T, root string) []fixtureCase {
	t.Helper()
	var cases []fixtureCase
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".diff") {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		parts := strings.Split(filepath.ToSlash(relative), "/")
		if len(parts) != 4 {
			t.Fatalf("the fixture %q does not follow the rule/language/case/name layout", relative)
		}
		name := strings.TrimSuffix(entry.Name(), ".diff")
		cases = append(cases, fixtureCase{
			Rule:         parts[0],
			Language:     parts[1],
			Kind:         parts[2],
			CaseName:     name,
			Directory:    filepath.Dir(path),
			DiffPath:     path,
			ExpectedPath: filepath.Join(filepath.Dir(path), name+".expected.json"),
		})
		return nil
	})
	if err != nil {
		t.Fatalf("the fixture corpus cannot be walked: %v", err)
	}
	sort.Slice(cases, func(i, j int) bool { return cases[i].Name() < cases[j].Name() })
	return cases
}

// TestSupportMatrixMatchesFixtures cross checks the support matrix that the
// engine publishes with the fixture corpus.
//
// A rule may not declare support for a language without positive and
// negative fixtures, and it may not have fixtures for a language that it
// declares as unsupported.
func TestSupportMatrixMatchesFixtures(t *testing.T) {
	cases := collectCases(t, fixtureRoot())
	fixtures := map[string]map[string]map[string]int{}
	for _, testCase := range cases {
		if fixtures[testCase.Rule] == nil {
			fixtures[testCase.Rule] = map[string]map[string]int{}
		}
		if fixtures[testCase.Rule][testCase.Language] == nil {
			fixtures[testCase.Rule][testCase.Language] = map[string]int{}
		}
		fixtures[testCase.Rule][testCase.Language][testCase.Kind]++
	}
	engine := analysis.New()
	for _, rule := range engine.Rules() {
		for language, level := range rule.Languages {
			kinds := fixtures[rule.ID][language]
			if level == domain.SupportNone {
				if len(kinds) > 0 {
					t.Errorf("the rule %s declares no support for %s but the corpus has %d cases for it", rule.ID, language, countCases(kinds))
				}
				continue
			}
			if kinds["positive"] == 0 {
				t.Errorf("the rule %s declares support for %s but the corpus has no positive case for it", rule.ID, language)
			}
			if kinds["negative"] == 0 {
				t.Errorf("the rule %s declares support for %s but the corpus has no negative case for it", rule.ID, language)
			}
		}
		for language := range fixtures[rule.ID] {
			level, ok := rule.Languages[language]
			if !ok {
				t.Errorf("the rule %s has fixtures for %s but does not declare a support level for it", rule.ID, language)
				continue
			}
			if level == domain.SupportNone {
				t.Errorf("the rule %s has fixtures for %s but declares no support for it", rule.ID, language)
			}
		}
	}
	for rule := range fixtures {
		found := false
		for _, published := range engine.Rules() {
			if published.ID == rule {
				found = true
			}
		}
		if !found {
			t.Errorf("the corpus has cases for the rule %s, which the catalog does not publish", rule)
		}
	}
}

// countCases returns the number of cases of a language.
func countCases(kinds map[string]int) int {
	total := 0
	for _, count := range kinds {
		total += count
	}
	return total
}

// TestRuleCatalogIsStable checks the identifiers, the versions and the
// severities of the catalog.
func TestRuleCatalogIsStable(t *testing.T) {
	engine := analysis.New()
	rules := engine.Rules()
	if len(rules) != 6 {
		t.Fatalf("the catalog publishes %d rules, want 6", len(rules))
	}
	want := map[string]domain.Severity{
		"CDX-IMP-001": domain.SeverityError,
		"CDX-REF-001": domain.SeverityError,
		"CDX-TST-001": domain.SeverityWarning,
		"CDX-SUP-001": domain.SeverityWarning,
		"CDX-ERR-001": domain.SeverityWarning,
		"CDX-PLH-001": domain.SeverityWarning,
	}
	for _, rule := range rules {
		severity, ok := want[rule.ID]
		if !ok {
			t.Errorf("the catalog publishes an unknown rule %q", rule.ID)
			continue
		}
		if rule.Severity != severity {
			t.Errorf("the severity of %s is %q, want %q", rule.ID, rule.Severity, severity)
		}
		if rule.Version == "" {
			t.Errorf("the rule %s does not publish a version", rule.ID)
		}
		if rule.Method == "" {
			t.Errorf("the rule %s does not publish a method", rule.ID)
		}
		if len(rule.Limitations) == 0 {
			t.Errorf("the rule %s does not publish its limitations", rule.ID)
		}
		for language, level := range rule.Languages {
			if level == domain.SupportResolved {
				t.Errorf("the rule %s claims a resolved support level for %s", rule.ID, language)
			}
		}
	}
}

// TestEngineSortsAndSummarizes checks the order of the findings and the
// summary of the run.
func TestEngineSortsAndSummarizes(t *testing.T) {
	changes := &domain.ChangeSet{
		Origin: domain.OriginBase,
		Base:   "base",
		Head:   "head",
		Files: []domain.FileChange{
			{
				Path:     "b/second.py",
				Status:   domain.FileModified,
				Language: domain.LangPython,
				Hunks: []domain.Hunk{
					{
						Index:    1,
						OldStart: 1,
						NewStart: 1,
						Lines: []domain.DiffLine{
							{Kind: domain.LineAdded, NewLine: 1, Content: "import httpx"},
						},
					},
				},
			},
			{
				Path:     "a/first.py",
				Status:   domain.FileModified,
				Language: domain.LangPython,
				Hunks: []domain.Hunk{
					{
						Index:    1,
						OldStart: 4,
						NewStart: 4,
						Lines: []domain.DiffLine{
							{Kind: domain.LineAdded, NewLine: 4, Content: "# TODO: implement"},
						},
					},
				},
			},
		},
	}
	engine := analysis.New(analysis.WithClock(fixedClock()))
	run, err := engine.Analyze(context.Background(), changes)
	if err != nil {
		t.Fatalf("the analysis returned an error: %v", err)
	}
	if len(run.Findings) < 2 {
		t.Fatalf("the analysis produced %d findings, want at least 2", len(run.Findings))
	}
	for i := 1; i < len(run.Findings); i++ {
		previous, current := run.Findings[i-1], run.Findings[i]
		if previous.Location.Path > current.Location.Path {
			t.Errorf("the findings are not sorted by path: %q before %q", previous.Location.Path, current.Location.Path)
		}
		if previous.Location.Path == current.Location.Path &&
			previous.Location.StartLine > current.Location.StartLine {
			t.Errorf("the findings are not sorted by line: %d before %d",
				previous.Location.StartLine, current.Location.StartLine)
		}
	}
	if run.Summary.Total != len(run.Findings) {
		t.Errorf("the total of the summary is %d, want %d", run.Summary.Total, len(run.Findings))
	}
	if run.Summary.ByLanguage[domain.LangPython] != run.Summary.Total {
		t.Errorf("the summary counts %d findings in python, want %d",
			run.Summary.ByLanguage[domain.LangPython], run.Summary.Total)
	}
	if run.Summary.ByRule["CDX-PLH-001"] == 0 {
		t.Errorf("the summary does not count the findings of CDX-PLH-001: %v", run.Summary.ByRule)
	}
	if run.Summary.ByEpistemicStatus[string(domain.EpistemicInferred)] == 0 {
		t.Errorf("the summary does not count the inferred findings: %v", run.Summary.ByEpistemicStatus)
	}
	if run.SchemaVersion != domain.SchemaVersion {
		t.Errorf("the schema version is %q, want %q", run.SchemaVersion, domain.SchemaVersion)
	}
	if run.GeneratedAt != "2026-01-02T03:04:05Z" {
		t.Errorf("the generated at is %q, want the fixed instant", run.GeneratedAt)
	}
	if run.Origin != string(domain.OriginBase) {
		t.Errorf("the origin is %q, want %q", run.Origin, domain.OriginBase)
	}
	if run.Revision.Base != "base" || run.Revision.Head != "head" {
		t.Errorf("the revision is %+v, want base and head", run.Revision)
	}
	if len(run.Rules) != 6 {
		t.Errorf("the run publishes %d rules, want 6", len(run.Rules))
	}
	if len(run.Limitations) == 0 {
		t.Error("the run does not publish its limitations")
	}
}

// TestEngineIgnoresUnknownLanguages checks that the files of an unknown
// language do not produce findings.
func TestEngineIgnoresUnknownLanguages(t *testing.T) {
	changes := &domain.ChangeSet{
		Files: []domain.FileChange{
			{
				Path:   "docs/notes.txt",
				Status: domain.FileModified,
				Hunks: []domain.Hunk{
					{
						Index: 1,
						Lines: []domain.DiffLine{
							{Kind: domain.LineAdded, NewLine: 1, Content: "TODO: import httpx"},
						},
					},
				},
			},
		},
	}
	engine := analysis.New(analysis.WithClock(fixedClock()))
	run, err := engine.Analyze(context.Background(), changes)
	if err != nil {
		t.Fatalf("the analysis returned an error: %v", err)
	}
	if len(run.Findings) != 0 {
		t.Errorf("the analysis produced %d findings for an unknown language, want 0", len(run.Findings))
	}
}

// TestEngineReportsADiagnosticWhenAFileCannotBeParsed checks that a file
// that cannot be parsed does not stop the analysis.
func TestEngineReportsADiagnosticWhenAFileCannotBeParsed(t *testing.T) {
	root := t.TempDir()
	broken := "package main\n\nfunc main() {\n\tthis is not go\n"
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte(broken), 0o600); err != nil {
		t.Fatalf("the file cannot be written: %v", err)
	}
	changes := &domain.ChangeSet{
		Root: root,
		Files: []domain.FileChange{
			{
				Path:   "main.go",
				Status: domain.FileModified,
				Hunks: []domain.Hunk{
					{
						Index: 1,
						Lines: []domain.DiffLine{
							{Kind: domain.LineAdded, NewLine: 4, Content: "\tthis is not go"},
						},
					},
				},
			},
		},
	}
	engine := analysis.New(analysis.WithClock(fixedClock()))
	run, err := engine.Analyze(context.Background(), changes)
	if err != nil {
		t.Fatalf("the analysis returned an error: %v", err)
	}
	if run.Summary.Total != 0 {
		t.Errorf("the analysis produced %d findings, want 0", run.Summary.Total)
	}
	if len(changes.Diagnostics) == 0 {
		t.Error("the analysis did not record a diagnostic for the file that cannot be parsed")
	}
}
