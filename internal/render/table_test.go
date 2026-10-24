package render

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"testing"

	"github.com/sphrilix/stadionzeitung-automation/internal/bfv"
)

func TestRenderTable(t *testing.T) {
	var logo bytes.Buffer
	if err := png.Encode(&logo, image.NewGray(image.Rect(0, 0, 4, 4))); err != nil {
		t.Fatal(err)
	}
	tbl := bfv.Table{
		League: "Landesliga Mitte",
		Zones:  bfv.Zones{Promotion: 1, Relegation: 1},
		Rows: []bfv.Standing{
			{Position: 1, Team: "FC Dingolfing", ClubID: "D", Played: 15, Won: 10, Drawn: 3, Lost: 2, GoalsFor: 36, GoalsAgainst: 15, HasGoals: true, GoalDiff: 21, Points: 33},
			{Position: 2, Team: "FC Sturm Hauzenberg", ClubID: "H", Played: 15, Won: 8, Drawn: 4, Lost: 3, GoalDiff: 23, Points: 28},
			{Position: 3, Team: "SV Grainet", Played: 15, Won: 2, Drawn: 4, Lost: 9, GoalsFor: 12, GoalsAgainst: 27, HasGoals: true, GoalDiff: -15, Points: 10},
		},
	}
	opts := TableOptions{
		// A broken logo is skipped instead of failing the PDF.
		Logos:     map[string][]byte{"D": logo.Bytes(), "H": []byte("not a png")},
		Highlight: "Hauzenberg",
	}
	var buf bytes.Buffer
	if err := RenderTable(&buf, tbl, Fonts{}, opts); err != nil {
		t.Fatal(err)
	}
	wantH := (2*edge + tHeaderHeight + 3*tRowHeight + 2*boxPadding) * 72 / 25.4
	wantBox := fmt.Sprintf("/MediaBox [0 0 %.2f %.2f]", (boxWidth+2*edge)*72/25.4, wantH)
	if !bytes.Contains(buf.Bytes(), []byte(wantBox)) {
		t.Errorf("PDF has no %q", wantBox)
	}
	if n := bytes.Count(buf.Bytes(), []byte("/Subtype /Image")); n != 1 {
		t.Errorf("PDF embeds %d images, want the one valid logo", n)
	}
}
