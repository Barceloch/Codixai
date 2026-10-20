// Package python implements the Python specific extraction of Codixai.
//
// The extraction is syntactic: imports, markers and exception handlers are
// recognized from the text of the changed lines with a scanner that knows
// the comment and string syntax of Python.
package python

import (
	"regexp"
	"strings"

	"github.com/Barceloch/Codixai/internal/domain"
	"github.com/Barceloch/Codixai/internal/languages/lexical"
)

// Profile returns the lexical profile and the syntactic markers of Python.
func Profile() lexical.Profile {
	return lexical.Profile{
		ID:            domain.LangPython,
		Name:          "Python",
		Extensions:    []string{".py"},
		LineComments:  []string{"#"},
		BlockComments: []lexical.BlockComment{},
		Strings: []lexical.StringSpec{
			{Delimiter: "\"\"\"", Escape: true, Multiline: true},
			{Delimiter: "'''", Escape: true, Multiline: true},
			{Delimiter: "\"", Escape: true},
			{Delimiter: "'", Escape: true},
		},
		RawPrefixes:   []string{"r", "R", "u", "U", "f", "F", "b", "B", "rb", "br", "fr", "rf"},
		RegexLiterals: false,
		Suppressions: []lexical.Marker{
			{
				Pattern:     regexp.MustCompile(`\bnoqa\b`),
				Description: "flake8 noqa suppression",
				Scope:       lexical.ScopeComment,
			},
			{
				Pattern:     regexp.MustCompile(`\btype:\s*ignore\b`),
				Description: "mypy type ignore suppression",
				Scope:       lexical.ScopeComment,
			},
			{
				Pattern:     regexp.MustCompile(`\bpylint:\s*disable\b`),
				Description: "pylint disable suppression",
				Scope:       lexical.ScopeComment,
			},
			{
				Pattern:     regexp.MustCompile(`\bmypy:\s*ignore\b`),
				Description: "mypy ignore suppression",
				Scope:       lexical.ScopeComment,
			},
		},
		Skips: []lexical.Marker{
			{
				Pattern:     regexp.MustCompile(`@pytest\.mark\.(skip|skipif|xfail)\b`),
				Description: "pytest marker that disables a test",
				Scope:       lexical.ScopeAny,
			},
			{
				Pattern:     regexp.MustCompile(`@unittest\.(skip|skipIf|skipUnless)\b`),
				Description: "unittest marker that disables a test",
				Scope:       lexical.ScopeAny,
			},
			{
				Pattern:     regexp.MustCompile(`\bskipTest\s*\(`),
				Description: "test skipped at run time",
				Scope:       lexical.ScopeCode,
			},
		},
		Assertions: []lexical.Marker{
			{
				Pattern:     regexp.MustCompile(`\bassert\b`),
				Description: "assert statement",
				Scope:       lexical.ScopeCode,
			},
			{
				Pattern:     regexp.MustCompile(`\bself\.assert[A-Za-z]*\s*\(`),
				Description: "unittest assertion",
				Scope:       lexical.ScopeCode,
			},
			{
				Pattern:     regexp.MustCompile(`\bpytest\.raises\s*\(`),
				Description: "pytest exception assertion",
				Scope:       lexical.ScopeCode,
			},
		},
		Placeholders: []lexical.Marker{
			{
				Pattern:     regexp.MustCompile(`\bTODO\b`),
				Description: "TODO marker",
				Scope:       lexical.ScopeComment,
			},
			{
				Pattern:     regexp.MustCompile(`\bFIXME\b`),
				Description: "FIXME marker",
				Scope:       lexical.ScopeComment,
			},
			{
				Pattern:     regexp.MustCompile(`not\s+implemented`),
				Description: "explicit not implemented marker in a comment",
				Scope:       lexical.ScopeComment,
				Explicit:    true,
			},
			{
				Pattern:     regexp.MustCompile(`not\s+implemented`),
				Description: "explicit not implemented marker in a thrown error",
				Scope:       lexical.ScopeAny,
				Require:     regexp.MustCompile(`(?i)\b(panic|throw|raise)\b`),
				Explicit:    true,
			},
			{
				Pattern:     regexp.MustCompile(`\bNotImplementedError\b`),
				Description: "explicit not implemented error",
				Scope:       lexical.ScopeCode,
				Explicit:    true,
			},
			{
				Pattern:     regexp.MustCompile(`\bplaceholder\b`),
				Description: "placeholder marker",
				Scope:       lexical.ScopeAny,
				Require:     regexp.MustCompile(`(?i)\b(pass|return\s+None)\b`),
			},
		},
		Swallows:  []lexical.Marker{},
		TestPaths: []string{"**/test_*.py", "**/*_test.py"},
	}
}

// stdlibModules holds the top level modules of the Python standard library.
//
// The list is used only to recognize standard library imports.
var stdlibModules = map[string]bool{}

