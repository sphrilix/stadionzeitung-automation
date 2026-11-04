package main

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/sphrilix/stadionzeitung-automation/internal/bfv"
	"github.com/sphrilix/stadionzeitung-automation/internal/fupa"
	"github.com/sphrilix/stadionzeitung-automation/internal/render"
)

// errNotConfigured stands for a source the entry names no team on.
var errNotConfigured = errors.New("no team configured")

// sourcesErr reports that neither source has something.
func sourcesErr(fupaErr, bfvErr error) error {
	return fmt.Errorf("FuPa: %w; BFV: %w", cmp.Or(fupaErr, errNotConfigured), cmp.Or(bfvErr, errNotConfigured))
}

// team is an entry's team, or its next league opponent, as FuPa and BFV
// know it. Either may be missing, with the reason in fupaErr or bfvErr.
type team struct {
	name string
	// fupa is the FuPa slug with season, bfvID the BFV team id.
	fupa, bfvID     string
	fupaErr, bfvErr error
}

// resolveTeam finds an entry's team, or its next league opponent, on
// FuPa and BFV; it fails only if neither has it.
func resolveTeam(ctx context.Context, bc *bfv.Client, fc *fupa.Client, t entry) (team, error) {
	tm := team{fupaErr: errNotConfigured, bfvErr: errNotConfigured}
	if t.FuPa != "" {
		tm.fupaErr = func() error {
			slug, err := fupa.ParseTeam(t.FuPa)
			if err != nil {
				return err
			}
			ft, err := fc.Team(ctx, slug)
			if err != nil {
				return err
			}
			if t.Opponent {
				m, err := fc.NextLeagueMatch(ctx, ft.Slug)
				if err != nil {
					return err
				}
				ft = m.Opponent(ft.Slug)
			}
			tm.fupa, tm.name = ft.Slug, ft.Name
			return nil
		}()
	}
	if t.Team != "" {
		tm.bfvErr = func() error {
			id, err := parseTeam(t.Team)
			if err != nil {
				return err
			}
			if t.Opponent {
				opp, err := bc.NextOpponent(ctx, id)
				if err != nil {
					return err
				}
				tm.bfvID, tm.name = opp.ID, cmp.Or(tm.name, opp.Name)
				return nil
			}
			tm.bfvID = id
			if tm.name == "" {
				info, err := bc.FetchTeamInfo(ctx, id)
				if err != nil {
					return err
				}
				tm.name = info.Name
			}
			return nil
		}()
	}
	if tm.fupa == "" && tm.bfvID == "" {
		return team{}, sourcesErr(tm.fupaErr, tm.bfvErr)
	}
	return tm, nil
}

