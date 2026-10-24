// Command bfv-miner writes PDFs with BFV league data: the previous, current
// and next matchday, the table, or a team's season fixtures.
//
//	bfv-miner [-o out.pdf] [-kind tabelle] [-matchday N] <bfv.de league URL | competition id>
//	bfv-miner [-o out.pdf] -kind spielplan <bfv.de team URL | team id>
//	bfv-miner -config stadionzeitung.json
//	bfv-miner -version
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
	"strconv"
	"strings"
	"time"

	"github.com/sphrilix/stadionzeitung-automation/internal/bfv"
	"github.com/sphrilix/stadionzeitung-automation/internal/render"
)

var (
	reCompoundID = regexp.MustCompile(`[0-9A-Z]{32}-G`)
	reTeamID     = regexp.MustCompile(`(?:^|/)([0-9A-Z]{32})(?:$|[/?#])`)
	reURLDay     = regexp.MustCompile(`/spieltag/(\d+)`)
	reUnsafe     = regexp.MustCompile(`[^\pL\pN]+`)
)

// version is set by release builds (bazel build --config=release).
var version = "dev"

func main() {
	configPath := flag.String("config", "", "write every PDF listed in this JSON config instead of a single league")
	out := flag.String("o", "", "output PDF (default: named after the league and kind)")
	matchday := flag.Int("matchday", 0, "treat this matchday as the current one (default: the matchday running today, else the next one)")
	kind := flag.String("kind", kindMatchdays, "what to write: "+strings.Join(kinds, ", "))
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "bfv-miner %s\nusage: %s [flags] <bfv.de league URL | competition id>\n       %s [flags] -kind spielplan <bfv.de team URL | team id>\n       %s -config <file.json>\n", version, os.Args[0], os.Args[0], os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()
	if *showVersion {
		fmt.Println("bfv-miner", version)
		return
	}

	var err error
	switch {
	case *configPath != "" && (flag.NArg() != 0 || *out != "" || *matchday != 0 || *kind != kindMatchdays):
		fmt.Fprintln(os.Stderr, "bfv-miner: -config takes no league argument and no -o, -matchday or -kind")
		os.Exit(2)
	case *configPath != "":
		err = runConfig(fromWorkingDir(*configPath))
	case flag.NArg() == 1:
		err = runSingle(flag.Arg(0), *out, *kind, *matchday)
	default:
		flag.Usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "bfv-miner:", err)
		os.Exit(1)
	}
}

func runSingle(arg, out, kind string, matchday int) error {
	t := team{Kind: kind, League: arg, matchday: matchday}
	if kind == kindSeason {
		t = team{Kind: kind, Team: arg}
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

// runConfig writes every configured PDF. A failing team does not stop the
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
	for _, t := range c.Teams {
		o, err := build(t, bc)
		if err == nil {
			err = write(filepath.Join(c.OutDir, t.File), o, fonts)
		}
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", t.File, err))
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
func build(t team, c *bfv.Client) (output, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	switch t.Kind {
	case kindTable:
		return buildTable(ctx, c, t)
	case kindSeason:
		return buildSeason(ctx, c, t)
	default:
		return buildMatchdays(ctx, c, t)
	}
}

func buildMatchdays(ctx context.Context, c *bfv.Client, t team) (output, error) {
	compoundID, matchday, err := parseLeague(t.League)
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

func buildTable(ctx context.Context, c *bfv.Client, t team) (output, error) {
	compoundID, _, err := parseLeague(t.League)
	if err != nil {
		return output{}, err
	}
	tbl, err := c.FetchTable(ctx, compoundID)
	if err != nil {
		return output{}, err
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
	return output{
		label: "Tabelle",
		name:  tbl.League + "-tabelle",
		render: func(out io.Writer, fonts render.Fonts) error {
			return render.RenderTable(out, tbl, fonts, render.TableOptions{Logos: logos, Highlight: t.Highlight})
		},
	}, nil
}

func buildSeason(ctx context.Context, c *bfv.Client, t team) (output, error) {
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
	m := reTeamID.FindStringSubmatch(arg)
	if m == nil {
		return "", errors.New("no team id in URL (expected something like …/mannschaften/fc-sturm-hauzenberg/016PE85GAS000000VV0AG811VTE5EA5R)")
	}
	return m[1], nil
}
