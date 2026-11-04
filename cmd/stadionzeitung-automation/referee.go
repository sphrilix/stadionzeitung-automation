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
// "Assistenten: Moritz Breitruck (TSV Bad Abbach), Dominik Von Maffei", and
// says where they are from. BFV lists the referee and the assistants;
// FuPa, if the team has it, adds the referee's club, and stands in while
// BFV has no referee. The other officials' clubs are looked up by name
// among the referees of the referee's group and of t.RefereeGroups.
func refereeRows(ctx context.Context, bc *bfv.Client, fc *fupa.Client, t entry) (rows []render.Row, source string, err error) {
	var (
		off     bfv.Officials
		fupaRef fupa.Referee
		fupaErr error
		bfvErr  error
	)
	source = "BFV"
	if t.Team != "" {
		off, bfvErr = bfvOfficials(ctx, bc, t.Team)
		if bfvErr != nil && t.FuPa == "" {
			return nil, "", bfvErr
		}
	}
	clubs := map[string]string{}
	if t.FuPa != "" {
		fupaRef, fupaErr = fupaReferee(ctx, fc, t.FuPa)
		switch {
		case t.Team == "" || bfvErr != nil:
			if fupaErr != nil {
				if bfvErr != nil {
					return nil, "", fmt.Errorf("BFV: %w; FuPa: %w", bfvErr, fupaErr)
				}
				return nil, "", fupaErr
			}
			off, source = bfv.Officials{Referee: fupaRef.Name}, "FuPa"
			if bfvErr != nil {
				source += ", BFV: " + bfvErr.Error()
			}
		}
		if fupaErr == nil && sameName(fupaRef.Name, off.Referee) {
			clubs[off.Referee] = fupaRef.Club
		}
		groupClubs(ctx, fc, fupaRef, t.RefereeGroups, append([]string{off.Referee}, off.Assistants...), clubs)
	}
	withClub := func(name string) string {
		if c := clubs[name]; c != "" {
			return name + " (" + c + ")"
		}
		return name
	}
	rows = []render.Row{{Label: "Schiedsrichter:", Text: withClub(off.Referee)}}
	if len(off.Assistants) > 0 {
		names := make([]string, len(off.Assistants))
		for i, a := range off.Assistants {
			names[i] = withClub(a)
		}
		rows = append(rows, render.Row{Label: "Assistenten:", Text: strings.Join(names, ", ")})
	}
	return rows, source, nil
}

// groupClubs adds the clubs of the officials not in clubs yet, looked up by
// name among the referees of the referee's group and of groups. A name
// found with more than one club is left without.
func groupClubs(ctx context.Context, fc *fupa.Client, ref fupa.Referee, groups []string, officials []string, clubs map[string]string) {
	var missing []string
	for _, o := range officials {
		if o != "" && clubs[o] == "" {
			missing = append(missing, o)
		}
	}
	if len(missing) == 0 {
		return
	}
	if ref.Slug != "" {
		if g, err := fc.RefereeGroup(ctx, ref.Slug); err == nil {
			groups = append([]string{g}, groups...)
		}
	}
	found := map[string]map[string]bool{}
	seen := map[string]bool{}
	for _, g := range groups {
		if seen[g] {
			continue
		}
		seen[g] = true
		members, err := fc.GroupReferees(ctx, g)
		if err != nil {
			warnf("referee group %s: %v", g, err)
			continue
		}
		for _, m := range members {
			for _, o := range missing {
				if m.Club != "" && sameName(m.Name, o) {
					if found[o] == nil {
						found[o] = map[string]bool{}
					}
					found[o][m.Club] = true
				}
			}
		}
	}
	for _, o := range missing {
		if len(found[o]) == 1 {
			for c := range found[o] {
				clubs[o] = c
			}
		}
	}
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
