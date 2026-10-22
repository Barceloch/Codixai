package baseline

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/barceloch/codixai/internal/domain"
)

// finding builds a finding of a rule on a line of a file.
func finding(ruleID, path string, line int) domain.Finding {
	return domain.Finding{
		RuleID:          ruleID,
		RuleVersion:     "1",
		Severity:        domain.SeverityWarning,
		EpistemicStatus: domain.EpistemicObserved,
		Message:         "a finding",
		Location:        domain.Location{Path: path, StartLine: line, EndLine: line},
	}
}

// changeset builds a change set with one file and one hunk.
func changeset(path string, lines ...domain.DiffLine) *domain.ChangeSet {
	return &domain.ChangeSet{
		Files: []domain.FileChange{{
			Path:   path,
			Status: domain.FileModified,
			Hunks:  []domain.Hunk{{Index: 0, Lines: lines}},
		}},
	}
}

// added builds an added line of a hunk.
func added(line int, content string) domain.DiffLine {
	return domain.DiffLine{Kind: domain.LineAdded, NewLine: line, Content: content}
}

func TestFingerprintIsStableAndHexadecimal(t *testing.T) {
	first := Fingerprint("CDX-SUP-001", "app.py", "return x")
	second := Fingerprint("CDX-SUP-001", "app.py", "return x")
	if first != second {
		t.Fatalf("the fingerprint is not stable across runs: %q and %q", first, second)
	}
	if len(first) != 16 {
		t.Errorf("the fingerprint is %d characters long, want 16: %q", len(first), first)
	}
	for _, character := range first {
		if !strings.ContainsRune("0123456789abcdef", character) {
			t.Errorf("the fingerprint %q holds the non hexadecimal character %q", first, character)
		}
	}
	if Fingerprint("CDX-SUP-001", "app.py", "return y") == first {
		t.Error("the fingerprint does not change when the context changes")
	}
	if Fingerprint("CDX-PLH-001", "app.py", "return x") == first {
		t.Error("the fingerprint does not change when the rule changes")
	}
	if Fingerprint("CDX-SUP-001", "other.py", "return x") == first {
		t.Error("the fingerprint does not change when the path changes")
	}
}

func TestFingerprintIgnoresTheLineNumber(t *testing.T) {
	// The same line, moved from line 3 to line 9 by the code above it.
	before := changeset("app.py", added(3, "    return value  # noqa"))
	after := changeset("app.py", added(9, "    return value  # noqa"))

	beforeFinding := finding("CDX-SUP-001", "app.py", 3)
	afterFinding := finding("CDX-SUP-001", "app.py", 9)

	beforeFingerprint := Fingerprint(beforeFinding.RuleID, beforeFinding.Location.Path, Context(before, beforeFinding))
	afterFingerprint := Fingerprint(afterFinding.RuleID, afterFinding.Location.Path, Context(after, afterFinding))

	if beforeFingerprint != afterFingerprint {
		t.Errorf("moving the line changed the fingerprint: %q then %q", beforeFingerprint, afterFingerprint)
	}
}

func TestFingerprintChangesWhenTheLineChanges(t *testing.T) {
	before := changeset("app.py", added(3, "    return value  # noqa"))
	after := changeset("app.py", added(3, "    return other  # noqa"))

	beforeFinding := finding("CDX-SUP-001", "app.py", 3)
	afterFinding := finding("CDX-SUP-001", "app.py", 3)

	beforeFingerprint := Fingerprint(beforeFinding.RuleID, beforeFinding.Location.Path, Context(before, beforeFinding))
	afterFingerprint := Fingerprint(afterFinding.RuleID, afterFinding.Location.Path, Context(after, afterFinding))

	if beforeFingerprint == afterFingerprint {
		t.Errorf("changing the line kept the fingerprint %q", beforeFingerprint)
	}
}

func TestFingerprintWithoutContextIdentifiesTheRuleAndThePath(t *testing.T) {
	changes := changeset("app.py", added(3, "    return value"))
	missing := finding("CDX-SUP-001", "app.py", 40)

	if context := Context(changes, missing); context != "" {
		t.Fatalf("the context of a line outside of the diff is %q, want the empty string", context)
	}
	fingerprint := Fingerprint(missing.RuleID, missing.Location.Path, Context(changes, missing))
	want := Fingerprint("CDX-SUP-001", "app.py", "")
	if fingerprint != want {
		t.Errorf("the fingerprint without a context is %q, want %q", fingerprint, want)
	}
}

func TestContextNormalizesTheWhitespace(t *testing.T) {
	changes := changeset("app.py", added(3, "\t  return \t value  \t # noqa   "))
	got := Context(changes, finding("CDX-SUP-001", "app.py", 3))
	want := "return value # noqa"
	if got != want {
		t.Errorf("the normalized context is %q, want %q", got, want)
	}
}

