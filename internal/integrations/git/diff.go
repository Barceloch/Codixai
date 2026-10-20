package git

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/barceloch/codixai/internal/domain"
)

// maxDiffLineSize bounds the size of a single line of a diff. A line longer
// than this is rejected instead of exhausting the memory of the process.
const maxDiffLineSize = 4 * 1024 * 1024

// hunkHeaderRe matches a unified diff hunk header, with or without the
// optional line counts.
var hunkHeaderRe = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@(.*)$`)

// ParseUnifiedDiff parses a unified diff into a domain.ChangeSet.
//
// The parser is pure: it does not need the git binary and can be tested with
// hand written fixtures. It understands the output produced by
// "git diff --no-ext-diff --no-textconv --find-renames", including quoted
// paths, renames, binary files and deleted files.
func ParseUnifiedDiff(r io.Reader) (*domain.ChangeSet, error) {
	lines, err := readDiffLines(r)
	if err != nil {
		return nil, err
	}
	changes := &domain.ChangeSet{Files: []domain.FileChange{}}

	var current *fileBuilder
	flush := func() {
		if current == nil {
			return
		}
		change := current.build()
		if change != nil {
			changes.Files = append(changes.Files, *change)
		}
		current = nil
	}

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if isFileHeader(line) {
			flush()
			current = newFileBuilder()
			current.parseGitHeader(strings.TrimPrefix(line, "diff --git "))
			continue
		}
		if current == nil {
			// Content outside of a file section is ignored.
			continue
		}
		if current.inHunk {
			consumed, err := current.consumeHunkLine(line)
			if err != nil {
				return nil, err
			}
			if consumed == 0 {
				// The line does not belong to the hunk: it has to be
				// processed again as a file section line.
				i--
			}
			continue
		}
		current.parseFileLine(line)
	}
	flush()
	return changes, nil
}

// readDiffLines reads the whole diff into memory, one entry per line.
func readDiffLines(r io.Reader) ([]string, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), maxDiffLineSize)
	var lines []string
	for scanner.Scan() {
		line := scanner.Text()
		lines = append(lines, strings.TrimSuffix(line, "\r"))
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("cannot read the diff: %w", err)
	}
	return lines, nil
}

// isFileHeader reports whether a line starts a new file section.
func isFileHeader(line string) bool {
	return strings.HasPrefix(line, "diff --git ")
}

// fileBuilder accumulates the state of the file section being parsed.
type fileBuilder struct {
	oldPath string
	newPath string
	status  string
	binary  bool
	hunks   []domain.Hunk
	hunk    *domain.Hunk
	oldLine int
	newLine int
	inHunk  bool
}

// newFileBuilder returns a builder for a new file section.
func newFileBuilder() *fileBuilder { return &fileBuilder{} }

// parseGitHeader reads the "diff --git a/x b/x" header. The header is only a
// hint: the authoritative paths come from the "---"/"+++" lines and from the
// "rename from"/"rename to" lines, because a path may contain spaces.
func (b *fileBuilder) parseGitHeader(rest string) {
	oldPath, newPath := splitGitHeaderPaths(rest)
	b.oldPath = oldPath
	b.newPath = newPath
}

// splitGitHeaderPaths splits the two paths of a "diff --git" header. Quoted
// paths are unquoted; unquoted paths cannot contain spaces, so the single
// " b/" separator is unambiguous.
func splitGitHeaderPaths(rest string) (string, string) {
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return "", ""
	}
	if strings.HasPrefix(rest, `"`) {
		first, remainder, ok := splitQuoted(rest)
		if !ok {
			return "", ""
		}
		second := strings.TrimSpace(remainder)
		if second != "" {
			second = unquotePath(second)
		}
		return stripPrefix(first, "a/"), stripPrefix(second, "b/")
	}
	idx := strings.Index(rest, " b/")
	if idx < 0 {
		return "", ""
	}
	return stripPrefix(rest[:idx], "a/"), stripPrefix(rest[idx+1:], "b/")
}

// splitQuoted splits a leading double quoted string from a string.
func splitQuoted(s string) (string, string, bool) {
	if len(s) == 0 || s[0] != '"' {
		return "", s, false
	}
	for i := 1; i < len(s); i++ {
		switch s[i] {
		case '\\':
			i++
		case '"':
			return s[1:i], s[i+1:], true
		}
	}
	return "", s, false
}

// parseFileLine reads a metadata line of a file section.
func (b *fileBuilder) parseFileLine(line string) {
	switch {
	case strings.HasPrefix(line, "new file mode "):
		b.status = domain.FileAdded
	case strings.HasPrefix(line, "deleted file mode "):
		b.status = domain.FileDeleted
	case strings.HasPrefix(line, "rename from "):
		b.oldPath = unquotePath(strings.TrimSpace(strings.TrimPrefix(line, "rename from ")))
		b.status = domain.FileRenamed
	case strings.HasPrefix(line, "rename to "):
		b.newPath = unquotePath(strings.TrimSpace(strings.TrimPrefix(line, "rename to ")))
		b.status = domain.FileRenamed
	case strings.HasPrefix(line, "copy from "):
		b.oldPath = unquotePath(strings.TrimSpace(strings.TrimPrefix(line, "copy from ")))
	case strings.HasPrefix(line, "copy to "):
		b.newPath = unquotePath(strings.TrimSpace(strings.TrimPrefix(line, "copy to ")))
		if b.status == "" {
			b.status = domain.FileAdded
		}
	case strings.HasPrefix(line, "--- "):
		b.oldPath = parseDiffPath(strings.TrimPrefix(line, "--- "), "a/")
	case strings.HasPrefix(line, "+++ "):
		b.newPath = parseDiffPath(strings.TrimPrefix(line, "+++ "), "b/")
	case strings.HasPrefix(line, "Binary files "), strings.HasPrefix(line, "GIT binary patch"):
		b.binary = true
	case strings.HasPrefix(line, "@@"):
		b.startHunk(line)
	}
}

