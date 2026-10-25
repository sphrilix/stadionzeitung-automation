// Command stadionzeitung-automation writes PDFs with BFV data: a league's previous, current
// and next matchday or its table, a team's season fixtures or top scorers,
// or a club's recent and upcoming youth matches. With FuPa's permission it
// adds what BFV lacks: squads, league top scorers and referees.
//
//	stadionzeitung-automation [-o out.pdf] [-kind tabelle] [-matchday N] <bfv.de league URL | competition id>
//	stadionzeitung-automation [-o out.pdf] -kind spielplan|torjaeger <bfv.de team URL | team id>
//	stadionzeitung-automation [-o out.pdf] -kind jugend-ergebnisse|jugend-vorschau <bfv.de club URL | club id>
//	stadionzeitung-automation [-o out.pdf] -kind kader|liga-torjaeger|schiedsrichter <fupa.net team URL | slug>
//	stadionzeitung-automation -config stadionzeitung.yaml
//	stadionzeitung-automation -version
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/sphrilix/stadionzeitung-automation/internal/bfv"
	"github.com/sphrilix/stadionzeitung-automation/internal/fupa"
	"github.com/sphrilix/stadionzeitung-automation/internal/render"
)

var (
	reCompoundID = regexp.MustCompile(`[0-9A-Z]{32}-G`)
	reID         = regexp.MustCompile(`(?:^|/)([0-9A-Z]{32})(?:$|[/?#])`)
	reURLDay     = regexp.MustCompile(`/spieltag/(\d+)`)
	reUnsafe     = regexp.MustCompile(`[^\pL\pN]+`)
)

// version is set by release builds (bazel build --config=release).
var version = "dev"

func main() {
	configPath := flag.String("config", "", "write every PDF listed in this YAML (or JSON) config instead of a single league")
	out := flag.String("o", "", "output PDF (default: named after the league and kind)")
	matchday := flag.Int("matchday", 0, "treat this matchday as the current one (default: the matchday running today, else the next one)")
	kind := flag.String("kind", kindMatchdays, "what to write: "+strings.Join(kinds, ", "))
	days := flag.Int("days", 0, fmt.Sprintf("for the jugend kinds: how many days back or ahead to list (default %d)", defaultDays))
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "stadionzeitung-automation %s\nusage: %s [flags] <bfv.de league URL | competition id>\n       %s [flags] -kind spielplan|torjaeger <bfv.de team URL | team id>\n       %s [flags] -kind jugend-ergebnisse|jugend-vorschau <bfv.de club URL | club id>\n       %s [flags] -kind kader|liga-torjaeger|schiedsrichter <fupa.net team URL | slug>\n       %s -config <file.yaml>\n", version, os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()
	if *showVersion {
		fmt.Println("stadionzeitung-automation", version)
		return
	}

	var err error
	switch {
	case *configPath != "" && (flag.NArg() != 0 || *out != "" || *matchday != 0 || *kind != kindMatchdays || *days != 0):
		fmt.Fprintln(os.Stderr, "stadionzeitung-automation: -config takes no league argument and no -o, -matchday, -kind or -days")
		os.Exit(2)
	case *configPath != "":
		err = runConfig(fromWorkingDir(*configPath))
	case flag.NArg() == 1:
		err = runSingle(flag.Arg(0), *out, *kind, *matchday, *days)
	default:
		flag.Usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "stadionzeitung-automation:", err)
		os.Exit(1)
	}
}

func runSingle(arg, out, kind string, matchday, days int) error {
	t := entry{Kind: kind, League: arg, matchday: matchday}
	switch {
	case !reCompoundID.MatchString(arg) && !slices.Contains(clubKinds, kind) && !slices.Contains(fupaKinds, kind):
		// A team URL: its league is looked up.
		t = entry{Kind: kind, Team: arg, matchday: matchday}
	case slices.Contains(teamKinds, kind):
		t = entry{Kind: kind, Team: arg}
	case slices.Contains(clubKinds, kind):
		t = entry{Kind: kind, Club: arg, Days: days}
	case slices.Contains(fupaKinds, kind):
		t = entry{Kind: kind, FuPa: arg}
	}
	if err := t.validate(); err != nil {
		return err
	}
	o, err := build(t, bfv.NewClient())
	if err != nil {
		return err
	}
	if out == "" {
		out = strings.Trim(reUnsafe.ReplaceAllString(o.name, "-"), "-") + ".pdf"
	}
	return write(fromWorkingDir(out), o, render.Fonts{})
}

