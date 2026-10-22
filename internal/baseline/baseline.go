// Package baseline implements the baseline of findings of a repository.
//
// A baseline is a versioned document that lives in the analyzed repository
// and that records the findings the repository already has. It lets a
// pipeline fail on the new findings without turning the existing debt into
// constant noise.
//
// The identity of a finding is its fingerprint: the first 16 hexadecimal
// characters of the SHA-256 of the rule identifier, the path of the file and
// a normalized context, joined by a null byte. The line number is never part
// of the identity, so a baseline survives the movement of the code, which is
// the normal case in a repository with activity.
//
// A finding whose line is not part of the compared range still has a
// fingerprint: the rule and the path identify it, so the baseline recognizes
// it while they do not change. The identity is coarser than the one of a
// finding with context, and it is reviewable in the baseline file.
//
// The format of the file and the behavior of the audit are fixed by the
// decision 0008, in docs/decisions/0008-baseline-and-fingerprint.md.
package baseline

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/barceloch/codixai/internal/domain"
)

const (
	// SchemaVersion is the version of the format of the baseline file.
	//
	// It is the version of the document of the analyzed repository, not the
	// version of the output of the audit: the audit keeps its own versioned
	// contract.
	SchemaVersion = "0.1"

	// fingerprintLength is the number of hexadecimal characters of a
	// fingerprint.
	fingerprintLength = 16

	// separator joins the values of a fingerprint. It cannot appear in a
	// path nor in the text of a line of code.
	separator = "\x00"
)

// Entry is one recorded finding of a baseline.
//
// The fingerprint is the identity of the finding. The line, the severity
// and the message are informative: they change when the code moves and they
// are never part of the identity.
type Entry struct {
	// Fingerprint is the identity of the finding.
	Fingerprint string `json:"fingerprint"`
	// RuleID is the identifier of the rule that produced the finding.
	RuleID string `json:"rule_id"`
	// Path is the repository relative path of the file of the finding.
	Path string `json:"path"`
	// Line is the line of the finding when it is known.
	Line int `json:"line"`
	// Severity is the severity of the finding when it is known.
	Severity string `json:"severity"`
	// Message is the message of the finding when it is known.
	Message string `json:"message"`
}

// File is the versioned document of a baseline.
//
// The file is written in the analyzed repository and it is committed with it.
type File struct {
	// SchemaVersion is the version of the format of the document.
	SchemaVersion string `json:"schema_version"`
	// CreatedAt is the instant of the creation of the document.
	CreatedAt string `json:"created_at"`
	// ToolVersion is the version of the tool that wrote the document.
	ToolVersion string `json:"tool_version"`
	// Findings holds the recorded findings, sorted by their fingerprint.
	Findings []Entry `json:"findings"`
}

// New returns an empty baseline of the supported version.
//
// An empty baseline is the baseline of a repository that has no known
// finding. It is not an error: a repository without a baseline has no known
// findings.
func New() *File {
	return &File{
		SchemaVersion: SchemaVersion,
		Findings:      []Entry{},
	}
}

// Fingerprint returns the identity of a finding.
//
// The identity is the first 16 hexadecimal characters of the SHA-256 of the
// rule identifier, the path and the normalized context, joined by a null
// byte. The line number is not part of the identity.
func Fingerprint(ruleID, path, context string) string {
	sum := sha256.Sum256([]byte(ruleID + separator + path + separator + context))
	return hex.EncodeToString(sum[:])[:fingerprintLength]
}

