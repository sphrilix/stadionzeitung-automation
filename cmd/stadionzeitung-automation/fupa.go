package main

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/sphrilix/stadionzeitung-automation/internal/bfv"
	"github.com/sphrilix/stadionzeitung-automation/internal/fupa"
	"github.com/sphrilix/stadionzeitung-automation/internal/render"
)

// buildFuPa builds the kinds read from FuPa: the squad, a team's or the
// league's top scorers, and the referee of the next league match.
func buildFuPa(ctx context.Context, c *fupa.Client, t entry) (output, error) {
	slug, err := fupa.ParseTeam(t.FuPa)
	if err != nil {
		return output{}, err
	}
	tm, err := c.Team(ctx, slug)
	if err != nil {
		return output{}, err
	}
	top := cmp.Or(t.Top, defaultTop)

	switch t.Kind {
	case kindLeagueTop:
		lines, err := leagueScorerLines(ctx, c, t)
		if err != nil {
			return output{}, err
		}
		return linesOutput("Liga-Torjäger", lines), nil

	case kindReferee:
		m, err := c.NextLeagueMatch(ctx, tm.Slug)
		if err != nil {
			return output{}, err
		}
		ref, err := c.Referee(ctx, m.Slug)
		if err != nil {
			return output{}, fmt.Errorf("%s - %s: %w", m.Home.Name, m.Away.Name, err)
		}
		line := ref.Name
		if ref.Club != "" {
			line += " (" + ref.Club + ")"
		}
		return linesOutput("Schiedsrichter", []string{line}), nil
	}

	// Kader, torjaeger and the guest page are about the team or its next
	// opponent.
	if t.Opponent {
		m, err := c.NextLeagueMatch(ctx, tm.Slug)
		if err != nil {
			return output{}, err
		}
		tm = m.Opponent(tm.Slug)
	}
	squad, err := c.Squad(ctx, tm.Slug)
	if err != nil {
		return output{}, err
	}
	if t.Kind == kindScorers {
		return linesOutput("Torjäger "+tm.Name, squadScorerLines(squad, top)), nil
	}
	roles := func(role string) bool { return len(t.Roles) == 0 || slices.Contains(t.Roles, role) }
	if t.coachesOnly {
		roles = isCoach
	}
	rows := squadRows(squad, roles, t.AllPlayers)
	if t.Kind == kindTeamPage {
		if lines := squadScorerLines(squad, top); len(lines) > 0 {
			rows = append(rows, render.Row{Label: "Torjäger:", Text: strings.Join(lines, "\n"), Gap: true})
		}
		g := render.TeamPage{Name: cmp.Or(t.Title, tm.Name), Rows: rows}
		// Without a photo the page is still useful; the layout shows a gap.
		if photo, err := c.TeamPhoto(ctx, squad); err == nil {
			g.Photo, g.PhotoCredit = photo, squad.PhotoCredit
		}
		return output{
			label: "Seite " + g.Name,
			name:  g.Name + "-seite",
			render: func(out io.Writer, fonts render.Fonts) error {
				return render.RenderTeamPage(out, g, fonts)
			},
		}, nil
	}
	return output{
		label: "Kader " + tm.Name,
		name:  tm.Name + "-kader",
		render: func(out io.Writer, fonts render.Fonts) error {
			return render.RenderRows(out, rows, fonts)
		},
	}, nil
}

// leagueScorerLines lists the top scorers of the league of an entry's FuPa
// team, e.g. "Almir Mesanovic (SV Türk Gücü Straubing, 16 Tore)".
func leagueScorerLines(ctx context.Context, c *fupa.Client, t entry) ([]string, error) {
	slug, err := fupa.ParseTeam(t.FuPa)
	if err != nil {
		return nil, err
	}
	tm, err := c.Team(ctx, slug)
	if err != nil {
		return nil, err
	}
	scorers, err := c.LeagueScorers(ctx, tm.Competition, cmp.Or(t.Top, defaultTop))
	if err != nil {
		return nil, err
	}
	lines := make([]string, len(scorers))
	for i, s := range scorers {
		lines[i] = fmt.Sprintf("%s (%s, %s)", s.Name, s.Team, render.Goals(s.Goals))
	}
	return lines, nil
}

// squadScorerLines lists a squad's top scorers, e.g. "J. Wagner (10 Tore)".
func squadScorerLines(squad fupa.Squad, top int) []string {
	players := make([]bfv.Player, len(squad.Players))
	for i, p := range squad.Players {
		players[i] = bfv.Player{Name: p.Name(), Matches: p.Matches, Goals: p.Goals}
	}
	var lines []string
	for _, p := range bfv.TopScorers(players, top) {
		lines = append(lines, render.ScorerLine(p))
	}
	return lines
}

// isCoach reports whether a FuPa staff role is a coach's, e.g. "Trainer",
// "Co-Trainer" or "Spielertrainer".
func isCoach(role string) bool { return strings.Contains(strings.ToLower(role), "trainer") }

func linesOutput(label string, lines []string) output {
	return output{
		label: label,
		name:  label,
		render: func(out io.Writer, fonts render.Fonts) error {
			return render.RenderLines(out, lines, fonts)
		},
	}
}

// positions maps FuPa's positions to the Kader pages' labels, in order.
var positions = []struct{ fupa, label string }{
	{"Torwart", "Tor:"},
	{"Abwehr", "Abwehr:"},
	{"Mittelfeld", "Mittelfeld:"},
	{"Angriff", "Angriff:"},
}

// shortRoles abbreviates staff roles that would not fit the label column.
var shortRoles = map[string]string{
	"Torwart-Trainer": "TW-Trainer",
	"Physiotherapeut": "Physio",
}

// squadRows lays out a squad as on the Kader pages: the staff whose role
// is wanted, by role, then the players by position, sorted by last name,
// the captain marked "(K)". Unless all is set, only players with a shirt
// number are listed.
func squadRows(s fupa.Squad, wanted func(role string) bool, all bool) []render.Row {
	var rows []render.Row
	byRole := map[string][]string{}
	var roles []string
	for _, st := range s.Staff {
		if !wanted(st.Role) {
			continue
		}
		if _, ok := byRole[st.Role]; !ok {
			roles = append(roles, st.Role)
		}
		byRole[st.Role] = append(byRole[st.Role], st.Name)
	}
	for _, role := range roles {
		label := cmp.Or(shortRoles[role], role)
		rows = append(rows, render.Row{Label: label + ":", Text: strings.Join(byRole[role], ", ")})
	}

	players := slices.Clone(s.Players)
	if !all {
		players = slices.Clone(s.Numbered())
	}
	slices.SortStableFunc(players, func(a, b fupa.Player) int {
		return cmp.Or(cmp.Compare(a.LastName, b.LastName), cmp.Compare(a.FirstName, b.FirstName))
	})
	known := map[string]bool{}
	first := true
	add := func(label string, match func(fupa.Player) bool) {
		var names []string
		for _, p := range players {
			if match(p) {
				n := p.Name()
				if p.Captain {
					n += " (K)"
				}
				names = append(names, n)
			}
		}
		if len(names) == 0 {
			return
		}
		rows = append(rows, render.Row{Label: label, Text: strings.Join(names, ", "), Gap: first})
		first = false
	}
	for _, pos := range positions {
		known[pos.fupa] = true
		add(pos.label, func(p fupa.Player) bool { return p.Position == pos.fupa })
	}
	add("Weitere:", func(p fupa.Player) bool { return !known[p.Position] })
	return rows
}
