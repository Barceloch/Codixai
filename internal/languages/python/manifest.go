package python

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Manifest holds the dependencies declared by a Python repository.
//
// The manifests are read directly from disk. No package manager is invoked
// and nothing is installed.
type Manifest struct {
	// Found states whether a manifest of the language was found.
	Found bool
	// Distributions are the normalized names of the declared distributions.
	Distributions []string
}

// ReadManifest reads the requirements.txt and the pyproject.toml files of
// the given repository root.
func ReadManifest(root string) Manifest {
	manifest := Manifest{}
	if root == "" {
		return manifest
	}
	if data, err := os.ReadFile(filepath.Join(root, "requirements.txt")); err == nil {
		manifest.Found = true
		manifest.Distributions = append(manifest.Distributions, parseRequirements(data)...)
	}
	if data, err := os.ReadFile(filepath.Join(root, "pyproject.toml")); err == nil {
		manifest.Found = true
		manifest.Distributions = append(manifest.Distributions, parsePyproject(data)...)
	}
	return manifest
}

// requirementLineRe matches a requirement specification.
var requirementLineRe = regexp.MustCompile(`^([A-Za-z0-9][A-Za-z0-9._-]*)`)

// parseRequirements extracts the distribution names of a requirements.txt
// file.
//
// Only the names are read. Version specifiers, environment markers, extras
// and the options of pip are ignored.
func parseRequirements(data []byte) []string {
	var names []string
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "-") {
			// Options such as "-r other.txt" or "-e ." are ignored.
			continue
		}
		if idx := strings.Index(line, "#"); idx >= 0 {
			line = strings.TrimSpace(line[:idx])
		}
		if idx := strings.Index(line, ";"); idx >= 0 {
			line = strings.TrimSpace(line[:idx])
		}
		if idx := strings.Index(line, "["); idx >= 0 {
			line = strings.TrimSpace(line[:idx])
		}
		match := requirementLineRe.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		names = append(names, Normalize(match[1]))
	}
	return names
}

// parsePyproject extracts the dependency names declared by a pyproject.toml
// file.
//
// The parser understands the "project.dependencies" and
// "project.optional-dependencies" arrays of PEP 621 and the
// "tool.poetry.dependencies" table. Other tables are ignored.
func parsePyproject(data []byte) []string {
	var names []string
	section := ""
	lines := strings.Split(string(data), "\n")
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(line[1 : len(line)-1])
			continue
		}
		eq := strings.Index(line, "=")
		if eq < 0 {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		value := strings.TrimSpace(line[eq+1:])
		switch {
		case section == "project" && key == "dependencies":
			names = append(names, readStringArray(lines, i, value)...)
			i = skipArrayLines(lines, i, value)
		case section == "project.optional-dependencies":
			names = append(names, readStringArray(lines, i, value)...)
			i = skipArrayLines(lines, i, value)
		case strings.HasPrefix(section, "tool.poetry") && strings.HasSuffix(section, "dependencies"):
			name, ok := poetryDependency(key)
			if ok {
				names = append(names, Normalize(name))
			}
		}
	}
	return names
}

// poetryDependency returns the distribution name of a poetry dependency
// entry.
func poetryDependency(key string) (string, bool) {
	name := strings.TrimSpace(key)
	if name == "" || name == "python" {
		return "", false
	}
	return name, true
}

// readStringArray reads the strings of an inline array that may continue in
// the following lines.
func readStringArray(lines []string, index int, value string) []string {
	var names []string
	rest := value
	for {
		for _, specification := range readQuotedStrings(rest) {
			name := distributionName(specification)
			if name == "" {
				continue
			}
			names = append(names, Normalize(name))
		}
		if strings.Contains(rest, "]") {
			return names
		}
		index++
		if index >= len(lines) {
			return names
		}
		rest = lines[index]
	}
}

// skipArrayLines returns the index of the last line of an inline array.
func skipArrayLines(lines []string, index int, value string) int {
	if strings.Contains(value, "]") {
		return index
	}
	for index < len(lines) {
		if strings.Contains(lines[index], "]") {
			return index
		}
		index++
	}
	return index
}

// quotedStringRe matches a quoted string.
var quotedStringRe = regexp.MustCompile(`"([^"]*)"|'([^']*)'`)

// readQuotedStrings returns the content of the quoted strings of a text.
func readQuotedStrings(text string) []string {
	var values []string
	for _, match := range quotedStringRe.FindAllStringSubmatch(text, -1) {
		if match[1] != "" {
			values = append(values, match[1])
			continue
		}
		values = append(values, match[2])
	}
	return values
}

// distributionName returns the name of a distribution from a requirement
// specification such as "httpx[http2]==0.27.0".
func distributionName(specification string) string {
	name := strings.TrimSpace(specification)
	if idx := strings.Index(name, "["); idx >= 0 {
		name = name[:idx]
	}
	if idx := strings.Index(name, ";"); idx >= 0 {
		name = name[:idx]
	}
	name = strings.TrimSpace(name)
	for _, separator := range []string{"==", ">=", "<=", "~=", "!=", ">", "<", "="} {
		if idx := strings.Index(name, separator); idx >= 0 {
			name = name[:idx]
			break
		}
	}
	return strings.TrimSpace(name)
}

// Normalize returns the normalized name of a distribution, following the
// rules of PEP 503.
func Normalize(name string) string {
	lower := strings.ToLower(strings.TrimSpace(name))
	var b strings.Builder
	previousSeparator := false
	for _, r := range lower {
		switch r {
		case '-', '_', '.':
			if !previousSeparator {
				b.WriteRune('-')
				previousSeparator = true
			}
		default:
			b.WriteRune(r)
			previousSeparator = false
		}
	}
	return b.String()
}

// Declares reports whether the manifest declares the given distribution.
func (m Manifest) Declares(name string) bool {
	normalized := Normalize(name)
	for _, declared := range m.Distributions {
		if declared == normalized {
			return true
		}
	}
	return false
}
