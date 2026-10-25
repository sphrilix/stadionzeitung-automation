# Stadionzeitung Automation

[![CI](https://github.com/sphrilix/stadionzeitung-automation/actions/workflows/ci.yml/badge.svg)](https://github.com/sphrilix/stadionzeitung-automation/actions/workflows/ci.yml)

Writes PDFs with data of any BFV league, ready to be placed into a layout:
without a heading (the layout provides that) and with the page cropped to the
content. By default it writes the previous, current and next matchday
("Spieltag") including results, as boxed tables. The current matchday is the
one running today (through its last scheduled day) or, between matchdays, the
next one.

```sh
stadionzeitung-automation -o ll.pdf \
  https://www.bfv.de/ergebnisse/wettbewerb/-/0307PL23RO000007VS5489BUVT69HBT7-G
```

The argument is a bfv.de league URL containing the competition id (`…-G`) or
the bare id; for `-kind spielplan` and `-kind torjaeger` it is a team URL
(`https://www.bfv.de/mannschaften/<name>/<id>`), for the `jugend` kinds a
club URL (`https://www.bfv.de/vereine/<name>/<id>`), for the FuPa kinds a
fupa.net team URL or slug, or the bare id. Flags:

- `-o` output file (default named after the league and kind, e.g.
  `Landesliga-Mitte-spieltag-16.pdf`)
- `-kind` what to write, see below (default `spieltage`)
- `-matchday N` centre the window on matchday N instead of the current one
- `-days N` for the `jugend` kinds, how many days back or ahead (default 7)
- `-version` print the version

| Kind | Content |
| --- | --- |
| `spieltage` | previous, current and next matchday with results |
| `tabelle` | current table with club logos, goals, and promotion/relegation zones coloured |
| `spielplan` | a team's season fixtures with matchday, kickoff and result, byes included; takes a team instead of a league |
| `torjaeger` | a team's top scorers, e.g. "J. Wagner (10 Tore)"; takes a team, needs its squad to be public on bfv.de |
| `jugend-ergebnisse` | results of all the club's youth teams over the last days, grouped by age group and competition; takes a club |
| `jugend-vorschau` | the same for the coming days, without results |
| `kader` | a team's squad by position, captain marked, with coaches and staff; from [FuPa](#fupa) |
| `liga-torjaeger` | the league's top scorers with their teams; from [FuPa](#fupa) |
| `schiedsrichter` | the referee of the team's next league match; from [FuPa](#fupa) |

> **Unofficial.** stadionzeitung-automation is not affiliated with or endorsed
> by the Bayerischer Fußball-Verband (BFV). It reads public bfv.de pages and
> the widget API behind BFV's embeddable widgets, both of which can change
> without notice and break the tool. Match data belongs to the BFV; check its
> terms before publishing generated tables beyond your own club's use.

## Installation

Download `stadionzeitung-automation_<version>_windows_amd64.zip` from the
[latest release](https://github.com/sphrilix/stadionzeitung-automation/releases/latest)
and unpack it. The archive holds `stadionzeitung-automation.exe`, this
README, the license and `stadionzeitung.example.yaml`; `checksums.txt` lists
the SHA-256 sums.

Releases contain only the Windows build by default. Linux and macOS archives
(`_linux_amd64`, `_linux_arm64`, `_darwin_amd64`, `_darwin_arm64`, as
`.tar.gz`) are added on request, see [Releasing](#releasing), or build from
source.

The binaries are not signed. On Windows, SmartScreen may warn on the first
start (*More info → Run anyway*); on macOS, clear the quarantine flag with
`xattr -d com.apple.quarantine stadionzeitung-automation`.

## Stadionzeitung (InDesign)

Made for a club's matchday programme ("Stadionzeitung"): an InDesign document
places one PDF per team as a linked graphic, and `-config` regenerates all of
them in one go, so updating the fixtures is: run the tool, update the links.

### One-time setup

1. **Get the tool.** Unpack
   `stadionzeitung-automation_<version>_windows_amd64.zip` from the
   [latest release](https://github.com/sphrilix/stadionzeitung-automation/releases/latest),
   e.g. into `C:\stadionzeitung-automation\`, on the PC that has the InDesign
   document's folder (e.g. synced via OneDrive).
2. **Write the config.** Copy `stadionzeitung.example.yaml` to
   `stadionzeitung.yaml` next to `stadionzeitung-automation.exe` and adjust it.
   The example is the club's complete setup, with every PDF of the programme
   and comments naming its page. The config is YAML: one `- file: …` block
   per PDF, `#` starts a comment, and indentation matters (use spaces, not
   tabs). Misspelt keys are reported as errors. JSON configs still work.
   - `outDir`: the folder the InDesign document links its graphics from. It
     must already exist. Write the path as it is (`C:\Users\…` or
     `C:/Users/…`); only inside "double quotes" must backslashes be doubled.
     Relative paths are resolved against the config file.
   - `fonts`: TrueType files for regular and bold text, typically the font of
     the layout (the example uses Calibri). Set both or neither; without them
     the Go font is used.
   - `teams`: one entry per PDF. `file` is the name inside `outDir`, `league`
     the team's league page on bfv.de (open the league under *Ergebnisse*
     and copy the address; it ends in `…-G`). `kind` picks the content (see
     the table above; default `spieltage`). For a `tabelle`, `highlight`
     sets the rows of teams whose name contains it in bold, e.g.
     `Hauzenberg` for the club's own teams. A `spielplan` takes `team`
     instead of `league`: the team's page on bfv.de (on the club page under
     *Mannschaften*, or a team name on the league page). So does
     `torjaeger`, which lists the `top` scorers (default 3, plus anyone tied
     with the last); with `opponent: true` it lists those of the team's
     next league opponent instead, for the guest page. Clubs that keep their
     squad private on bfv.de have no scorers to list; that file then fails
     with a message and the previous one is kept. The `jugend` kinds take
     `club`, the club's page on bfv.de, and list the matches of every team
     linked there, over the last (`jugend-ergebnisse`) or next
     (`jugend-vorschau`) `days` days (default 7). The FuPa kinds take
     `fupa`, see [FuPa](#fupa).
3. **Generate the PDFs once** (see below), so there is something to place.
4. **Place each PDF in InDesign.** Draw a graphic frame where the table goes,
   then *File → Place* (*Datei → Platzieren*), choose e.g.
   `Spielplan-H1.pdf` and tick *Show Import Options*
   (*Importoptionen anzeigen*). In the dialog, set *Crop to: Media*
   (*Zuschneiden auf: Medien*), since each page is already cropped to the
   table, and enable *Transparent Background*
   (*Transparenter Hintergrund*).
5. **Let the frame follow the table.** With the frame selected,
   *Object → Fitting → Frame Fitting Options*
   (*Objekt → Anpassen → Rahmeneinpassungsoptionen*): enable *Auto-Fit*
   (*Automatisch einpassen*), set *Fitting* to *Fit Content Proportionally*
   (*Inhalt proportional anpassen*) and the reference point to top left. The
   table then refits when a matchday has more or fewer matches.
6. **Optional: a double-click shortcut.** Save this as
   `Spielplan aktualisieren.bat` next to `stadionzeitung-automation.exe`:

   ```bat
   @echo off
   cd /d "%~dp0"
   stadionzeitung-automation.exe -config stadionzeitung.yaml
   pause
   ```

### For every issue

1. Run the tool: double-click the `.bat` file, or in a terminal in the
   tool's folder

   ```sh
   stadionzeitung-automation.exe -config stadionzeitung.yaml
   ```

   It writes every PDF listed in the config with the previous, current and
   next matchday. A file is replaced in one step, so InDesign or OneDrive
   never see a half-written PDF; if a league fails, its old PDF is kept and
   the error is printed.
2. Open the InDesign document. When it asks about modified links, choose
   *Update Links* (*Verknüpfungen aktualisieren*). If it is already open:
   *Window → Links* (*Fenster → Verknüpfungen*), select the generated PDFs
   (e.g. `Spielplan-H*.pdf`, `Tabelle-H*.pdf`, `Jugend-*.pdf`) with the
   yellow warning sign and click *Update Link*
   (*Verknüpfung aktualisieren*).
3. Check the tables, then export or print as usual.

## FuPa

BFV does not publish positions, shirt numbers, staff, the league's top
scorers or referees, and some clubs keep their squad private there.
[FuPa](https://www.fupa.net) has all of that.

> **Permission required.** FuPa's
> [terms of use](https://www.fupa.net/about/terms-of-use) (§4.1.4) forbid
> downloading or reusing its data without FuPa's prior written consent.
> Only configure FuPa once you have it. stadionzeitung-automation contacts
> FuPa only for entries with a `fupa` team, and identifies itself in its requests.

`fupa` is the team's FuPa page, e.g. `https://www.fupa.net/team/fc-sturm-hauzenberg-m1`,
or just its slug `fc-sturm-hauzenberg-m1`. Without the season suffix
(`-2026-27`) it always means the current season.

```yaml
  - file: Kader-H1.pdf
    kind: kader
    fupa: fc-sturm-hauzenberg-m1
    roles: [Trainer, Torwart-Trainer, Fitnesstrainer, Betreuer]

  - file: Kader-Gast.pdf
    kind: kader
    fupa: fc-sturm-hauzenberg-m1
    opponent: true
    allPlayers: true

  - file: Torjaeger-Gast.pdf
    kind: torjaeger
    fupa: fc-sturm-hauzenberg-m1
    opponent: true

  - file: Liga-Torjaeger-H1.pdf
    kind: liga-torjaeger
    fupa: fc-sturm-hauzenberg-m1

  - file: Schiedsrichter.pdf
    kind: schiedsrichter
    fupa: fc-sturm-hauzenberg-m1
```

- `kader` lists staff by role, then the players by position (Tor, Abwehr,
  Mittelfeld, Angriff), sorted by last name, the captain marked "(K)".
  FuPa squads often include players of the club's other teams without a
  shirt number, so only players with a number are listed, unless
  `allPlayers` is set (useful when a club numbers nobody or only part of
  its squad). `roles` limits the staff to these roles, as FuPa names
  them; without it all are listed. `opponent: true` shows the next
  league opponent's squad, for the guest page.
- `torjaeger` with `fupa` reads the goals from FuPa instead of BFV, so it
  also works for clubs whose BFV squad is private.
- `liga-torjaeger` lists the `top` scorers (default 3, plus ties) of the
  team's league, e.g. "Almir Mesanovic (SV Türk Gücü Straubing, 16 Tore)".
- `schiedsrichter` gives the referee and their club, e.g. "Marijo Kraljic
  (SC Wörnsmühl)". FuPa lists the referee only, not the assistants, and
  only once it is known, often shortly before the match; until then the
  entry fails with a message and the previous PDF is kept.

## How it works

Matchday structure comes from the bfv.de league pages, whose team names and
scores are obfuscated; the readable fixtures come from the BFV widget API and
are joined by match id.

## Development

Bazel (via bazelisk) downloads everything else, including the Go SDK.

```sh
bazel run //cmd/stadionzeitung-automation -- <league URL>   # run from source
bazel test //...
bazel run @rules_go//go -- mod tidy   # after changing imports
bazel run //:gazelle                  # regenerate BUILD files

# cross-compile, e.g. for Windows
bazel build --platforms=@rules_go//go/toolchain:windows_amd64 //cmd/stadionzeitung-automation
# → bazel-bin/cmd/stadionzeitung-automation/stadionzeitung-automation_/stadionzeitung-automation.exe
```

CI runs on every push to `main` and every pull request. It runs the tests,
fails if `go.mod`/`go.sum` or the BUILD files are not regenerated, and checks
that the Windows build still compiles.

### Releasing

```sh
git tag v0.1.0
git push origin v0.1.0
```

The release workflow builds the optimised Windows binary with
`--config=release`, which stamps the tag into `-version`, and publishes it
with `checksums.txt` as a GitHub Release with generated notes.

To add other platforms to an existing release, run the workflow by hand
(*Actions → Release → Run workflow*, or
`gh workflow run release.yml -f tag=v0.1.0 -f platforms="linux_amd64 darwin_arm64"`).
Supported: `windows_amd64`, `linux_amd64`, `linux_arm64`, `darwin_amd64`,
`darwin_arm64`. The archives are added to the release and `checksums.txt` is
regenerated for all of them. Locally, `bazel build --config=release` stamps
`git describe`; plain builds report `dev`.

## License

[MIT](LICENSE)
