// Package policies loads and validates the configuration of the analyzed
// repository.
//
// The configuration lives in the ".codixai.yml" file of the repository root.
// A missing file is not an error: the documented defaults are returned. A
// malformed file is an error and the message always names the file and the
// offending line.
//
// The package has no third party dependencies, so it ships a small YAML
// parser that understands only the subset of YAML the configuration needs:
//
//   - comments, both on their own line and at the end of a line;
//   - "key: value" entries, with the value on the same line;
//   - nested mappings defined by indentation;
//   - sequences written as block items ("- item") or inline ("[a, b, c]");
//   - sequences of mappings ("- rule: CDX-001" plus indented sibling keys);
//   - single and double quoted scalars.
//
// The parser does not support anchors, aliases, tags, multiple documents,
// flow mappings, block scalars ("|" and ">"), merge keys or complex keys.
// A document using any of those features is rejected with a clear error
// instead of being silently misinterpreted.
package policies

import (
	"errors"
	"fmt"
	"strings"
)

// yamlKind classifies a node of the parsed document.
type yamlKind int

const (
	yamlNull yamlKind = iota
	yamlScalar
	yamlMapping
	yamlSequence
)

// yamlNode is a node of the parsed document.
type yamlNode struct {
	kind  yamlKind
	line  int
	value string
	keys  []string
	items map[string]*yamlNode
	list  []*yamlNode
}

// get returns the child of a mapping node, or nil when it does not exist.
func (n *yamlNode) get(key string) *yamlNode {
	if n == nil || n.kind != yamlMapping {
		return nil
	}
	return n.items[key]
}

// yamlError describes a problem of the parsed document.
type yamlError struct {
	line    int
	message string
}

func (e *yamlError) Error() string {
	return fmt.Sprintf("line %d: %s", e.line, e.message)
}

// yamlLine is a significant line of the document.
type yamlLine struct {
	num    int
	indent int
	text   string
}

// yamlParser reads a document line by line.
type yamlParser struct {
	lines []yamlLine
	pos   int
}

// parseYAML parses a YAML document into a tree of nodes.
func parseYAML(data []byte) (*yamlNode, error) {
	lines, err := scanYAML(data)
	if err != nil {
		return nil, err
	}
	if len(lines) == 0 {
		return &yamlNode{kind: yamlMapping, items: map[string]*yamlNode{}}, nil
	}
	parser := &yamlParser{lines: lines}
	node, err := parser.parseValue(lines[0].indent)
	if err != nil {
		return nil, err
	}
	if parser.pos < len(parser.lines) {
		line := parser.lines[parser.pos]
		return nil, &yamlError{line: line.num, message: "unexpected content"}
	}
	return node, nil
}

// scanYAML removes the comments and the blank lines of a document and
// computes the indentation of every remaining line.
func scanYAML(data []byte) ([]yamlLine, error) {
	raw := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	lines := make([]yamlLine, 0, len(raw))
	for i, text := range raw {
		num := i + 1
		indent := 0
		for indent < len(text) {
			switch text[indent] {
			case ' ':
				indent++
				continue
			case '\t':
				return nil, &yamlError{line: num, message: "tabs are not allowed for indentation, use spaces"}
			}
			break
		}
		rest := text[indent:]
		stripped, err := stripComment(rest)
		if err != nil {
			return nil, &yamlError{line: num, message: err.Error()}
		}
		stripped = strings.TrimRight(stripped, " \t")
		if stripped == "" {
			continue
		}
		if stripped == "---" || stripped == "..." {
			// Document markers are ignored: the configuration has a
			// single document.
			continue
		}
		lines = append(lines, yamlLine{num: num, indent: indent, text: stripped})
	}
	return lines, nil
}

// stripComment removes the trailing comment of a line, ignoring the "#"
// characters that are inside a quoted scalar.
func stripComment(text string) (string, error) {
	inSingle, inDouble := false, false
	for i := 0; i < len(text); i++ {
		c := text[i]
		switch {
		case inDouble:
			if c == '\\' {
				i++
				continue
			}
			if c == '"' {
				inDouble = false
			}
		case inSingle:
			if c == '\'' {
				inSingle = false
			}
		case c == '"':
			inDouble = true
		case c == '\'':
			inSingle = true
		case c == '#':
			if i == 0 || text[i-1] == ' ' || text[i-1] == '\t' {
				return text[:i], nil
			}
		}
	}
	if inDouble || inSingle {
		return "", errors.New("unterminated quoted string")
	}
	return text, nil
}

