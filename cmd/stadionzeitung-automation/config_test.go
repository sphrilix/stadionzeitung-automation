package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeConfig(t *testing.T, yaml string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "sz.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadConfigExpandsTeams(t *testing.T) {
	path := writeConfig(t, `# comment
outDir: Dateien
fonts: {regular: fonts/calibri.ttf, bold: /abs/calibrib.ttf}
club: https://www.bfv.de/vereine/fc-sturm-hauzenberg/00ES8GNI7C00000KVV0AG08LVUPGND5I
jugend: [ergebnisse]
teams:
  H2:
    bfv: https://www.bfv.de/mannschaften/fc-sturm-hauzenberg-2/02T8FR5G5O000000VS5489BRVSH2JAIF
    pdfs: [spielplan]
  H1:
    bfv: https://www.bfv.de/mannschaften/fc-sturm-hauzenberg/016PE85GAS000000VV0AG811VTE5EA5R
    fupa: fc-sturm-hauzenberg-m1
    pdfs: [spieltage, tabelle, kader]   # trailing comment
    gast: [kader, torjaeger]
    roles: [Trainer, Betreuer]
    top: 4
`)
	c, err := loadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Dir(path)
	if c.OutDir != filepath.Join(dir, "Dateien") || c.Fonts.Regular != filepath.Join(dir, "fonts/calibri.ttf") || c.Fonts.Bold != "/abs/calibrib.ttf" {
		t.Errorf("paths = %q, %+v; want them resolved against the config", c.OutDir, c.Fonts)
	}
	var files []string
	for _, e := range c.entries {
		files = append(files, e.File)
	}
	// Teams in name order, then the youth PDFs.
	want := "Spielplan-H1.pdf Tabelle-H1.pdf Kader-H1.pdf Kader-H1-Gast.pdf Torjaeger-H1-Gast.pdf Saisonspielplan-H2.pdf Jugend-Ergebnisse.pdf"
	if got := strings.Join(files, " "); got != want {
		t.Errorf("files = %s\nwant    %s", got, want)
	}
	kader, gast := c.entries[2], c.entries[3]
	if kader.FuPa != "fc-sturm-hauzenberg-m1" || kader.Top != 4 || len(kader.Roles) != 2 || kader.Opponent || kader.coachesOnly {
		t.Errorf("Kader-H1 = %+v", kader)
	}
	// The guest gets every player and its coaches only, not the team's roles.
	if !gast.Opponent || !gast.AllPlayers || !gast.coachesOnly || gast.Roles != nil {
		t.Errorf("Kader-H1-Gast = %+v", gast)
	}
	if e := c.entries[6]; e.Kind != kindYouthDone || e.Club == "" {
		t.Errorf("Jugend-Ergebnisse = %+v", e)
	}
}

func TestLoadConfigErrors(t *testing.T) {
	const bfvTeam = "https://www.bfv.de/mannschaften/fc-sturm-hauzenberg/016PE85GAS000000VV0AG811VTE5EA5R"
	for _, tc := range []struct{ yaml, want string }{
		{"teams:\n  H1:\n    bfv: " + bfvTeam + "\n    pdf: [tabelle]\n", `"pdf"`},
		{"teams:\n  H1:\n    bfv: " + bfvTeam + "\n    pdfs: [tabele]\n", `unknown pdf "tabele"`},
		{"teams:\n  H1:\n    bfv: " + bfvTeam + "\n    pdfs: [jugend-ergebnisse]\n", `unknown pdf`},
		{"teams:\n  H1:\n    bfv: " + bfvTeam + "\n    gast: [spielplan]\n", `not available for the opponent`},
		{"teams:\n  H1:\n    bfv: " + bfvTeam + "\n    pdfs: [kader]\n", `team H1: kader: needs a fupa team`},
		{"teams:\n  H1:\n    fupa: x-m1\n    pdfs: [spielplan]\n", `needs a bfv team`},
		{"teams:\n  H1:\n    fupa: x-m1\n    pdfs: [tabelle]\n", `needs a bfv team or league`},
		{"teams:\n  H1:\n    bfv: " + bfvTeam + "\n    pdfs: [tabelle, tabelle]\n", `listed twice`},
		{"teams:\n  H 1:\n    bfv: " + bfvTeam + "\n    pdfs: [tabelle]\n", `file names`},
		{"jugend: [ergebnisse]\n", `jugend: ergebnisse: needs a club`},
		{"club: x\njugend: [resultate]\n", `unknown pdf "resultate"`},
		{"outDir: x\n", `no PDFs configured`},
		{"teams:\n  - file: Tabelle-H1.pdf\n    kind: tabelle\n", `the old format`},
	} {
		_, err := loadConfig(writeConfig(t, tc.yaml))
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("config\n%s\nerr = %v, want it to contain %s", tc.yaml, err, tc.want)
		}
	}
}

