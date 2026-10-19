package bfv

import (
	"os"
	"slices"
	"testing"
	"time"
)

func TestParsePage(t *testing.T) {
	html, err := os.ReadFile("testdata/spieltag13.html")
	if err != nil {
		t.Fatal(err)
	}
	p, err := parsePage(html)
	if err != nil {
		t.Fatal(err)
	}
	if p.LeagueName != "KL-Gr2 SW" {
		t.Errorf("LeagueName = %q, want %q", p.LeagueName, "KL-Gr2 SW")
	}
	if p.Matchday != 13 || p.MaxMatchday != 30 {
		t.Errorf("Matchday, MaxMatchday = %d, %d; want 13, 30", p.Matchday, p.MaxMatchday)
	}
	if len(p.Ends) != 30 || !p.Ends[1].Equal(time.Date(2026, 7, 19, 0, 0, 0, 0, berlin)) {
		t.Errorf("Ends has %d entries, matchday 1 ends %v; want 30, 19.07.2026", len(p.Ends), p.Ends[1])
	}
	want := []string{
		"0317T41PHS000000VS5489BUVSBBVPEU",
		"0317T41PIO000000VS5489BUVSBBVPEU",
		"0317T41PGS000000VS5489BUVSBBVPEU",
		"0317T41PJK000000VS5489BUVSBBVPEU",
		"0317T41PKK000000VS5489BUVSBBVPEU",
		"0317T41PMG000000VS5489BUVSBBVPEU",
		"0317T41PNG000000VS5489BUVSBBVPEU",
	}
	if !slices.Equal(p.MatchIDs, want) {
		t.Errorf("MatchIDs = %v, want %v", p.MatchIDs, want)
	}
}

func TestParsePageWithoutMatchday(t *testing.T) {
	if _, err := parsePage([]byte("<title>Pokal | BFV</title>")); err == nil {
		t.Error("expected an error for a page without matchdays")
	}
}

func TestRunningOrNext(t *testing.T) {
	// Matchdays on the weekends ending 11.10., 18.10. and 25.10.2026.
	day := func(m time.Month, d, h int) time.Time { return time.Date(2026, m, d, h, 0, 0, 0, berlin) }
	p := PageInfo{Matchday: 1, MaxMatchday: 3, Ends: map[int]time.Time{
		1: day(10, 11, 0), 2: day(10, 18, 0), 3: day(10, 25, 0),
	}}
	for _, tc := range []struct {
		now  time.Time
		want int
	}{
		{day(9, 1, 12), 1},   // before the season
		{day(10, 9, 19), 1},  // Friday of matchday 1
		{day(10, 11, 23), 1}, // last day of matchday 1, still running
		{day(10, 12, 0), 2},  // Monday after: next matchday
		{day(10, 19, 8), 3},
		{day(12, 1, 12), 3}, // after the season
		{time.Date(2026, 10, 11, 22, 30, 0, 0, time.UTC), 2}, // 00:30 on 12.10. in Berlin
	} {
		if got := p.RunningOrNext(tc.now); got != tc.want {
			t.Errorf("RunningOrNext(%v) = %d, want %d", tc.now, got, tc.want)
		}
	}
	if got := (PageInfo{Matchday: 7}).RunningOrNext(day(10, 12, 0)); got != 7 {
		t.Errorf("without schedule: RunningOrNext = %d, want the page's matchday 7", got)
	}
}
