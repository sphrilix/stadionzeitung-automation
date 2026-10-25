package render

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"testing"

	"github.com/sphrilix/stadionzeitung-automation/internal/bfv"
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

func TestRenderGuestPage(t *testing.T) {
	var photo bytes.Buffer
	if err := jpeg.Encode(&photo, image.NewGray(image.Rect(0, 0, 160, 90)), nil); err != nil {
		t.Fatal(err)
	}
	g := GuestPage{
		Name:  "SC Luhe-Wildenau",
		Photo: photo.Bytes(), PhotoCredit: "Robert Geisler",
		Rows: []Row{
			{Label: "Trainer:", Text: "Benjamin Urban"},
			{Label: "Torjäger:", Text: "N. Argauer (7 Tore)\nD. Bezdicka (6 Tore)", Gap: true},
		},
	}
	var with, without bytes.Buffer
	if err := RenderGuestPage(&with, g, Fonts{}); err != nil {
		t.Fatal(err)
	}
	if n := bytes.Count(with.Bytes(), []byte("/Subtype /Image")); n != 1 {
		t.Errorf("PDF embeds %d images, want the photo", n)
	}
	// A broken download leaves the photo out instead of failing the page.
	g.Photo = []byte("not a jpeg")
	if err := RenderGuestPage(&without, g, Fonts{}); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(without.Bytes(), []byte("/Subtype /Image")) {
		t.Error("broken photo was embedded")
	}
	// The photo is 160 mm wide at 16:9, so the page is 90 mm taller.
	if d := pageHeight(t, with.Bytes()) - pageHeight(t, without.Bytes()); d < 90*72/25.4 {
		t.Errorf("photo adds %.1f pt, want at least 90 mm", d)
	}
}

func TestRenderTableScorers(t *testing.T) {
	tbl := bfv.Table{Rows: []bfv.Standing{{Position: 1, Team: "FC Dingolfing", Played: 15}}}
	var plain, combined bytes.Buffer
	if err := RenderTable(&plain, tbl, Fonts{}, TableOptions{}); err != nil {
		t.Fatal(err)
	}
	scorers := []string{"Kai Dirmeier (ASV Burglengenfeld, 13 Tore)", "Almir Mesanovic (SV Türk Gücü Straubing, 12 Tore)"}
	if err := RenderTableScorers(&combined, tbl, TableOptions{}, scorers, Fonts{}); err != nil {
		t.Fatal(err)
	}
	// Two scorer lines and the gap below the table.
	want := (blockGap + 2*textLine) * 72 / 25.4
	if d := pageHeight(t, combined.Bytes()) - pageHeight(t, plain.Bytes()); d < want-0.1 || d > want+0.1 {
		t.Errorf("scorers add %.2f pt, want %.2f", d, want)
	}
}
