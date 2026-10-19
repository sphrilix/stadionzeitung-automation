package bfv

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const testCID = "03135AVGGC000009VS5489BTVU7GTVLE-G"

// pageHTML mimics the escaped flight data of a bfv.de matchday page.
// Matchday n is scheduled for the weekend ending on 11.10.2026 + 7(n-1) days.
func pageHTML(matchday, maxMatchday int, ids ...string) string {
	var b strings.Builder
	fmt.Fprintf(&b, `<title>Test Liga | Saison 26/27 | BFV</title><script>self.__next_f.push([1,"\"matchDay\":\"%d\",\"matchDays\":[`, matchday)
	for n := 1; n <= maxMatchday; n++ {
		end := time.Date(2026, 10, 11+7*(n-1), 0, 0, 0, 0, time.UTC)
		fmt.Fprintf(&b, `{\"number\":%d,\"period\":[\"%s\",\"%s\"]},`, n, end.AddDate(0, 0, -2).Format("02.01."), end.Format("02.01.2006"))
	}
	b.WriteString(`],\"matches\":[`)
	for _, id := range ids {
		fmt.Fprintf(&b, `{\"id\":\"%s\",\"status\":\"acknowledged\"},`, id)
	}
	b.WriteString(`]"])</script>`)
	return b.String()
}

func id(s string) string { return s + strings.Repeat("0", 32-len(s)) }

func newTestServer(t *testing.T) *Client {
	t.Helper()
	routes := map[string]string{
		// BFV's default page shows the last matchday played.
		"/ergebnisse/wettbewerb/-/" + testCID:              pageHTML(1, 3, id("M1"), id("M2")),
		"/ergebnisse/wettbewerb/-/" + testCID + "/spieltag/1": pageHTML(1, 3, id("M1"), id("M2")),
		"/ergebnisse/wettbewerb/-/" + testCID + "/spieltag/2": pageHTML(2, 3, id("M3"), id("BYE")),
		"/ergebnisse/wettbewerb/-/" + testCID + "/spieltag/3": pageHTML(3, 3, id("M4")),
		"/widget/competition/" + testCID + "/table": `{"state":0,"data":{"table":[
			{"team":{"permanentId":"A","name":"Alpha"}},
			{"team":{"permanentId":"B","name":"Bravo"}},
			{"team":{"permanentId":"C","name":"Charlie"}}]}}`,
		"/widget/team/A/matches": fmt.Sprintf(`{"state":0,"data":{"matches":[
			{"matchId":%q,"compoundId":%q,"kickoffDate":"11.10.2026","kickoffTime":"15:00","homeTeamName":"Alpha","guestTeamName":"Bravo","result":"0:4"},
			{"matchId":%q,"compoundId":%q,"kickoffDate":"18.10.2026","kickoffTime":"15:00","homeTeamName":"Charlie","guestTeamName":"Alpha","result":""},
			{"matchId":%q,"compoundId":%q,"kickoffDate":"25.10.2026","kickoffTime":"15:00","homeTeamName":"Bravo","guestTeamName":"Alpha","result":""},
			{"matchId":"FRIENDLY","compoundId":"OTHER","kickoffDate":"01.07.2026","kickoffTime":"18:00","homeTeamName":"Alpha","guestTeamName":"Zulu","result":"1:1"}]}}`,
			id("M1"), testCID, id("M3"), testCID, id("M4"), testCID),
		"/widget/team/B/matches": fmt.Sprintf(`{"state":0,"data":{"matches":[
			{"matchId":%q,"compoundId":%q,"kickoffDate":"11.10.2026","kickoffTime":"15:00","homeTeamName":"Alpha","guestTeamName":"Bravo","result":"0:4"},
			{"matchId":%q,"compoundId":%q,"kickoffDate":"18.10.2026","kickoffTime":"","homeTeamName":"Bravo","guestTeamName":"SPIELFREI","result":""}]}}`,
			id("M1"), testCID, id("BYE"), testCID),
		"/widget/team/C/matches": fmt.Sprintf(`{"state":0,"data":{"matches":[
			{"matchId":%q,"compoundId":%q,"kickoffDate":"10.10.2026","kickoffTime":"19:00","homeTeamName":"Charlie","guestTeamName":"Delta","result":"2:1"},
			{"matchId":%q,"compoundId":%q,"kickoffDate":"18.10.2026","kickoffTime":"15:00","homeTeamName":"Charlie","guestTeamName":"Alpha","result":""}]}}`,
			id("M2"), testCID, id("M3"), testCID),
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := routes[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, body)
	}))
	t.Cleanup(srv.Close)
	return &Client{HTTP: srv.Client(), BaseWeb: srv.URL, BaseWidget: srv.URL + "/widget"}
}

func TestFetchWindow(t *testing.T) {
	c := newTestServer(t)
	// Monday after matchday 1: matchday 2 is the next one.
	c.Now = func() time.Time { return time.Date(2026, 10, 12, 12, 0, 0, 0, berlin) }
	w, err := c.FetchWindow(context.Background(), testCID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if w.League != "Test Liga" || w.Current != 2 {
		t.Errorf("League, Current = %q, %d; want %q, 2", w.League, w.Current, "Test Liga")
	}
	if len(w.Days) != 3 || w.Days[0].Number != 1 || w.Days[2].Number != 3 {
		t.Fatalf("Days = %+v, want matchdays 1 to 3", w.Days)
	}
	got := func(d Matchday) []string {
		var s []string
		for _, m := range d.Matches {
			s = append(s, m.Home+"-"+m.Guest+" "+m.Result)
		}
		return s
	}
	// Sorted by kickoff: Charlie-Delta (10.10.) before Alpha-Bravo (11.10.).
	if g := strings.Join(got(w.Days[0]), ", "); g != "Charlie-Delta 2:1, Alpha-Bravo 0:4" {
		t.Errorf("matchday 1 = %s", g)
	}
	// The bye listed on the matchday page is left out.
	if g := strings.Join(got(w.Days[1]), ", "); g != "Charlie-Alpha " {
		t.Errorf("matchday 2 = %s", g)
	}
	if g := strings.Join(got(w.Days[2]), ", "); g != "Bravo-Alpha " {
		t.Errorf("matchday 3 = %s", g)
	}
	if m := w.Days[0].Matches[0]; !m.HasTime || m.Kickoff.Format("02.01.2006 15:04 MST") != "10.10.2026 19:00 CEST" {
		t.Errorf("kickoff = %v (HasTime %v)", m.Kickoff, m.HasTime)
	}
}

func TestFetchWindowOverrideClampsToSeason(t *testing.T) {
	c := newTestServer(t)
	for _, tc := range []struct{ override, from, to int }{{1, 1, 2}, {3, 2, 3}} {
		w, err := c.FetchWindow(context.Background(), testCID, tc.override)
		if err != nil {
			t.Fatal(err)
		}
		if w.Current != tc.override || len(w.Days) != 2 || w.Days[0].Number != tc.from || w.Days[1].Number != tc.to {
			t.Errorf("override %d: Current = %d, Days = %+v; want matchdays %d to %d", tc.override, w.Current, w.Days, tc.from, tc.to)
		}
	}
}

func TestTeamMatchesSkipsOtherCompetitions(t *testing.T) {
	c := newTestServer(t)
	ms, err := c.TeamMatches(context.Background(), "A", testCID)
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 3 {
		t.Errorf("TeamMatches = %+v, want the three league matches, not the friendly", ms)
	}
}
