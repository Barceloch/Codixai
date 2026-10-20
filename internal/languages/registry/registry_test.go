package registry_test

import (
	"testing"

	"codixai/internal/domain"
	"codixai/internal/languages/registry"
)

// TestDetect checks the detection of the language of a file.
func TestDetect(t *testing.T) {
	cases := []struct {
		path string
		want string
		ok   bool
	}{
		{"main.go", domain.LangGo, true},
		{"internal/app/service.go", domain.LangGo, true},
		{"app/main.py", domain.LangPython, true},
		{"src/Service.php", domain.LangPHP, true},
		{"src/index.js", domain.LangJavaScript, true},
		{"src/index.mjs", domain.LangJavaScript, true},
		{"src/index.ts", domain.LangTypeScript, true},
		{"src/index.tsx", domain.LangTypeScript, true},
		{"README.md", "", false},
		{"Makefile", "", false},
		{"", "", false},
	}
	for _, testCase := range cases {
		language, ok := registry.Detect(testCase.path)
		if language != testCase.want || ok != testCase.ok {
			t.Errorf("Detect(%q) is (%q, %v), want (%q, %v)", testCase.path, language, ok, testCase.want, testCase.ok)
		}
	}
}

// TestLanguages checks the list of the supported languages.
func TestLanguages(t *testing.T) {
	languages := registry.Languages()
	want := []string{
		domain.LangGo,
		domain.LangJavaScript,
		domain.LangPHP,
		domain.LangPython,
		domain.LangTypeScript,
	}
	if len(languages) != len(want) {
		t.Fatalf("the registry publishes %d languages, want %d", len(languages), len(want))
	}
	for i, language := range want {
		if languages[i] != language {
			t.Errorf("the language %d is %q, want %q", i, languages[i], language)
		}
	}
}

// TestProfileFor checks the profile of every supported language.
func TestProfileFor(t *testing.T) {
	for _, language := range registry.Languages() {
		profile, ok := registry.ProfileFor(language)
		if !ok {
			t.Errorf("the registry has no profile for %s", language)
			continue
		}
		if profile.ID != language {
			t.Errorf("the profile of %s has the identifier %q", language, profile.ID)
		}
		if len(profile.LineComments) == 0 {
			t.Errorf("the profile of %s has no line comment", language)
		}
		if len(profile.TestPaths) == 0 {
			t.Errorf("the profile of %s has no test path", language)
		}
	}
	if _, ok := registry.ProfileFor("cobol"); ok {
		t.Error("the registry has a profile for an unsupported language")
	}
}

// TestProfileForPath checks the profile of the language of a file.
func TestProfileForPath(t *testing.T) {
	profile, ok := registry.ProfileForPath("src/index.ts")
	if !ok {
		t.Fatal("the registry has no profile for a TypeScript file")
	}
	if profile.ID != domain.LangTypeScript {
		t.Errorf("the profile of a TypeScript file is %q", profile.ID)
	}
	if _, ok := registry.ProfileForPath("notes.txt"); ok {
		t.Error("the registry has a profile for an unknown file")
	}
}