// Context returns the normalized text of the line a finding points at.
//
// The text is taken from the diff of the compared range, so it describes
// the code that the finding observed. The normalization removes the leading
// and the trailing spaces and collapses every run of internal whitespace
// into a single space: the fingerprint then survives a reindentation and
// the movement of the code around the finding.
//
// A finding carries the number of the line but not the side of the diff it
// belongs to. The new side is the state of the file under review, so it is
// read first; the old side is read only when the new side holds no line with
// that number, which is the case of a removed line, for example the line of
// a deleted test. When both sides hold a line with the same number, the new
// side is the one that describes the finding.
//
// The context is empty when the line of the finding is not part of the diff.
// The caller must never hide a finding it cannot identify.
func Context(changes *domain.ChangeSet, finding domain.Finding) string {
	if changes == nil {
		return ""
	}
	path := finding.Location.Path
	line := finding.Location.StartLine
	if path == "" || line <= 0 {
		return ""
	}
	for _, file := range changes.Files {
		if file.Path != path {
			continue
		}
		if content, ok := contentAt(file, line, false); ok {
			return normalize(content)
		}
		if content, ok := contentAt(file, line, true); ok {
			return normalize(content)
		}
		return ""
	}
	return ""
}

// contentAt returns the content of the line with a number on one side of the
// diff.
//
// The new side holds the context lines, which exist on both sides, and the
// added lines. The old side holds only the removed lines.
func contentAt(file domain.FileChange, number int, oldSide bool) (string, bool) {
	for _, hunk := range file.Hunks {
		for _, line := range hunk.Lines {
			if oldSide {
				if line.Kind == domain.LineRemoved && line.OldLine == number {
					return line.Content, true
				}
				continue
			}
			if line.Kind != domain.LineRemoved && line.NewLine == number {
				return line.Content, true
			}
		}
	}
	return "", false
}

// normalize removes the leading and the trailing spaces of a text and
// collapses every run of internal whitespace into a single space.
func normalize(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

// Entries returns the entries of a set of findings.
//
// The entries are sorted by their fingerprint, so the document of a baseline
// is stable and its diffs are readable. Two findings with the same
// fingerprint are the same finding for the baseline, so they are recorded
// once.
func Entries(findings []domain.Finding, changes *domain.ChangeSet) []Entry {
	entries := make([]Entry, 0, len(findings))
	seen := make(map[string]bool, len(findings))
	for _, finding := range findings {
		context := Context(changes, finding)
		fingerprint := Fingerprint(finding.RuleID, finding.Location.Path, context)
		if seen[fingerprint] {
			continue
		}
		seen[fingerprint] = true
		entries = append(entries, Entry{
			Fingerprint: fingerprint,
			RuleID:      finding.RuleID,
			Path:        finding.Location.Path,
			Line:        finding.Location.StartLine,
			Severity:    string(finding.Severity),
			Message:     finding.Message,
		})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Fingerprint < entries[j].Fingerprint })
	return entries
}

// Contains reports whether the baseline records a fingerprint.
func (f *File) Contains(fingerprint string) bool {
	if f == nil || fingerprint == "" {
		return false
	}
	for _, entry := range f.Findings {
		if entry.Fingerprint == fingerprint {
			return true
		}
	}
	return false
}

// Knows reports whether the baseline already knows a finding.
//
// The identity of a finding is its fingerprint: the rule, the path and the
// normalized context of the line it points at. When the line is not part of
// the diff, the context is empty and the fingerprint holds the rule and the
// path only, which is still a stable and reviewable identity: the baseline
// recognizes the finding as long as the rule and the path do not change.
func (f *File) Knows(changes *domain.ChangeSet, finding domain.Finding) bool {
	fingerprint := Fingerprint(finding.RuleID, finding.Location.Path, Context(changes, finding))
	return f.Contains(fingerprint)
}

