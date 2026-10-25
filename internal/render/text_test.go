package render

import (
	"bytes"
	"fmt"
	"testing"
)

func TestRenderRows(t *testing.T) {
	rows := []Row{
		{Label: "spielender Co-Trainer:", Text: "Tobias Richter"},
		{Label: "Abwehr:", Text: "Moritz Gerhartinger, Alexander Kretz, Nikolas Löblein, Philipp Michl, Philipp Reitberger, Philipp Schneider, Jonas Windpassinger, Nico Wipplinger", Gap: true},
		{Label: "Angriff:"},
	}
	var buf bytes.Buffer
	if err := RenderRows(&buf, rows, Fonts{}); err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(buf.Bytes(), []byte("%PDF-")) {
		t.Error("output is not a PDF")
	}
	// The long defenders' row wraps: the page is taller than one line per
	// row plus the gap.
	if minH := 2*edge + rowGap + 3*textLine; pageHeight(t, buf.Bytes()) <= minH*72/25.4 {
		t.Errorf("page is not taller than %v mm; the long row did not wrap", minH)
	}
}

func TestRenderLines(t *testing.T) {
	for _, lines := range [][]string{{"Marijo Kraljic (SC Wörnsmühl)"}, nil} {
		var buf bytes.Buffer
		if err := RenderLines(&buf, lines, Fonts{}); err != nil {
			t.Fatal(err)
		}
		if !bytes.HasPrefix(buf.Bytes(), []byte("%PDF-")) {
			t.Errorf("%d lines: output is not a PDF", len(lines))
		}
	}
}

// pageHeight returns the height in points from a PDF's MediaBox.
func pageHeight(t *testing.T, pdf []byte) float64 {
	t.Helper()
	i := bytes.Index(pdf, []byte("/MediaBox [0 0 "))
	if i < 0 {
		t.Fatal("no MediaBox")
	}
	var w, h float64
	if _, err := fmt.Sscan(string(pdf[i+len("/MediaBox [0 0 "):i+len("/MediaBox [0 0 ")+30]), &w, &h); err != nil {
		t.Fatal(err)
	}
	return h
}
