package policies

import "testing"

func TestMatchGlob(t *testing.T) {
	cases := []struct {
		pattern string
		path    string
		want    bool
	}{
		{"vendor/**", "vendor/pkg/file.go", true},
		{"vendor/**", "vendor/file.go", true},
		{"vendor/**", "vendor", true},
		{"vendor/**", "src/vendor/file.go", false},
		{"vendor/**", "internal/file.go", false},
		{"node_modules/**", "node_modules/react/index.js", true},
		{"**/*.go", "main.go", true},
		{"**/*.go", "internal/app.go", true},
		{"**/*.go", "internal/app.py", false},
		{"src/*.go", "src/app.go", true},
		{"src/*.go", "src/deep/app.go", false},
		{"src/**", "src/deep/app.go", true},
		{"src/**", "src", true},
		{"*.log", "debug.log", true},
		{"*.log", "logs/debug.log", true},
		{"*.log", "debug.txt", false},
		{"dist/", "dist/app.js", true},
		{".git/**", ".git/config", true},
		{".git/**", ".gitignore", false},
		{"**", "any/path/file.go", true},
		{"a?c.go", "abc.go", true},
		{"a?c.go", "ac.go", false},
		{"file.go", "file.go", true},
		{"file.go", "src/file.go", true},
	}
	for _, tc := range cases {
		got, err := MatchGlob(tc.pattern, tc.path)
		if err != nil {
			t.Fatalf("MatchGlob(%q, %q) returned an error: %v", tc.pattern, tc.path, err)
		}
		if got != tc.want {
			t.Errorf("MatchGlob(%q, %q) = %v, want %v", tc.pattern, tc.path, got, tc.want)
		}
	}
}

func TestMatchGlobRejectsEmptyPattern(t *testing.T) {
	if _, err := MatchGlob("  ", "file.go"); err == nil {
		t.Error("expected an error for an empty pattern")
	}
}
