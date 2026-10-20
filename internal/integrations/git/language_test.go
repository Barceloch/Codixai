package git

import "testing"

func TestLanguageForPath(t *testing.T) {
	cases := []struct {
		path string
		want string
	}{
		{"internal/app.go", LanguageGo},
		{"main.py", LanguagePython},
		{"index.php", LanguagePHP},
		{"app.js", LanguageJavaScript},
		{"app.mjs", LanguageJavaScript},
		{"app.cjs", LanguageJavaScript},
		{"view.jsx", LanguageJavaScript},
		{"app.ts", LanguageTypeScript},
		{"view.tsx", LanguageTypeScript},
		{"mod.mts", LanguageTypeScript},
		{"mod.cts", LanguageTypeScript},
		{"MAIN.GO", LanguageGo},
		{"src/deep/path/file.py", LanguagePython},
		{"README.md", ""},
		{"Makefile", ""},
		{"archive.tar.gz", ""},
		{"", ""},
	}
	for _, tc := range cases {
		if got := LanguageForPath(tc.path); got != tc.want {
			t.Errorf("LanguageForPath(%q) = %q, want %q", tc.path, got, tc.want)
		}
	}
}
