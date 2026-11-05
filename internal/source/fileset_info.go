// Package source provides file discovery and reading capabilities for full directory analysis.
package source

import (
	"github.com/barceloch/codixai/internal/domain"
)

// FileSetInfo contains information about files discovered in a directory.
type FileSetInfo struct {
	// Root is the absolute, symlink free path of the directory.
	Root string
	// Files is the list of analyzed files with their content.
	Files []domain.DirectoryFile
	// Diagnostics collects problems found during discovery.
	Diagnostics []string
	// Discovered is the number of non directory entries that were walked.
	Discovered int
	// Excluded is the number of files excluded by the configuration.
	Excluded int
	// ExcludedPaths is the list of paths excluded by the configuration.
	ExcludedPaths []string
	// Skipped is the number of files that were walked but not analyzed,
	// because the language is not supported, the file is too large, the file
	// is binary or it could not be read.
	Skipped int
}