// peek returns the next line without consuming it.
func (p *yamlParser) peek() (yamlLine, bool) {
	if p.pos >= len(p.lines) {
		return yamlLine{}, false
	}
	return p.lines[p.pos], true
}

// parseValue parses a block of the document at the given indentation.
func (p *yamlParser) parseValue(indent int) (*yamlNode, error) {
	line, ok := p.peek()
	if !ok {
		return &yamlNode{kind: yamlNull}, nil
	}
	if line.indent != indent {
		return nil, &yamlError{line: line.num, message: "unexpected indentation"}
	}
	if isSequenceEntry(line.text) {
		return p.parseSequence(indent)
	}
	return p.parseMapping(indent)
}

// isSequenceEntry reports whether a line starts a block sequence item.
func isSequenceEntry(text string) bool {
	return text == "-" || strings.HasPrefix(text, "- ")
}

// parseMapping parses a block mapping.
func (p *yamlParser) parseMapping(indent int) (*yamlNode, error) {
	node := &yamlNode{kind: yamlMapping, items: map[string]*yamlNode{}}
	if err := p.parseMappingInto(node, indent); err != nil {
		return nil, err
	}
	return node, nil
}

// parseMappingInto reads the entries of a block mapping into a node.
func (p *yamlParser) parseMappingInto(node *yamlNode, indent int) error {
	for {
		line, ok := p.peek()
		if !ok || line.indent != indent {
			return nil
		}
		if isSequenceEntry(line.text) {
			return nil
		}
		key, rest, ok := splitMappingEntry(line.text)
		if !ok {
			return &yamlError{line: line.num, message: fmt.Sprintf("invalid mapping entry %q", line.text)}
		}
		p.pos++
		child, err := p.parseChild(rest, indent, line.num)
		if err != nil {
			return err
		}
		if _, exists := node.items[key]; exists {
			return &yamlError{line: line.num, message: fmt.Sprintf("duplicate key %q", key)}
		}
		if node.line == 0 {
			node.line = line.num
		}
		node.keys = append(node.keys, key)
		node.items[key] = child
	}
}

// parseChild parses the value of a mapping entry: an inline value, an inline
// sequence or a nested block.
func (p *yamlParser) parseChild(rest string, indent, line int) (*yamlNode, error) {
	rest = strings.TrimSpace(rest)
	if rest != "" {
		if strings.HasPrefix(rest, "[") {
			return parseFlowSequence(rest, line)
		}
		return &yamlNode{kind: yamlScalar, line: line, value: unquoteScalar(rest)}, nil
	}
	next, ok := p.peek()
	if !ok || next.indent <= indent {
		return &yamlNode{kind: yamlNull, line: line}, nil
	}
	return p.parseValue(next.indent)
}

// parseSequence parses a block sequence.
func (p *yamlParser) parseSequence(indent int) (*yamlNode, error) {
	node := &yamlNode{kind: yamlSequence, list: []*yamlNode{}}
	for {
		line, ok := p.peek()
		if !ok || line.indent != indent || !isSequenceEntry(line.text) {
			return node, nil
		}
		rest := line.text[1:]
		trimmed := strings.TrimLeft(rest, " ")
		contentStart := 1 + (len(rest) - len(trimmed))
		content := strings.TrimRight(trimmed, " ")
		p.pos++
		if node.line == 0 {
			node.line = line.num
		}

		if content == "" {
			child, err := p.parseNestedBlock(indent, line.num)
			if err != nil {
				return nil, err
			}
			node.list = append(node.list, child)
			continue
		}
		if key, rest, ok := splitMappingEntry(content); ok {
			item := &yamlNode{kind: yamlMapping, line: line.num, items: map[string]*yamlNode{}}
			child, err := p.parseChild(rest, indent+contentStart, line.num)
			if err != nil {
				return nil, err
			}
			item.keys = append(item.keys, key)
			item.items[key] = child
			if err := p.parseMappingInto(item, indent+contentStart); err != nil {
				return nil, err
			}
			node.list = append(node.list, item)
			continue
		}
		if strings.HasPrefix(content, "[") {
			child, err := parseFlowSequence(content, line.num)
			if err != nil {
				return nil, err
			}
			node.list = append(node.list, child)
			continue
		}
		node.list = append(node.list, &yamlNode{kind: yamlScalar, line: line.num, value: unquoteScalar(content)})
	}
}

