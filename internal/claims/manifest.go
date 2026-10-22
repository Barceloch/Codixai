package claims

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/barceloch/codixai/internal/languages/go"
	"github.com/barceloch/codixai/internal/languages/javascript"
	"github.com/barceloch/codixai/internal/languages/php"
	"github.com/barceloch/codixai/internal/languages/python"
)

// manifestPaths lists the manifests and the lockfiles of the supported
// ecosystems.
//
// The list holds the two kinds of file together because a change of a
// lockfile is a change of the dependencies of the project, even when the
// manifest itself stays untouched. The names are the ones the ecosystems
// use in the root of the repository.
//
// The dependency declarations are read with the readers of the language
// packages. Only the manifests are read: a lockfile is not a declaration of
// dependencies.
var manifestPaths = []string{
	// Go.
	"go.mod",
	"go.sum",
	// Python.
	"requirements.txt",
	"pyproject.toml",
	"poetry.lock",
	"Pipfile",
	"Pipfile.lock",
	// PHP.
	"composer.json",
	"composer.lock",
	// JavaScript and TypeScript.
	"package.json",
	"package-lock.json",
	"yarn.lock",
	"pnpm-lock.yaml",
}

// manifests holds the dependency declarations of an analyzed repository.
type manifests struct {
	goManifest         golang.Manifest
	pythonManifest     python.Manifest
	phpManifest        php.Manifest
	javascriptManifest javascript.Manifest
}

// readManifests reads the manifests of the repository located at root.
//
// The files are read directly from the disk. No package manager is invoked,
// nothing is installed and no network is used.
func readManifests(root string) manifests {
	if root == "" {
		return manifests{}
	}
	return manifests{
		goManifest:         golang.ReadManifest(root),
		pythonManifest:     python.ReadManifest(root),
		phpManifest:        php.ReadManifest(root),
		javascriptManifest: javascript.ReadManifest(root),
	}
}

// found reports whether the repository declares its dependencies in a
// manifest of any supported language.
func (m manifests) found() bool {
	return m.goManifest.Found || m.pythonManifest.Found || m.phpManifest.Found || m.javascriptManifest.Found
}

// declares reports whether a dependency is declared in a manifest of the
// repository.
func (m manifests) declares(name string) bool {
	if m.goManifest.Declares(name) {
		return true
	}
	if m.pythonManifest.Declares(name) {
		return true
	}
	if m.phpManifest.Declares(name) {
		return true
	}
	return m.javascriptManifest.Declares(name)
}

// isManifestPath reports whether a repository relative path is a manifest or
// a lockfile. The name is matched at any depth, because a manifest of a
// subdirectory is still a manifest.
func isManifestPath(path string) bool {
	name := filepath.Base(strings.ReplaceAll(path, "\\", "/"))
	for _, manifest := range manifestPaths {
		if manifest == name {
			return true
		}
	}
	return false
}

// rootHasManifest reports whether the root of the repository holds a
// manifest of any supported language.
func rootHasManifest(root string) bool {
	if root == "" {
		return false
	}
	for _, manifest := range manifestPaths {
		if _, err := os.Stat(filepath.Join(root, manifest)); err == nil {
			return true
		}
	}
	return false
}
