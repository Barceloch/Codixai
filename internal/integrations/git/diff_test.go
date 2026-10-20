package git

import (
	"strings"
	"testing"

	"codixai/internal/domain"
)

func TestParseUnifiedDiffModifiedFile(t *testing.T) {
	diff := strings.Join([]string{
		"diff --git a/internal/app.go b/internal/app.go",
		"index 1234567..89abcde 100644",
		"--- a/internal/app.go",
		"+++ b/internal/app.go",
		"@@ -10,6 +10,6 @@ func Run() {",
		" \tctx := context.Background()",
		"-\tif err != nil {",
		"+\tif err != nil { // report",
		" \t\treturn err",
		" \t}",
		" \treturn nil",
		" }",
		"",
	}, "\n")

	changes, err := ParseUnifiedDiff(strings.NewReader(diff))
	if err != nil {
		t.Fatalf("ParseUnifiedDiff returned an error: %v", err)
	}
	if len(changes.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(changes.Files))
	}
	file := changes.Files[0]
	if file.Path != "internal/app.go" {
		t.Errorf("path: got %q, want %q", file.Path, "internal/app.go")
	}
	if file.Status != domain.FileModified {
		t.Errorf("status: got %q, want %q", file.Status, domain.FileModified)
	}
	if file.Language != LanguageGo {
		t.Errorf("language: got %q, want %q", file.Language, LanguageGo)
	}
	if file.OldPath != "" {
		t.Errorf("old path: got %q, want empty", file.OldPath)
	}
	if len(file.Hunks) != 1 {
		t.Fatalf("expected 1 hunk, got %d", len(file.Hunks))
	}
	hunk := file.Hunks[0]
	if hunk.Index != 1 {
		t.Errorf("hunk index: got %d, want 1", hunk.Index)
	}
	if hunk.OldStart != 10 || hunk.OldLines != 6 {
		t.Errorf("old range: got %d,%d want 10,6", hunk.OldStart, hunk.OldLines)
	}
	if hunk.NewStart != 10 || hunk.NewLines != 6 {
		t.Errorf("new range: got %d,%d want 10,6", hunk.NewStart, hunk.NewLines)
	}
	if hunk.Header != "func Run() {" {
		t.Errorf("header: got %q, want %q", hunk.Header, "func Run() {")
	}
	if len(hunk.Lines) != 7 {
		t.Fatalf("expected 7 hunk lines, got %d", len(hunk.Lines))
	}
	first := hunk.Lines[0]
	if first.Kind != domain.LineContext || first.OldLine != 10 || first.NewLine != 10 {
		t.Errorf("first line: got %+v", first)
	}
	removed := hunk.Lines[1]
	if removed.Kind != domain.LineRemoved || removed.OldLine != 11 || removed.NewLine != 0 {
		t.Errorf("removed line: got %+v", removed)
	}
	added := hunk.Lines[2]
	if added.Kind != domain.LineAdded || added.OldLine != 0 || added.NewLine != 11 {
		t.Errorf("added line: got %+v", added)
	}
	last := hunk.Lines[6]
	if last.Kind != domain.LineContext || last.OldLine != 15 || last.NewLine != 15 {
		t.Errorf("last line: got %+v", last)
	}
}

func TestParseUnifiedDiffAddedFile(t *testing.T) {
	diff := strings.Join([]string{
		"diff --git a/pkg/service.py b/pkg/service.py",
		"new file mode 100644",
		"index 0000000..3b18e51",
		"--- /dev/null",
		"+++ b/pkg/service.py",
		"@@ -0,0 +1,3 @@",
		"+class Service:",
		"+    pass",
		"+",
		"",
	}, "\n")

	changes, err := ParseUnifiedDiff(strings.NewReader(diff))
	if err != nil {
		t.Fatalf("ParseUnifiedDiff returned an error: %v", err)
	}
	file := changes.Files[0]
	if file.Path != "pkg/service.py" {
		t.Errorf("path: got %q", file.Path)
	}
	if file.Status != domain.FileAdded {
		t.Errorf("status: got %q, want %q", file.Status, domain.FileAdded)
	}
	if file.Language != LanguagePython {
		t.Errorf("language: got %q", file.Language)
	}
	if file.OldPath != "" {
		t.Errorf("old path: got %q, want empty", file.OldPath)
	}
	hunk := file.Hunks[0]
	if hunk.OldStart != 0 || hunk.OldLines != 0 || hunk.NewStart != 1 || hunk.NewLines != 3 {
		t.Errorf("hunk range: got %+v", hunk)
	}
	if len(hunk.Lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(hunk.Lines))
	}
	if hunk.Lines[2].Kind != domain.LineAdded || hunk.Lines[2].Content != "" {
		t.Errorf("last line: got %+v", hunk.Lines[2])
	}
	if hunk.Lines[2].NewLine != 3 {
		t.Errorf("last line number: got %d, want 3", hunk.Lines[2].NewLine)
	}
}

