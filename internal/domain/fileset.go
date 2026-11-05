package domain

// DirectoryFile represents a single file discovered in a directory for full
// analysis. Unlike FileChange, which describes a change, DirectoryFile
// describes the current content of a file.
type DirectoryFile struct {
	// Path is the path relative to the directory root.
	Path string `json:"path"`
	// Language is the detected programming language (go, python, etc.).
	Language string `json:"language,omitempty"`
	// Content is the full text content of the file.
	Content string `json:"content"`
	// Hash is an optional hash of the content for deduplication.
	Hash string `json:"hash,omitempty"`
	// Size is the size in bytes of the file content.
	Size int64 `json:"size"`
	// IsBinary indicates if the file is a binary file.
	IsBinary bool `json:"is_binary"`
}