// runConfig writes every configured PDF. A failing PDF does not stop the
// others; all failures are reported together.
func runConfig(path string) error {
	c, err := loadConfig(path)
	if err != nil {
		return err
	}
	fonts, err := c.loadFonts()
	if err != nil {
		return err
	}
	bc := bfv.NewClient()
	var errs []error
	for _, e := range c.entries {
		o, err := build(e, bc)
		if err == nil {
			err = write(filepath.Join(c.OutDir, e.File), o, fonts)
		}
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", e.File, err))
		}
	}
	return errors.Join(errs...)
}

// output is a fetched PDF, ready to be rendered.
type output struct {
	// label describes the content in the progress line, name is the default
	// file name (without extension) when writing a single PDF.
	label, name string
	render      func(io.Writer, render.Fonts) error
}

// build fetches the data for one config entry.
func build(t entry, c *bfv.Client) (output, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	switch t.Kind {
	case kindTable:
		return buildTable(ctx, c, t)
	case kindSeason:
		return buildSeason(ctx, c, t)
	case kindYouthDone, kindYouthNext:
		return buildYouth(ctx, c, t)
	case kindSquad, kindLeagueTop, kindReferee, kindGuestPage:
		return buildFuPa(ctx, fupa.NewClient(), t)
	case kindTableTop:
		return buildTableScorers(ctx, c, fupa.NewClient(), t)
	case kindScorers:
		if t.FuPa != "" {
			return buildFuPa(ctx, fupa.NewClient(), t)
		}
		return buildScorers(ctx, c, t)
	default:
		return buildMatchdays(ctx, c, t)
	}
}

// league returns the competition id of an entry's league, the matchday a
// league URL names (0 if none), and the name of the entry's team, if any.
func league(ctx context.Context, c *bfv.Client, t entry) (compoundID string, matchday int, team string, err error) {
	if t.League != "" {
		compoundID, matchday, err = parseLeague(t.League)
		return compoundID, matchday, "", err
	}
	teamID, err := parseTeam(t.Team)
	if err != nil {
		return "", 0, "", err
	}
	info, err := c.FetchTeamInfo(ctx, teamID)
	return info.CompoundID, 0, info.Name, err
}

func buildMatchdays(ctx context.Context, c *bfv.Client, t entry) (output, error) {
	compoundID, matchday, _, err := league(ctx, c, t)
	if err != nil {
		return output{}, err
	}
	if t.matchday != 0 {
		matchday = t.matchday
	}
	w, err := c.FetchWindow(ctx, compoundID, matchday)
	if err != nil {
		return output{}, err
	}
	return output{
		label: fmt.Sprintf("Spieltag %d", w.Current),
		name:  fmt.Sprintf("%s-spieltag-%d", w.League, w.Current),
		render: func(out io.Writer, fonts render.Fonts) error {
			return render.Render(out, w.Days, fonts)
		},
	}, nil
}

func buildTable(ctx context.Context, c *bfv.Client, t entry) (output, error) {
	tbl, opts, err := fetchTable(ctx, c, t)
	if err != nil {
		return output{}, err
	}
	return output{
		label: "Tabelle",
		name:  tbl.League + "-tabelle",
		render: func(out io.Writer, fonts render.Fonts) error {
			return render.RenderTable(out, tbl, fonts, opts)
		},
	}, nil
}

// fetchTable loads the table of an entry's league with the clubs' logos,
// the entry's team set in bold.
func fetchTable(ctx context.Context, c *bfv.Client, t entry) (bfv.Table, render.TableOptions, error) {
	compoundID, _, highlight, err := league(ctx, c, t)
	if err != nil {
		return bfv.Table{}, render.TableOptions{}, err
	}
	tbl, err := c.FetchTable(ctx, compoundID)
	if err != nil {
		return bfv.Table{}, render.TableOptions{}, err
	}
	// A missing logo only leaves its cell empty.
	logos := map[string][]byte{}
	for _, r := range tbl.Rows {
		if r.ClubID == "" {
			continue
		}
		if b, err := c.ClubLogo(ctx, r.ClubID); err == nil {
			logos[r.ClubID] = b
		}
	}
	return tbl, render.TableOptions{Logos: logos, Highlight: highlight}, nil
}

// buildTableScorers builds the table with the league's top scorers from
// FuPa below it, as the table pages show them.
func buildTableScorers(ctx context.Context, c *bfv.Client, fc *fupa.Client, t entry) (output, error) {
	tbl, opts, err := fetchTable(ctx, c, t)
	if err != nil {
		return output{}, err
	}
	scorers, err := leagueScorerLines(ctx, fc, t)
	if err != nil {
		return output{}, err
	}
	return output{
		label: "Tabelle und Torjäger",
		name:  tbl.League + "-tabelle-torjaeger",
		render: func(out io.Writer, fonts render.Fonts) error {
			return render.RenderTableScorers(out, tbl, opts, scorers, fonts)
		},
	}, nil
}

