package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
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

func TestPreviewStart(t *testing.T) {
	const team = "016PE85GAS000000VV0AG811VTE5EA5R"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/widget/team/"+team+"/matches" {
			http.NotFound(w, r)
			return
		}
		fmt.Fprintf(w, `{"state":0,"data":{"team":{"compoundId":"L"},"matches":[
			{"matchId":"A","compoundId":"L","kickoffDate":"31.10.2026","kickoffTime":"14:00","homeTeamName":"X","guestTeamName":"H"},
			{"matchId":"B","compoundId":"L","kickoffDate":"07.11.2026","kickoffTime":"14:00","homeTeamName":"H","guestTeamName":"Y"}]}}`)
	}))
	defer srv.Close()
	berlin, _ := time.LoadLocation("Europe/Berlin")
	kickoff := time.Date(2026, 11, 7, 14, 0, 0, 0, berlin)
	for _, tc := range []struct {
		name      string
		now, want time.Time
	}{
		{"before the match: its kickoff", time.Date(2026, 11, 4, 20, 0, 0, 0, berlin), kickoff},
		{"after kickoff: now", time.Date(2026, 11, 7, 15, 0, 0, 0, berlin), time.Date(2026, 11, 7, 15, 0, 0, 0, berlin)},
	} {
		c := &bfv.Client{HTTP: srv.Client(), BaseWidget: srv.URL + "/widget", Now: func() time.Time { return tc.now }}
		if got := previewStart(context.Background(), c, team, tc.now); !got.Equal(tc.want) {
			t.Errorf("%s: %v, want %v", tc.name, got, tc.want)
		}
	}
	// Without the team's matches the preview starts now.
	c := &bfv.Client{HTTP: srv.Client(), BaseWidget: srv.URL + "/widget"}
	now := time.Date(2026, 11, 4, 20, 0, 0, 0, berlin)
	if got := previewStart(context.Background(), c, "016PE85GAS000000VV0AG811VTE5EAXX", now); !got.Equal(now) {
		t.Errorf("unknown team: %v, want now", got)
	}
}

func TestYouthWindowKeepsWeekendsTogether(t *testing.T) {
	day := func(d int) time.Time { return time.Date(2026, 11, d, 0, 0, 0, 0, time.UTC) }
	for _, tc := range []struct {
		name    string
		today   time.Time
		wantEnd time.Time
	}{
		// From Saturday 21.11., the week would end after Saturday 28.11.
		{"from a Saturday", day(21), day(30)},
		{"from a Thursday", day(19), day(27)},
	} {
		if _, to := youthWindow(tc.today, tc.today.Add(14*time.Hour), 7, false); !to.Equal(tc.wantEnd) {
			t.Errorf("%s: preview ends %v, want %v", tc.name, to, tc.wantEnd)
		}
	}
}
