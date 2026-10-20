package golang

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ImportPackageNames returns the name that every import of the source
// declares.
//
// The name of a package does not have to match the last element of its
// import path: a package of a directory named "go" is commonly imported
// under the name "golang". For an import inside the module of the
// repository, the package clause of the target directory is read, so the
// real name is used. For an import outside the module, the last element of
// the path is used, which is the convention of the language.
func ImportPackageNames(root, module string, src []byte) (map[string]string, error) {
	names := map[string]string{}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "source.go", src, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		names[path] = PackageName(root, module, path)
	}
	return names, nil
}

// PackageName returns the name of the package that an import path
// declares.
//
// The package clause of the target directory decides the name when the
// import belongs to the module of the repository and the directory can be
// read. Otherwise the last element of the path is used.
func PackageName(root, module, importPath string) string {
	fallback := defaultPackageName(importPath)
	if root == "" || module == "" {
		return fallback
	}
	if importPath != module && !strings.HasPrefix(importPath, module+"/") {
		return fallback
	}
	relative := strings.TrimPrefix(strings.TrimPrefix(importPath, module), "/")
	directory := filepath.Join(root, filepath.FromSlash(relative))
	entries, err := os.ReadDir(directory)
	if err != nil {
		return fallback
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		if strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		name := packageClauseName(filepath.Join(directory, entry.Name()))
		if name != "" {
			return name
		}
	}
	return fallback
}

// packageClauseName returns the name of the package that a Go file
// declares, or an empty string when the file cannot be read or parsed.
func packageClauseName(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, data, parser.PackageClauseOnly)
	if err != nil || file == nil || file.Name == nil {
		return ""
	}
	return file.Name.Name
}
