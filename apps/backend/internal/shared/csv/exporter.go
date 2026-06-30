// Package csv writes streaming, spreadsheet-injection-safe CSV.
package csv

import (
	stdcsv "encoding/csv"
	"io"
	"strings"
)

const injectionPrefixes = "=+-@"

// Writer wraps encoding/csv.Writer and escapes formula-injection-prone cells.
type Writer struct {
	w *stdcsv.Writer
}

// NewWriter returns a Writer that writes to w.
func NewWriter(w io.Writer) *Writer { return &Writer{w: stdcsv.NewWriter(w)} }

// Write escapes any cell starting with =, +, -, or @ before passing to the
// underlying encoder. This prevents spreadsheet formula injection (CSV injection).
func (w *Writer) Write(record []string) error {
	safe := make([]string, len(record))
	for i, cell := range record {
		safe[i] = escapeCell(cell)
	}
	return w.w.Write(safe)
}

// Flush flushes any buffered data and returns any accumulated error.
func (w *Writer) Flush() error {
	w.w.Flush()
	return w.w.Error()
}

// escapeCell prefixes a single quote to any cell that starts with a character
// a spreadsheet would interpret as a formula.
func escapeCell(s string) string {
	if s != "" && strings.ContainsRune(injectionPrefixes, rune(s[0])) {
		return "'" + s
	}
	return s
}
