package bfv

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestFetchSquad(t *testing.T) {
	c := serve(t, map[string]string{
		"/widget/team/A/squad": `{"state":0,"data":{"public":true,"players":[
			{"player":{"name":"Joris Wagner"},"statistics":{"matchesPlayed":16,"minutesPlayed":1171,"goals":11}},
			{"player":{"name":"Florian Seibold"},"statistics":{"matchesPlayed":17,"goals":0}}]}}`,
		"/widget/team/P/squad": `{"state":0,"data":{"public":false,"players":null}}`,
	})
	players, err := c.FetchSquad(context.Background(), "A")
	if err != nil {
		t.Fatal(err)
	}
	if len(players) != 2 || players[0] != (Player{Name: "Joris Wagner", Matches: 16, Goals: 11}) {
		t.Errorf("FetchSquad = %+v", players)
	}
	if _, err := c.FetchSquad(context.Background(), "P"); !errors.Is(err, ErrSquadPrivate) {
		t.Errorf("private squad: err = %v, want ErrSquadPrivate", err)
	}
}

func TestNextOpponent(t *testing.T) {
	c := serve(t, map[string]string{
		"/widget/team/A/matches": fmt.Sprintf(`{"state":0,"data":{"team":{"compoundId":%q},"matches":[
			{"matchId":"M1","compoundId":%q,"kickoffDate":"10.10.2026","kickoffTime":"15:00","homeTeamName":"Alpha","homeTeamPermanentId":"A","guestTeamName":"Bravo","guestTeamPermanentId":"B","result":"1:0"},
			{"matchId":"F","compoundId":"OTHER","kickoffDate":"17.10.2026","kickoffTime":"11:00","homeTeamName":"Alpha","homeTeamPermanentId":"A","guestTeamName":"Zulu","guestTeamPermanentId":"Z"},
			{"matchId":"BYE","compoundId":%q,"kickoffDate":"17.10.2026","homeTeamName":"Alpha","homeTeamPermanentId":"A","guestTeamName":"SPIELFREI"},
			{"matchId":"M3","compoundId":%q,"kickoffDate":"24.10.2026","kickoffTime":"15:00","homeTeamName":"Delta","homeTeamPermanentId":"D","guestTeamName":"Alpha","guestTeamPermanentId":"A"},
			{"matchId":"M2","compoundId":%q,"kickoffDate":"18.10.2026","kickoffTime":"15:00","homeTeamName":"Alpha","homeTeamPermanentId":"A","guestTeamName":"Charlie","guestTeamPermanentId":"C"}]}}`,
			testCID, testCID, testCID, testCID, testCID),
	})
	for _, tc := range []struct {
		now  time.Time
		want Team
	}{
		// The bye and the friendly are skipped; M2 comes first despite being
		// listed last.
		{time.Date(2026, 10, 17, 9, 0, 0, 0, berlin), Team{ID: "C", Name: "Charlie"}},
		// On match day, after kickoff, the day's opponent still counts.
		{time.Date(2026, 10, 18, 18, 0, 0, 0, berlin), Team{ID: "C", Name: "Charlie"}},
		{time.Date(2026, 10, 19, 0, 30, 0, 0, berlin), Team{ID: "D", Name: "Delta"}},
	} {
		c.Now = func() time.Time { return tc.now }
		got, err := c.NextOpponent(context.Background(), "A")
		if err != nil || got != tc.want {
			t.Errorf("NextOpponent at %v = %+v, %v; want %+v", tc.now, got, err, tc.want)
		}
	}
	c.Now = func() time.Time { return time.Date(2026, 10, 25, 0, 0, 0, 0, berlin) }
	if _, err := c.NextOpponent(context.Background(), "A"); err == nil {
		t.Error("expected an error after the last match")
	}
}

func TestTopScorers(t *testing.T) {
	players := []Player{
		{Name: "Bastian Schmid", Goals: 4}, {Name: "Joris Wagner", Goals: 10},
		{Name: "Florian Seibold"}, {Name: "Luca Canales Uhrmann", Goals: 4},
		{Name: "Manuel Mader", Goals: 8}, {Name: "Philipp Michl", Goals: 2},
	}
	var got []string
	for _, p := range TopScorers(players, 3) {
		got = append(got, p.Name)
	}
	// Ties with the third are included, ordered by name.
	want := "[Joris Wagner Manuel Mader Bastian Schmid Luca Canales Uhrmann]"
	if fmt.Sprint(got) != want {
		t.Errorf("TopScorers = %v, want %s", got, want)
	}
	if top := TopScorers([]Player{{Name: "Nobody"}}, 3); len(top) != 0 {
		t.Errorf("TopScorers without goals = %+v, want none", top)
	}
}
