// Package ui centralizes the presentation of the command line interface.
//
// The package is the only place where the CLI decides how something looks.
// It knows about colors, styles, tables and severity labels, and it knows
// nothing about the analysis: the commands render the values of the domain
// with it.
//
// The renderer has three modes:
//
//   - colored: ANSI codes are emitted, for an interactive terminal;
//   - plain: no color and no decoration, for a readable text output;
//   - disabled: the fallback used when the destination is not a terminal.
//
// The color is disabled automatically when the destination is not a
// terminal, when the NO_COLOR environment variable holds any non empty
// value and when the --no-color flag is passed. The --plain flag removes
// every decoration on top of that.
//
// The presentation is always a layer on top of the result: the text of a
// severity, of a rule identifier or of a path never changes because of a
// color. The JSON output is produced by the encoding/json package and
// never goes through the renderer, so it never carries an escape sequence.
package ui

import (
	"io"
	"os"
	"strings"

	"github.com/barceloch/codixai/internal/domain"
)

// The ANSI escape sequences used by the renderer.
const (
	reset       = "\x1b[0m"
	bold        = "\x1b[1m"
	dim         = "\x1b[2m"
	colorRed    = "\x1b[31m"
	colorGreen  = "\x1b[32m"
	colorYellow = "\x1b[33m"
	colorCyan   = "\x1b[36m"
	colorViolet = "\x1b[35m"
)

// Options configures a Renderer.
type Options struct {
	// NoColor disables every escape sequence.
	NoColor bool
	// Plain disables every decoration, the color included.
	Plain bool
	// TTY reports whether the destination is an interactive terminal.
	TTY bool
	// NoColorEnv holds the value of the NO_COLOR environment variable.
	// Any non empty value disables the escape sequences.
	NoColorEnv string
}

// OptionsFor derives the options of a renderer from a destination, the
// command line flags and an environment lookup.
//
// The lookup may be nil, in which case the environment is not inspected.
func OptionsFor(destination io.Writer, noColor, plain bool, lookup func(string) string) Options {
	value := ""
	if lookup != nil {
		value = lookup("NO_COLOR")
	}
	return Options{
		NoColor:    noColor,
		Plain:      plain,
		TTY:        IsTerminal(destination),
		NoColorEnv: value,
	}
}

// IsTerminal reports whether the destination is a character device.
//
// The check is deliberately conservative: anything that is not a file
// descriptor of the process is treated as a pipe, so a redirected output
// never receives an escape sequence.
func IsTerminal(destination io.Writer) bool {
	file, ok := destination.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

// Renderer renders the presentation of the CLI.
//
// A Renderer is safe for concurrent use: it never mutates its state.
type Renderer struct {
	ansi bool
}

// NewRenderer builds a renderer from its options.
//
// The escape sequences are emitted only when the destination is an
// interactive terminal and when neither the --no-color flag, the --plain
// flag nor the NO_COLOR environment variable ask for a plain output. A
// redirected output never receives an escape sequence.
func NewRenderer(options Options) *Renderer {
	return &Renderer{
		ansi: options.TTY && !options.NoColor && options.NoColorEnv == "" && !options.Plain,
	}
}

// ColorEnabled reports whether the renderer emits color codes.
func (r *Renderer) ColorEnabled() bool { return r.ansi }

// DecorEnabled reports whether the renderer emits decorations such as the
// bold or the dim attributes.
func (r *Renderer) DecorEnabled() bool { return r.ansi }

// style describes how a piece of text is painted.
type style struct {
	// color is the ANSI color code, emitted when the color is enabled.
	color string
	// code is an ANSI attribute code, emitted when the decoration is
	// enabled.
	code string
	// decor states whether the attribute code belongs to the decoration.
	decor bool
}

// The styles of the presentation.
var (
	titleStyle   = style{color: colorCyan, code: bold, decor: true}
	pathStyle    = style{color: colorViolet}
	dimStyle     = style{code: dim, decor: true}
	errorStyle   = style{color: colorRed, code: bold, decor: true}
	warningStyle = style{color: colorYellow, code: bold, decor: true}
	infoStyle    = style{color: colorGreen, code: bold, decor: true}
)

// paint applies a style to a text.
func (r *Renderer) paint(s style, text string) string {
	if text == "" {
		return text
	}
	var prefix strings.Builder
	if s.color != "" && r.ansi {
		prefix.WriteString(s.color)
	}
	if s.code != "" && s.decor && r.ansi {
		prefix.WriteString(s.code)
	}
	if prefix.Len() == 0 {
		return text
	}
	return prefix.String() + text + reset
}

// Title paints a title of a section.
func (r *Renderer) Title(text string) string { return r.paint(titleStyle, text) }

// Bold paints an emphasized text.
func (r *Renderer) Bold(text string) string {
	return r.paint(style{code: bold, decor: true}, text)
}

// Dim paints a secondary text.
func (r *Renderer) Dim(text string) string { return r.paint(dimStyle, text) }

// Path paints a repository relative path.
func (r *Renderer) Path(text string) string { return r.paint(pathStyle, text) }

// Severity paints the textual label of a severity.
//
// The label is always present: the color only emphasizes it.
func (r *Renderer) Severity(severity domain.Severity) string {
	return r.paint(styleOf(severity), SeverityLabel(severity))
}

// styleOf returns the style of a severity.
func styleOf(severity domain.Severity) style {
	switch severity {
	case domain.SeverityError:
		return errorStyle
	case domain.SeverityWarning:
		return warningStyle
	case domain.SeverityInfo:
		return infoStyle
	default:
		return dimStyle
	}
}

// SeverityLabel returns the textual label of a severity.
//
// The label never depends on the color: the severity of a finding has to
// stay understandable in a pipe, in a log file and in a screen reader.
func SeverityLabel(severity domain.Severity) string {
	switch severity {
	case domain.SeverityError:
		return "ERROR"
	case domain.SeverityWarning:
		return "WARNING"
	case domain.SeverityInfo:
		return "INFO"
	case "":
		return "UNKNOWN"
	default:
		return strings.ToUpper(string(severity))
	}
}

// SupportLabel returns the textual label of a support level.
func SupportLabel(level domain.SupportLevel) string {
	switch level {
	case domain.SupportSyntactic:
		return "syntactic"
	case domain.SupportResolved:
		return "resolved"
	case domain.SupportNone:
		return "none"
	case "":
		return "unknown"
	default:
		return string(level)
	}
}
