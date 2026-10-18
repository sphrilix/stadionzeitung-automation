// Command bfv-miner writes a PDF with the previous, current and next matchday
// of a BFV league.
//
//	bfv-miner [-o out.pdf] [-matchday N] [-title NAME] <bfv.de league URL | competition id>
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/sphrilix/bfv-miner/internal/bfv"
	"github.com/sphrilix/bfv-miner/internal/render"
)

var (
	reCompoundID = regexp.MustCompile(`[0-9A-Z]{32}-G`)
	reURLDay     = regexp.MustCompile(`/spieltag/(\d+)`)
	reUnsafe     = regexp.MustCompile(`[^\pL\pN]+`)
)

func main() {
	out := flag.String("o", "", "output PDF (default: <league>-spieltag-<N>.pdf)")
	matchday := flag.Int("matchday", 0, "treat this matchday as the current one (default: BFV's current matchday)")
	title := flag.String("title", "", `league name for the heading (default: BFV's name, e.g. "KL-Gr2 SW")`)
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: %s [flags] <bfv.de league URL | competition id>\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}
	if err := run(flag.Arg(0), *out, *matchday, *title); err != nil {
		fmt.Fprintln(os.Stderr, "bfv-miner:", err)
		os.Exit(1)
	}
}

func run(arg, out string, matchday int, title string) error {
	compoundID, urlDay, err := parseLeague(arg)
	if err != nil {
		return err
	}
	if matchday == 0 {
		matchday = urlDay
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	w, err := bfv.NewClient().FetchWindow(ctx, compoundID, matchday)
	if err != nil {
		return err
	}
	if title == "" {
		title = w.League
	}
	if out == "" {
		out = fmt.Sprintf("%s-spieltag-%d.pdf", strings.Trim(reUnsafe.ReplaceAllString(title, "-"), "-"), w.Current)
	}

	// Under `bazel run` the process starts in the runfiles tree; resolve
	// relative paths against the directory the user invoked bazel from.
	if wd := os.Getenv("BUILD_WORKING_DIRECTORY"); wd != "" && !filepath.IsAbs(out) {
		out = filepath.Join(wd, out)
	}

	f, err := os.Create(out)
	if err != nil {
		return err
	}
	if err := render.Render(f, title, w.Days); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	fmt.Println(out)
	return nil
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
