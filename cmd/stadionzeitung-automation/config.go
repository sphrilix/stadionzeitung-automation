package main

import (
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"sigs.k8s.io/yaml"

	"github.com/sphrilix/stadionzeitung-automation/internal/render"
)

// config describes the PDFs a newspaper layout links to, per team: its
// addresses on bfv.de and FuPa, and which PDFs to write for it. Relative
// paths are resolved against the config file's directory. The file is YAML.
type config struct {
	// OutDir is where the PDFs are written, e.g. the folder the InDesign
	// document links its placed files from.
	OutDir string `json:"outDir"`
	Fonts  struct {
		Regular string `json:"regular"`
		Bold    string `json:"bold"`
	} `json:"fonts"`
	// Club is the club's bfv.de page, for the youth PDFs.
	Club string `json:"club"`
	// Jugend lists the club-wide youth PDFs: "ergebnisse", "vorschau".
	Jugend []string `json:"jugend"`
	// Days is how far back (ergebnisse) or ahead (vorschau) the youth PDFs
	// list matches; 0 means defaultDays.
	Days int `json:"days"`
	// Teams maps a short name used in the file names, e.g. "H1", to a team.
	Teams map[string]teamConfig `json:"teams"`

	// entries are the PDFs to write, expanded from the above.
	entries []entry
}

// teamConfig is one team and the PDFs wanted for it.
type teamConfig struct {
	// BFV is the team's bfv.de page; it also gives the team's league.
	BFV string `json:"bfv"`
	// FuPa is the team's fupa.net page or slug. Only set it with FuPa's
	// permission.
	FuPa string `json:"fupa"`
	// PDFs are the kinds to write for the team.
	PDFs []string `json:"pdfs"`
	// Gast are the kinds to write for the team's next league opponent.
	Gast []string `json:"gast"`
	// Top is how many scorers to list; 0 means defaultTop.
	Top int `json:"top"`
	// Roles limits the team's kader staff to these roles (as FuPa names
	// them, e.g. "Trainer"); empty means all.
	Roles []string `json:"roles"`
	// AllPlayers lists every player in the team's kader, not only those
	// with a shirt number.
	AllPlayers bool `json:"allPlayers"`
}

// entry is one PDF to write.
type entry struct {
	// File is the PDF's name inside OutDir, e.g. "Tabelle-H1.pdf".
	File string
	// Kind is what the PDF shows, one of kinds; empty means kindMatchdays.
	Kind string
	// League is a bfv.de league URL or competition id; when empty, the
	// league of Team is used.
	League string
	// Team is a bfv.de team URL or team id.
	Team string
	// Club is a bfv.de club URL or club id, for kinds about all its teams.
	Club string
	// FuPa is a fupa.net team URL or slug. Kinds that need FuPa read it
	// from there; torjaeger prefers it over BFV, whose squads can be
	// private.
	FuPa string
	// Opponent writes the PDF for the team's next league opponent.
	Opponent bool
	// Top is how many scorers to list (ties with the last included);
	// 0 means defaultTop.
	Top int
	// Days is how far back (results) or ahead (preview) club matches are
	// listed; 0 means defaultDays.
	Days int
	// Roles limits a kader's staff to these roles; empty means all.
	// coachesOnly keeps only the coaches instead, as for a guest.
	Roles       []string
	coachesOnly bool
	// AllPlayers lists every player in a kader, not only those with a
	// shirt number.
	AllPlayers bool

	// matchday overrides the current matchday (command line only).
	matchday int
}

const (
	kindMatchdays = "spieltage"
	kindTable     = "tabelle"
	kindSeason    = "spielplan"
	kindScorers   = "torjaeger"
	kindYouthDone = "jugend-ergebnisse"
	kindYouthNext = "jugend-vorschau"
	kindSquad     = "kader"
	kindLeagueTop = "liga-torjaeger"
	kindReferee   = "schiedsrichter"

	defaultTop  = 3
	defaultDays = 7
)

var kinds = []string{kindMatchdays, kindTable, kindSeason, kindScorers, kindYouthDone, kindYouthNext,
	kindSquad, kindLeagueTop, kindReferee}

// teamKinds are the kinds about one team rather than a league, clubKinds
// those about all teams of a club, fupaKinds those only FuPa has data for.
var (
	teamKinds = []string{kindSeason, kindScorers}
	clubKinds = []string{kindYouthDone, kindYouthNext}
	fupaKinds = []string{kindSquad, kindLeagueTop, kindReferee}
	// gastKinds can be written for a team's next opponent.
	gastKinds = []string{kindSquad, kindScorers}
)

// filePrefix names each kind's PDFs: <prefix>-<team>.pdf. The matchdays
// keep "Spielplan", the name the Stadionzeitung's layout links them by.
var filePrefix = map[string]string{
	kindMatchdays: "Spielplan",
	kindTable:     "Tabelle",
	kindSeason:    "Saisonspielplan",
	kindScorers:   "Torjaeger",
	kindSquad:     "Kader",
	kindLeagueTop: "Liga-Torjaeger",
	kindReferee:   "Schiedsrichter",
	kindYouthDone: "Jugend-Ergebnisse",
	kindYouthNext: "Jugend-Vorschau",
}

