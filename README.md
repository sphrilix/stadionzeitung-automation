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

Per team, the config lists which of these PDFs to write. The pages refer to the 40-page issue of 17.10.2026; see
[docs/automation.md](docs/automation.md) for the full page-by-page
inventory, including what stays manual.

| PDF | Content | Programme page | Source |
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
`.exe`. The example is FC Sturm Hauzenberg's complete setup. Adjust
`outDir`, and for another club the addresses.

Each team needs its bfv.de page, plus its FuPa page for the FuPa PDFs.
Everything else (league, table, opponent, team name) is looked up from
there. Then list the PDFs you want:

```yaml
outDir: C:/Users/<user>/OneDrive/Stadionzeitung/Dateien
fonts:
  regular: C:/Windows/Fonts/calibri.ttf
  bold: C:/Windows/Fonts/calibrib.ttf

club: https://www.bfv.de/vereine/fc-sturm-hauzenberg/00ES8GNI7C00000KVV0AG08LVUPGND5I
jugend: [ergebnisse, vorschau]

teams:
  H1:
    bfv: https://www.bfv.de/mannschaften/fc-sturm-hauzenberg/016PE85GAS000000VV0AG811VTE5EA5R
    fupa: https://www.fupa.net/team/fc-sturm-hauzenberg-m1
    pdfs: [spieltage, tabelle, spielplan, torjaeger, kader, liga-torjaeger, schiedsrichter]
    gast: [kader, torjaeger]
  H2:
    bfv: https://www.bfv.de/mannschaften/fc-sturm-hauzenberg-2/02T8FR5G5O000000VS5489BRVSH2JAIF
    pdfs: [spieltage, tabelle]
```

That writes `Spielplan-H1.pdf`, `Tabelle-H1.pdf`, …, `Kader-H1-Gast.pdf`,
`Spielplan-H2.pdf`, `Tabelle-H2.pdf`, `Jugend-Ergebnisse.pdf` and
`Jugend-Vorschau.pdf`; see [Config reference](#config-reference).

The config is YAML: `#` starts a comment, lists are written `[a, b]`, and
indentation matters (spaces, not tabs). A misspelt key is reported as an
error.

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

## Config reference

**Top level**

- `outDir`: the folder the InDesign document links its graphics from. It
  must exist. Write the path as it is (`C:\Users\…` or `C:/Users/…`); only
  inside "double quotes" must backslashes be doubled. Relative paths are
  resolved against the config file.
- `fonts`: `regular` and `bold` TrueType files, usually the layout's font.
  Set both or neither; without them the built-in Go font is used.
- `club`: the club's bfv.de page, e.g.
  `https://www.bfv.de/vereine/fc-sturm-hauzenberg/00ES8GNI7C00000KVV0AG08LVUPGND5I`.
  Only needed for `jugend`.
- `jugend`: `ergebnisse` (last days' results of all youth teams) and/or
  `vorschau` (coming days' matches), written as `Jugend-Ergebnisse.pdf` and
  `Jugend-Vorschau.pdf`.
- `days`: how many days back or ahead `jugend` covers (default 7).
- `teams`: one block per team. Its name (e.g. `H1`) becomes part of the file
  names, so use letters, digits, `-` and `_`.

**Per team**

- `bfv`: the team's bfv.de page, linked from the club page under
  *Mannschaften* or from a team name on the league page, e.g.
  `https://www.bfv.de/mannschaften/fc-sturm-hauzenberg/016PE85GAS000000VV0AG811VTE5EA5R`.
  Its league, and the matchdays, table and opponent, are found from there.
- `fupa`: the team's FuPa page, see [FuPa](#fupa).
- `pdfs`: which PDFs to write for the team:

  | PDF | File | Needs |
  | --- | --- | --- |
  | `spieltage` | `Spielplan-<Team>.pdf` | `bfv` |
  | `tabelle` | `Tabelle-<Team>.pdf` | `bfv` |
  | `spielplan` | `Saisonspielplan-<Team>.pdf` | `bfv` |
  | `torjaeger` | `Torjaeger-<Team>.pdf` | `bfv` or `fupa` |
  | `kader` | `Kader-<Team>.pdf` | `fupa` |
  | `liga-torjaeger` | `Liga-Torjaeger-<Team>.pdf` | `fupa` |
  | `schiedsrichter` | `Schiedsrichter-<Team>.pdf` | `fupa` |

  The matchdays are named `Spielplan-…`, the name the Stadionzeitung's
  layout has always linked them by; the season schedule is
  `Saisonspielplan-…`.
- `gast`: `kader` and/or `torjaeger` for the team's next league opponent
  (the first league match today or later), written as
  `Kader-<Team>-Gast.pdf` and `Torjaeger-<Team>-Gast.pdf`. Generated before
  a home game, that is the guest. The guest's Kader lists all its players
  and only its coaches.
- `top`: how many scorers `torjaeger` and `liga-torjaeger` list (default
  3); anyone tied with the last is listed too, as the programme does.
- `roles`, `allPlayers`: for the team's `kader`, see [FuPa](#fupa).

Notes:

- The current matchday is the one running today (through its last
  scheduled day) or, between matchdays, the next one.
- In the table, the team's own row is set in bold.
- `torjaeger` reads FuPa when the team has `fupa`, else BFV. BFV only has
  scorers for clubs that publish their squad on bfv.de; many don't.
- `jugend` finds the club's teams through the links on its bfv.de page; a
  team not linked there is missed.
- Team names are spelt as BFV has them, e.g. "(SG) Haag 1/Hauzenberg 3".

## FuPa

BFV does not publish positions, shirt numbers, staff, a league's top
scorers or referees, and some clubs keep their squad private there.
[FuPa](https://www.fupa.net) has all of that.

> **Permission required.** FuPa's
> [terms of use](https://www.fupa.net/about/terms-of-use) (§4.1.4) forbid
> downloading or reusing its data without FuPa's prior written consent.
> Only add `fupa` to a team once you have it. The tool contacts FuPa only
> for teams with `fupa` and identifies itself in its requests. Without
> `fupa`, leave `kader`, `liga-torjaeger` and `schiedsrichter` out of the
> team's `pdfs` and `kader` out of its `gast`.

`fupa` is the team's FuPa page, e.g.
`https://www.fupa.net/team/fc-sturm-hauzenberg-m1`, or just
`fc-sturm-hauzenberg-m1`. Without a season suffix (`-2026-27`) it always
means the current season. Some teams, such as joint teams (SG), only work
with the suffix; update it each summer.

- `kader` lists the staff by role, then the players by position (Tor,
  Abwehr, Mittelfeld, Angriff), sorted by last name, the captain marked
  "(K)". FuPa squads often include players of the club's other teams
  without a shirt number, so only players with a number are listed, unless
  the team sets `allPlayers: true`. `roles` limits the staff to these roles
  as FuPa names them, e.g. `[Trainer, Torwart-Trainer, Betreuer]`; without
  it all are listed. A guest's Kader always lists all players and only the
  coaches.
- `torjaeger` reads the goals from FuPa when the team has `fupa`.
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
  https://www.bfv.de/mannschaften/fc-sturm-hauzenberg/016PE85GAS000000VV0AG811VTE5EA5R
```

The argument is the bfv.de team page (for `spieltage` and `tabelle` a league
page works too), the club page for `jugend-ergebnisse` and
`jugend-vorschau`, or the FuPa team for `kader`, `liga-torjaeger` and
`schiedsrichter`. Flags:

- `-kind` the PDF to write, as in `pdfs` (default `spieltage`)
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