// buildTeam builds the kinds about one team or its next opponent: the
// squad, its top scorers and its whole page. Each part is read from FuPa,
// or from BFV where FuPa lacks it: BFV lists the players and their goals,
// without positions, shirt numbers or staff. The guest's page also lists
// the officials of the match.
func buildTeam(ctx context.Context, bc *bfv.Client, fc *fupa.Client, t entry) (output, error) {
	tm, err := resolveTeam(ctx, bc, fc, t)
	if err != nil {
		return output{}, err
	}
	top := cmp.Or(t.Top, defaultTop)

	var fs *fupa.Squad
	fupaErr := tm.fupaErr
	if tm.fupa != "" {
		if s, err := fc.Squad(ctx, tm.fupa); err == nil {
			fs = &s
		} else {
			fupaErr = err
		}
	}
	// BFV's squad is loaded only when FuPa's lacks what is needed.
	var (
		bfvPlayers []bfv.Player
		bfvErr     = tm.bfvErr
		bfvLoaded  bool
	)
	bfvSquad := func() ([]bfv.Player, error) {
		if !bfvLoaded && tm.bfvID != "" {
			bfvPlayers, bfvErr = bc.FetchSquad(ctx, tm.bfvID)
			bfvLoaded = true
		}
		return bfvPlayers, bfvErr
	}
	// scorers lists the team's top scorers from FuPa, else BFV; nobody
	// having scored yet is no error.
	scorers := func() ([]string, error) {
		if fs != nil {
			if lines := squadScorerLines(*fs, top); len(lines) > 0 {
				return lines, nil
			}
		}
		players, err := bfvSquad()
		switch {
		case err == nil:
			return scorerLines(players, top), nil
		case fs != nil:
			return nil, nil
		}
		return nil, sourcesErr(fupaErr, err)
	}

	if t.Kind == kindScorers {
		lines, err := scorers()
		if err != nil {
			return output{}, fmt.Errorf("%s: no scorers: %w", tm.name, err)
		}
		return linesOutput("Torjäger "+tm.name, lines), nil
	}

	var rows []render.Row
	if fs != nil {
		roles := func(role string) bool { return len(t.Roles) == 0 || slices.Contains(t.Roles, role) }
		if t.coachesOnly {
			roles = isCoach
		}
		rows = squadRows(*fs, roles, t.AllPlayers)
	} else {
		players, err := bfvSquad()
		if err != nil {
			return output{}, fmt.Errorf("%s: no squad: %w", tm.name, sourcesErr(fupaErr, err))
		}
		warnf("%s: squad from BFV, without positions or staff (FuPa: %v)", tm.name, fupaErr)
		rows = bfvSquadRows(players)
	}
	if t.Kind != kindTeamPage {
		return output{
			label: "Kader " + tm.name,
			name:  tm.name + "-kader",
			render: func(out io.Writer, fonts render.Fonts) error {
				return render.RenderRows(out, rows, fonts)
			},
		}, nil
	}

	if lines, err := scorers(); err != nil {
		warnf("%s: no scorers: %v", tm.name, err)
	} else if len(lines) > 0 {
		rows = append(rows, render.Row{Label: "Torjäger:", Text: strings.Join(lines, "\n"), Gap: true})
	}
	if t.Opponent {
		// Officials are often appointed only days before the match; the
		// page is still written without them.
		if refs, _, err := refereeRows(ctx, bc, fc, t); err == nil {
			refs[0].Gap = true
			rows = append(rows, refs...)
		} else {
			warnf("%s: no referees: %v", tm.name, err)
		}
	}
	g := render.TeamPage{Name: cmp.Or(t.Title, tm.name), Rows: rows}
	g.Photo, g.PhotoCredit = teamPhoto(ctx, bc, fc, tm, fs, fupaErr)
	return output{
		label: "Seite " + g.Name,
		name:  g.Name + "-seite",
		render: func(out io.Writer, fonts render.Fonts) error {
			return render.RenderTeamPage(out, g, fonts)
		},
	}, nil
}

// teamPhoto returns a team's photo and its photographer: FuPa's, else
// BFV's, else FuPa's placeholder. Without any the page is still useful;
// the layout shows a gap.
func teamPhoto(ctx context.Context, bc *bfv.Client, fc *fupa.Client, tm team, fs *fupa.Squad, fupaErr error) ([]byte, string) {
	var placeholder []byte
	if fs != nil {
		b, err := fc.TeamPhoto(ctx, *fs)
		switch {
		case err != nil:
			fupaErr = err
		case fs.PhotoPlaceholder:
			placeholder, fupaErr = b, errors.New("only a placeholder")
		default:
			return b, fs.PhotoCredit
		}
	}
	bfvErr := tm.bfvErr
	if tm.bfvID != "" {
		b, err := bc.TeamPhoto(ctx, tm.bfvID)
		if err == nil {
			return b, ""
		}
		bfvErr = err
	}
	if placeholder != nil {
		warnf("%s has no team photo on FuPa or BFV; the page shows FuPa's placeholder", tm.name)
		return placeholder, ""
	}
	warnf("%s: no team photo: %v", tm.name, sourcesErr(fupaErr, bfvErr))
	return nil, ""
}

