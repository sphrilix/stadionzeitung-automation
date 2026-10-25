# What in the Staffelberg-Kicker can be automated

An inventory of the matchday programme, page by page, based on the issue
for FC Sturm Hauzenberg – SC Luhe-Wildenau (17.10.2026, 40 pages). For
each block: where the data comes from and whether stadionzeitung-automation can generate
it. Sponsor pages (every even page) are left out.

Data sources used:

- **Widget API** (`widget-prod.bfv.de/api/service/widget/v1`): the JSON
  API behind BFV's embeddable widgets. Readable, but unofficial.
- **bfv.de pages**: league, club and match pages. Their structure
  (matchday numbers, match and team IDs) is readable, but names, dates and
  results are obfuscated with a per-request font, so the readable details
  always come from the widget API.
- **FuPa API** (`api.fupa.net/v1`): fills what BFV lacks. FuPa's terms of
  use require its **written permission** to reuse the data; the FuPa kinds
  are only for use once that is granted.

## Automated

| Page | Block | Kind | Source | PR |
| --- | --- | --- | --- | --- |
| 15, 31 | Aktuelle Spieltage (previous, current, next matchday) | `spieltage` | league pages + widget team matches | already in `main` |
| 13, 29, 37 | Tabelle with logos and zone colours | `tabelle` | widget `/competition/{id}/table`; goals summed from all results; logos from BFV's media service | #2 |
| 17, 33, 35 | Saisonspielplan of a team | `spielplan` | widget `/team/{id}/matches`; matchday numbers from the league pages | #3 |
| 11, 27 | Torjäger of the own teams | `torjaeger` | widget `/team/{id}/squad` (goals per player) | #4 |
| 5 | Torjäger of today's guest | `torjaeger` + `opponent` | next league match → opponent's squad | #4 |
| 21 | Spielergebnisse der Jugendmannschaften | `jugend-ergebnisse` | club page (team list) + widget team matches | #5 |
| 23 | Kommende Spiele der Jugendmannschaften | `jugend-vorschau` | as above | #5 |
| 5, 11, 27 | Kader: staff, players by position, captain | `kader` (+ `opponent` for the guest) | FuPa `/teams/{slug}/squad` | #7 |
| 5 | Torjäger of a guest with a private BFV squad | `torjaeger` + `fupa` | FuPa squad | #7 |
| 13, 29 | Top-Torjäger of the league | `liga-torjaeger` | FuPa `/competitions/{slug}/seasons/current/scorers` | #7 |
| 3, 5 | Referee | `schiedsrichter` | FuPa `/matches/{slug}` | #7 |

Limits:

- **Squads can be private on BFV.** `torjaeger` from BFV needs the club to
  publish its squad on bfv.de; in the Landesliga 8 of 18 clubs don't (e.g.
  SV Türk Gücü Straubing). With a `fupa` team it reads FuPa instead.
- **Referees:** FuPa lists the referee only, not the assistants, and only
  once known, often shortly before the match.
- **Teams not linked from the club page** are missed by the `jugend`
  kinds; in the sample issue that was one F-Junioren tournament game.
- **Team names** are BFV's spelling, e.g. "(SG) Haag 1/Hauzenberg 3"
  rather than "SG Haag I/Hauzenberg III".

## Could be automated next

Data that is available but not generated yet:

| Page | Block | Idea | Source |
| --- | --- | --- | --- |
| 1 | Cover: league, matchday, opponent, date and kickoff | small text PDF (or text file for InDesign data merge) for the next home match | widget team matches + matchday pages |
| 3 | Grußwort facts: "zum 8. Heimspiel", last result, table position and gap of each team | a fact sheet for whoever writes the Grußwort, not the text itself | team matches + table |
| 8, 9 | Kader photo pages with shirt numbers | photos and numbers exist on FuPa and BFV (with photographer credit) | FuPa / widget squad |

## Not automatable

| Page | Block | Why |
| --- | --- | --- |
| 3, 5 | Assistant referees | neither BFV nor FuPa publishes them |
| 3 | Grußwort text | written by the board |
| 19 | Macher des Spiels | editorial choice |
| 24, 25, 39 | Sturm-Freunde, sponsor thanks, Impressum | club content |