func TestParseUnifiedDiffDeletedFile(t *testing.T) {
	diff := strings.Join([]string{
		"diff --git a/legacy/old.php b/legacy/old.php",
		"deleted file mode 100644",
		"index 3b18e51..0000000",
		"--- a/legacy/old.php",
		"+++ /dev/null",
		"@@ -1,2 +0,0 @@",
		"-<?php",
		"-echo 1;",
		"",
	}, "\n")

	changes, err := ParseUnifiedDiff(strings.NewReader(diff))
	if err != nil {
		t.Fatalf("ParseUnifiedDiff returned an error: %v", err)
	}
	file := changes.Files[0]
	if file.Path != "legacy/old.php" {
		t.Errorf("path: got %q", file.Path)
	}
	if file.Status != domain.FileDeleted {
		t.Errorf("status: got %q, want %q", file.Status, domain.FileDeleted)
	}
	if file.Language != LanguagePHP {
		t.Errorf("language: got %q", file.Language)
	}
	if len(file.Hunks) != 1 || len(file.Hunks[0].Lines) != 2 {
		t.Fatalf("expected one hunk with two lines, got %+v", file.Hunks)
	}
	if file.Hunks[0].Lines[0].OldLine != 1 || file.Hunks[0].Lines[1].OldLine != 2 {
		t.Errorf("old line numbers: %+v", file.Hunks[0].Lines)
	}
}

func TestParseUnifiedDiffRenamedFile(t *testing.T) {
	diff := strings.Join([]string{
		"diff --git a/src/old.ts b/src/new.ts",
		"similarity index 100%",
		"rename from src/old.ts",
		"rename to src/new.ts",
		"",
	}, "\n")

	changes, err := ParseUnifiedDiff(strings.NewReader(diff))
	if err != nil {
		t.Fatalf("ParseUnifiedDiff returned an error: %v", err)
	}
	file := changes.Files[0]
	if file.Path != "src/new.ts" {
		t.Errorf("path: got %q, want %q", file.Path, "src/new.ts")
	}
	if file.OldPath != "src/old.ts" {
		t.Errorf("old path: got %q, want %q", file.OldPath, "src/old.ts")
	}
	if file.Status != domain.FileRenamed {
		t.Errorf("status: got %q, want %q", file.Status, domain.FileRenamed)
	}
	if file.Language != LanguageTypeScript {
		t.Errorf("language: got %q", file.Language)
	}
	if len(file.Hunks) != 0 {
		t.Errorf("expected no hunks, got %d", len(file.Hunks))
	}
}

func TestParseUnifiedDiffRenamedAndModifiedFile(t *testing.T) {
	diff := strings.Join([]string{
		"diff --git a/src/old.ts b/src/new.ts",
		"similarity index 87%",
		"rename from src/old.ts",
		"rename to src/new.ts",
		"index 1234567..89abcde 100644",
		"--- a/src/old.ts",
		"+++ b/src/new.ts",
		"@@ -1,2 +1,2 @@",
		"-const a = 1",
		"+const a = 2",
		"",
	}, "\n")

	changes, err := ParseUnifiedDiff(strings.NewReader(diff))
	if err != nil {
		t.Fatalf("ParseUnifiedDiff returned an error: %v", err)
	}
	file := changes.Files[0]
	if file.Status != domain.FileRenamed {
		t.Errorf("status: got %q, want %q", file.Status, domain.FileRenamed)
	}
	if file.Path != "src/new.ts" || file.OldPath != "src/old.ts" {
		t.Errorf("paths: got %q / %q", file.Path, file.OldPath)
	}
	if len(file.Hunks) != 1 || len(file.Hunks[0].Lines) != 2 {
		t.Fatalf("expected one hunk with two lines, got %+v", file.Hunks)
	}
}

func TestParseUnifiedDiffBinaryFile(t *testing.T) {
	diff := strings.Join([]string{
		"diff --git a/assets/logo.png b/assets/logo.png",
		"new file mode 100644",
		"index 0000000..3b18e51",
		"Binary files /dev/null and b/assets/logo.png differ",
		"",
	}, "\n")

	changes, err := ParseUnifiedDiff(strings.NewReader(diff))
	if err != nil {
		t.Fatalf("ParseUnifiedDiff returned an error: %v", err)
	}
	file := changes.Files[0]
	if !file.IsBinary {
		t.Error("expected the file to be marked as binary")
	}
	if file.Path != "assets/logo.png" {
		t.Errorf("path: got %q", file.Path)
	}
	if file.Status != domain.FileAdded {
		t.Errorf("status: got %q, want %q", file.Status, domain.FileAdded)
	}
	if len(file.Hunks) != 0 {
		t.Errorf("expected no hunks, got %d", len(file.Hunks))
	}
}

