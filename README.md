# bfv-miner

[![CI](https://github.com/sphrilix/bfv-miner/actions/workflows/ci.yml/badge.svg)](https://github.com/sphrilix/bfv-miner/actions/workflows/ci.yml)

Writes a PDF with the previous, current and next matchday ("Spieltag") of any
BFV league, including results. The current matchday is the one running today
(through its last scheduled day) or, between matchdays, the next one.

```sh
bfv-miner -title "Landesliga Mitte" -o ll.pdf \
  https://www.bfv.de/ergebnisse/wettbewerb/-/0307PL23RO000007VS5489BUVT69HBT7-G
```

The argument is a bfv.de league URL containing the competition id (`…-G`) or
the bare id. Flags:

- `-o` output file (default `<league>-spieltag-<N>.pdf`)
- `-matchday N` centre the window on matchday N instead of the current one
- `-title` heading name (BFV's own names are short, e.g. `KL-Gr2 SW`)
- `-version` print the version

## Installation

Download the archive for your system from the
[latest release](https://github.com/sphrilix/bfv-miner/releases/latest) and
unpack it. Each archive holds the binary, this README and
`stadionzeitung.example.json`; `checksums.txt` lists their SHA-256 sums.

| System | Archive |
| --- | --- |
| Windows | `bfv-miner_<version>_windows_amd64.zip` |
| macOS (Apple silicon / Intel) | `bfv-miner_<version>_darwin_arm64.tar.gz` / `_darwin_amd64.tar.gz` |
| Linux (x86-64 / ARM64) | `bfv-miner_<version>_linux_amd64.tar.gz` / `_linux_arm64.tar.gz` |

The binaries are not signed. On Windows, SmartScreen may warn on the first
start (*More info → Run anyway*); on macOS, clear the quarantine flag with
`xattr -d com.apple.quarantine bfv-miner`.

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

On the Windows PC that holds the OneDrive folder, unpack
`bfv-miner_<version>_windows_amd64.zip` next to the config and run
`bfv-miner.exe -config stadionzeitung.json`.

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

The release workflow builds optimised binaries for Linux, macOS and Windows
(amd64 and arm64; Windows amd64 only) with `--config=release`, which stamps
the tag into `-version`, and publishes them with `checksums.txt` as a GitHub
Release with generated notes. Locally, `bazel build --config=release` stamps
`git describe`; plain builds report `dev`.
