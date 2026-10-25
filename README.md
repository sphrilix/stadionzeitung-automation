# Stadionzeitung Automation

[![CI](https://github.com/sphrilix/stadionzeitung-automation/actions/workflows/ci.yml/badge.svg)](https://github.com/sphrilix/stadionzeitung-automation/actions/workflows/ci.yml)

Generates the automatable parts of a club's matchday programme
("Stadionzeitung") as PDFs: tables, matchdays, season schedules, scorers,
youth results, squads and more. It was built for the *Staffelberg-Kicker*
of FC Sturm Hauzenberg, but works for any club in the Bavarian football
association (BFV).

The InDesign document places each PDF as a linked graphic. Before every
issue, one command regenerates them all from current data, and InDesign
picks up the changes when you update its links. Each PDF has no heading
(the layout provides that), and its page is cropped to the content, so it
drops straight into a frame.

```sh
stadionzeitung-automation -config stadionzeitung.yaml
```

> **Unofficial.** This tool is not affiliated with or endorsed by the
> Bayerischer Fußball-Verband (BFV) or FuPa. It reads public bfv.de pages
> and the widget API behind BFV's embeddable widgets, both of which can
> change without notice and break the tool. Match data belongs to the BFV;
> check its terms before publishing generated tables beyond your own club's
> use.

## What it generates

Each entry in the config writes one PDF; its `kind` says what goes in it.
The pages refer to the 40-page issue of 17.10.2026; see
[docs/automation.md](docs/automation.md) for the full page-by-page
inventory, including what stays manual.

| Kind | Content | Programme page | Source |
| --- | --- | --- | --- |
| `spieltage` | previous, current and next matchday with results | Aktuelle Spieltage (15, 31) | BFV |
| `tabelle` | league table with club logos, goals and promotion/relegation zones in colour | Tabelle (13, 29, 37) | BFV |
| `spielplan` | a team's season fixtures: matchday, date, kickoff, result, byes included | Saisonspielplan (17, 33, 35) | BFV |
| `torjaeger` | a team's top scorers, e.g. "J. Wagner (10 Tore)" | Kader (11, 27), guest (5) | BFV or FuPa |
| `jugend-ergebnisse` | results of all the club's youth teams over the last days | Jugend (21) | BFV |
| `jugend-vorschau` | the youth teams' upcoming matches | Jugend (23) | BFV |
| `kader` | squad by position with captain, coaches and staff | Kader (11, 27), guest (5) | FuPa |
| `liga-torjaeger` | the league's top scorers with their teams | Tabelle (13, 29) | FuPa |
| `schiedsrichter` | the referee of the next league match and their club | Grußwort (3), guest (5) | FuPa |

The FuPa kinds need FuPa's written permission, see [FuPa](#fupa).

## Setup

### 1. Get the tool

Download `stadionzeitung-automation_<version>_windows_amd64.zip` from the
[latest release](https://github.com/sphrilix/stadionzeitung-automation/releases/latest)
and unpack it, e.g. into `C:\stadionzeitung-automation\`, on the PC that has
the InDesign document's folder (e.g. synced via OneDrive). The archive holds
`stadionzeitung-automation.exe`, this README, the license and
`stadionzeitung.example.yaml`; `checksums.txt` lists the SHA-256 sums.

The binary is not signed, so Windows SmartScreen may warn on the first
start (*More info → Run anyway*). For Linux and macOS, see
[Releasing](#releasing) or build from source.

### 2. Write the config

Copy `stadionzeitung.example.yaml` to `stadionzeitung.yaml` next to the
`.exe`. The example is FC Sturm Hauzenberg's complete setup: every PDF of
the programme, each commented with the page it fills. Adjust `outDir`, and
for another club the URLs.

```yaml
outDir: C:/Users/<user>/OneDrive/Stadionzeitung/Dateien
fonts:
  regular: C:/Windows/Fonts/calibri.ttf
  bold: C:/Windows/Fonts/calibrib.ttf

teams:
  - file: Tabelle-H1.pdf              # Tabelle (S. 13)
    kind: tabelle
    league: https://www.bfv.de/ergebnisse/wettbewerb/-/0307PL23RO000007VS5489BUVT69HBT7-G
    highlight: Hauzenberg
```

The config is YAML: one `- file: …` block per PDF, `#` starts a comment, and
indentation matters (spaces, not tabs). A misspelt key is reported as an
error. JSON configs work too.

- `outDir`: the folder the InDesign document links its graphics from. It must
  exist. Write the path as it is (`C:\Users\…` or `C:/Users/…`); only inside
  "double quotes" must backslashes be doubled. Relative paths are resolved
  against the config file.
- `fonts`: TrueType files for regular and bold text, usually the layout's
  font. Set both or neither; without them the built-in Go font is used.
- `teams`: the PDFs, see [Config entries](#config-entries).

### 3. Place the PDFs in InDesign (once)

1. Generate the PDFs once (see [Every issue](#every-issue)), so there is
   something to place.
2. Draw a graphic frame, then *File → Place* (*Datei → Platzieren*), choose
   e.g. `Tabelle-H1.pdf` and tick *Show Import Options*
   (*Importoptionen anzeigen*). Set *Crop to: Media*
   (*Zuschneiden auf: Medien*), since each page is already cropped to its
   content, and enable *Transparent Background*
   (*Transparenter Hintergrund*).
3. With the frame selected, *Object → Fitting → Frame Fitting Options*
   (*Objekt → Anpassen → Rahmeneinpassungsoptionen*): enable *Auto-Fit*
   (*Automatisch einpassen*), set *Fitting* to *Fit Content Proportionally*
   (*Inhalt proportional anpassen*) and the reference point to top left.
   The content then refits when, say, a matchday has more matches.

### 4. Optional: a double-click shortcut

Save this as `Stadionzeitung aktualisieren.bat` next to the `.exe`:

```bat
@echo off
cd /d "%~dp0"
stadionzeitung-automation.exe -config stadionzeitung.yaml
pause
```

## Every issue

1. Run the tool on the weekend before the home game: double-click the
   `.bat`, or in a terminal in the tool's folder

   ```sh
   stadionzeitung-automation.exe -config stadionzeitung.yaml
   ```

   It prints each PDF it writes. Each file is replaced in one step, so
   InDesign and OneDrive never see a half-written PDF. If an entry fails
   (e.g. the referee is not known yet), its previous PDF is kept, the other
   PDFs are still written, and all errors are listed at the end.
2. Open the InDesign document and choose *Update Links*
   (*Verknüpfungen aktualisieren*) when it asks about modified links. If it
   is already open: *Window → Links* (*Fenster → Verknüpfungen*), select the
   PDFs with the yellow warning sign, *Update Link*
   (*Verknüpfung aktualisieren*).
3. Check the pages, fill in what stays manual (Grußwort, assistant referees,
   Macher des Spiels, …), then export or print as usual.

## Config entries

Every entry has a `file` (the PDF's name in `outDir`) and a `kind`
(default `spieltage`). What else it needs depends on the kind:

| Kind | Needs | Options |
| --- | --- | --- |
| `spieltage`, `tabelle` | `league` | `highlight` (tabelle) |
| `spielplan` | `team` | |
| `torjaeger` | `team` or `fupa` | `top`, `opponent` |
| `jugend-ergebnisse`, `jugend-vorschau` | `club` | `days` |
| `kader` | `fupa` | `opponent`, `allPlayers`, `roles` |
| `liga-torjaeger` | `fupa` | `top` |
| `schiedsrichter` | `fupa` | |

Where to find the addresses on bfv.de:

- `league`: the league's page under *Ergebnisse*, e.g.
  `https://www.bfv.de/ergebnisse/wettbewerb/-/0307PL23RO000007VS5489BUVT69HBT7-G`
  (it ends in `…-G`).
- `team`: the team's page, linked from the club page under *Mannschaften*
  or from a team name on the league page, e.g.
  `https://www.bfv.de/mannschaften/fc-sturm-hauzenberg/016PE85GAS000000VV0AG811VTE5EA5R`.
- `club`: the club's page, e.g.
  `https://www.bfv.de/vereine/fc-sturm-hauzenberg/00ES8GNI7C00000KVV0AG08LVUPGND5I`.

Options:

- `highlight`: table rows of teams whose name contains it are set in bold,
  e.g. `Hauzenberg` for the club's own teams.
- `top`: how many scorers (default 3); anyone tied with the last is listed
  too, as the programme does.
- `opponent: true`: instead of the team itself, use its next league
  opponent (the first league match today or later). Generated before a
  home game, that is the guest.
- `days`: how far back (`jugend-ergebnisse`) or ahead (`jugend-vorschau`)
  to list matches (default 7).
- `allPlayers`, `roles`: see [FuPa](#fupa).

Notes:

- The current matchday is the one running today (through its last
  scheduled day) or, between matchdays, the next one.
- `torjaeger` from BFV needs the club to publish its squad on bfv.de; many
  don't. With `fupa` instead of `team` it reads FuPa, which also covers
  those clubs.
- The `jugend` kinds find the club's teams through the links on its bfv.de
  page; a team not linked there is missed.
- Team names are spelt as BFV has them, e.g. "(SG) Haag 1/Hauzenberg 3".

## FuPa

BFV does not publish positions, shirt numbers, staff, a league's top
scorers or referees, and some clubs keep their squad private there.
[FuPa](https://www.fupa.net) has all of that.

> **Permission required.** FuPa's
> [terms of use](https://www.fupa.net/about/terms-of-use) (§4.1.4) forbid
> downloading or reusing its data without FuPa's prior written consent.
> Only configure FuPa entries once you have it. The tool contacts FuPa only
> for entries with a `fupa` team and identifies itself in its requests. In
> the example config, all FuPa entries are in one block at the end.

`fupa` is the team's FuPa page, e.g.
`https://www.fupa.net/team/fc-sturm-hauzenberg-m1`, or just
`fc-sturm-hauzenberg-m1`. Without a season suffix (`-2026-27`) it always
means the current season. Some teams, such as joint teams (SG), only work
with the suffix; update it each summer.

- `kader` lists the staff by role, then the players by position (Tor,
  Abwehr, Mittelfeld, Angriff), sorted by last name, the captain marked
  "(K)". FuPa squads often include players of the club's other teams
  without a shirt number, so only players with a number are listed, unless
  `allPlayers: true` (useful for clubs that number nobody or only part of
  their squad). `roles` limits the staff to these roles as FuPa names them,
  e.g. `[Trainer, Torwart-Trainer, Betreuer]`; without it all are listed.
- `torjaeger` with `fupa` reads the goals from FuPa.
- `liga-torjaeger` lists the top scorers of the team's league, e.g.
  "Almir Mesanovic (SV Türk Gücü Straubing, 16 Tore)".
- `schiedsrichter` gives the referee and their club, e.g. "Marijo Kraljic
  (SC Wörnsmühl)". FuPa lists the referee only, not the assistants, and
  only once known, often shortly before the match; until then the entry
  fails with a message and the previous PDF is kept.

## Single PDFs from the command line

For trying things out, the tool also writes one PDF without a config:

```sh
stadionzeitung-automation -kind tabelle -o tabelle.pdf \
  https://www.bfv.de/ergebnisse/wettbewerb/-/0307PL23RO000007VS5489BUVT69HBT7-G
```

The argument is what the kind needs (see [Config entries](#config-entries)):
a league, team or club URL, a FuPa team, or the bare id. Flags:

- `-kind` what to write (default `spieltage`)
- `-o` output file (default named after the content, e.g.
  `Landesliga-Mitte-spieltag-16.pdf`)
- `-matchday N` for `spieltage`: centre on matchday N instead of the
  current one
- `-days N` for the `jugend` kinds (default 7)
- `-version` print the version

## How it works

- **Matchdays and fixtures:** the bfv.de league pages give the structure
  (matchday numbers, match IDs), but obfuscate names and results, so the
  readable fixtures come from the BFV widget API and are joined by match ID.
- **Tables:** from the widget API. It only carries the goal difference, so
  goals for and against are summed from all results; where they do not add
  up (e.g. a match awarded by the association) the cell stays empty. Logos
  come from BFV's media service.
- **Season schedules:** the team's fixtures from the widget API, numbered by
  the league's matchday pages, so a postponed match keeps its matchday.
- **Scorers:** BFV's squad statistics, or FuPa's.
- **Youth:** the club's teams from its bfv.de page, their matches from the
  widget API.
- **Squads, league scorers, referees:** the FuPa API.

## Development

Bazel (via bazelisk) downloads everything else, including the Go SDK.

```sh
bazel run //cmd/stadionzeitung-automation -- -config stadionzeitung.yaml
bazel test //...
bazel run @rules_go//go -- mod tidy   # after changing imports
bazel run //:gazelle                  # regenerate BUILD files

# cross-compile, e.g. for Windows
bazel build --platforms=@rules_go//go/toolchain:windows_amd64 //cmd/stadionzeitung-automation
# → bazel-bin/cmd/stadionzeitung-automation/stadionzeitung-automation_/stadionzeitung-automation.exe
```

CI runs on every push to `main` and every pull request. It runs the tests
(including one that loads `stadionzeitung.example.yaml`), fails if
`go.mod`/`go.sum` or the BUILD files are not regenerated, and checks that
the Windows build still compiles.

### Releasing

```sh
git tag v0.2.0
git push origin v0.2.0
```

The release workflow builds the optimised Windows binary with
`--config=release`, which stamps the tag into `-version`, and publishes it
with `checksums.txt` as a GitHub Release with generated notes.

To add other platforms to an existing release, run the workflow by hand
(*Actions → Release → Run workflow*, or
`gh workflow run release.yml -f tag=v0.2.0 -f platforms="linux_amd64 darwin_arm64"`).
Supported: `windows_amd64`, `linux_amd64`, `linux_arm64`, `darwin_amd64`,
`darwin_arm64`; they are added as `.tar.gz` archives and `checksums.txt` is
regenerated. On macOS, clear the quarantine flag of the unsigned binary
with `xattr -d com.apple.quarantine stadionzeitung-automation`. Locally,
`bazel build --config=release` stamps `git describe`; plain builds report
`dev`.

## License

[MIT](LICENSE)
