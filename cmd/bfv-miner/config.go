package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/sphrilix/stadionzeitung-automation/internal/render"
)

// config describes a batch of PDFs, typically the ones a newspaper layout
// links to. Relative paths are resolved against the config file's directory.
type config struct {
	// OutDir is where the PDFs are written, e.g. the folder the InDesign
	// document links its placed files from.
	OutDir string `json:"outDir"`
	Fonts  struct {
		Regular string `json:"regular"`
		Bold    string `json:"bold"`
	} `json:"fonts"`
	Teams []team `json:"teams"`
}

type team struct {
	// File is the PDF's name inside OutDir, e.g. "Spielplan-H1.pdf".
	File string `json:"file"`
	// Kind is what the PDF shows, one of kinds; empty means kindMatchdays.
	Kind string `json:"kind"`
	// League is a bfv.de league URL or competition id.
	League string `json:"league"`
	// Team is a bfv.de team URL or team id, for kinds about one team.
	Team string `json:"team"`
	// Top is how many scorers to list (ties with the last included);
	// 0 means defaultTop.
	Top int `json:"top"`
	// Opponent lists the scorers of the team's next league opponent
	// instead of the team's own.
	Opponent bool `json:"opponent"`
	// Highlight sets table rows of teams whose name contains it in bold.
	Highlight string `json:"highlight"`

	// matchday overrides the current matchday (command line only).
	matchday int
}

const (
	kindMatchdays = "spieltage"
	kindTable     = "tabelle"
	kindSeason    = "spielplan"
	kindScorers   = "torjaeger"

	defaultTop = 3
)

var kinds = []string{kindMatchdays, kindTable, kindSeason, kindScorers}

// teamKinds are the kinds about one team rather than a league.
var teamKinds = []string{kindSeason, kindScorers}

func (t team) validate() error {
	switch {
	case t.Kind != "" && !slices.Contains(kinds, t.Kind):
		return fmt.Errorf("unknown kind %q (want one of %s)", t.Kind, strings.Join(kinds, ", "))
	case slices.Contains(teamKinds, t.Kind) && t.Team == "":
		return errors.New("needs a team")
	case !slices.Contains(teamKinds, t.Kind) && t.League == "":
		return errors.New("needs a league")
	case t.Top < 0:
		return errors.New("top must not be negative")
	}
	return nil
}

func loadConfig(path string) (config, error) {
	var c config
	b, err := os.ReadFile(path)
	if err != nil {
		return c, err
	}
	if err := json.Unmarshal(b, &c); err != nil {
		return c, fmt.Errorf("%s: %w", path, err)
	}
	if len(c.Teams) == 0 {
		return c, fmt.Errorf("%s: no teams configured", path)
	}
	for i, t := range c.Teams {
		if t.File == "" {
			return c, fmt.Errorf("%s: team %d needs a file", path, i+1)
		}
		if err := t.validate(); err != nil {
			return c, fmt.Errorf("%s: team %d (%s): %w", path, i+1, t.File, err)
		}
	}
	dir := filepath.Dir(path)
	resolve := func(p string) string {
		if p == "" || filepath.IsAbs(p) {
			return p
		}
		return filepath.Join(dir, p)
	}
	c.OutDir = resolve(c.OutDir)
	c.Fonts.Regular = resolve(c.Fonts.Regular)
	c.Fonts.Bold = resolve(c.Fonts.Bold)
	return c, nil
}

func (c config) loadFonts() (render.Fonts, error) {
	var f render.Fonts
	if (c.Fonts.Regular == "") != (c.Fonts.Bold == "") {
		return f, errors.New("fonts: set both regular and bold, or neither")
	}
	if c.Fonts.Regular == "" {
		return f, nil
	}
	var err error
	if f.Regular, err = os.ReadFile(c.Fonts.Regular); err != nil {
		return f, err
	}
	if f.Bold, err = os.ReadFile(c.Fonts.Bold); err != nil {
		return f, err
	}
	return f, nil
}

// writeFileAtomic writes path via a temporary file in the same directory, so
// programs watching it (InDesign links, OneDrive) never see a partial file.
func writeFileAtomic(path string, write func(io.Writer) error) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if err := write(tmp); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	// CreateTemp makes the file private; keep the replaced file's mode.
	mode := os.FileMode(0o644)
	if fi, err := os.Stat(path); err == nil {
		mode = fi.Mode().Perm()
	}
	if err := os.Chmod(tmp.Name(), mode); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
