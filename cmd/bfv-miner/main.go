// Command bfv-miner writes PDFs with the previous, current and next matchday
// of BFV leagues.
//
//	bfv-miner [-o out.pdf] [-matchday N] <bfv.de league URL | competition id>
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

	"github.com/sphrilix/bfv-miner/internal/bfv"
	"github.com/sphrilix/bfv-miner/internal/render"
)

var (
	reCompoundID = regexp.MustCompile(`[0-9A-Z]{32}-G`)
	reURLDay     = regexp.MustCompile(`/spieltag/(\d+)`)
	reUnsafe     = regexp.MustCompile(`[^\pL\pN]+`)
)

// version is set by release builds (bazel build --config=release).
var version = "dev"

func main() {
	configPath := flag.String("config", "", "write every PDF listed in this JSON config instead of a single league")
	out := flag.String("o", "", "output PDF (default: <league>-spieltag-<N>.pdf)")
	matchday := flag.Int("matchday", 0, "treat this matchday as the current one (default: the matchday running today, else the next one)")
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "bfv-miner %s\nusage: %s [flags] <bfv.de league URL | competition id>\n       %s -config <file.json>\n", version, os.Args[0], os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()
	if *showVersion {
		fmt.Println("bfv-miner", version)
		return
	}

	var err error
	switch {
	case *configPath != "" && (flag.NArg() != 0 || *out != "" || *matchday != 0):
		fmt.Fprintln(os.Stderr, "bfv-miner: -config takes no league argument and no -o or -matchday")
		os.Exit(2)
	case *configPath != "":
		err = runConfig(fromWorkingDir(*configPath))
	case flag.NArg() == 1:
		err = runSingle(flag.Arg(0), *out, *matchday)
	default:
		flag.Usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "bfv-miner:", err)
		os.Exit(1)
	}
}

func runSingle(league, out string, matchday int) error {
	compoundID, urlDay, err := parseLeague(league)
	if err != nil {
		return err
	}
	if matchday == 0 {
		matchday = urlDay
	}
	w, err := fetch(compoundID, matchday)
	if err != nil {
		return err
	}
	if out == "" {
		out = fmt.Sprintf("%s-spieltag-%d.pdf", strings.Trim(reUnsafe.ReplaceAllString(w.League, "-"), "-"), w.Current)
	}
	return write(fromWorkingDir(out), w, render.Fonts{})
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
	var errs []error
	for _, t := range c.Teams {
		if err := runTeam(c.OutDir, t, fonts); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", t.File, err))
		}
	}
	return errors.Join(errs...)
}

func runTeam(outDir string, t team, fonts render.Fonts) error {
	compoundID, matchday, err := parseLeague(t.League)
	if err != nil {
		return err
	}
	w, err := fetch(compoundID, matchday)
	if err != nil {
		return err
	}
	return write(filepath.Join(outDir, t.File), w, fonts)
}

func fetch(compoundID string, matchday int) (bfv.Window, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	return bfv.NewClient().FetchWindow(ctx, compoundID, matchday)
}

func write(path string, w bfv.Window, fonts render.Fonts) error {
	err := writeFileAtomic(path, func(out io.Writer) error {
		return render.Render(out, w.Days, fonts)
	})
	if err != nil {
		return err
	}
	fmt.Printf("%s (Spieltag %d)\n", path, w.Current)
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
