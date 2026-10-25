package fupa

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// serve returns a Client whose API answers with the body routed by path
// and query; "/teams/x-m1" redirects to the current season like FuPa.
func serve(t *testing.T, routes map[string]string) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != userAgent {
			t.Errorf("User-Agent = %q", r.Header.Get("User-Agent"))
		}
		if r.URL.Path == "/teams/hz-m1" {
			http.Redirect(w, r, "/teams/hz-m1-2026-27", http.StatusTemporaryRedirect)
			return
		}
		body, ok := routes[r.URL.RequestURI()]
		if !ok {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, body)
	}))
	t.Cleanup(srv.Close)
	return &Client{HTTP: srv.Client(), Base: srv.URL}
}

func TestParseTeam(t *testing.T) {
	for in, want := range map[string]string{
		"fc-sturm-hauzenberg-m1":                                           "fc-sturm-hauzenberg-m1",
		"https://www.fupa.net/team/fc-sturm-hauzenberg-m1-2026-27":         "fc-sturm-hauzenberg-m1-2026-27",
		"https://www.fupa.net/team/fc-sturm-hauzenberg-m1-2026-27/matches": "fc-sturm-hauzenberg-m1-2026-27",
	} {
		if got, err := ParseTeam(in); err != nil || got != want {
			t.Errorf("ParseTeam(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	if got, err := ParseTeam("https://www.fupa.net/club/fc-sturm-hauzenberg/x y"); err == nil {
		t.Errorf("ParseTeam(club URL) = %q, want an error", got)
	}
}

func TestTeamFollowsSeasonRedirect(t *testing.T) {
	c := serve(t, map[string]string{
		"/teams/hz-m1-2026-27": `{"slug":"hz-m1-2026-27","name":{"full":"FC Sturm Hauzenberg"},"competition":{"slug":"bayern-landesliga-mitte"}}`,
	})
	tm, err := c.Team(context.Background(), "hz-m1")
	if err != nil {
		t.Fatal(err)
	}
	if tm != (Team{Slug: "hz-m1-2026-27", Name: "FC Sturm Hauzenberg", Competition: "bayern-landesliga-mitte"}) {
		t.Errorf("Team = %+v", tm)
	}
}

func TestSquad(t *testing.T) {
	c := serve(t, map[string]string{
		"/teams/hz-m1-2026-27/squad": `{"players":[
			{"firstName":"Michael","lastName":"Hellauer","position":"Torwart","jerseyNumber":null,"flags":["new"]},
			{"firstName":"Christoph","lastName":"Obermüller","position":"Torwart","jerseyNumber":1,"matches":8,"flags":["captain"]},
			{"firstName":"Joris","lastName":"Wagner","position":"Angriff","jerseyNumber":9,"matches":16,"goals":11,"flags":[]},
			{"firstName":"Ex","lastName":"Player","position":"Angriff","jerseyNumber":99,"isDeactivated":true}],
			"coaches":[{"firstName":"Matthias","lastName":"Süß","role":"Trainer"},{"firstName":"Gone","lastName":"Coach","role":"Trainer","isDeactivated":true}]}`,
	})
	s, err := c.Squad(context.Background(), "hz-m1-2026-27")
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Players) != 3 || len(s.Staff) != 1 || s.Staff[0] != (Staff{Name: "Matthias Süß", Role: "Trainer"}) {
		t.Fatalf("Squad = %+v, want 3 active players and 1 coach", s)
	}
	if p := s.Players[1]; !p.Captain || p.Number != 1 || p.Name() != "Christoph Obermüller" {
		t.Errorf("captain = %+v", p)
	}
	if n := s.Numbered(); len(n) != 2 {
		t.Errorf("Numbered = %+v, want the two with a number", n)
	}
	if n := (Squad{Players: []Player{{LastName: "A"}, {LastName: "B"}}}).Numbered(); len(n) != 2 {
		t.Errorf("Numbered without any numbers = %+v, want all", n)
	}
}

func TestNextLeagueMatchAndReferee(t *testing.T) {
	c := serve(t, map[string]string{
		"/teams/hz-m1-2026-27/matches?flavor=current&limit=60": `[
			{"slug":"lam-hz-261010","kickoff":"2026-10-10T16:00:00+02:00","category":"league","homeTeam":{"slug":"lam-m1-2026-27","name":{"full":"SpVgg Lam"}},"awayTeam":{"slug":"hz-m1-2026-27","name":{"full":"FC Sturm Hauzenberg"}}},
			{"slug":"cup-261016","kickoff":"2026-10-16T19:00:00+02:00","category":"cup","homeTeam":{"slug":"hz-m1-2026-27"},"awayTeam":{"slug":"x-m1-2026-27"}},
			{"slug":"hz-luhe-261017","kickoff":"2026-10-17T15:00:00+02:00","category":"league","homeTeam":{"slug":"hz-m1-2026-27","name":{"full":"FC Sturm Hauzenberg"}},"awayTeam":{"slug":"luhe-m1-2026-27","name":{"full":"SC Luhe-Wildenau"}}}]`,
		"/matches/hz-luhe-261017": `{"referee":{"name":"Marijo Kraljic","club":{"name":"SC Wörnsmühl "}}}`,
		"/matches/lam-hz-261010":  `{"referee":null}`,
	})
	// The cup match on Friday is skipped; on Saturday evening the day's
	// match still counts.
	for _, now := range []time.Time{time.Date(2026, 10, 15, 12, 0, 0, 0, berlin), time.Date(2026, 10, 17, 20, 0, 0, 0, berlin)} {
		c.Now = func() time.Time { return now }
		m, err := c.NextLeagueMatch(context.Background(), "hz-m1-2026-27")
		if err != nil || m.Slug != "hz-luhe-261017" {
			t.Fatalf("NextLeagueMatch at %v = %+v, %v", now, m, err)
		}
		if opp := m.Opponent("hz-m1-2026-27"); opp != (Team{Slug: "luhe-m1-2026-27", Name: "SC Luhe-Wildenau"}) {
			t.Errorf("Opponent = %+v", opp)
		}
	}
	ref, err := c.Referee(context.Background(), "hz-luhe-261017")
	if err != nil || ref != (Referee{Name: "Marijo Kraljic", Club: "SC Wörnsmühl"}) {
		t.Errorf("Referee = %+v, %v", ref, err)
	}
	if _, err := c.Referee(context.Background(), "lam-hz-261010"); !errors.Is(err, ErrNoReferee) {
		t.Errorf("Referee without one: err = %v, want ErrNoReferee", err)
	}
	c.Now = func() time.Time { return time.Date(2026, 10, 18, 0, 0, 0, 0, berlin) }
	if _, err := c.NextLeagueMatch(context.Background(), "hz-m1-2026-27"); err == nil {
		t.Error("expected an error without upcoming league matches")
	}
}

func TestLeagueScorers(t *testing.T) {
	c := serve(t, map[string]string{
		"/competitions/ll/seasons/current/scorers?limit=12": `[
			{"player":{"firstName":"Almir","lastName":"Mesanovic"},"team":{"name":{"full":"SV Türk Gücü Straubing"}},"statistics":{"goals":16}},
			{"player":{"firstName":"Marcel","lastName":"Müller"},"team":{"name":{"full":"FC Dingolfing"}},"statistics":{"goals":14}},
			{"player":{"firstName":"Patrick","lastName":"Käufer"},"team":{"name":{"full":"ASV Burglengenfeld"}},"statistics":{"goals":14}},
			{"player":{"firstName":"Kai","lastName":"Dirmeier"},"team":{"name":{"full":"ASV Burglengenfeld"}},"statistics":{"goals":13}}]`,
	})
	s, err := c.LeagueScorers(context.Background(), "ll", 2)
	if err != nil {
		t.Fatal(err)
	}
	// Käufer is tied with the second.
	if len(s) != 3 || s[2] != (Scorer{Name: "Patrick Käufer", Team: "ASV Burglengenfeld", Goals: 14}) {
		t.Errorf("LeagueScorers = %+v", s)
	}
}

func TestTeamPhoto(t *testing.T) {
	c := serve(t, map[string]string{
		"/teams/luhe-m1-2026-27/squad": `{"players":[],"coaches":[],"info":{"teamImage":{"path":"https://image.fupa.net/team-image/abc/","source":" Robert Geisler "}}}`,
		"/team-image/abc/" + photoSize:  "JPEGDATA",
	})
	s, err := c.Squad(context.Background(), "luhe-m1-2026-27")
	if err != nil {
		t.Fatal(err)
	}
	if s.Photo != "https://image.fupa.net/team-image/abc/" || s.PhotoCredit != "Robert Geisler" {
		t.Errorf("Photo, PhotoCredit = %q, %q", s.Photo, s.PhotoCredit)
	}
	// The photo is fetched from the address FuPa gives, here the test server.
	s.Photo = c.Base + "/team-image/abc/"
	b, err := c.TeamPhoto(context.Background(), s)
	if err != nil || string(b) != "JPEGDATA" {
		t.Errorf("TeamPhoto = %q, %v", b, err)
	}
	if _, err := c.TeamPhoto(context.Background(), Squad{}); err == nil {
		t.Error("expected an error without a photo")
	}
}