func buildSeason(ctx context.Context, c *bfv.Client, t entry) (output, error) {
	teamID, err := parseTeam(t.Team)
	if err != nil {
		return output{}, err
	}
	s, err := c.FetchSeason(ctx, teamID)
	if err != nil {
		return output{}, err
	}
	return output{
		label: "Spielplan " + s.Team,
		name:  s.Team + "-spielplan",
		render: func(out io.Writer, fonts render.Fonts) error {
			return render.RenderSeason(out, s, fonts)
		},
	}, nil
}

func buildScorers(ctx context.Context, c *bfv.Client, t entry) (output, error) {
	teamID, err := parseTeam(t.Team)
	if err != nil {
		return output{}, err
	}
	who := "own team"
	if t.Opponent {
		opp, err := c.NextOpponent(ctx, teamID)
		if err != nil {
			return output{}, err
		}
		teamID, who = opp.ID, opp.Name
	}
	squad, err := c.FetchSquad(ctx, teamID)
	if err != nil {
		return output{}, fmt.Errorf("%s: %w", who, err)
	}
	top := t.Top
	if top == 0 {
		top = defaultTop
	}
	scorers := bfv.TopScorers(squad, top)
	return output{
		label: "Torjäger " + who,
		name:  "torjaeger",
		render: func(out io.Writer, fonts render.Fonts) error {
			return render.RenderScorers(out, scorers, fonts)
		},
	}, nil
}

func buildYouth(ctx context.Context, c *bfv.Client, t entry) (output, error) {
	clubID, err := parseID(t.Club)
	if err != nil {
		return output{}, err
	}
	days := t.Days
	if days == 0 {
		days = defaultDays
	}
	// Results: the last days up to now; preview: from now through the
	// coming days.
	now := time.Now()
	y, m, d := now.Date()
	today := time.Date(y, m, d, 0, 0, 0, 0, now.Location())
	from, to := today.AddDate(0, 0, -days), now
	label := "Jugend-Ergebnisse"
	if t.Kind == kindYouthNext {
		from, to = now, today.AddDate(0, 0, days+1)
		label = "Jugend-Vorschau"
	}
	all, err := c.FetchClubMatches(ctx, clubID, from, to)
	if err != nil {
		return output{}, err
	}
	var youth []bfv.ClubMatch
	for _, m := range all {
		if m.IsYouth() {
			youth = append(youth, m)
		}
	}
	return output{
		label: fmt.Sprintf("%s, %d Spiele", label, len(youth)),
		name:  strings.ToLower(label),
		render: func(out io.Writer, fonts render.Fonts) error {
			return render.RenderClubMatches(out, youth, t.Kind == kindYouthDone, fonts)
		},
	}, nil
}

func write(path string, o output, fonts render.Fonts) error {
	err := writeFileAtomic(path, func(out io.Writer) error {
		return o.render(out, fonts)
	})
	if err != nil {
		return err
	}
	fmt.Printf("%s (%s)\n", path, o.label)
	return nil
}

// fromWorkingDir resolves a relative path against the directory the user ran
// the command from. Under `bazel run` the process starts in the runfiles tree
// instead, with the original directory in BUILD_WORKING_DIRECTORY.
func fromWorkingDir(p string) string {
	if wd := os.Getenv("BUILD_WORKING_DIRECTORY"); wd != "" && !filepath.IsAbs(p) {
		return filepath.Join(wd, p)
	}
	return p
}

// parseLeague extracts the competition id and an optional matchday from a
// bfv.de URL or a bare competition id.
func parseLeague(arg string) (string, int, error) {
	ids := reCompoundID.FindAllString(arg, -1)
	if len(ids) == 0 {
		return "", 0, errors.New("no competition id in URL (expected something like …/03135AVGGC000009VS5489BTVU7GTVLE-G)")
	}
	day := 0
	if m := reURLDay.FindStringSubmatch(arg); m != nil {
		day, _ = strconv.Atoi(m[1])
	}
	return ids[len(ids)-1], day, nil
}

// parseTeam extracts the team id from a bfv.de team URL or a bare team id.
func parseTeam(arg string) (string, error) {
	id, err := parseID(arg)
	if err != nil {
		return "", errors.New("no team id in URL (expected something like …/mannschaften/fc-sturm-hauzenberg/016PE85GAS000000VV0AG811VTE5EA5R)")
	}
	return id, nil
}

// parseID extracts a team or club id from a bfv.de URL or a bare id.
func parseID(arg string) (string, error) {
	m := reID.FindStringSubmatch(arg)
	if m == nil {
		return "", errors.New("no team or club id in URL (expected something like …/vereine/fc-sturm-hauzenberg/00ES8GNI7C00000KVV0AG08LVUPGND5I)")
	}
	return m[1], nil
}