func init() {
	names := []string{
		"__future__", "_ast", "_thread", "abc", "argparse", "array", "ast",
		"asyncio", "atexit", "base64", "bdb", "binascii", "bisect", "builtins",
		"bz2", "calendar", "cmath", "cmd", "code", "codecs", "collections",
		"colorsys", "compileall", "concurrent", "configparser", "contextlib",
		"contextvars", "copy", "copyreg", "csv", "ctypes", "curses", "dataclasses",
		"datetime", "dbm", "decimal", "difflib", "dis", "doctest", "email",
		"encodings", "enum", "errno", "faulthandler", "fcntl", "filecmp",
		"fileinput", "fnmatch", "fractions", "ftplib", "functools", "gc",
		"getopt", "getpass", "gettext", "glob", "graphlib", "grp", "gzip",
		"hashlib", "heapq", "hmac", "html", "http", "imaplib", "importlib",
		"inspect", "io", "ipaddress", "itertools", "json", "keyword", "linecache",
		"locale", "logging", "lzma", "mailbox", "marshal", "math", "mimetypes",
		"mmap", "modulefinder", "multiprocessing", "netrc", "numbers", "operator",
		"optparse", "os", "pathlib", "pdb", "pickle", "pickletools", "pkgutil",
		"platform", "plistlib", "poplib", "posixpath", "pprint", "profile",
		"pstats", "pty", "pwd", "py_compile", "pyclbr", "pydoc", "queue",
		"quopri", "random", "re", "readline", "reprlib", "resource", "rlcompleter",
		"runpy", "sched", "secrets", "select", "selectors", "shelve", "shlex",
		"shutil", "signal", "site", "smtplib", "socket", "socketserver",
		"sqlite3", "ssl", "stat", "statistics", "string", "struct", "subprocess",
		"sys", "sysconfig", "syslog", "tabnanny", "tarfile", "tempfile", "termios",
		"textwrap", "threading", "time", "timeit", "tkinter", "token",
		"tokenize", "trace", "traceback", "tracemalloc", "tty", "turtle",
		"types", "typing", "unicodedata", "unittest", "urllib", "uuid", "venv",
		"warnings", "wave", "weakref", "webbrowser", "winreg", "wsgiref", "xml",
		"xmlrpc", "zipapp", "zipfile", "zipimport", "zlib", "zoneinfo",
	}
	for _, name := range names {
		stdlibModules[name] = true
	}
}

// IsStdlib reports whether the given module belongs to the Python standard
// library.
func IsStdlib(module string) bool {
	root := module
	if idx := strings.Index(module, "."); idx > 0 {
		root = module[:idx]
	}
	return stdlibModules[root]
}

// importStatementRe matches a Python import statement.
var importStatementRe = regexp.MustCompile(`^\s*import\s+(.+)$`)

// fromStatementRe matches a Python from import statement.
var fromStatementRe = regexp.MustCompile(`^\s*from\s+([.\w]+)\s+import\b`)

// ExtractImports returns the modules imported by the given source lines.
//
// The relative imports, which always point to the local project, are
// returned with the Relative flag set. Only the code of the line is
// considered, so an import written inside a string or a comment is not
// reported.
func ExtractImports(lines []lexical.Line, segments map[int][]lexical.Segment) []lexical.ImportSpec {
	var imports []lexical.ImportSpec
	for _, line := range lines {
		text := lexical.CodeOrText(line, segments)
		if match := fromStatementRe.FindStringSubmatch(text); match != nil {
			module := match[1]
			imports = append(imports, lexical.ImportSpec{
				Module:   module,
				Relative: strings.HasPrefix(module, "."),
				Line:     line.Number,
			})
			continue
		}
		match := importStatementRe.FindStringSubmatch(text)
		if match == nil {
			continue
		}
		for _, part := range strings.Split(match[1], ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			fields := strings.Fields(part)
			module := fields[0]
			if module == "" {
				continue
			}
			imports = append(imports, lexical.ImportSpec{
				Module:   module,
				Relative: strings.HasPrefix(module, "."),
				Line:     line.Number,
			})
		}
	}
	return imports
}

// exceptRe matches the header of an exception handler.
var exceptRe = regexp.MustCompile(`^\s*except\b.*:\s*$`)

// exceptInlineRe matches an exception handler with an inline body.
var exceptInlineRe = regexp.MustCompile(`^\s*except\b.*:\s*pass\s*$`)

// EmptyHandlers returns the numbers of the lines that open an exception
// handler with an empty body.
//
// Only the handlers whose body is visible in the given lines are reported.
func EmptyHandlers(lines []lexical.Line, segments map[int][]lexical.Segment) []int {
	var found []int
	for i, line := range lines {
		trimmed := strings.TrimSpace(lexical.CodeOrText(line, segments))
		if exceptInlineRe.MatchString(trimmed) {
			found = append(found, line.Number)
			continue
		}
		if !exceptRe.MatchString(trimmed) {
			continue
		}
		if i+1 < len(lines) && strings.TrimSpace(lexical.CodeOrText(lines[i+1], segments)) == "pass" {
			found = append(found, line.Number)
		}
	}
	return found
}
