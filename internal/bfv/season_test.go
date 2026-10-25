package bfv

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func TestFetchSeason(t *testing.T) {
	c := serve(t, map[string]string{
		"/ergebnisse/wettbewerb/-/" + testCID:                 pageHTML(2, 3, id("M3"), id("BYE")),
		"/ergebnisse/wettbewerb/-/" + testCID + "/spieltag/1": pageHTML(1, 3, id("M1"), id("M2")),
		"/ergebnisse/wettbewerb/-/" + testCID + "/spieltag/3": pageHTML(3, 3, id("M4")),
		// M1 of matchday 1 was postponed past matchday 2; the widget lists
		// matches by date.
		"/widget/team/B/matches": fmt.Sprintf(`{"state":0,"data":{
			"team":{"name":"Bravo","compoundId":%q,"competitionName":"Test Liga"},
			"matches":[
			{"matchId":"FRIENDLY","compoundId":"OTHER","kickoffDate":"01.07.2026","kickoffTime":"18:00","homeTeamName":"Bravo","guestTeamName":"Zulu"},
			{"matchId":%q,"compoundId":%q,"kickoffDate":"18.10.2026","homeTeamName":"Bravo","guestTeamName":"SPIELFREI"},
			{"matchId":%q,"compoundId":%q,"kickoffDate":"21.10.2026","kickoffTime":"19:00","homeTeamName":"Alpha","guestTeamName":"Bravo","result":"0:4"},
			{"matchId":%q,"compoundId":%q,"kickoffDate":"25.10.2026","kickoffTime":"15:00","homeTeamName":"Bravo","guestTeamName":"Alpha"}]}}`,
			testCID, id("BYE"), testCID, id("M1"), testCID, id("M4"), testCID),
	})
	s, err := c.FetchSeason(context.Background(), "B")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, m := range s.Matches {
		got = append(got, fmt.Sprintf("%d %s-%s", m.Matchday, m.Home, m.Guest))
	}
	want := "1 Alpha-Bravo, 2 Bravo-SPIELFREI, 3 Bravo-Alpha"
	if s.Team != "Bravo" || s.League != "Test Liga" || strings.Join(got, ", ") != want {
		t.Errorf("Season = %q %q %s; want Bravo, Test Liga, %s", s.Team, s.League, strings.Join(got, ", "), want)
	}
}

func TestFetchSeasonWithoutLeague(t *testing.T) {
	c := serve(t, map[string]string{
		"/widget/team/F/matches": `{"state":0,"data":{"team":{"name":"Friends"},"matches":[]}}`,
	})
	if _, err := c.FetchSeason(context.Background(), "F"); err == nil {
		t.Error("expected an error for a team without a league")
	}
}

func TestFetchTeamInfo(t *testing.T) {
	c := serve(t, map[string]string{
		"/widget/team/A/matches": `{"state":0,"data":{"team":{"name":"FC Sturm Hauzenberg","compoundId":"` + testCID + `","competitionName":"Landesliga Mitte"},"matches":[]}}`,
		"/widget/team/F/matches": `{"state":0,"data":{"team":{"name":"Friends"},"matches":[]}}`,
	})
	info, err := c.FetchTeamInfo(context.Background(), "A")
	if err != nil || info != (TeamInfo{Name: "FC Sturm Hauzenberg", CompoundID: testCID, League: "Landesliga Mitte"}) {
		t.Errorf("FetchTeamInfo = %+v, %v", info, err)
	}
	if _, err := c.FetchTeamInfo(context.Background(), "F"); err == nil {
		t.Error("expected an error for a team without a league")
	}
}
