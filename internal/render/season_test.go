package render

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/sphrilix/stadionzeitung-automation/internal/bfv"
)

func TestRenderSeason(t *testing.T) {
	kick := time.Date(2026, 7, 18, 15, 0, 0, 0, time.UTC)
	s := bfv.Season{Team: "FC Sturm Hauzenberg", League: "Landesliga Mitte", Matches: []bfv.SeasonMatch{
		{Matchday: 1, Match: bfv.Match{Kickoff: kick, HasTime: true, Home: "FC Sturm Hauzenberg", Guest: "SV Türk Gücü Straubing", Result: "1:1"}},
		{Matchday: 2, Match: bfv.Match{Kickoff: kick.AddDate(0, 0, 7), Home: "FC Sturm Hauzenberg", Guest: "SPIELFREI"}},
		{Matchday: 3, Match: bfv.Match{Kickoff: kick.AddDate(0, 0, 14), HasTime: true, Home: "DJK Vornbach", Guest: "FC Sturm Hauzenberg"}},
	}}
	var buf bytes.Buffer
	if err := RenderSeason(&buf, s, Fonts{}); err != nil {
		t.Fatal(err)
	}
	wantH := (2*edge + tHeaderHeight + 3*sRowHeight + 2*boxPadding) * 72 / 25.4
	wantBox := fmt.Sprintf("/MediaBox [0 0 %.2f %.2f]", (boxWidth+2*edge)*72/25.4, wantH)
	if !bytes.Contains(buf.Bytes(), []byte(wantBox)) {
		t.Errorf("PDF has no %q", wantBox)
	}
}
