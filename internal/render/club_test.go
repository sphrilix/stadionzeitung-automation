package render

import (
	"bytes"
	"fmt"
	"math"
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
	// The page has a programme page's proportions, however many matches.
	wantBox := fmt.Sprintf("/MediaBox [0 0 %.2f %.2f]", (boxWidth+2*edge)*72/25.4, clubPageHeight*72/25.4)
	if !bytes.Contains(buf.Bytes(), []byte(wantBox)) {
		t.Errorf("PDF has no %q", wantBox)
	}
	// A week without matches still gives a PDF to link.
	if err := RenderClubMatches(&buf, nil, false, Fonts{}); err != nil {
		t.Errorf("no matches: %v", err)
	}
}

func TestLayoutClubPage(t *testing.T) {
	avail := clubPageHeight - 2*edge - 2*boxPadding
	groups := func(sizes ...int) [][]bfv.ClubMatch {
		var gs [][]bfv.ClubMatch
		for _, n := range sizes {
			gs = append(gs, make([]bfv.ClubMatch, n))
		}
		return gs
	}
	height := func(gs [][]bfv.ClubMatch, l clubLayout) float64 {
		h := float64(len(gs)-1) * l.gap
		for _, g := range gs {
			h += l.group + float64(len(g))*l.row
		}
		return h
	}
	for _, tc := range []struct {
		name  string
		sizes []int
	}{
		{"a quiet weekend", []int{3, 2, 4, 1, 2}},
		{"a usual weekend", []int{3, 2, 4, 1, 2, 3, 2}},
		{"too many for natural size", []int{10, 10, 10, 10}},
	} {
		gs := groups(tc.sizes...)
		l := layoutClubPage(gs)
		if h := height(gs, l); math.Abs(h-avail) > 1e-6 {
			t.Errorf("%s: content is %.2f mm, want the page's %.2f", tc.name, h, avail)
		}
		if l.scale > cMaxStretch {
			t.Errorf("%s: rows stretched %.2f times, more than %v", tc.name, l.scale, cMaxStretch)
		}
	}
	// With few matches the rows and gaps grow only so far; the rest stays
	// at the bottom.
	for _, sizes := range [][]int{{1}, {1, 2}} {
		gs := groups(sizes...)
		l := layoutClubPage(gs)
		if l.scale != cMaxStretch || l.gap > cMaxGap || height(gs, l) > avail {
			t.Errorf("groups %v: %+v, want rows at the most stretched, gaps at most %v and within the page", sizes, l, cMaxGap)
		}
	}
	if l := layoutClubPage(groups(10, 10, 10, 10)); l.scale >= 1 {
		t.Errorf("40 matches: scale %v, want them shrunk", l.scale)
	}
}
