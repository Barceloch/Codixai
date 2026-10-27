package golang

import (
	"os"
	"path/filepath"
	"strings"
)

// Manifest holds the dependencies declared by the go.mod file of a
// repository.
//
// The file is read directly. No package manager is invoked and no module is
// downloaded.
type Manifest struct {
	// Found states whether a go.mod file was found in the repository root.
	Found bool
	// Module is the module path declared by the go.mod file.
	Module string
	// Requires are the module paths required by the go.mod file.
	Requires []string
}

// ReadManifest reads the go.mod file of the given repository root.
func ReadManifest(root string) Manifest {
	manifest := Manifest{}
	if root == "" {
		return manifest
	}
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return manifest
	}
	manifest.Found = true
	manifest.Module, manifest.Requires = parseGoMod(data)
	return manifest
}

// parseGoMod extracts the module path and the required modules of a go.mod
// file.
//
// Only the "module" and the "require" directives are read. The "replace",
// "exclude", "retract" and "toolchain" blocks are ignored.
func parseGoMod(data []byte) (string, []string) {
	module := ""
	var requires []string
	lines := strings.Split(string(data), "\n")
	block := ""
	for _, raw := range lines {
		line := raw
		if idx := strings.Index(line, "//"); idx >= 0 {
			line = line[:idx]
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if block != "" {
			if line == ")" {
				block = ""
				continue
			}
			if block == "require" {
				if name, ok := firstField(line); ok {
					requires = append(requires, name)
				}
			}
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		switch fields[0] {
		case "module":
			if len(fields) > 1 {
				module = fields[1]
			}
		case "require":
			if len(fields) == 1 || (len(fields) == 2 && fields[1] == "(") {
				block = "require"
				continue
			}
			requires = append(requires, fields[1])
		case "replace", "exclude", "retract", "toolchain", "godebug":
			if len(fields) == 1 {
				block = fields[0]
			}
		}
	}
	return module, requires
}

// firstField returns the first whitespace separated field of a line.
func firstField(line string) (string, bool) {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return "", false
	}
	return fields[0], true
}

// Declares reports whether the manifest declares the given import path.
//
// A required module covers every package below its path.
func (m Manifest) Declares(path string) bool {
	for _, required := range m.Requires {
		if path == required || strings.HasPrefix(path, required+"/") {
			return true
		}
	}
	return false
}

// IsLocal reports whether the import path belongs to the module of the
// repository.
func (m Manifest) IsLocal(path string) bool {
	if m.Module == "" {
		return false
	}
	return path == m.Module || strings.HasPrefix(path, m.Module+"/")
}
