package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sphrilix/stadionzeitung-automation/internal/bfv"
	"github.com/sphrilix/stadionzeitung-automation/internal/fupa"
	"golang.org/x/image/font/gofont/goregular"
)

const (
	refTeam = "016PE85GAS000000VV0AG811VTE5EA5R"
	refCID  = "03135AVGGC000009VS5489BTVU7GTVLE-G"
)

// refServers fakes BFV and FuPa for the next league match, on 07.11.2026.
// BFV lists the given officials ("" for none yet), FuPa the given referee
// ("" for none yet).
func refServers(t *testing.T, bfvReferee string, bfvAssistants []string, fupaReferee string) (*bfv.Client, *fupa.Client) {
	t.Helper()
	value := func(s string) string { return `<span class="P"><span class="value-text">` + s + `</span></span>` }
	page := `<div class="StageMatchOverviewFooter_container__P" style="--value-text-font:font-f">` +
		`<div class="StageMatchOverviewFooter_referee__C"><span class="H">Schiedsrichter</span>` + value(bfvReferee) + `</div>` +
		`<div class="StageMatchOverviewFooter_assistants__F"><span class="H">Assistenten</span>`
	for _, a := range bfvAssistants {
		page += value(a)
	}
	page += `</div></div>`
	fupaRef := `null`
	if fupaReferee != "" {
		fupaRef = fmt.Sprintf(`{"name":%q,"club":{"name":"SC Wörnsmühl"}}`, fupaReferee)
	}
	routes := map[string]string{
		"/bfv/widget/team/" + refTeam + "/matches": fmt.Sprintf(`{"state":0,"data":{"team":{"compoundId":%q},"matches":[
			{"matchId":"M1","compoundId":%q,"kickoffDate":"31.10.2026","kickoffTime":"15:00","homeTeamName":"SV Türk Gücü Straubing","guestTeamName":"FC Sturm Hauzenberg","guestTeamPermanentId":%q},
			{"matchId":"M2","compoundId":%q,"kickoffDate":"07.11.2026","kickoffTime":"14:00","homeTeamName":"FC Sturm Hauzenberg","homeTeamPermanentId":%q,"guestTeamName":"DJK Vornbach"}]}}`,
			refCID, refCID, refTeam, refCID, refTeam),
		"/bfv/ergebnisse/spiel/-/M2": page,
		"/bfv/font/f/type/font":      string(goregular.TTF),

		"/fupa/teams/hz-m1": `{"slug":"hz-m1","name":{"full":"FC Sturm Hauzenberg"}}`,
		"/fupa/teams/hz-m1/matches?flavor=current&limit=60": `[
			{"slug":"hz-vornbach","kickoff":"2026-11-07T14:00:00+01:00","category":"league","homeTeam":{"slug":"hz-m1","name":{"full":"FC Sturm Hauzenberg"}},"awayTeam":{"slug":"vornbach-m1","name":{"full":"DJK Vornbach"}}}]`,
		"/fupa/matches/hz-vornbach": `{"referee":` + fupaRef + `}`,
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := routes[r.URL.RequestURI()]
		if !ok {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, body)
	}))
	t.Cleanup(srv.Close)
	now := func() time.Time { return time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC) }
	bc := &bfv.Client{HTTP: srv.Client(), BaseWeb: srv.URL + "/bfv", BaseWidget: srv.URL + "/bfv/widget", BaseFont: srv.URL + "/bfv/font/", Now: now}
	fc := &fupa.Client{HTTP: srv.Client(), Base: srv.URL + "/fupa", Now: now}
	return bc, fc
}

func TestRefereeRows(t *testing.T) {
	assistants := []string{"Simon Engl", "David Gomer"}
	for _, tc := range []struct {
		name              string
		bfvRef, fupaRef   string
		bfvTeam, fupaTeam bool
		want, wantSource  string
	}{
		{"BFV only", "Fabian Buechner", "", true, false,
			"[{Schiedsrichter: Fabian Buechner false} {Assistenten: Simon Engl, David Gomer false}]", "BFV"},
		{"FuPa adds the club", "Fabian Buechner", "Fabian  Büchner", true, true,
			"[{Schiedsrichter: Fabian Buechner (SC Wörnsmühl) false} {Assistenten: Simon Engl, David Gomer false}]", "BFV"},
		{"FuPa names someone else", "Fabian Buechner", "Marijo Kraljic", true, true,
			"[{Schiedsrichter: Fabian Buechner false} {Assistenten: Simon Engl, David Gomer false}]", "BFV"},
		{"FuPa without a referee yet", "Fabian Buechner", "", true, true,
			"[{Schiedsrichter: Fabian Buechner false} {Assistenten: Simon Engl, David Gomer false}]", "BFV"},
		{"BFV without a referee yet", "", "Marijo Kraljic", true, true,
			"[{Schiedsrichter: Marijo Kraljic (SC Wörnsmühl) false}]", "FuPa, BFV: FC Sturm Hauzenberg - DJK Vornbach: BFV has no referee for the match yet"},
		{"FuPa only", "Fabian Buechner", "Marijo Kraljic", false, true,
			"[{Schiedsrichter: Marijo Kraljic (SC Wörnsmühl) false}]", "FuPa"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bc, fc := refServers(t, tc.bfvRef, assistants, tc.fupaRef)
			e := entry{Kind: kindReferee}
			if tc.bfvTeam {
				e.Team = refTeam
			}
			if tc.fupaTeam {
				e.FuPa = "hz-m1"
			}
			rows, source, err := refereeRows(context.Background(), bc, fc, e)
			if err != nil {
				t.Fatal(err)
			}
			if got := fmt.Sprint(rows); got != tc.want || source != tc.wantSource {
				t.Errorf("refereeRows = %s (%s)\nwant %s (%s)", got, source, tc.want, tc.wantSource)
			}
		})
	}
}

func TestRefereeRowsUnknown(t *testing.T) {
	bc, fc := refServers(t, "", nil, "")
	_, _, err := refereeRows(context.Background(), bc, fc, entry{Kind: kindReferee, Team: refTeam})
	if err == nil || !strings.Contains(err.Error(), "BFV has no referee") {
		t.Errorf("BFV only, no referee: err = %v", err)
	}
	_, _, err = refereeRows(context.Background(), bc, fc, entry{Kind: kindReferee, Team: refTeam, FuPa: "hz-m1"})
	if err == nil || !strings.Contains(err.Error(), "BFV: ") || !strings.Contains(err.Error(), "FuPa has no referee") {
		t.Errorf("neither has a referee: err = %v", err)
	}
}

func TestValidateReferee(t *testing.T) {
	for _, e := range []entry{{Team: refTeam}, {FuPa: "hz-m1"}, {Team: refTeam, FuPa: "hz-m1"}} {
		e.Kind = kindReferee
		if err := e.validate(); err != nil {
			t.Errorf("%+v: %v", e, err)
		}
	}
	if err := (entry{Kind: kindReferee}).validate(); err == nil {
		t.Error("schiedsrichter without a team: expected an error")
	}
}

func TestSameName(t *testing.T) {
	for _, tc := range []struct {
		a, b string
		want bool
	}{
		{"Dominik Von Maffei", "Dominik  von Maffei", true},
		{"Fabian Buechner", "Fabian Büchner", true},
		{"Jörg Weiß", "Joerg Weiss", true},
		{"Fabian Buechner", "Marijo Kraljic", false},
	} {
		if got := sameName(tc.a, tc.b); got != tc.want {
			t.Errorf("sameName(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}
