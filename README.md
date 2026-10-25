# bfv-miner

[![CI](https://github.com/sphrilix/stadionzeitung-automation/actions/workflows/ci.yml/badge.svg)](https://github.com/sphrilix/stadionzeitung-automation/actions/workflows/ci.yml)

Writes PDFs with data of any BFV league, ready to be placed into a layout:
without a heading (the layout provides that) and with the page cropped to the
content. By default it writes the previous, current and next matchday
("Spieltag") including results, as boxed tables. The current matchday is the
one running today (through its last scheduled day) or, between matchdays, the
next one.

```sh
bfv-miner -o ll.pdf \
  https://www.bfv.de/ergebnisse/wettbewerb/-/0307PL23RO000007VS5489BUVT69HBT7-G
```

The argument is a bfv.de league URL containing the competition id (`…-G`) or
the bare id. Flags:

- `-o` output file (default named after the league and kind, e.g.
  `Landesliga-Mitte-spieltag-16.pdf`)
- `-kind` what to write, see below (default `spieltage`)
- `-matchday N` centre the window on matchday N instead of the current one
- `-version` print the version

| Kind | Content |
| --- | --- |
| `spieltage` | previous, current and next matchday with results |
| `tabelle` | current table with club logos, goals, and promotion/relegation zones coloured |

> **Unofficial.** bfv-miner is not affiliated with or endorsed by the
> Bayerischer Fußball-Verband (BFV). It reads public bfv.de pages and the
> widget API behind BFV's embeddable widgets, both of which can change
> without notice and break the tool. Match data belongs to the BFV; check
> its terms before publishing generated tables beyond your own club's use.

## Installation

Download `bfv-miner_<version>_windows_amd64.zip` from the
[latest release](https://github.com/sphrilix/stadionzeitung-automation/releases/latest) and
unpack it. The archive holds `bfv-miner.exe`, this README, the license and
`stadionzeitung.example.json`; `checksums.txt` lists the SHA-256 sums.

Releases contain only the Windows build by default. Linux and macOS archives
(`_linux_amd64`, `_linux_arm64`, `_darwin_amd64`, `_darwin_arm64`, as
`.tar.gz`) are added on request, see [Releasing](#releasing), or build from
source.

The binaries are not signed. On Windows, SmartScreen may warn on the first
start (*More info → Run anyway*); on macOS, clear the quarantine flag with
`xattr -d com.apple.quarantine bfv-miner`.

## Stadionzeitung (InDesign)

Made for a club's matchday programme ("Stadionzeitung"): an InDesign document
places one PDF per team as a linked graphic, and `-config` regenerates all of
them in one go, so updating the fixtures is: run the tool, update the links.

### One-time setup

1. **Get the tool.** Unpack `bfv-miner_<version>_windows_amd64.zip` from the
   [latest release](https://github.com/sphrilix/stadionzeitung-automation/releases/latest),
   e.g. into `C:\bfv-miner\`, on the PC that has the InDesign document's
   folder (e.g. synced via OneDrive).
2. **Write the config.** Copy `stadionzeitung.example.json` to
   `stadionzeitung.json` next to `bfv-miner.exe` and adjust it:
   - `outDir`: the folder the InDesign document links its graphics from. It
     must already exist. Use forward slashes (`C:/Users/…`) or doubled
     backslashes (`C:\\Users\\…`); relative paths are resolved against the
     config file.
   - `fonts`: TrueType files for regular and bold text, typically the font of
     the layout (the example uses Calibri). Set both or neither; without them
     the Go font is used.
   - `teams`: one entry per PDF. `file` is the name inside `outDir`, `league`
     the team's league page on bfv.de (open the league under *Ergebnisse*
     and copy the address; it ends in `…-G`). `kind` picks the content (see
     the table above; default `spieltage`). For a `tabelle`, `highlight`
     sets the rows of teams whose name contains it in bold, e.g.
     `"Hauzenberg"` for the club's own teams.
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
   `Spielplan aktualisieren.bat` next to `bfv-miner.exe`:

   ```bat
   @echo off
   cd /d "%~dp0"
   bfv-miner.exe -config stadionzeitung.json
   pause
   ```

### For every issue

1. Run the tool: double-click the `.bat` file, or in a terminal in the
   tool's folder

   ```sh
   bfv-miner.exe -config stadionzeitung.json
   ```

   It writes every PDF listed in the config with the previous, current and
   next matchday. A file is replaced in one step, so InDesign or OneDrive
   never see a half-written PDF; if a league fails, its old PDF is kept and
   the error is printed.
2. Open the InDesign document. When it asks about modified links, choose
   *Update Links* (*Verknüpfungen aktualisieren*). If it is already open:
   *Window → Links* (*Fenster → Verknüpfungen*), select the generated PDFs
   (e.g. `Spielplan-H*.pdf`, `Tabelle-H*.pdf`) with the yellow warning sign
   and click
   *Update Link* (*Verknüpfung aktualisieren*).
3. Check the tables, then export or print as usual.

## How it works

Matchday structure comes from the bfv.de league pages, whose team names and
scores are obfuscated; the readable fixtures come from the BFV widget API and
are joined by match id.

## Development

Bazel (via bazelisk) downloads everything else, including the Go SDK.

```sh
bazel run //cmd/bfv-miner -- <league URL>   # run from source
bazel test //...
bazel run @rules_go//go -- mod tidy   # after changing imports
bazel run //:gazelle                  # regenerate BUILD files

# cross-compile, e.g. for Windows
bazel build --platforms=@rules_go//go/toolchain:windows_amd64 //cmd/bfv-miner
# → bazel-bin/cmd/bfv-miner/bfv-miner_/bfv-miner.exe
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
