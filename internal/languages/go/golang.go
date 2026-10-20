// Package golang implements the Go specific extraction of Codixai.
//
// Import paths and comment directives are extracted with the Go standard
// library toolchain (go/parser and go/ast with ParseComments). No external
// tool is executed and no module is downloaded: the parser only reads the
// source text that the analysis already has.
package golang

import (
	"regexp"
	"strings"

	"github.com/Barceloch/Codixai/internal/domain"
	"github.com/Barceloch/Codixai/internal/languages/lexical"
)

// Profile returns the lexical profile and the syntactic markers of Go.
func Profile() lexical.Profile {
	return lexical.Profile{
		ID:            domain.LangGo,
		Name:          "Go",
		Extensions:    []string{".go"},
		LineComments:  []string{"//"},
		BlockComments: []lexical.BlockComment{{Start: "/*", End: "*/"}},
		Strings: []lexical.StringSpec{
			{Delimiter: "`", Escape: false, Multiline: true},
			{Delimiter: "\"", Escape: true},
			{Delimiter: "'", Escape: true},
		},
		RawPrefixes:   []string{"r"},
		RegexLiterals: false,
		Suppressions: []lexical.Marker{
			{
				Pattern:     regexp.MustCompile(`\bnolint\b`),
				Description: "nolint analyzer suppression directive",
				Scope:       lexical.ScopeComment,
			},
		},
		Skips: []lexical.Marker{
			{
				Pattern:     regexp.MustCompile(`\bt\.(Skip|Skipf|SkipNow)\s*\(`),
				Description: "test skipped at run time",
				Scope:       lexical.ScopeCode,
			},
			{
				Pattern:     regexp.MustCompile(`\btesting\.Short\s*\(\)`),
				Description: "test skipped in short mode",
				Scope:       lexical.ScopeCode,
			},
		},
		Assertions: []lexical.Marker{
			{
				Pattern:     regexp.MustCompile(`\b(assert|require)\.[A-Za-z_][A-Za-z0-9_]*\s*\(`),
				Description: "testify style assertion",
				Scope:       lexical.ScopeCode,
			},
			{
				Pattern:     regexp.MustCompile(`\bt\.(Error|Errorf|Fatal|Fatalf|Fail|FailNow)\s*\(`),
				Description: "standard library test failure call",
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
		},
		Swallows: []lexical.Marker{
			{
				Pattern:     regexp.MustCompile(`_\s*=\s*[A-Za-z0-9_.]*[Ee]rr\b`),
				Description: "error value assigned to the blank identifier",
				Scope:       lexical.ScopeCode,
			},
		},
		TestPaths: []string{"**/*_test.go"},
	}
}

// stdlibPackages holds the names of the packages of the Go standard library.
//
// The list is a snapshot of the standard library and it is used only to
// recognize standard library imports. It is not used to resolve versions.
var stdlibPackages = map[string]bool{}

func init() {
	names := []string{
		"archive", "arena", "bufio", "bytes", "cmp", "cmp/cmpopts", "compress/bzip2",
		"compress/flate", "compress/gzip", "compress/lzw", "compress/zlib",
		"container/heap", "container/list", "container/ring", "context", "crypto",
		"crypto/aes", "crypto/cipher", "crypto/des", "crypto/dsa", "crypto/ecdsa",
		"crypto/ed25519", "crypto/elliptic", "crypto/hmac", "crypto/md5",
		"crypto/rand", "crypto/rsa", "crypto/sha1", "crypto/sha256", "crypto/sha512",
		"crypto/subtle", "crypto/tls", "crypto/x509", "database/sql", "database/sql/driver",
		"debug/dwarf", "debug/elf", "debug/gosym", "debug/macho", "debug/pe",
		"debug/plan9obj", "debug/strace", "embed", "encoding", "encoding/ascii85",
		"encoding/asn1", "encoding/base32", "encoding/base64", "encoding/binary",
		"encoding/csv", "encoding/gob", "encoding/hex", "encoding/json",
		"encoding/json/v2", "encoding/pem", "encoding/xml", "errors", "expvar",
		"flag", "fmt", "go/ast", "go/build", "go/constant", "go/doc",
		"go/doc/comment", "go/format", "go/parser", "go/printer", "go/scanner",
		"go/token", "go/types", "hash", "hash/adler32", "hash/crc32",
		"hash/crc64", "hash/fnv", "hash/maphash", "html", "html/template",
		"image", "image/color", "image/draw", "image/gif", "image/jpeg", "image/png",
		"index/suffixarray", "io", "io/fs", "io/fs/glob", "io/ioutil", "log",
		"log/slog", "maps", "math", "math/big", "math/bits", "math/cmplx",
		"math/rand", "math/rand/v2", "mime", "mime/multipart", "mime/quotedprintable",
		"net", "net/http", "net/http/cookiejar", "net/http/httptest", "net/http/httptrace",
		"net/http/httputil", "net/mail", "net/netip", "net/rpc", "net/rpc/jsonrpc",
		"net/smtp", "net/textproto", "net/url", "os", "os/exec", "os/signal",
		"os/user", "path", "path/filepath", "plugin", "reflect", "regexp",
		"regexp/syntax", "runtime", "runtime/debug", "runtime/metrics",
		"runtime/pprof", "runtime/trace", "slices", "sort", "strconv", "strings",
		"sync", "sync/atomic", "syscall", "testing", "testing/fstest", "testing/iotest",
		"testing/quick", "text/scanner", "text/tabwriter", "text/template",
		"text/template/parse", "time", "time/tzdata", "unicode", "unicode/utf16",
		"unicode/utf8", "unsafe", "weak",
	}
	for _, name := range names {
		stdlibPackages[name] = true
	}
}

// IsStdlib reports whether the given import path belongs to the Go standard
// library.
func IsStdlib(path string) bool {
	return stdlibPackages[path]
}

// IsThirdPartyShape reports whether the import path has the shape of a
// third party module, that is, its first element contains a dot and looks
// like a domain.
//
// The shape is what separates an unambiguous external dependency from an
// import that may belong to another package of the same module.
func IsThirdPartyShape(path string) bool {
	first := path
	if idx := strings.Index(path, "/"); idx >= 0 {
		first = path[:idx]
	}
	return strings.Contains(first, ".")
}

// ExtractImports returns the import paths declared by the given source
// lines.
//
// It is the textual fallback used when the whole file is not available. The
// Go specific extraction based on go/parser is preferred when it can be
// applied.
func ExtractImports(lines []lexical.Line) []lexical.ImportSpec {
	var imports []lexical.ImportSpec
	inBlock := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line.Text)
		if inBlock {
			if trimmed == ")" {
				inBlock = false
				continue
			}
			if path, ok := importPathOf(trimmed); ok {
				imports = append(imports, lexical.ImportSpec{Module: path, Line: line.Number})
			}
			continue
		}
		if !strings.HasPrefix(trimmed, "import") {
			continue
		}
		rest := strings.TrimSpace(strings.TrimPrefix(trimmed, "import"))
		if rest == "(" {
			inBlock = true
			continue
		}
		if rest == "" {
			continue
		}
		if path, ok := importPathOf(rest); ok {
			imports = append(imports, lexical.ImportSpec{Module: path, Line: line.Number})
		}
	}
	return imports
}

// importPathOf returns the quoted import path of an import specification.
func importPathOf(spec string) (string, bool) {
	start := strings.Index(spec, "\"")
	if start < 0 {
		return "", false
	}
	end := strings.Index(spec[start+1:], "\"")
	if end < 0 {
		return "", false
	}
	return spec[start+1 : start+1+end], true
}
