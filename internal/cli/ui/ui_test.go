package ui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/barceloch/codixai/internal/domain"
)

// ansiSequence detects an ANSI escape sequence in a text.
func ansiSequence(text string) bool {
	return strings.Contains(text, "\x1b[")
}

func TestRendererDisablesColorWhenTheDestinationIsNotATerminal(t *testing.T) {
	renderer := NewRenderer(Options{TTY: false})
	if renderer.ColorEnabled() {
		t.Error("the color must be disabled when the destination is not a terminal")
	}
	if got := renderer.Severity(domain.SeverityError); ansiSequence(got) {
		t.Errorf("the severity label must not carry an escape sequence, got %q", got)
	}
}

func TestRendererDisablesColorWithTheNoColorFlag(t *testing.T) {
	renderer := NewRenderer(Options{TTY: true, NoColor: true})
	if renderer.ColorEnabled() {
		t.Error("the color must be disabled when the no color flag is set")
	}
	if got := renderer.Title("title"); ansiSequence(got) {
		t.Errorf("the title must not carry an escape sequence, got %q", got)
	}
	if got := renderer.Dim("text"); ansiSequence(got) {
		t.Errorf("the dim text must not carry an escape sequence, got %q", got)
	}
}

func TestRendererDisablesColorWithTheNoColorEnvironmentVariable(t *testing.T) {
	for _, value := range []string{"1", "true", "no", " "} {
		renderer := NewRenderer(Options{TTY: true, NoColorEnv: value})
		if renderer.ColorEnabled() {
			t.Errorf("the color must be disabled when NO_COLOR holds %q", value)
		}
	}
}

func TestRendererKeepsTheColorWithAnEmptyNoColorEnvironmentVariable(t *testing.T) {
	renderer := NewRenderer(Options{TTY: true})
	if !renderer.ColorEnabled() {
		t.Error("the color must be enabled in an interactive terminal")
	}
	if got := renderer.Severity(domain.SeverityError); !ansiSequence(got) {
		t.Errorf("the severity label must be painted in an interactive terminal, got %q", got)
	}
}

func TestPlainModeRemovesEveryDecoration(t *testing.T) {
	renderer := NewRenderer(Options{TTY: true, Plain: true})
	if renderer.ColorEnabled() {
		t.Error("the color must be disabled in plain mode")
	}
	if renderer.DecorEnabled() {
		t.Error("the decoration must be disabled in plain mode")
	}
	for name, painted := range map[string]string{
		"title":    renderer.Title("title"),
		"bold":     renderer.Bold("bold"),
		"dim":      renderer.Dim("dim"),
		"path":     renderer.Path("path"),
		"severity": renderer.Severity(domain.SeverityWarning),
	} {
		if ansiSequence(painted) {
			t.Errorf("the %s must not carry an escape sequence in plain mode, got %q", name, painted)
		}
	}
}

func TestNoColorRemovesEveryEscapeSequence(t *testing.T) {
	renderer := NewRenderer(Options{TTY: true, NoColor: true})
	if renderer.ColorEnabled() {
		t.Error("the color must be disabled by the no color flag")
	}
	for name, painted := range map[string]string{
		"title":    renderer.Title("title"),
		"bold":     renderer.Bold("bold"),
		"dim":      renderer.Dim("dim"),
		"path":     renderer.Path("path"),
		"severity": renderer.Severity(domain.SeverityError),
	} {
		if ansiSequence(painted) {
			t.Errorf("the %s must not carry an escape sequence with the no color flag, got %q", name, painted)
		}
	}
}

func TestSeverityLabelIsAlwaysTextual(t *testing.T) {
	cases := map[domain.Severity]string{
		domain.SeverityError:   "ERROR",
		domain.SeverityWarning: "WARNING",
		domain.SeverityInfo:    "INFO",
		"":                     "UNKNOWN",
	}
	for severity, want := range cases {
		if got := SeverityLabel(severity); got != want {
			t.Errorf("SeverityLabel(%q) = %q, want %q", severity, got, want)
		}
	}
}

func TestSeverityLabelSurvivesTheColoredRenderer(t *testing.T) {
	renderer := NewRenderer(Options{TTY: true})
	for _, severity := range []domain.Severity{domain.SeverityError, domain.SeverityWarning, domain.SeverityInfo} {
		painted := renderer.Severity(severity)
		if !strings.Contains(painted, SeverityLabel(severity)) {
			t.Errorf("the painted severity %q must contain the label %q", painted, SeverityLabel(severity))
		}
	}
}

func TestTableAlignsTheColumns(t *testing.T) {
	table := NewTable("RULE", "SEVERITY")
	table.Append("CDX-SUP-001", "warning")
	table.Append("CDX-IMP-001", "error")

	var buffer bytes.Buffer
	table.Render(&buffer, NewRenderer(Options{}))
	lines := strings.Split(strings.TrimRight(buffer.String(), "\n"), "\n")
	if len(lines) != 4 {
		t.Fatalf("expected a header, a separator and two rows, got %d lines: %q", len(lines), buffer.String())
	}
	column := strings.Index(lines[0], "SEVERITY")
	if column < 0 {
		t.Fatalf("the header does not hold the second column: %q", lines[0])
	}
	if !strings.HasPrefix(lines[2], "CDX-SUP-001") {
		t.Errorf("unexpected first row %q", lines[2])
	}
	if !strings.HasPrefix(lines[3], "CDX-IMP-001") {
		t.Errorf("unexpected second row %q", lines[3])
	}
	if got := strings.Index(lines[2], "warning"); got != column {
		t.Errorf("the second column starts at %d, want %d", got, column)
	}
}

func TestTableDoesNotEmitEscapeSequencesWhenTheColorIsDisabled(t *testing.T) {
	table := NewTable("RULE")
	table.Append("CDX-SUP-001")
	var buffer bytes.Buffer
	table.Render(&buffer, NewRenderer(Options{}))
	if ansiSequence(buffer.String()) {
		t.Errorf("the table must not carry an escape sequence, got %q", buffer.String())
	}
}

func TestIsTerminalRejectsABuffer(t *testing.T) {
	if IsTerminal(&bytes.Buffer{}) {
		t.Error("a buffer is not a terminal")
	}
}

func TestOptionsForReadsTheNoColorEnvironmentVariable(t *testing.T) {
	lookup := func(name string) string {
		if name == "NO_COLOR" {
			return "1"
		}
		return ""
	}
	options := OptionsFor(&bytes.Buffer{}, false, false, lookup)
	if options.NoColorEnv != "1" {
		t.Errorf("the NO_COLOR value was not read, got %q", options.NoColorEnv)
	}
	if NewRenderer(options).ColorEnabled() {
		t.Error("the color must be disabled when NO_COLOR is set")
	}
}
