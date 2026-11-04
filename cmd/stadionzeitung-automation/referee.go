package main

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/sphrilix/stadionzeitung-automation/internal/bfv"
	"github.com/sphrilix/stadionzeitung-automation/internal/fupa"
	"github.com/sphrilix/stadionzeitung-automation/internal/render"
)

// buildReferee builds the officials of a team's next league match.
func buildReferee(ctx context.Context, bc *bfv.Client, fc *fupa.Client, t entry) (output, error) {
	rows, source, err := refereeRows(ctx, bc, fc, t)
	if err != nil {
		return output{}, err
	}
	return output{
		label: "Schiedsrichter (" + source + ")",
		name:  "Schiedsrichter",
		render: func(out io.Writer, fonts render.Fonts) error {
			return render.RenderRows(out, rows, fonts)
		},
	}, nil
}

// refereeRows lays out the officials of a team's next league match, e.g.
// "Schiedsrichter: Marijo Kraljic (SC Wörnsmühl)" and
// "Assistenten: Moritz Breitruck, Dominik Von Maffei", and says where they
// are from. BFV lists the referee and the assistants; FuPa, if the team has
// it, adds the referee's club, and stands in while BFV has no referee.
func refereeRows(ctx context.Context, bc *bfv.Client, fc *fupa.Client, t entry) (rows []render.Row, source string, err error) {
	var (
		off    bfv.Officials
		club   string
		bfvErr error
	)
	source = "BFV"
	if t.Team != "" {
		off, bfvErr = bfvOfficials(ctx, bc, t.Team)
		if bfvErr != nil && t.FuPa == "" {
			return nil, "", bfvErr
		}
	}
	if t.FuPa != "" {
		ref, err := fupaReferee(ctx, fc, t.FuPa)
		switch {
		case t.Team == "" || bfvErr != nil:
			if err != nil {
				if bfvErr != nil {
					return nil, "", fmt.Errorf("BFV: %w; FuPa: %w", bfvErr, err)
				}
				return nil, "", err
			}
			off, club, source = bfv.Officials{Referee: ref.Name}, ref.Club, "FuPa"
			if bfvErr != nil {
				source += ", BFV: " + bfvErr.Error()
			}
		case err == nil && sameName(ref.Name, off.Referee):
			club = ref.Club
		}
	}
	ref := off.Referee
	if club != "" {
		ref += " (" + club + ")"
	}
	rows = []render.Row{{Label: "Schiedsrichter:", Text: ref}}
	if len(off.Assistants) > 0 {
		rows = append(rows, render.Row{Label: "Assistenten:", Text: strings.Join(off.Assistants, ", ")})
	}
	return rows, source, nil
}

// bfvOfficials returns the officials of the next league match of a bfv.de
// team.
func bfvOfficials(ctx context.Context, c *bfv.Client, team string) (bfv.Officials, error) {
	id, err := parseTeam(team)
	if err != nil {
		return bfv.Officials{}, err
	}
	m, err := c.NextLeagueMatch(ctx, id)
	if err != nil {
		return bfv.Officials{}, err
	}
	off, err := c.MatchOfficials(ctx, m.ID)
	if err != nil {
		return bfv.Officials{}, fmt.Errorf("%s - %s: %w", m.Home, m.Guest, err)
	}
	return off, nil
}

// fupaReferee returns the referee of the next league match of a FuPa team.
func fupaReferee(ctx context.Context, c *fupa.Client, team string) (fupa.Referee, error) {
	slug, err := fupa.ParseTeam(team)
	if err != nil {
		return fupa.Referee{}, err
	}
	tm, err := c.Team(ctx, slug)
	if err != nil {
		return fupa.Referee{}, err
	}
	m, err := c.NextLeagueMatch(ctx, tm.Slug)
	if err != nil {
		return fupa.Referee{}, err
	}
	ref, err := c.Referee(ctx, m.Slug)
	if err != nil {
		return fupa.Referee{}, fmt.Errorf("%s - %s: %w", m.Home.Name, m.Away.Name, err)
	}
	return ref, nil
}

// sameName reports whether BFV and FuPa name the same person, ignoring
// case and spacing ("Dominik Von Maffei", "Dominik von Maffei") and how
// umlauts are spelt: BFV writes "Fabian Buechner", FuPa "Fabian Büchner".
func sameName(a, b string) bool {
	norm := func(s string) string {
		return umlauts.Replace(strings.ToLower(strings.Join(strings.Fields(s), " ")))
	}
	return norm(a) == norm(b)
}

var umlauts = strings.NewReplacer("ä", "ae", "ö", "oe", "ü", "ue", "ß", "ss")
