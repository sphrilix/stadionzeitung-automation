package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sphrilix/stadionzeitung-automation/internal/bfv"
	"github.com/sphrilix/stadionzeitung-automation/internal/fupa"
)

// sources fakes BFV under /bfv and FuPa under /fupa with the given routes;
// a route with a "->" value redirects there.
func sources(t *testing.T, routes map[string]string) (*bfv.Client, *fupa.Client) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := routes[r.URL.RequestURI()]
		switch {
		case !ok:
			http.NotFound(w, r)
		case strings.HasPrefix(body, "->"):
			http.Redirect(w, r, strings.TrimPrefix(body, "->"), http.StatusFound)
		case strings.HasSuffix(r.URL.Path, ".jpg") || strings.HasSuffix(r.URL.Path, ".jpeg"):
			w.Header().Set("Content-Type", "image/jpeg")
			fmt.Fprint(w, body)
		default:
			fmt.Fprint(w, body)
		}
	}))
	t.Cleanup(srv.Close)
	bc := &bfv.Client{HTTP: srv.Client(), BaseWeb: srv.URL + "/bfv", BaseWidget: srv.URL + "/bfv/widget",
		BaseLogo: srv.URL + "/bfv/logo/", BaseMedia: srv.URL + "/bfv/media/"}
	fc := &fupa.Client{HTTP: srv.Client(), Base: srv.URL + "/fupa"}
	return bc, fc
}

const (
	hzTeam = "016PE85GAS000000VV0AG811VTE5EA5R"
	hzCID  = "0307PL23RO000007VS5489BUVT69HBT7-G"
)

var hzRoutes = map[string]string{
	"/bfv/widget/team/" + hzTeam + "/matches": fmt.Sprintf(`{"state":0,"data":{"team":{"name":"FC Sturm Hauzenberg","compoundId":%q}}}`, hzCID),
	"/bfv/widget/team/" + hzTeam + "/squad": `{"state":0,"data":{"public":true,"players":[
		{"player":{"name":"Joris Wagner"},"statistics":{"goals":11}},
		{"player":{"name":"Florian Seibold"},"statistics":{"goals":0}}]}}`,
	"/bfv/media/" + hzTeam + "/TYP/50/SZ/0": "->/img/hz.jpg",
	"/img/hz.jpg":                           "bfv photo",
}

func TestBuildTeamFallsBackToBFV(t *testing.T) {
	// FuPa does not know the team at all.
	bc, fc := sources(t, hzRoutes)
	e := entry{Kind: kindScorers, Team: hzTeam, FuPa: "unknown-m1"}
	o, err := buildTeam(context.Background(), bc, fc, e)
	if err != nil {
		t.Fatal(err)
	}
	if o.label != "Torjäger FC Sturm Hauzenberg" {
		t.Errorf("label = %q", o.label)
	}
	tm, err := resolveTeam(context.Background(), bc, fc, e)
	if err != nil {
		t.Fatal(err)
	}
	if photo, _ := teamPhoto(context.Background(), bc, fc, tm, nil, tm.fupaErr); string(photo) != "bfv photo" {
		t.Errorf("teamPhoto = %q, want BFV's", photo)
	}
	// Neither source knows the team.
	_, err = buildTeam(context.Background(), bc, fc, entry{Kind: kindSquad, FuPa: "unknown-m1"})
	if err == nil || !strings.Contains(err.Error(), "FuPa: ") || !strings.Contains(err.Error(), "BFV: no team configured") {
		t.Errorf("buildTeam without any source: err = %v", err)
	}
}

func TestTeamPhotoPrefersFuPa(t *testing.T) {
	routes := map[string]string{
		"/fupa/teams/hz-m1":        `{"slug":"hz-m1","name":{"full":"FC Sturm Hauzenberg"}}`,
		"/img/fupa/1920x1080.jpeg": "fupa photo",
		"/img/none/1920x1080.jpeg": "fupa placeholder",
	}
	for k, v := range hzRoutes {
		routes[k] = v
	}
	bc, fc := sources(t, routes)
	srvURL := strings.TrimSuffix(fc.Base, "/fupa")
	tm := team{name: "FC Sturm Hauzenberg", fupa: "hz-m1", bfvID: hzTeam}
	for _, tc := range []struct {
		squad fupa.Squad
		bfvID string
		want  string
	}{
		{fupa.Squad{Photo: srvURL + "/img/fupa/", PhotoCredit: "R. Geisler"}, hzTeam, "fupa photo"},
		{fupa.Squad{Photo: srvURL + "/img/none/", PhotoPlaceholder: true}, hzTeam, "bfv photo"},
		{fupa.Squad{Photo: srvURL + "/img/none/", PhotoPlaceholder: true}, "", "fupa placeholder"},
		{fupa.Squad{}, "", ""},
	} {
		tm.bfvID = tc.bfvID
		if photo, _ := teamPhoto(context.Background(), bc, fc, tm, &tc.squad, nil); string(photo) != tc.want {
			t.Errorf("teamPhoto(%+v, bfv %q) = %q, want %q", tc.squad, tc.bfvID, photo, tc.want)
		}
	}
}

