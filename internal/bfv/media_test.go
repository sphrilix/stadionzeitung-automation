package bfv

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTeamPhoto(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/media/A/TYP/50/SZ/0":
			http.Redirect(w, r, "/userfiles/a_t0.jpg", http.StatusFound)
		case "/media/P/TYP/50/SZ/0":
			http.Redirect(w, r, "/static/placeholder_team_image_890x500.png", http.StatusFound)
		case "/userfiles/a_t0.jpg":
			w.Header().Set("Content-Type", "image/jpeg")
			w.Write([]byte("jpeg"))
		case "/static/placeholder_team_image_890x500.png":
			w.Header().Set("Content-Type", "image/png")
			w.Write([]byte("png"))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	c := &Client{HTTP: srv.Client(), BaseMedia: srv.URL + "/media/"}
	if b, err := c.TeamPhoto(context.Background(), "A"); err != nil || string(b) != "jpeg" {
		t.Errorf("TeamPhoto(A) = %q, %v", b, err)
	}
	if _, err := c.TeamPhoto(context.Background(), "P"); !errors.Is(err, ErrNoTeamPhoto) {
		t.Errorf("TeamPhoto(placeholder) err = %v, want ErrNoTeamPhoto", err)
	}
	if _, err := c.TeamPhoto(context.Background(), "X"); err == nil {
		t.Error("TeamPhoto(unknown) = nil error")
	}
}

func TestLeagueScorers(t *testing.T) {
	c := serve(t, map[string]string{
		"/widget/competition/" + testCID + "/topscorer": `{"state":0,"data":{"scorers":[
			{"name":"Shaban Rugovaj","team":{"name":"TSV Forst"},"rank":1,"goals":17},
			{"name":"Julian Östreicher","team":{"name":"TSV 07 Grettstadt"},"rank":2,"goals":16},
			{"name":"Bastian Stößel","team":{"name":"SG Eltmann"},"rank":3,"goals":12},
			{"name":"Tied Player","team":{"name":"SG Eltmann"},"rank":3,"goals":12},
			{"name":"Below","team":{"name":"X"},"rank":5,"goals":11}]}}`,
	})
	got, err := c.LeagueScorers(context.Background(), testCID, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 4 || got[0] != (Scorer{Name: "Shaban Rugovaj", Team: "TSV Forst", Goals: 17}) || got[3].Name != "Tied Player" {
		t.Errorf("LeagueScorers = %+v, want the top 3 and the tie", got)
	}
}