func TestParseUnifiedDiffQuotedPath(t *testing.T) {
	diff := strings.Join([]string{
		`diff --git "a/dir with space/na\303\257ve file.go" "b/dir with space/na\303\257ve file.go"`,
		"index 1234567..89abcde 100644",
		`--- "a/dir with space/na\303\257ve file.go"` + "\t",
		`+++ "b/dir with space/na\303\257ve file.go"` + "\t",
		"@@ -1,1 +1,2 @@",
		" hello",
		"+world",
		"",
	}, "\n")

	changes, err := ParseUnifiedDiff(strings.NewReader(diff))
	if err != nil {
		t.Fatalf("ParseUnifiedDiff returned an error: %v", err)
	}
	file := changes.Files[0]
	want := "dir with space/naïve file.go"
	if file.Path != want {
		t.Errorf("path: got %q, want %q", file.Path, want)
	}
	if file.Language != LanguageGo {
		t.Errorf("language: got %q, want %q", file.Language, LanguageGo)
	}
	if len(file.Hunks) != 1 || len(file.Hunks[0].Lines) != 2 {
		t.Fatalf("expected one hunk with two lines, got %+v", file.Hunks)
	}
}

func TestParseUnifiedDiffMultipleFilesAndHunks(t *testing.T) {
	diff := strings.Join([]string{
		"diff --git a/a.js b/a.js",
		"index 1111111..2222222 100644",
		"--- a/a.js",
		"+++ b/a.js",
		"@@ -1,3 +1,3 @@",
		" one",
		"-two",
		"+TWO",
		"@@ -10,2 +10,3 @@",
		" ten",
		"+eleven",
		" twelve",
		"diff --git a/b.md b/b.md",
		"index 3333333..4444444 100644",
		"--- a/b.md",
		"+++ b/b.md",
		"@@ -1,1 +1,1 @@",
		"-# title",
		"+# Title",
		"",
	}, "\n")

	changes, err := ParseUnifiedDiff(strings.NewReader(diff))
	if err != nil {
		t.Fatalf("ParseUnifiedDiff returned an error: %v", err)
	}
	if len(changes.Files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(changes.Files))
	}
	first := changes.Files[0]
	if first.Path != "a.js" || first.Language != LanguageJavaScript {
		t.Errorf("first file: %+v", first)
	}
	if len(first.Hunks) != 2 {
		t.Fatalf("expected 2 hunks, got %d", len(first.Hunks))
	}
	if first.Hunks[1].Index != 2 {
		t.Errorf("second hunk index: got %d, want 2", first.Hunks[1].Index)
	}
	if first.Hunks[1].OldStart != 10 || first.Hunks[1].NewStart != 10 {
		t.Errorf("second hunk range: %+v", first.Hunks[1])
	}
	second := changes.Files[1]
	if second.Path != "b.md" {
		t.Errorf("second file path: got %q", second.Path)
	}
	if second.Language != "" {
		t.Errorf("expected an unknown language for a markdown file, got %q", second.Language)
	}
}

func TestParseUnifiedDiffNoNewlineMarker(t *testing.T) {
	diff := strings.Join([]string{
		"diff --git a/a.txt b/a.txt",
		"index 1111111..2222222 100644",
		"--- a/a.txt",
		"+++ b/a.txt",
		"@@ -1,1 +1,1 @@",
		"-old",
		"\\ No newline at end of file",
		"+new",
		"\\ No newline at end of file",
		"",
	}, "\n")

	changes, err := ParseUnifiedDiff(strings.NewReader(diff))
	if err != nil {
		t.Fatalf("ParseUnifiedDiff returned an error: %v", err)
	}
	lines := changes.Files[0].Hunks[0].Lines
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if lines[0].OldLine != 1 || lines[1].NewLine != 1 {
		t.Errorf("line numbers: %+v", lines)
	}
}

func TestParseUnifiedDiffEmptyInput(t *testing.T) {
	changes, err := ParseUnifiedDiff(strings.NewReader(""))
	if err != nil {
		t.Fatalf("ParseUnifiedDiff returned an error: %v", err)
	}
	if len(changes.Files) != 0 {
		t.Errorf("expected no files, got %d", len(changes.Files))
	}
	if changes.Files == nil {
		t.Error("expected an initialized slice of files")
	}
}

func TestUnquotePath(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{`plain/path.go`, `plain/path.go`},
		{`"a/dir with space/file.go"`, `a/dir with space/file.go`},
		{`"a/na\303\257ve.go"`, "a/naïve.go"},
		{`"a/quo\"te.go"`, `a/quo"te.go`},
		{`"a/back\\slash.go"`, `a/back\slash.go`},
		{`"a/tab\tchar.go"`, "a/tab\tchar.go"},
	}
	for _, tc := range cases {
		if got := unquotePath(tc.in); got != tc.want {
			t.Errorf("unquotePath(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