func TestLeagueScorerLinesFallsBackToBFV(t *testing.T) {
	routes := map[string]string{
		"/fupa/teams/hz-m1": `{"slug":"hz-m1","name":{"full":"FC Sturm Hauzenberg"},"competition":{"slug":"ll-mitte"}}`,
		// FuPa has no scorers yet.
		"/fupa/competitions/ll-mitte/seasons/current/scorers?limit=13": `[]`,
		"/bfv/widget/competition/" + hzCID + "/topscorer": `{"state":0,"data":{"scorers":[
			{"name":"Shaban Rugovaj","team":{"name":"TSV Forst"},"goals":17}]}}`,
	}
	for k, v := range hzRoutes {
		routes[k] = v
	}
	bc, fc := sources(t, routes)
	lines, err := leagueScorerLines(context.Background(), bc, fc, entry{Team: hzTeam, FuPa: "hz-m1"})
	if err != nil || fmt.Sprint(lines) != "[Shaban Rugovaj (TSV Forst, 17 Tore)]" {
		t.Errorf("leagueScorerLines = %q, %v", lines, err)
	}
	// Without BFV, FuPa's empty list stands.
	lines, err = leagueScorerLines(context.Background(), bc, fc, entry{FuPa: "hz-m1"})
	if err != nil || len(lines) != 0 {
		t.Errorf("leagueScorerLines(FuPa only) = %q, %v", lines, err)
	}
	if _, err := leagueScorerLines(context.Background(), bc, fc, entry{FuPa: "unknown-m1"}); err == nil {
		t.Error("leagueScorerLines with no source = nil error")
	}
}

func TestFuPaLogos(t *testing.T) {
	_, fc := sources(t, map[string]string{
		"/logo/a/200x200.png": "a",
		"/logo/b/200x200.png": "b",
		"/fupa/teams/hz-m1":   `{"slug":"hz-m1","name":{"full":"FC Sturm Hauzenberg"},"competition":{"slug":"ll-mitte"}}`,
		"/fupa/standings?competition=ll-mitte": `{"standings":[
			{"rank":1,"matches":18,"points":42,"team":{"name":{"full":"A"},"image":{"path":"/logo/a/"}}},
			{"rank":2,"matches":18,"points":39,"team":{"name":{"full":"B"},"image":{"path":"/logo/b/"}}}]}`,
	})
	base := strings.TrimSuffix(fc.Base, "/fupa")
	// FuPa's image paths are absolute on its image server.
	fc.HTTP.Transport = rewrite{base: base, next: fc.HTTP.Transport}
	logos := map[string][]byte{}
	fupaLogos(context.Background(), fc, "hz-m1", []bfv.Standing{
		{Position: 1, TeamID: "A", Played: 18, Points: 42},
		// BFV has a newer result: the rows do not agree, so no logo.
		{Position: 2, TeamID: "B", Played: 19, Points: 40},
	}, logos)
	if len(logos) != 1 || string(logos["A"]) != "a" {
		t.Errorf("fupaLogos = %v, want only A's", logos)
	}
}

// rewrite sends requests for a relative URL path to base.
type rewrite struct {
	base string
	next http.RoundTripper
}

func (r rewrite) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Host == "" {
		u, _ := req.URL.Parse(r.base + req.URL.Path)
		req.URL = u
	}
	return r.next.RoundTrip(req)
}

func TestBFVSquadRows(t *testing.T) {
	got := fmt.Sprint(bfvSquadRows([]bfv.Player{{Name: "Joris Wagner"}, {Name: "Luca Canales Uhrmann"}, {Name: "Florian Seibold"}}))
	if want := "[{Spieler: Florian Seibold, Luca Canales Uhrmann, Joris Wagner true}]"; got != want {
		t.Errorf("bfvSquadRows = %s, want %s", got, want)
	}
}
