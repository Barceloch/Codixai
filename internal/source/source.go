package source

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/barceloch/codixai/internal/domain"
	"github.com/barceloch/codixai/internal/languages/registry"
	"github.com/barceloch/codixai/internal/policies"
)

// Default limits for directory analysis.
const (
	// MaxFileSize is the largest file that directory analysis reads.
	MaxFileSize = 10 * 1024 * 1024 // 10 MB
	// MaxFileCount is the largest number of files that directory analysis
	// analyzes in one run.
	MaxFileCount = 10000
	// binaryProbeSize is the number of leading bytes inspected to decide
	// whether a file is binary.
	binaryProbeSize = 8000
)

// DirectorySource implements safe file discovery and reading from a directory.
//
// Every path it reads is resolved and checked to belong to the analyzed root:
// a symbolic link that points outside the root is never followed.
type DirectorySource struct {
	root   string
	config *policies.Config
}

// NewDirectorySource creates a new DirectorySource for the given root.
//
// The root is resolved with its symbolic links, so a root that is itself a
// link is analyzed and every containment check compares canonical paths.
func NewDirectorySource(root string, config *policies.Config) (*DirectorySource, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve directory root %q: %w", root, err)
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve the directory root %q: %w", root, err)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return nil, fmt.Errorf("cannot open directory root %q: %w", root, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("root %q is not a directory", root)
	}
	if config == nil {
		config = policies.Defaults()
	}
	return &DirectorySource{root: resolved, config: config}, nil
}

// Root returns the absolute, symlink free path of the directory.
func (s *DirectorySource) Root() string { return s.root }

// Files discovers and reads files in the directory according to policies and limits.
//
// The counters have a stable meaning:
//   - Discovered counts every non directory entry that was walked, whether it
//     was analyzed, excluded or skipped.
//   - Excluded counts the files excluded by the configuration.
//   - Skipped counts the files that were walked but not analyzed, because the
//     language is not supported, the file is too large, the file is binary or
//     it could not be read.
func (s *DirectorySource) Files(ctx context.Context) (*FileSetInfo, error) {
	fileSet := &FileSetInfo{Root: s.root}

	err := filepath.WalkDir(s.root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			fileSet.Diagnostics = append(fileSet.Diagnostics, fmt.Sprintf("cannot access path: %v", walkErr))
			return nil
		}

		if ctx.Err() != nil {
			return ctx.Err()
		}

		if d.IsDir() {
			rel, err := filepath.Rel(s.root, path)
			if err != nil {
				return nil
			}
			if rel != "." && s.config.Excluded(rel) {
				fileSet.ExcludedPaths = append(fileSet.ExcludedPaths, rel)
				return filepath.SkipDir
			}
			return nil
		}

		fileSet.Discovered++

		if len(fileSet.Files) >= MaxFileCount {
			fileSet.Diagnostics = append(fileSet.Diagnostics, fmt.Sprintf("file limit reached (%d files)", MaxFileCount))
			return filepath.SkipAll
		}

		rel, err := filepath.Rel(s.root, path)
		if err != nil {
			fileSet.Skipped++
			return nil
		}

		if s.config.Excluded(rel) {
			fileSet.Excluded++
			fileSet.ExcludedPaths = append(fileSet.ExcludedPaths, rel)
			return nil
		}

		language, ok := registry.Detect(rel)
		if !ok {
			// The file has no language that a rule supports: it is part of the
			// directory but it is not analyzed.
			fileSet.Skipped++
			return nil
		}

		content, size, err := s.readFile(path)
		if err != nil {
			fileSet.Skipped++
			fileSet.Diagnostics = append(fileSet.Diagnostics, fmt.Sprintf("cannot read file %s: %v", rel, err))
			return nil
		}
		if size > MaxFileSize {
			fileSet.Skipped++
			fileSet.Diagnostics = append(fileSet.Diagnostics, fmt.Sprintf("file too large (%d bytes), max is %d", size, MaxFileSize))
			return nil
		}

		binary := isBinary(content)
		if binary {
			fileSet.Skipped++
			return nil
		}

		fileSet.Files = append(fileSet.Files, domain.DirectoryFile{
			Path:     rel,
			Content:  content,
			Language: language,
			Size:     size,
			Hash:     computeHash(content),
			IsBinary: false,
		})

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking directory: %w", err)
	}

	return fileSet, nil
}

// readFile reads a file only when its resolved path belongs to the root.
//
// The path is resolved with its symbolic links before it is compared with the
// root: filepath.Clean and filepath.Abs normalize the text of a path, but they
// do not resolve a link, so they cannot guarantee the containment by
// themselves. The comparison uses filepath.Rel, so a sibling directory whose
// name shares a prefix with the root (for example /tmp/project and
// /tmp/project-external) is never mistaken for the root.
//
// The size is the size of the resolved file. A file larger than MaxFileSize is
// not read: the caller receives its size with empty content, so the limit is
// enforced before the bytes reach memory.
func (s *DirectorySource) readFile(path string) (string, int64, error) {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", 0, err
	}
	if !withinRoot(s.root, resolved) {
		return "", 0, fmt.Errorf("access denied: the path resolves outside the analyzed root")
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", 0, err
	}
	if !info.Mode().IsRegular() {
		return "", 0, fmt.Errorf("access denied: the path is not a regular file")
	}
	if info.Size() > MaxFileSize {
		return "", info.Size(), nil
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		return "", 0, err
	}
	return string(data), info.Size(), nil
}

// withinRoot reports whether target is root or lives below it.
func withinRoot(root, target string) bool {
	relative, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	if relative == "." {
		return true
	}
	if filepath.IsAbs(relative) {
		return false
	}
	return relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

// isBinary reports whether the content looks like a binary file.
func isBinary(content string) bool {
	probe := content
	if len(probe) > binaryProbeSize {
		probe = probe[:binaryProbeSize]
	}
	return bytes.IndexByte([]byte(probe), 0) >= 0
}

// computeHash returns a stable short hash of the content of a file.
func computeHash(content string) string {
	sum := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", sum)[:16]
}
