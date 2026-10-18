package bfv

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const testCID = "03135AVGGC000009VS5489BTVU7GTVLE-G"

// pageHTML mimics the escaped flight data of a bfv.de matchday page.
func pageHTML(matchday, maxMatchday int, ids ...string) string {
	var b strings.Builder
	fmt.Fprintf(&b, `<title>Test Liga | Saison 26/27 | BFV</title><script>self.__next_f.push([1,"\"matchDay\":\"%d\",\"matchDays\":[`, matchday)
	for n := 1; n <= maxMatchday; n++ {
		fmt.Fprintf(&b, `{\"number\":%d,\"period\":[\"01.01.\",\"02.01.2027\"]},`, n)
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
		"/ergebnisse/wettbewerb/-/" + testCID:              pageHTML(1, 2, id("M1"), id("M2")),
		"/ergebnisse/wettbewerb/-/" + testCID + "/spieltag/1": pageHTML(1, 2, id("M1"), id("M2")),
		"/ergebnisse/wettbewerb/-/" + testCID + "/spieltag/2": pageHTML(2, 2, id("M3"), id("BYE")),
		"/widget/competition/" + testCID + "/table": `{"state":0,"data":{"table":[
			{"team":{"permanentId":"A","name":"Alpha"}},
			{"team":{"permanentId":"B","name":"Bravo"}},
			{"team":{"permanentId":"C","name":"Charlie"}}]}}`,
		"/widget/team/A/matches": fmt.Sprintf(`{"state":0,"data":{"matches":[
			{"matchId":%q,"compoundId":%q,"kickoffDate":"11.10.2026","kickoffTime":"15:00","homeTeamName":"Alpha","guestTeamName":"Bravo","result":"0:4"},
			{"matchId":%q,"compoundId":%q,"kickoffDate":"18.10.2026","kickoffTime":"15:00","homeTeamName":"Charlie","guestTeamName":"Alpha","result":""},
			{"matchId":"FRIENDLY","compoundId":"OTHER","kickoffDate":"01.07.2026","kickoffTime":"18:00","homeTeamName":"Alpha","guestTeamName":"Zulu","result":"1:1"}]}}`,
			id("M1"), testCID, id("M3"), testCID),
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
	w, err := c.FetchWindow(context.Background(), testCID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if w.League != "Test Liga" || w.Current != 1 {
		t.Errorf("League, Current = %q, %d; want %q, 1", w.League, w.Current, "Test Liga")
	}
	// Matchday 0 does not exist, so the window is clamped to 1..2.
	if len(w.Days) != 2 || w.Days[0].Number != 1 || w.Days[1].Number != 2 {
		t.Fatalf("Days = %+v, want matchdays 1 and 2", w.Days)
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
	if g := strings.Join(got(w.Days[1]), ", "); g != "Charlie-Alpha " {
		t.Errorf("matchday 2 = %s", g)
	}
	if m := w.Days[0].Matches[0]; !m.HasTime || m.Kickoff.Format("02.01.2006 15:04 MST") != "10.10.2026 19:00 CEST" {
		t.Errorf("kickoff = %v (HasTime %v)", m.Kickoff, m.HasTime)
	}
}

func TestFetchWindowLastMatchday(t *testing.T) {
	c := newTestServer(t)
	w, err := c.FetchWindow(context.Background(), testCID, 2)
	if err != nil {
		t.Fatal(err)
	}
	if w.Current != 2 || len(w.Days) != 2 || w.Days[0].Number != 1 || w.Days[1].Number != 2 {
		t.Errorf("Current = %d, Days = %+v; want matchdays 1 and 2", w.Current, w.Days)
	}
}

func TestTeamMatchesSkipsOtherCompetitions(t *testing.T) {
	c := newTestServer(t)
	ms, err := c.TeamMatches(context.Background(), "A", testCID)
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 2 {
		t.Errorf("TeamMatches = %+v, want the two league matches", ms)
	}
}
