package render

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/sphrilix/stadionzeitung-automation/internal/bfv"
)

func TestRenderClubMatches(t *testing.T) {
	kick := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	match := func(teamType, comp, home string) bfv.ClubMatch {
		return bfv.ClubMatch{TeamType: teamType, Competition: comp,
			Match: bfv.Match{Kickoff: kick, HasTime: true, Home: home, Guest: "FC Sturm Hauzenberg", Result: "2:3"}}
	}
	ms := []bfv.ClubMatch{
		match("A-Junioren", "U 19 (A-Jun.) BOL Niederbayern", "FC Fürstenzell"),
		match("D-Junioren", "(U13) D-Junioren Bezirksliga", "JFG Straubing-Bogen Stadt und Land e.V."),
		match("D-Junioren", "(U13) D-Junioren Früchte Eder-Cup - Runde 3", "(SG) DJK Holzfreyung"),
		match("D-Junioren", "(U13) D-Junioren Bezirksliga", "VfB Passau-Grubweg"),
	}
	if g := groupClubMatches(ms); len(g) != 3 || len(g[1]) != 2 {
		t.Errorf("groups = %v, want 3 with the two Bezirksliga matches together", g)
	}
	var buf bytes.Buffer
	if err := RenderClubMatches(&buf, ms, true, Fonts{}); err != nil {
		t.Fatal(err)
	}
	wantH := (2*edge + 2*boxPadding + 2*cGroupGap + 3*cGroupHeight + 4*cRowHeight) * 72 / 25.4
	wantBox := fmt.Sprintf("/MediaBox [0 0 %.2f %.2f]", (boxWidth+2*edge)*72/25.4, wantH)
	if !bytes.Contains(buf.Bytes(), []byte(wantBox)) {
		t.Errorf("PDF has no %q", wantBox)
	}
	// A week without matches still gives a PDF to link.
	if err := RenderClubMatches(&buf, nil, false, Fonts{}); err != nil {
		t.Errorf("no matches: %v", err)
	}
}
