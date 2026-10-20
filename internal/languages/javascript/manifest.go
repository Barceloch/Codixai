package javascript

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// Manifest holds the dependencies declared by the package.json file of a
// repository.
//
// The file is read directly. No package manager is invoked and no package is
// installed.
type Manifest struct {
	// Found states whether a package.json file was found.
	Found bool
	// Packages are the names of the declared packages.
	Packages []string
}

// packageFile is the subset of package.json that the analysis reads.
type packageFile struct {
	Dependencies         map[string]string `json:"dependencies"`
	DevDependencies      map[string]string `json:"devDependencies"`
	OptionalDependencies map[string]string `json:"optionalDependencies"`
}

// ReadManifest reads the package.json file of the given repository root.
func ReadManifest(root string) Manifest {
	manifest := Manifest{}
	if root == "" {
		return manifest
	}
	data, err := os.ReadFile(filepath.Join(root, "package.json"))
	if err != nil {
		return manifest
	}
	var parsed packageFile
	if err := json.Unmarshal(data, &parsed); err != nil {
		return manifest
	}
	manifest.Found = true
	for _, group := range []map[string]string{
		parsed.Dependencies,
		parsed.DevDependencies,
		parsed.OptionalDependencies,
	} {
		for name := range group {
			manifest.Packages = append(manifest.Packages, strings.ToLower(name))
		}
	}
	return manifest
}

// Declares reports whether the manifest declares the given package.
func (m Manifest) Declares(name string) bool {
	lowered := strings.ToLower(name)
	for _, declared := range m.Packages {
		if declared == lowered {
			return true
		}
	}
	return false
}