// parseNestedBlock parses the block that belongs to a sequence item without
// an inline value.
func (p *yamlParser) parseNestedBlock(indent, line int) (*yamlNode, error) {
	next, ok := p.peek()
	if !ok || next.indent <= indent {
		return &yamlNode{kind: yamlNull, line: line}, nil
	}
	return p.parseValue(next.indent)
}

// splitMappingEntry splits a line into its key and the rest of the line.
// The second return value is false when the line is not a mapping entry.
func splitMappingEntry(text string) (string, string, bool) {
	for i := 0; i < len(text); i++ {
		if text[i] != ':' {
			continue
		}
		if i+1 < len(text) && text[i+1] != ' ' && text[i+1] != '\t' {
			// Something like "http://example.com" is not a key.
			continue
		}
		key := strings.TrimSpace(text[:i])
		if !isPlainKey(key) {
			return "", "", false
		}
		return key, strings.TrimSpace(text[i+1:]), true
	}
	return "", "", false
}

// isPlainKey reports whether a key is a plain scalar key.
func isPlainKey(key string) bool {
	if key == "" {
		return false
	}
	for i := 0; i < len(key); i++ {
		c := key[i]
		switch {
		case c >= 'a' && c <= 'z':
		case c >= 'A' && c <= 'Z':
		case c >= '0' && c <= '9':
		case c == '_' || c == '-' || c == '.':
		default:
			return false
		}
	}
	return true
}

// parseFlowSequence parses an inline sequence such as "[a, b, c]".
func parseFlowSequence(text string, line int) (*yamlNode, error) {
	trimmed := strings.TrimSpace(text)
	if !strings.HasPrefix(trimmed, "[") {
		return nil, &yamlError{line: line, message: "invalid inline sequence"}
	}
	end := strings.LastIndex(trimmed, "]")
	if end < 0 {
		return nil, &yamlError{line: line, message: "unterminated inline sequence"}
	}
	node := &yamlNode{kind: yamlSequence, line: line, list: []*yamlNode{}}
	for _, part := range splitFlowItems(trimmed[1:end]) {
		item := strings.TrimSpace(part)
		if item == "" {
			continue
		}
		node.list = append(node.list, &yamlNode{kind: yamlScalar, line: line, value: unquoteScalar(item)})
	}
	return node, nil
}

// splitFlowItems splits the items of an inline sequence on the commas that
// are not inside a quoted scalar.
func splitFlowItems(text string) []string {
	var items []string
	var current strings.Builder
	inSingle, inDouble := false, false
	for i := 0; i < len(text); i++ {
		c := text[i]
		switch {
		case inDouble:
			if c == '\\' && i+1 < len(text) {
				current.WriteByte(c)
				i++
				current.WriteByte(text[i])
				continue
			}
			if c == '"' {
				inDouble = false
			}
		case inSingle:
			if c == '\'' {
				inSingle = false
			}
		case c == '"':
			inDouble = true
		case c == '\'':
			inSingle = true
		case c == ',':
			items = append(items, current.String())
			current.Reset()
			continue
		}
		current.WriteByte(c)
	}
	items = append(items, current.String())
	return items
}

// unquoteScalar removes the quotes of a scalar and resolves its escapes.
func unquoteScalar(value string) string {
	if len(value) >= 2 && value[0] == '\'' && value[len(value)-1] == '\'' {
		return strings.ReplaceAll(value[1:len(value)-1], "''", "'")
	}
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		body := value[1 : len(value)-1]
		var b strings.Builder
		for i := 0; i < len(body); i++ {
			if body[i] != '\\' || i+1 >= len(body) {
				b.WriteByte(body[i])
				continue
			}
			i++
			switch body[i] {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case 'r':
				b.WriteByte('\r')
			case '"':
				b.WriteByte('"')
			case '\\':
				b.WriteByte('\\')
			default:
				b.WriteByte('\\')
				b.WriteByte(body[i])
			}
		}
		return b.String()
	}
	return value
}