func TestContextReadsTheOldSideOfARemovedLine(t *testing.T) {
	// A deleted line only exists on the old side of the diff.
	changes := changeset("app.py",
		domain.DiffLine{Kind: domain.LineContext, OldLine: 6, NewLine: 6, Content: "def test_value():"},
		domain.DiffLine{Kind: domain.LineRemoved, OldLine: 7, Content: "    assert value() == 1  # noqa"},
	)
	removed := finding("CDX-SUP-001", "app.py", 7)
	if got := Context(changes, removed); got != "assert value() == 1 # noqa" {
		t.Errorf("the context of the removed line is %q, want %q", got, "assert value() == 1 # noqa")
	}
}

func TestContextPrefersTheNewSideOfAModifiedLine(t *testing.T) {
	// The line 7 is replaced in place: the new side is the state under
	// review, so it is the text the finding points at.
	changes := changeset("app.py",
		domain.DiffLine{Kind: domain.LineRemoved, OldLine: 7, Content: "    return old  # noqa"},
		domain.DiffLine{Kind: domain.LineAdded, NewLine: 7, Content: "    return new  # noqa"},
	)
	got := Context(changes, finding("CDX-SUP-001", "app.py", 7))
	if got != "return new # noqa" {
		t.Errorf("the context of the modified line is %q, want %q", got, "return new # noqa")
	}
}

func TestFingerprintChangesWhenTheAddedLineChangesBesideARemovedLine(t *testing.T) {
	// A removed line with the same number must not shadow the added line:
	// the fingerprint has to change when the added line changes.
	before := changeset("app.py",
		domain.DiffLine{Kind: domain.LineRemoved, OldLine: 7, Content: "    return old  # noqa"},
		domain.DiffLine{Kind: domain.LineAdded, NewLine: 7, Content: "    return new  # noqa"},
	)
	after := changeset("app.py",
		domain.DiffLine{Kind: domain.LineRemoved, OldLine: 7, Content: "    return old  # noqa"},
		domain.DiffLine{Kind: domain.LineAdded, NewLine: 7, Content: "    return changed  # noqa"},
	)
	beforeFingerprint := Fingerprint("CDX-SUP-001", "app.py", Context(before, finding("CDX-SUP-001", "app.py", 7)))
	afterFingerprint := Fingerprint("CDX-SUP-001", "app.py", Context(after, finding("CDX-SUP-001", "app.py", 7)))
	if beforeFingerprint == afterFingerprint {
		t.Errorf("the fingerprint ignored the change of the added line: %q", beforeFingerprint)
	}
}

func TestContextIgnoresAnotherFile(t *testing.T) {
	changes := changeset("app.py", added(3, "    return value"))
	if got := Context(changes, finding("CDX-SUP-001", "other.py", 3)); got != "" {
		t.Errorf("the context of a finding of another file is %q, want the empty string", got)
	}
	if got := Context(nil, finding("CDX-SUP-001", "app.py", 3)); got != "" {
		t.Errorf("the context of a nil change set is %q, want the empty string", got)
	}
}

func TestEntriesAreSortedAndUnique(t *testing.T) {
	changes := changeset("app.py",
		added(3, "    return a"),
		added(4, "    return b"),
	)
	findings := []domain.Finding{
		finding("CDX-PLH-001", "app.py", 4),
		finding("CDX-SUP-001", "app.py", 3),
		finding("CDX-SUP-001", "app.py", 3),
	}
	entries := Entries(findings, changes)
	if len(entries) != 2 {
		t.Fatalf("the entries are %d, want 2: %+v", len(entries), entries)
	}
	if entries[0].Fingerprint > entries[1].Fingerprint {
		t.Errorf("the entries are not sorted by their fingerprint: %q then %q",
			entries[0].Fingerprint, entries[1].Fingerprint)
	}
	for _, entry := range entries {
		if entry.RuleID == "" {
			t.Errorf("an entry has no rule identifier: %+v", entry)
		}
		if entry.Path != "app.py" {
			t.Errorf("an entry has the path %q, want app.py", entry.Path)
		}
		if entry.Line <= 0 {
			t.Errorf("an entry has no line: %+v", entry)
		}
		if entry.Severity != string(domain.SeverityWarning) {
			t.Errorf("an entry has the severity %q, want warning", entry.Severity)
		}
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, ".codixai", "baseline.json")
	file := &File{
		SchemaVersion: SchemaVersion,
		CreatedAt:     "2026-10-22T10:00:00Z",
		ToolVersion:   "0.1.0",
		Findings: []Entry{
			{Fingerprint: "bbbbbbbbbbbbbbbb", RuleID: "CDX-PLH-001", Path: "b.py", Line: 2, Severity: "warning", Message: "second"},
			{Fingerprint: "aaaaaaaaaaaaaaaa", RuleID: "CDX-SUP-001", Path: "a.py", Line: 1, Severity: "error", Message: "first"},
		},
	}
	if err := Save(path, file); err != nil {
		t.Fatalf("cannot save the baseline: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read the baseline: %v", err)
	}
	if !strings.HasSuffix(string(data), "\n") {
		t.Error("the baseline does not end with a newline")
	}
	if !json.Valid(data) {
		t.Fatalf("the baseline is not a JSON document:\n%s", data)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("cannot load the baseline: %v", err)
	}
	if loaded.SchemaVersion != SchemaVersion {
		t.Errorf("the loaded schema version is %q, want %q", loaded.SchemaVersion, SchemaVersion)
	}
	if loaded.CreatedAt != "2026-10-22T10:00:00Z" || loaded.ToolVersion != "0.1.0" {
		t.Errorf("the loaded metadata is %+v", loaded)
	}
	if len(loaded.Findings) != 2 {
		t.Fatalf("the loaded baseline holds %d findings, want 2", len(loaded.Findings))
	}
	if loaded.Findings[0].Fingerprint != "aaaaaaaaaaaaaaaa" {
		t.Errorf("the saved entries are not sorted by their fingerprint: %+v", loaded.Findings)
	}
	if !loaded.Contains("bbbbbbbbbbbbbbbb") {
		t.Error("the loaded baseline does not contain a fingerprint that it holds")
	}
	if loaded.Contains("cccccccccccccccc") || loaded.Contains("") {
		t.Error("the baseline contains a fingerprint that it does not hold")
	}
}

func TestLoadAcceptsAMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "absent.json")
	file, err := Load(path)
	if err != nil {
		t.Fatalf("a missing baseline is not an error, got: %v", err)
	}
	if file == nil || len(file.Findings) != 0 {
		t.Fatalf("the baseline of a missing file is %+v, want an empty baseline", file)
	}
	if file.SchemaVersion != SchemaVersion {
		t.Errorf("the empty baseline has the schema version %q, want %q", file.SchemaVersion, SchemaVersion)
	}
}

func TestLoadRejectsAMalformedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "baseline.json")
	content := "{\n  \"schema_version\": \"0.1\",\n  \"findings\": [\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("cannot write the baseline: %v", err)
	}
	_, err := Load(path)
	if err == nil {
		t.Fatal("the malformed baseline was accepted")
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("the error does not name the file: %v", err)
	}
	if !strings.Contains(err.Error(), "line") {
		t.Errorf("the error does not name the line: %v", err)
	}
}

func TestLoadRejectsAnotherSchemaVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "baseline.json")
	content := "{\"schema_version\": \"0.2\", \"findings\": []}\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("cannot write the baseline: %v", err)
	}
	_, err := Load(path)
	if err == nil {
		t.Fatal("the baseline of an unsupported version was accepted")
	}
	for _, want := range []string{path, "0.2", SchemaVersion} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the error does not mention %q: %v", want, err)
		}
	}
}

func TestLoadRejectsAnEntryWithoutIdentity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "baseline.json")
	content := "{\"schema_version\": \"0.1\", \"findings\": [{\"rule_id\": \"CDX-SUP-001\"}]}\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("cannot write the baseline: %v", err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("an entry without a fingerprint was accepted")
	}
}

func TestSaveRejectsAnotherSchemaVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "baseline.json")
	file := &File{SchemaVersion: "0.2"}
	if err := Save(path, file); err == nil {
		t.Fatal("a baseline of an unsupported version was written")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("the rejected baseline was written")
	}
}

func TestKnowsRequiresAnIdentifiableContext(t *testing.T) {
	changes := changeset("app.py", added(3, "    return value  # noqa"))
	recorded := finding("CDX-SUP-001", "app.py", 3)
	file := &File{
		SchemaVersion: SchemaVersion,
		Findings:      Entries([]domain.Finding{recorded}, changes),
	}

	if !file.Knows(changes, recorded) {
		t.Error("the baseline does not know the finding it recorded")
	}

	// The same rule and path, but the line is not in the diff: the
	// baseline must not hide a finding it cannot identify.
	unidentifiable := finding("CDX-SUP-001", "app.py", 40)
	if file.Knows(changes, unidentifiable) {
		t.Error("the baseline hides a finding without a context")
	}

	// The same line with another content: a different finding.
	movedChanges := changeset("app.py", added(3, "    return other  # noqa"))
	if file.Knows(movedChanges, finding("CDX-SUP-001", "app.py", 3)) {
		t.Error("the baseline hides a finding whose code changed")
	}
}

func TestResolveKeepsTheBaselineInsideTheRepository(t *testing.T) {
	root := t.TempDir()
	inside, err := Resolve(root, ".codixai/baseline.json")
	if err != nil {
		t.Fatalf("cannot resolve a path inside the repository: %v", err)
	}
	if inside != filepath.Join(root, ".codixai", "baseline.json") {
		t.Errorf("the resolved path is %q", inside)
	}
	if _, err := Resolve(root, "../baseline.json"); err == nil {
		t.Error("a path outside of the repository was accepted")
	}
	if _, err := Resolve(root, " "); err == nil {
		t.Error("an empty path was accepted")
	}
}
