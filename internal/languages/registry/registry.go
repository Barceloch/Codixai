// Package registry resolves the language of a file and exposes the profile
// of every supported language.
//
// The registry is the only entry point that the analysis engine uses to know
// which language a file belongs to. Files with an unknown extension are not
// analyzed by any language specific rule.
package registry

import (
	"path/filepath"
	"strings"

	"codixai/internal/domain"
	"codixai/internal/languages/go"
	"codixai/internal/languages/javascript"
	"codixai/internal/languages/lexical"
	"codixai/internal/languages/php"
	"codixai/internal/languages/python"
	"codixai/internal/languages/typescript"
)

// profiles holds the profile of every supported language, indexed by the
// language identifier of the domain model.
var profiles = map[string]lexical.Profile{
	domain.LangGo:         golang.Profile(),
	domain.LangPython:     python.Profile(),
	domain.LangPHP:        php.Profile(),
	domain.LangJavaScript: javascript.Profile(),
	domain.LangTypeScript: typescript.Profile(),
}

// extensions maps a lower case file extension to its language identifier.
var extensions = map[string]string{}

func init() {
	for _, profile := range profiles {
		for _, extension := range profile.Extensions {
			extensions[extension] = profile.ID
		}
	}
}

// Languages returns the identifiers of the supported languages, sorted.
func Languages() []string {
	return []string{
		domain.LangGo,
		domain.LangJavaScript,
		domain.LangPHP,
		domain.LangPython,
		domain.LangTypeScript,
	}
}

// Detect returns the language of a file from its extension.
//
// It reports false when the extension is not known. The caller must then
// ignore the file: no language specific rule applies to it.
func Detect(path string) (string, bool) {
	if path == "" {
		return "", false
	}
	base := filepath.Base(strings.ReplaceAll(path, "\\", "/"))
	extension := strings.ToLower(filepath.Ext(base))
	language, ok := extensions[extension]
	if !ok {
		return "", false
	}
	return language, true
}

// ProfileFor returns the profile of a language.
func ProfileFor(language string) (lexical.Profile, bool) {
	profile, ok := profiles[language]
	return profile, ok
}

// ProfileForPath returns the profile of the language of a file.
func ProfileForPath(path string) (lexical.Profile, bool) {
	language, ok := Detect(path)
	if !ok {
		return lexical.Profile{}, false
	}
	return ProfileFor(language)
}