// buildLeagueScorers builds the top scorers of an entry's league.
func buildLeagueScorers(ctx context.Context, bc *bfv.Client, fc *fupa.Client, t entry) (output, error) {
	lines, err := leagueScorerLines(ctx, bc, fc, t)
	if err != nil {
		return output{}, err
	}
	return linesOutput("Liga-Torjäger", lines), nil
}

// leagueScorerLines lists the top scorers of the league of an entry's
// team, e.g. "Almir Mesanovic (SV Türk Gücü Straubing, 16 Tore)", from
// FuPa, else BFV. A list without scorers is taken only if neither source
// has one.
func leagueScorerLines(ctx context.Context, bc *bfv.Client, fc *fupa.Client, t entry) ([]string, error) {
	top := cmp.Or(t.Top, defaultTop)
	fupaErr, bfvErr := errNotConfigured, errNotConfigured
	var lines []string
	found := false
	if t.FuPa != "" {
		fupaErr = func() error {
			slug, err := fupa.ParseTeam(t.FuPa)
			if err != nil {
				return err
			}
			tm, err := fc.Team(ctx, slug)
			if err != nil {
				return err
			}
			scorers, err := fc.LeagueScorers(ctx, tm.Competition, top)
			if err != nil {
				return err
			}
			lines, found = nil, true
			for _, s := range scorers {
				lines = append(lines, leagueScorerLine(s.Name, s.Team, s.Goals))
			}
			return nil
		}()
		if len(lines) > 0 {
			return lines, nil
		}
	}
	if t.League != "" || t.Team != "" {
		bfvErr = func() error {
			compoundID, _, _, err := league(ctx, bc, t)
			if err != nil {
				return err
			}
			scorers, err := bc.LeagueScorers(ctx, compoundID, top)
			if err != nil {
				return err
			}
			lines, found = nil, true
			for _, s := range scorers {
				lines = append(lines, leagueScorerLine(s.Name, s.Team, s.Goals))
			}
			return nil
		}()
	}
	if !found {
		return nil, fmt.Errorf("no league scorers: %w", sourcesErr(fupaErr, bfvErr))
	}
	return lines, nil
}

func leagueScorerLine(name, team string, goals int) string {
	return fmt.Sprintf("%s (%s, %s)", name, team, render.Goals(goals))
}

// squadScorerLines lists a FuPa squad's top scorers, e.g. "J. Wagner (10 Tore)".
func squadScorerLines(squad fupa.Squad, top int) []string {
	players := make([]bfv.Player, len(squad.Players))
	for i, p := range squad.Players {
		players[i] = bfv.Player{Name: p.Name(), Matches: p.Matches, Goals: p.Goals}
	}
	return scorerLines(players, top)
}

// scorerLines lists the top scorers among players.
func scorerLines(players []bfv.Player, top int) []string {
	var lines []string
	for _, p := range bfv.TopScorers(players, top) {
		lines = append(lines, render.ScorerLine(p))
	}
	return lines
}

// bfvSquadRows lays out a BFV squad, which has neither positions nor
// staff: all players in one row, sorted by last name.
func bfvSquadRows(players []bfv.Player) []render.Row {
	lastName := func(n string) string { f := strings.Fields(n); return f[len(f)-1] }
	sorted := slices.Clone(players)
	slices.SortStableFunc(sorted, func(a, b bfv.Player) int {
		if a.Name == "" || b.Name == "" {
			return cmp.Compare(a.Name, b.Name)
		}
		return cmp.Or(cmp.Compare(lastName(a.Name), lastName(b.Name)), cmp.Compare(a.Name, b.Name))
	})
	names := make([]string, 0, len(sorted))
	for _, p := range sorted {
		if p.Name != "" {
			names = append(names, p.Name)
		}
	}
	return []render.Row{{Label: "Spieler:", Text: strings.Join(names, ", "), Gap: true}}
}

// warnf reports a problem that does not stop a PDF from being written.
func warnf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "warning: "+format+"\n", args...)
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
