package bfv

import (
	"os"
	"slices"
	"testing"
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
