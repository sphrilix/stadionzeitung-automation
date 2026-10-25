package bfv

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestFetchClubMatches(t *testing.T) {
	c := serve(t, map[string]string{
		// Team links appear several times, once inside escaped flight data.
		"/vereine/-/CLUB": `<a href="/mannschaften/fc-test/` + id("A") + `">A-Jun.</a>
			<a href="/mannschaften/fc-test-2/` + id("D") + `">D-Jun.</a>
			<script>self.__next_f.push([1,"\"href\":\"/mannschaften/fc-test/` + id("A") + `\""])</script>`,
		"/widget/team/" + id("A") + "/matches": `{"state":0,"data":{"matches":[
			{"matchId":"OLD","teamType":"A-Junioren","competitionName":"U 19 BOL","kickoffDate":"01.10.2026","kickoffTime":"12:00","homeTeamName":"FC Test","guestTeamName":"X","result":"1:0"},
			{"matchId":"A1","teamType":"A-Junioren","competitionName":"U 19 BOL","kickoffDate":"10.10.2026","kickoffTime":"12:00","homeTeamName":"FC Fürstenzell","guestTeamName":"FC Test","result":"2:3"},
			{"matchId":"BYE","teamType":"A-Junioren","competitionName":"U 19 BOL","kickoffDate":"11.10.2026","homeTeamName":"FC Test","guestTeamName":"SPIELFREI"},
			{"matchId":"DERBY","teamType":"A-Junioren","competitionName":"U 19 BOL","kickoffDate":"11.10.2026","kickoffTime":"10:00","homeTeamName":"FC Test","guestTeamName":"FC Test 2","result":"4:4"}]}}`,
		"/widget/team/" + id("D") + "/matches": `{"state":0,"data":{"matches":[
			{"matchId":"D1","teamType":"D-Junioren","competitionName":"(U13) Bezirksliga","kickoffDate":"10.10.2026","kickoffTime":"18:00","homeTeamName":"FC Test 2","guestTeamName":"JFG Y","result":"0:0"},
			{"matchId":"DERBY","teamType":"A-Junioren","competitionName":"U 19 BOL","kickoffDate":"11.10.2026","kickoffTime":"10:00","homeTeamName":"FC Test","guestTeamName":"FC Test 2","result":"4:4"},
			{"matchId":"NEXT","teamType":"D-Junioren","competitionName":"(U13) Bezirksliga","kickoffDate":"17.10.2026","kickoffTime":"11:00","homeTeamName":"FC Test 2","guestTeamName":"JFG Z"}]}}`,
	})
	teams, err := c.ClubTeams(context.Background(), "CLUB")
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(teams) != fmt.Sprint([]string{id("A"), id("D")}) {
		t.Errorf("ClubTeams = %v, want A and D once each", teams)
	}

	from := time.Date(2026, 10, 9, 0, 0, 0, 0, berlin)
	to := time.Date(2026, 10, 12, 0, 0, 0, 0, berlin)
	ms, err := c.FetchClubMatches(context.Background(), "CLUB", from, to)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, m := range ms {
		got = append(got, m.ID+" "+m.TeamType+" "+m.Competition)
	}
	// Outside the window, byes and the derby's second copy are left out; A
	// comes before D.
	want := "A1 A-Junioren U 19 BOL, DERBY A-Junioren U 19 BOL, D1 D-Junioren (U13) Bezirksliga"
	if g := strings.Join(got, ", "); g != want {
		t.Errorf("FetchClubMatches = %s\nwant %s", g, want)
	}
	if !ms[0].IsYouth() || (ClubMatch{TeamType: "Herren"}).IsYouth() {
		t.Error("IsYouth misclassifies")
	}
}
