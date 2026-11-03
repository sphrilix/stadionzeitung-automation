package main

import (
	"testing"
	"time"

	"github.com/sphrilix/stadionzeitung-automation/internal/bfv"
)

func TestFillYouthPage(t *testing.T) {
	now := time.Date(2026, 11, 3, 18, 0, 0, 0, time.UTC)
	today := time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)
	// n matches on each of the given days from today (negative: past).
	matches := func(n int, offsets ...int) []bfv.ClubMatch {
		var ms []bfv.ClubMatch
		for _, off := range offsets {
			for range n {
				ms = append(ms, bfv.ClubMatch{Match: bfv.Match{Kickoff: today.AddDate(0, 0, off).Add(14 * time.Hour)}})
			}
		}
		return ms
	}
	// The page holds 10 matches.
	fits := func(ms []bfv.ClubMatch) bool { return len(ms) <= 10 }
	for _, tc := range []struct {
		name     string
		ms       []bfv.ClubMatch
		results  bool
		wantN    int
		wantDays int
	}{
		{"busy week stays one week", matches(8, -3, -10), true, 8, 7},
		{"busy week overflowing stays one week", matches(12, -3, -10), true, 12, 7},
		{"quiet weeks add older results", matches(3, -3, -10, -17, -24), true, 9, 21},
		{"at most four weeks", matches(1, -3, -10, -17, -24, -27), true, 5, 28},
		{"quiet preview adds later fixtures", matches(4, 4, 11, 18), false, 8, 14},
		{"preview ignores the past", matches(4, -3, 4), false, 4, 28},
	} {
		got, days := fillYouthPage(tc.ms, today, now, tc.results, fits)
		if len(got) != tc.wantN || days != tc.wantDays {
			t.Errorf("%s: %d matches over %d days, want %d over %d", tc.name, len(got), days, tc.wantN, tc.wantDays)
		}
	}
}