// youthKinds maps the config's jugend entries to their kinds.
var youthKinds = map[string]string{"ergebnisse": kindYouthDone, "vorschau": kindYouthNext}

// reTeamName limits team names to what is safe in a file name.
var reTeamName = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func (e entry) validate() error {
	switch {
	case e.Kind != "" && !slices.Contains(kinds, e.Kind):
		return fmt.Errorf("unknown kind %q (want one of %s)", e.Kind, strings.Join(kinds, ", "))
	case slices.Contains(fupaKinds, e.Kind):
		if e.FuPa == "" {
			return errors.New("needs a fupa team")
		}
	case e.Kind == kindScorers && e.FuPa != "":
		// FuPa replaces the BFV team.
	case slices.Contains(teamKinds, e.Kind):
		if e.Team == "" {
			return errors.New("needs a bfv team")
		}
	case slices.Contains(clubKinds, e.Kind):
		if e.Club == "" {
			return errors.New("needs a club")
		}
	case e.League == "" && e.Team == "":
		return errors.New("needs a bfv team or league")
	}
	if e.Top < 0 || e.Days < 0 {
		return errors.New("top and days must not be negative")
	}
	return nil
}

func loadConfig(path string) (config, error) {
	var c config
	b, err := os.ReadFile(path)
	if err != nil {
		return c, err
	}
	// Strict, so a misspelt key is reported instead of silently ignored.
	if err := yaml.UnmarshalStrict(b, &c); err != nil {
		var old struct {
			Teams []any `json:"teams"`
		}
		if yaml.Unmarshal(b, &old) == nil && len(old.Teams) > 0 {
			return c, fmt.Errorf("%s: teams is a list of PDFs, the old format; list each team once with its addresses and pdfs instead, see stadionzeitung.example.yaml", path)
		}
		return c, fmt.Errorf("%s: %w", path, err)
	}
	if err := c.expand(); err != nil {
		return c, fmt.Errorf("%s: %w", path, err)
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

// expand turns the teams' wanted PDFs and the youth PDFs into entries,
// teams in name order.
func (c *config) expand() error {
	names := slices.Sorted(maps.Keys(c.Teams))
	for _, name := range names {
		tc := c.Teams[name]
		if !reTeamName.MatchString(name) {
			return fmt.Errorf("team %q: use only letters, digits, - and _ in team names, they become file names", name)
		}
		add := func(kind string, gast bool) error {
			if gast && !slices.Contains(gastKinds, kind) {
				return fmt.Errorf("team %s: gast: %q is not available for the opponent (want one of %s)", name, kind, strings.Join(gastKinds, ", "))
			}
			if !gast && (!slices.Contains(kinds, kind) || slices.Contains(clubKinds, kind)) {
				return fmt.Errorf("team %s: unknown pdf %q (want one of %s)", name, kind, strings.Join(teamPDFs(), ", "))
			}
			e := entry{
				File: filePrefix[kind] + "-" + name + ".pdf",
				Kind: kind, Team: tc.BFV, FuPa: tc.FuPa, Top: tc.Top,
				Roles: tc.Roles, AllPlayers: tc.AllPlayers,
			}
			if gast {
				// Guests often number only part of their squad, and the
				// guest page lists their coaches only.
				e.File = filePrefix[kind] + "-" + name + "-Gast.pdf"
				e.Opponent, e.AllPlayers, e.Roles, e.coachesOnly = true, true, nil, true
			}
			if slices.ContainsFunc(c.entries, func(o entry) bool { return o.File == e.File }) {
				return fmt.Errorf("team %s: %s is listed twice", name, kind)
			}
			if err := e.validate(); err != nil {
				return fmt.Errorf("team %s: %s: %w", name, kind, err)
			}
			c.entries = append(c.entries, e)
			return nil
		}
		for _, kind := range tc.PDFs {
			if err := add(kind, false); err != nil {
				return err
			}
		}
		for _, kind := range tc.Gast {
			if err := add(kind, true); err != nil {
				return err
			}
		}
	}
	for _, j := range c.Jugend {
		kind, ok := youthKinds[j]
		if !ok {
			return fmt.Errorf("jugend: unknown pdf %q (want ergebnisse or vorschau)", j)
		}
		e := entry{File: filePrefix[kind] + ".pdf", Kind: kind, Club: c.Club, Days: c.Days}
		if err := e.validate(); err != nil {
			return fmt.Errorf("jugend: %s: %w", j, err)
		}
		c.entries = append(c.entries, e)
	}
	if len(c.entries) == 0 {
		return errors.New("no PDFs configured")
	}
	return nil
}

// teamPDFs are the kinds a team's pdfs can list.
func teamPDFs() []string {
	var out []string
	for _, k := range kinds {
		if !slices.Contains(clubKinds, k) {
			out = append(out, k)
		}
	}
	return out
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
