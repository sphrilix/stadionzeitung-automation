package render

import (
	"bytes"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/sphrilix/bfv-miner/internal/bfv"
)

func TestRender(t *testing.T) {
	kick := time.Date(2026, 10, 10, 15, 0, 0, 0, time.UTC)
	days := []bfv.Matchday{
		{Number: 15, Matches: []bfv.Match{
			{Kickoff: kick, HasTime: true, Home: "TSV 1883 Bogen", Guest: "SV Türk Gücü Straubing", Result: "0:4"},
			{Kickoff: kick, HasTime: true, Home: "1. FC Bad Kötzting", Guest: "TSV Seebach", Result: "1:1"},
		}},
		{Number: 16, Matches: []bfv.Match{
			{Kickoff: kick.AddDate(0, 0, 7), HasTime: true, Home: "1. FC Passau", Guest: "FC Amberg"},
		}},
	}
	var buf bytes.Buffer
	if err := Render(&buf, "Landesliga Mitte", days, Fonts{}); err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(buf.Bytes(), []byte("%PDF-")) || buf.Len() < 1000 {
		t.Errorf("output does not look like a PDF (%d bytes)", buf.Len())
	}
	// The page is cropped to the content: 2 boxes with 2 and 1 rows below the
	// title, in points (mm * 72/25.4).
	wantH := (2*edge + titleHeight + titleGap + boxHeight(days[0]) + boxGap + boxHeight(days[1])) * 72 / 25.4
	wantBox := fmt.Sprintf("/MediaBox [0 0 %.2f %.2f]", (boxWidth+2*edge)*72/25.4, wantH)
	if !bytes.Contains(buf.Bytes(), []byte(wantBox)) {
		t.Errorf("PDF has no %q", wantBox)
	}
}

func TestRenderRejectsInvalidFont(t *testing.T) {
	err := Render(io.Discard, "Liga", nil, Fonts{Regular: []byte("not a font"), Bold: []byte("not a font")})
	if err == nil {
		t.Error("expected an error for an invalid font")
	}
}

func TestFormatResult(t *testing.T) {
	for in, want := range map[string]string{"": "-", "0:4": "0 - 4", " 10 : 2 ": "10 - 2", "abgesetzt": "abgesetzt"} {
		if got := formatResult(in); got != want {
			t.Errorf("formatResult(%q) = %q, want %q", in, got, want)
		}
	}
}
