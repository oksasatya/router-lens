package csv

import (
	"bytes"
	"strings"
	"testing"
)

func TestExporter(t *testing.T) {
	t.Run("escapes formula-injection cells", func(t *testing.T) {
		var buf bytes.Buffer
		w := NewWriter(&buf)
		if err := w.Write([]string{"=cmd()", "+1", "-2", "@x", "safe"}); err != nil {
			t.Fatalf("write: %v", err)
		}
		if err := w.Flush(); err != nil {
			t.Fatalf("flush: %v", err)
		}
		out := buf.String()
		for _, dangerous := range []string{"'=cmd()", "'+1", "'-2", "'@x"} {
			if !strings.Contains(out, dangerous) {
				t.Fatalf("expected %q escaped in output: %q", dangerous, out)
			}
		}
		if !strings.Contains(out, "safe") || strings.Contains(out, "'safe") {
			t.Fatalf("safe cell must not be escaped: %q", out)
		}
	})
}
