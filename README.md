# bfv-miner

Writes a PDF with the previous, current and next matchday ("Spieltag") of any
BFV league, including results.

```sh
bazel run //cmd/bfv-miner -- -title "Landesliga Mitte" -o ll.pdf \
  https://www.bfv.de/ergebnisse/wettbewerb/-/0307PL23RO000007VS5489BUVT69HBT7-G
```

The argument is a bfv.de league URL containing the competition id (`…-G`) or
the bare id. Flags:

- `-o` output file (default `<league>-spieltag-<N>.pdf`)
- `-matchday N` centre the window on matchday N instead of BFV's current one
- `-title` heading name (BFV's own names are short, e.g. `KL-Gr2 SW`)

Matchday structure comes from the bfv.de league pages, whose team names and
scores are obfuscated; the readable fixtures come from the BFV widget API and
are joined by match id.

## Development

```sh
bazel test //...
bazel run @rules_go//go -- mod tidy   # after changing imports
bazel run //:gazelle                  # regenerate BUILD files
```