// Load reads the baseline of a repository.
//
// A missing file is an empty baseline and not an error: a repository that
// has no baseline has no known finding. A malformed file is an error whose
// message names the file and the line of the problem.
func Load(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return New(), nil
		}
		return nil, fmt.Errorf("cannot read the baseline %s: %w", path, err)
	}
	file := &File{}
	if err := json.Unmarshal(data, file); err != nil {
		hint := "fix the document or recreate it with 'codixai baseline create --force'"
		if line := lineOf(data, err); line > 0 {
			return nil, fmt.Errorf("%s: line %d: the baseline is not a valid JSON document, %s: %w", path, line, hint, err)
		}
		return nil, fmt.Errorf("%s: the baseline is not a valid JSON document, %s: %w", path, hint, err)
	}
	if file.SchemaVersion == "" {
		return nil, fmt.Errorf("%s: the baseline does not declare a schema_version, the supported version is %q: "+
			"recreate the baseline with 'codixai baseline create --force'", path, SchemaVersion)
	}
	if file.SchemaVersion != SchemaVersion {
		return nil, fmt.Errorf("%s: unsupported baseline schema version %q, the supported version is %q: "+
			"recreate the baseline with 'codixai baseline create --force'", path, file.SchemaVersion, SchemaVersion)
	}
	for index, entry := range file.Findings {
		if entry.Fingerprint == "" {
			return nil, fmt.Errorf("%s: the finding %d has no fingerprint, every entry of a baseline needs the "+
				"identity of its finding: recreate the baseline with 'codixai baseline create --force'", path, index)
		}
		if entry.RuleID == "" {
			return nil, fmt.Errorf("%s: the finding %d has no rule identifier, every entry of a baseline needs "+
				"the rule that produced it: recreate the baseline with 'codixai baseline create --force'", path, index)
		}
	}
	if file.Findings == nil {
		file.Findings = []Entry{}
	}
	return file, nil
}

// Save writes the baseline of a repository.
//
// The directory of the path is created when it is needed. The entries are
// sorted by their fingerprint and the document ends with a newline, so the
// file is stable and its diffs are readable.
func Save(path string, file *File) error {
	if file == nil {
		return errors.New("the baseline is required")
	}
	if file.SchemaVersion == "" {
		file.SchemaVersion = SchemaVersion
	}
	if file.SchemaVersion != SchemaVersion {
		return fmt.Errorf("cannot write the baseline %s: unsupported schema version %q, the supported version is %q", path, file.SchemaVersion, SchemaVersion)
	}
	document := *file
	document.Findings = sortedEntries(file.Findings)

	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return fmt.Errorf("cannot encode the baseline %s: %w", path, err)
	}
	data = append(data, '\n')

	directory := filepath.Dir(path)
	if directory != "" && directory != "." {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			return fmt.Errorf("cannot create the directory of the baseline %s: %w", path, err)
		}
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("cannot write the baseline %s: %w", path, err)
	}
	return nil
}

// sortedEntries returns the entries of a baseline in a stable order.
func sortedEntries(entries []Entry) []Entry {
	sorted := make([]Entry, len(entries))
	copy(sorted, entries)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Fingerprint < sorted[j].Fingerprint })
	return sorted
}

// Resolve returns the absolute path of a baseline inside a repository.
//
// A relative path is resolved against the root of the repository. A path
// that escapes the root is rejected: the access to the files of the
// analyzed repository is bounded to its root.
func Resolve(root, path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", errors.New("the path of the baseline is empty")
	}
	if strings.TrimSpace(root) == "" {
		return "", errors.New("the root of the repository is required to resolve the path of the baseline")
	}
	absolute := trimmed
	if !filepath.IsAbs(absolute) {
		absolute = filepath.Join(root, absolute)
	}
	absolute = filepath.Clean(absolute)
	cleanRoot := filepath.Clean(root)
	relative, err := filepath.Rel(cleanRoot, absolute)
	if err != nil {
		return "", fmt.Errorf("cannot resolve the path of the baseline %q inside the repository %q: %w", path, root, err)
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("the path of the baseline %q is outside of the repository %q: it must be a path inside the analyzed repository", path, root)
	}
	return absolute, nil
}

// lineOf returns the line of a JSON document where a decoding error
// happened.
//
// The line is 0 when the error does not carry a position.
func lineOf(data []byte, err error) int {
	offset := int64(-1)
	var syntax *json.SyntaxError
	if errors.As(err, &syntax) {
		offset = syntax.Offset
	}
	var typed *json.UnmarshalTypeError
	if errors.As(err, &typed) {
		offset = typed.Offset
	}
	if offset <= 0 {
		return 0
	}
	line := 1
	for index := int64(0); index < offset-1 && index < int64(len(data)); index++ {
		if data[index] == '\n' {
			line++
		}
	}
	return line
}
