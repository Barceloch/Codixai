package glob

import "testing"

func TestMatch(t *testing.T) {
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
		{"internal/payment/**", "internal/payment/service.go", true},
		{"internal/payment/**", "internal/payment/service_test.go", true},
		{"internal/payment/**", "internal/billing/service.go", false},
	}
	for _, tc := range cases {
		got, err := Match(tc.pattern, tc.path)
		if err != nil {
			t.Fatalf("Match(%q, %q) returned an error: %v", tc.pattern, tc.path, err)
		}
		if got != tc.want {
			t.Errorf("Match(%q, %q) = %v, want %v", tc.pattern, tc.path, got, tc.want)
		}
	}
}

func TestMatchRejectsEmptyPattern(t *testing.T) {
	if _, err := Match("  ", "file.go"); err == nil {
		t.Error("expected an error for an empty pattern")
	}
}

func TestCompileKeepsThePattern(t *testing.T) {
	matcher, err := Compile("src/**")
	if err != nil {
		t.Fatalf("Compile returned an error: %v", err)
	}
	if matcher.Pattern() != "src/**" {
		t.Errorf("Pattern() = %q, want %q", matcher.Pattern(), "src/**")
	}
}

func TestCompileIsDeterministic(t *testing.T) {
	first, err := Compile("internal/**")
	if err != nil {
		t.Fatalf("Compile returned an error: %v", err)
	}
	second, err := Compile("internal/**")
	if err != nil {
		t.Fatalf("Compile returned an error: %v", err)
	}
	for _, path := range []string{"internal/a.go", "internal/a/b.go", "other/a.go"} {
		if first.Matches(path) != second.Matches(path) {
			t.Errorf("the matchers disagree about %q", path)
		}
	}
}
