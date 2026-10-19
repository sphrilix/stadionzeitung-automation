# bfv-miner

[![CI](https://github.com/sphrilix/bfv-miner/actions/workflows/ci.yml/badge.svg)](https://github.com/sphrilix/bfv-miner/actions/workflows/ci.yml)

Writes a PDF with the previous, current and next matchday ("Spieltag") of any
BFV league, including results, as boxed tables without a heading (the layout
it is placed into provides that). The current matchday is the one running today
(through its last scheduled day) or, between matchdays, the next one.

```sh
bfv-miner -o ll.pdf \
  https://www.bfv.de/ergebnisse/wettbewerb/-/0307PL23RO000007VS5489BUVT69HBT7-G
```

The argument is a bfv.de league URL containing the competition id (`…-G`) or
the bare id. Flags:

- `-o` output file (default `<league>-spieltag-<N>.pdf`)
- `-matchday N` centre the window on matchday N instead of the current one
- `-version` print the version

## Installation

Download `bfv-miner_<version>_windows_amd64.zip` from the
[latest release](https://github.com/sphrilix/bfv-miner/releases/latest) and
unpack it. The archive holds `bfv-miner.exe`, this README and
`stadionzeitung.example.json`; `checksums.txt` lists the SHA-256 sums.

Releases contain only the Windows build by default. Linux and macOS archives
(`_linux_amd64`, `_linux_arm64`, `_darwin_amd64`, `_darwin_arm64`, as
`.tar.gz`) are added on request, see [Releasing](#releasing), or build from
source.

The binaries are not signed. On Windows, SmartScreen may warn on the first
start (*More info → Run anyway*); on macOS, clear the quarantine flag with
`xattr -d com.apple.quarantine bfv-miner`.

## Stadionzeitung (InDesign)

The Stadionzeitung places `SZ/Dateien/Spielplan-H1.pdf`, `-H2.pdf` and
`-H3.pdf` as linked graphics. `-config` regenerates all of them in one go:

```sh
bfv-miner -config stadionzeitung.json
```

See `stadionzeitung.example.json`: `outDir` is the linked folder (replace
`<user>` with your Windows user), `fonts` points at Calibri (any TrueType
font works; without it the Go font is used), and each team maps a file name
to its bfv.de league. Relative paths are
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