// The shipped example is the club's full setup; it must stay loadable.
func TestExampleConfigLoads(t *testing.T) {
	c, err := loadConfig("../../stadionzeitung.example.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if len(c.entries) < 20 {
		t.Errorf("example has %d PDFs, want the full setup", len(c.entries))
	}
	if c.Fonts.Regular == "" || c.Fonts.Bold == "" {
		t.Errorf("fonts = %+v, want both set", c.Fonts)
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
	if err := (entry{Kind: kindSeason, League: "0307PL23RO000007VS5489BUVT69HBT7-G"}).validate(); err == nil {
		t.Error("expected an error for a spielplan without team")
	}
	if err := (entry{Kind: kindSeason, Team: "016PE85GAS000000VV0AG811VTE5EA5R"}).validate(); err != nil {
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
	if err := (entry{Kind: kindScorers, Team: teamID, Opponent: true, Top: 4}).validate(); err != nil {
		t.Errorf("torjaeger with team: %v", err)
	}
	if err := (entry{Kind: kindScorers, Team: teamID, Top: -1}).validate(); err == nil {
		t.Error("expected an error for a negative top")
	}
	if err := (entry{Kind: kindScorers, League: "0307PL23RO000007VS5489BUVT69HBT7-G"}).validate(); err == nil {
		t.Error("expected an error for torjaeger without team")
	}
}

func TestValidateYouth(t *testing.T) {
	const club = "https://www.bfv.de/vereine/fc-sturm-hauzenberg/00ES8GNI7C00000KVV0AG08LVUPGND5I"
	if err := (entry{Kind: kindYouthDone, Club: club}).validate(); err != nil {
		t.Errorf("jugend-ergebnisse with club: %v", err)
	}
	if err := (entry{Kind: kindYouthNext, Team: "016PE85GAS000000VV0AG811VTE5EA5R"}).validate(); err == nil {
		t.Error("expected an error for jugend-vorschau without club")
	}
	if err := (entry{Kind: kindYouthNext, Club: club, Days: -1}).validate(); err == nil {
		t.Error("expected an error for negative days")
	}
	if id, err := parseID(club); err != nil || id != "00ES8GNI7C00000KVV0AG08LVUPGND5I" {
		t.Errorf("parseID(club URL) = %q, %v", id, err)
	}
}

func TestValidateFuPa(t *testing.T) {
	for _, kind := range fupaKinds {
		if err := (entry{Kind: kind, FuPa: "fc-sturm-hauzenberg-m1"}).validate(); err != nil {
			t.Errorf("%s with fupa: %v", kind, err)
		}
		if err := (entry{Kind: kind, Team: "016PE85GAS000000VV0AG811VTE5EA5R"}).validate(); err == nil {
			t.Errorf("%s without fupa: expected an error", kind)
		}
	}
	// FuPa can stand in for the BFV team.
	if err := (entry{Kind: kindScorers, FuPa: "fc-sturm-hauzenberg-m1", Opponent: true}).validate(); err != nil {
		t.Errorf("torjaeger with fupa only: %v", err)
	}
}
