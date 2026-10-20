package php

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// Manifest holds the dependencies declared by the composer.json file of a
// repository.
//
// The file is read directly. Composer is never invoked.
type Manifest struct {
	// Found states whether a composer.json file was found.
	Found bool
	// Packages are the declared package names, in the "vendor/name" form.
	Packages []string
	// AutoloadPrefixes are the namespace prefixes that the project maps to
	// its own directories. They are local code, not third party packages.
	AutoloadPrefixes []string
}

// composerFile is the subset of composer.json that the analysis reads.
type composerFile struct {
	Require    map[string]string `json:"require"`
	RequireDev map[string]string `json:"require-dev"`
	Autoload   struct {
		PSR4 map[string]string `json:"psr-4"`
		PSR0 map[string]string `json:"psr-0"`
	} `json:"autoload"`
}

// ReadManifest reads the composer.json file of the given repository root.
func ReadManifest(root string) Manifest {
	manifest := Manifest{}
	if root == "" {
		return manifest
	}
	data, err := os.ReadFile(filepath.Join(root, "composer.json"))
	if err != nil {
		return manifest
	}
	var parsed composerFile
	if err := json.Unmarshal(data, &parsed); err != nil {
		return manifest
	}
	manifest.Found = true
	for name := range parsed.Require {
		manifest.Packages = append(manifest.Packages, strings.ToLower(name))
	}
	for name := range parsed.RequireDev {
		manifest.Packages = append(manifest.Packages, strings.ToLower(name))
	}
	for prefix := range parsed.Autoload.PSR4 {
		manifest.AutoloadPrefixes = append(manifest.AutoloadPrefixes, strings.TrimSuffix(prefix, "\\"))
	}
	for prefix := range parsed.Autoload.PSR0 {
		manifest.AutoloadPrefixes = append(manifest.AutoloadPrefixes, strings.TrimSuffix(prefix, "\\"))
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

// IsLocal reports whether the namespace belongs to the project itself.
func (m Manifest) IsLocal(namespace string) bool {
	for _, prefix := range m.AutoloadPrefixes {
		if prefix == "" {
			continue
		}
		if namespace == prefix || strings.HasPrefix(namespace, prefix+"\\") {
			return true
		}
	}
	return false
}

// PackageOf returns the composer package that may provide the given
// namespace.
//
// Composer packages are named "vendor/name". A namespace such as
// "Acme\Library\Client" is provided by the package "acme/library".
func PackageOf(namespace string) (string, bool) {
	parts := strings.Split(namespace, "\\")
	if len(parts) < 2 {
		return "", false
	}
	return strings.ToLower(parts[0] + "/" + parts[1]), true
}
