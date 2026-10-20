package ui

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

// Table renders a set of rows in aligned columns.
//
// The table is a presentation helper: it never changes the content of a
// cell and it never truncates it, so a long description stays readable.
type Table struct {
	headers []string
	rows    [][]string
}

// NewTable builds a table with the given headers.
func NewTable(headers ...string) *Table {
	return &Table{headers: append([]string(nil), headers...)}
}

// Append adds a row to the table.
//
// The missing cells are filled with empty strings and the extra cells are
// dropped, so a malformed row never breaks the layout.
func (t *Table) Append(cells ...string) {
	row := make([]string, len(t.headers))
	for i := range row {
		if i < len(cells) {
			row[i] = cells[i]
		}
	}
	t.rows = append(t.rows, row)
}

// Rows returns the number of rows of the table.
func (t *Table) Rows() int { return len(t.rows) }

// Render writes the table to the given writer.
func (t *Table) Render(w io.Writer, r *Renderer) {
	widths := make([]int, len(t.headers))
	for i, header := range t.headers {
		widths[i] = utf8.RuneCountInString(header)
	}
	for _, row := range t.rows {
		for i, cell := range row {
			if length := utf8.RuneCountInString(cell); length > widths[i] {
				widths[i] = length
			}
		}
	}

	if len(t.headers) > 0 {
		fmt.Fprintln(w, r.Title(t.join(t.headers, widths)))
		fmt.Fprintln(w, r.Dim(t.separator(widths)))
	}
	for _, row := range t.rows {
		fmt.Fprintln(w, t.join(row, widths))
	}
}

// join renders a row with the widths of the columns.
func (t *Table) join(cells []string, widths []int) string {
	parts := make([]string, 0, len(cells))
	for i, cell := range cells {
		if i == len(cells)-1 {
			parts = append(parts, cell)
			continue
		}
		padding := widths[i] - utf8.RuneCountInString(cell)
		if padding < 0 {
			padding = 0
		}
		parts = append(parts, cell+strings.Repeat(" ", padding))
	}
	return strings.TrimRight(strings.Join(parts, "  "), " ")
}

// separator renders the rule that separates the header from the rows.
func (t *Table) separator(widths []int) string {
	parts := make([]string, 0, len(widths))
	for _, width := range widths {
		if width < 3 {
			width = 3
		}
		parts = append(parts, strings.Repeat("-", width))
	}
	return strings.Join(parts, "  ")
}
