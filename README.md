# bfv-miner

Writes a PDF with the previous, current and next matchday ("Spieltag") of any
BFV league, including results. The current matchday is the one running today
(through its last scheduled day) or, between matchdays, the next one.

```sh
bazel run //cmd/bfv-miner -- -title "Landesliga Mitte" -o ll.pdf \
  https://www.bfv.de/ergebnisse/wettbewerb/-/0307PL23RO000007VS5489BUVT69HBT7-G
```

The argument is a bfv.de league URL containing the competition id (`…-G`) or
the bare id. Flags:

- `-o` output file (default `<league>-spieltag-<N>.pdf`)
- `-matchday N` centre the window on matchday N instead of the current one
- `-title` heading name (BFV's own names are short, e.g. `KL-Gr2 SW`)

## Stadionzeitung (InDesign)

The Stadionzeitung places `SZ/Dateien/Spielplan-H1.pdf`, `-H2.pdf` and
`-H3.pdf` as linked graphics. `-config` regenerates all of them in one go:

```sh
bfv-miner -config stadionzeitung.json
```

See `stadionzeitung.example.json`: `outDir` is the linked folder, `fonts`
points at Calibri (any TrueType font works; without it the Go font is used),
and each team maps a file name to its bfv.de league. Relative paths are
resolved against the config file. Each PDF's page is cropped to its content
and replaced atomically, so InDesign never links a half-written file.

In InDesign:

1. Links panel → select the modified `Spielplan-H*.pdf` → *Update Link*.
2. Once per frame: *Object → Fitting → Frame Fitting Options*, enable
   *Auto-Fit* with *Fit Content Proportionally*, so the block refits when a
   matchday has more or fewer matches.

To run it on the Windows PC that holds the OneDrive folder:

```sh
bazel build --platforms=@rules_go//go/toolchain:windows_amd64 //cmd/bfv-miner
# → bazel-bin/cmd/bfv-miner/bfv-miner_/bfv-miner.exe
```

## How it works

Matchday structure comes from the bfv.de league pages, whose team names and
scores are obfuscated; the readable fixtures come from the BFV widget API and
are joined by match id.

## Development

```sh
bazel test //...
bazel run @rules_go//go -- mod tidy   # after changing imports
bazel run //:gazelle                  # regenerate BUILD files
```
