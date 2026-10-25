package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigResolvesRelativePaths(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sz.json")
	err := os.WriteFile(path, []byte(`{
		"outDir": "Dateien",
		"fonts": {"regular": "fonts/calibri.ttf", "bold": "/abs/calibrib.ttf"},
		"teams": [{"file": "Spielplan-H1.pdf", "league": "0307PL23RO000007VS5489BUVT69HBT7-G"}]
	}`), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	c, err := loadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(dir, "Dateien"); c.OutDir != want {
		t.Errorf("OutDir = %q, want %q", c.OutDir, want)
	}
	if want := filepath.Join(dir, "fonts/calibri.ttf"); c.Fonts.Regular != want {
		t.Errorf("Fonts.Regular = %q, want %q", c.Fonts.Regular, want)
	}
	if c.Fonts.Bold != "/abs/calibrib.ttf" {
		t.Errorf("Fonts.Bold = %q, want it unchanged", c.Fonts.Bold)
	}
}

func TestLoadConfigYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sz.yaml")
	err := os.WriteFile(path, []byte(`# comment
outDir: C:\Users\x\Dateien
teams:
  - file: Kader-Gast.pdf   # trailing comment
    kind: kader
    fupa: fc-sturm-hauzenberg-m1
    opponent: true
    roles: [Trainer, Co-Trainer]
`), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	c, err := loadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	// Unquoted YAML keeps backslashes as they are. (Off Windows the path is
	// relative, hence joined with the config's directory.)
	if !strings.HasSuffix(c.OutDir, `C:\Users\x\Dateien`) {
		t.Errorf("OutDir = %q", c.OutDir)
	}
	tm := c.Teams[0]
	if tm.Kind != kindSquad || !tm.Opponent || len(tm.Roles) != 2 || tm.Roles[1] != "Co-Trainer" {
		t.Errorf("team = %+v", tm)
	}
}

func TestLoadConfigRejectsUnknownKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sz.yaml")
	if err := os.WriteFile(path, []byte("teams:\n  - file: x.pdf\n    kinde: tabelle\n    league: 0307PL23RO000007VS5489BUVT69HBT7-G\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadConfig(path); err == nil || !strings.Contains(err.Error(), "kinde") {
		t.Errorf("err = %v, want one naming the misspelt key", err)
	}
}

// The shipped example is the club's full setup; it must stay loadable.
func TestExampleConfigLoads(t *testing.T) {
	c, err := loadConfig("../../stadionzeitung.example.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Teams) < 20 {
		t.Errorf("example has %d entries, want the full setup", len(c.Teams))
	}
	if c.Fonts.Regular == "" || c.Fonts.Bold == "" {
		t.Errorf("fonts = %+v, want both set", c.Fonts)
	}
}

func TestLoadConfigRejectsIncompleteTeam(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sz.json")
	if err := os.WriteFile(path, []byte(`{"teams": [{"file": "Spielplan-H1.pdf"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadConfig(path); err == nil {
		t.Error("expected an error for a team without league")
	}
}

func TestLoadConfigRejectsUnknownKind(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sz.json")
	err := os.WriteFile(path, []byte(`{"teams": [
		{"file": "Tabelle-H1.pdf", "kind": "tabelle", "league": "0307PL23RO000007VS5489BUVT69HBT7-G"},
		{"file": "x.pdf", "kind": "kaffee", "league": "0307PL23RO000007VS5489BUVT69HBT7-G"}]}`), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := loadConfig(path); err == nil || !strings.Contains(err.Error(), "team 2") {
		t.Errorf("err = %v, want an unknown kind error for team 2", err)
	}
}

func TestWriteFileAtomicKeepsOldFileOnError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Spielplan-H1.pdf")
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := writeFileAtomic(path, func(w io.Writer) error {
		io.WriteString(w, "partial")
		return errors.New("boom")
	})
	if err == nil {
		t.Fatal("expected the write error")
	}
	if b, _ := os.ReadFile(path); string(b) != "old" {
		t.Errorf("file = %q, want the old content", b)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 1 {
		t.Errorf("leftover temp files: %v", entries)
	}

	if err := writeFileAtomic(path, func(w io.Writer) error { _, err := io.WriteString(w, "new"); return err }); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(path); string(b) != "new" {
		t.Errorf("file = %q, want the new content", b)
	}
}

func TestValidateSeasonNeedsTeam(t *testing.T) {
	if err := (team{Kind: kindSeason, League: "0307PL23RO000007VS5489BUVT69HBT7-G"}).validate(); err == nil {
		t.Error("expected an error for a spielplan without team")
	}
	if err := (team{Kind: kindSeason, Team: "016PE85GAS000000VV0AG811VTE5EA5R"}).validate(); err != nil {
		t.Errorf("spielplan with team: %v", err)
	}
}

func TestParseTeam(t *testing.T) {
	for in, want := range map[string]string{
		"https://www.bfv.de/mannschaften/fc-sturm-hauzenberg/016PE85GAS000000VV0AG811VTE5EA5R": "016PE85GAS000000VV0AG811VTE5EA5R",
		"016PE85GAS000000VV0AG811VTE5EA5R": "016PE85GAS000000VV0AG811VTE5EA5R",
	} {
		if got, err := parseTeam(in); err != nil || got != want {
			t.Errorf("parseTeam(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	// A league URL carries a competition id, not a team id.
	if got, err := parseTeam("https://www.bfv.de/ergebnisse/wettbewerb/-/0307PL23RO000007VS5489BUVT69HBT7-G"); err == nil {
		t.Errorf("parseTeam(league URL) = %q, want an error", got)
	}
}

func TestValidateScorers(t *testing.T) {
	const teamID = "016PE85GAS000000VV0AG811VTE5EA5R"
	if err := (team{Kind: kindScorers, Team: teamID, Opponent: true, Top: 4}).validate(); err != nil {
		t.Errorf("torjaeger with team: %v", err)
	}
	if err := (team{Kind: kindScorers, Team: teamID, Top: -1}).validate(); err == nil {
		t.Error("expected an error for a negative top")
	}
	if err := (team{Kind: kindScorers, League: "0307PL23RO000007VS5489BUVT69HBT7-G"}).validate(); err == nil {
		t.Error("expected an error for torjaeger without team")
	}
}

func TestValidateYouth(t *testing.T) {
	const club = "https://www.bfv.de/vereine/fc-sturm-hauzenberg/00ES8GNI7C00000KVV0AG08LVUPGND5I"
	if err := (team{Kind: kindYouthDone, Club: club}).validate(); err != nil {
		t.Errorf("jugend-ergebnisse with club: %v", err)
	}
	if err := (team{Kind: kindYouthNext, Team: "016PE85GAS000000VV0AG811VTE5EA5R"}).validate(); err == nil {
		t.Error("expected an error for jugend-vorschau without club")
	}
	if err := (team{Kind: kindYouthNext, Club: club, Days: -1}).validate(); err == nil {
		t.Error("expected an error for negative days")
	}
	if id, err := parseID(club); err != nil || id != "00ES8GNI7C00000KVV0AG08LVUPGND5I" {
		t.Errorf("parseID(club URL) = %q, %v", id, err)
	}
}

func TestValidateFuPa(t *testing.T) {
	for _, kind := range fupaKinds {
		if err := (team{Kind: kind, FuPa: "fc-sturm-hauzenberg-m1"}).validate(); err != nil {
			t.Errorf("%s with fupa: %v", kind, err)
		}
		if err := (team{Kind: kind, Team: "016PE85GAS000000VV0AG811VTE5EA5R"}).validate(); err == nil {
			t.Errorf("%s without fupa: expected an error", kind)
		}
	}
	// FuPa can stand in for the BFV team.
	if err := (team{Kind: kindScorers, FuPa: "fc-sturm-hauzenberg-m1", Opponent: true}).validate(); err != nil {
		t.Errorf("torjaeger with fupa only: %v", err)
	}
}
