package domain

// ChangeOrigin identifies where the analyzed changes come from.
type ChangeOrigin string

const (
	// OriginBase analyzes commits between a base ref and HEAD.
	OriginBase ChangeOrigin = "base"
	// OriginStaged analyzes the staging area.
	OriginStaged ChangeOrigin = "staged"
	// OriginWorktree analyzes unstaged working tree changes.
	OriginWorktree ChangeOrigin = "worktree"
)

// FileChange status values.
const (
	FileAdded    = "added"
	FileModified = "modified"
	FileDeleted  = "deleted"
	FileRenamed  = "renamed"
)

// DiffLine kinds.
const (
	LineContext = "context"
	LineAdded   = "added"
	LineRemoved = "removed"
)

// DiffLine is a single line inside a hunk.
type DiffLine struct {
	Kind    string `json:"kind"`
	OldLine int    `json:"old_line,omitempty"`
	NewLine int    `json:"new_line,omitempty"`
	Content string `json:"content"`
}

// Hunk is a contiguous block of changes inside a file diff.
type Hunk struct {
	Index    int        `json:"index"`
	OldStart int        `json:"old_start"`
	OldLines int        `json:"old_lines"`
	NewStart int        `json:"new_start"`
	NewLines int        `json:"new_lines"`
	Header   string     `json:"header,omitempty"`
	Lines    []DiffLine `json:"lines"`
}

// FileChange describes the change applied to one file.
type FileChange struct {
	Path     string `json:"path"`
	OldPath  string `json:"old_path,omitempty"`
	Status   string `json:"status"`
	Language string `json:"language,omitempty"`
	IsBinary bool   `json:"is_binary,omitempty"`
	Hunks    []Hunk `json:"hunks,omitempty"`
}

// ChangeSet is the normalized set of changes under analysis.
type ChangeSet struct {
	// Root is the absolute path of the analyzed repository. It is used to read
	// manifests (go.mod, package.json, ...) without executing repository code.
	Root        string       `json:"root,omitempty"`
	Origin      ChangeOrigin `json:"origin"`
	Base        string       `json:"base,omitempty"`
	Head        string       `json:"head,omitempty"`
	Files       []FileChange `json:"files"`
	Diagnostics []Diagnostic `json:"diagnostics,omitempty"`
}
