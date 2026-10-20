package git

import (
	"path/filepath"
	"strings"
)

// Language identifiers used by the analysis engine. They match the language
// identifiers of the domain package.
const (
	LanguageGo         = "go"
	LanguagePython     = "python"
	LanguagePHP        = "php"
	LanguageJavaScript = "javascript"
	LanguageTypeScript = "typescript"
)

// extensions maps a file extension to the language of the file. The
// extension is the one of the file name, in lower case.
var extensions = map[string]string{
	".go":  LanguageGo,
	".py":  LanguagePython,
	".php": LanguagePHP,
	".js":  LanguageJavaScript,
	".mjs": LanguageJavaScript,
	".cjs": LanguageJavaScript,
	".jsx": LanguageJavaScript,
	".ts":  LanguageTypeScript,
	".tsx": LanguageTypeScript,
	".mts": LanguageTypeScript,
	".cts": LanguageTypeScript,
}

// LanguageForPath returns the language of a file from its extension.
//
// An unknown extension is not an error: the function returns an empty string,
// meaning that the file is not analyzed by any language specific rule.
func LanguageForPath(path string) string {
	if path == "" {
		return ""
	}
	base := filepath.Base(strings.ReplaceAll(path, "\\", "/"))
	ext := strings.ToLower(filepath.Ext(base))
	return extensions[ext]
}