// parseDiffPath normalizes the path of a "---" or "+++" line. Git appends a
// tab after a quoted path and uses /dev/null for an absent side.
func parseDiffPath(rest, prefix string) string {
	rest = strings.TrimRight(rest, "\t")
	rest = strings.TrimSpace(rest)
	if rest == "" || rest == "/dev/null" {
		return ""
	}
	rest = unquotePath(rest)
	return stripPrefix(rest, prefix)
}

// stripPrefix removes a path prefix when it is present.
func stripPrefix(path, prefix string) string {
	if strings.HasPrefix(path, prefix) {
		return path[len(prefix):]
	}
	return path
}

// startHunk begins a new hunk from its header.
func (b *fileBuilder) startHunk(header string) {
	m := hunkHeaderRe.FindStringSubmatch(header)
	if m == nil {
		return
	}
	if b.hunk != nil {
		// Close the hunk that was still open.
		b.hunks = append(b.hunks, *b.hunk)
		b.hunk = nil
	}
	oldStart, _ := strconv.Atoi(m[1])
	newStart, _ := strconv.Atoi(m[3])
	b.hunk = &domain.Hunk{
		Index:    len(b.hunks) + 1,
		OldStart: oldStart,
		OldLines: countOrOne(m[2]),
		NewStart: newStart,
		NewLines: countOrOne(m[4]),
		Header:   strings.TrimSpace(m[5]),
		Lines:    []domain.DiffLine{},
	}
	b.oldLine = oldStart
	b.newLine = newStart
	b.inHunk = true
}

// countOrOne converts an optional line count, where an omitted count means
// a single line.
func countOrOne(value string) int {
	if value == "" {
		return 1
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return 1
	}
	return n
}

// consumeHunkLine reads one line of a hunk body. It returns the number of
// consumed lines: zero when the line does not belong to the hunk.
func (b *fileBuilder) consumeHunkLine(line string) (int, error) {
	if line == "" {
		// A context line of an empty line may lose its marker.
		b.appendLine(domain.LineContext, "")
		return 1, nil
	}
	switch line[0] {
	case ' ':
		b.appendLine(domain.LineContext, line[1:])
		return 1, nil
	case '+':
		b.appendLine(domain.LineAdded, line[1:])
		return 1, nil
	case '-':
		b.appendLine(domain.LineRemoved, line[1:])
		return 1, nil
	case '\\':
		// "\ No newline at end of file" does not consume a line number.
		return 1, nil
	default:
		// The hunk ended: the line belongs to the file section.
		b.inHunk = false
		return 0, nil
	}
}

// appendLine adds a line to the current hunk and advances the line counters.
func (b *fileBuilder) appendLine(kind, content string) {
	if b.hunk == nil {
		return
	}
	line := domain.DiffLine{Kind: kind, Content: content}
	switch kind {
	case domain.LineContext:
		line.OldLine = b.oldLine
		line.NewLine = b.newLine
		b.oldLine++
		b.newLine++
	case domain.LineAdded:
		line.NewLine = b.newLine
		b.newLine++
	case domain.LineRemoved:
		line.OldLine = b.oldLine
		b.oldLine++
	}
	b.hunk.Lines = append(b.hunk.Lines, line)
}

// build finalizes the file section into a domain.FileChange. It returns nil
// when the section carries no information at all.
func (b *fileBuilder) build() *domain.FileChange {
	if b.hunk != nil {
		b.hunks = append(b.hunks, *b.hunk)
		b.hunk = nil
	}
	path := b.newPath
	if path == "" {
		path = b.oldPath
	}
	if path == "" && b.status == "" && !b.binary {
		return nil
	}
	change := domain.FileChange{
		Path:     path,
		Status:   b.status,
		Language: LanguageForPath(path),
		IsBinary: b.binary,
		Hunks:    b.hunks,
	}
	if change.Status == "" {
		change.Status = domain.FileModified
	}
	if b.oldPath != "" && b.oldPath != path {
		change.OldPath = b.oldPath
	}
	return &change
}

// unquotePath decodes a path quoted by git in the C style used by
// core.quotePath. Unquoted paths are returned unchanged.
func unquotePath(path string) string {
	if len(path) < 2 || path[0] != '"' || path[len(path)-1] != '"' {
		return path
	}
	body := path[1 : len(path)-1]
	var b strings.Builder
	b.Grow(len(body))
	for i := 0; i < len(body); {
		c := body[i]
		if c != '\\' {
			b.WriteByte(c)
			i++
			continue
		}
		i++
		if i >= len(body) {
			b.WriteByte('\\')
			break
		}
		switch body[i] {
		case 'a':
			b.WriteByte('\a')
			i++
		case 'b':
			b.WriteByte('\b')
			i++
		case 'f':
			b.WriteByte('\f')
			i++
		case 'n':
			b.WriteByte('\n')
			i++
		case 'r':
			b.WriteByte('\r')
			i++
		case 't':
			b.WriteByte('\t')
			i++
		case 'v':
			b.WriteByte('\v')
			i++
		case '"':
			b.WriteByte('"')
			i++
		case '\\':
			b.WriteByte('\\')
			i++
		default:
			if body[i] >= '0' && body[i] <= '7' && i+3 <= len(body) {
				if value, err := strconv.ParseUint(body[i:i+3], 8, 8); err == nil {
					b.WriteByte(byte(value))
					i += 3
					continue
				}
			}
			b.WriteByte('\\')
			b.WriteByte(body[i])
			i++
		}
	}
	return b.String()
}
