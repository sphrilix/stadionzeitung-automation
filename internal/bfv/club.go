package bfv

import (
	"cmp"
	"context"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

// ClubMatch is a match of one of a club's teams in any competition.
type ClubMatch struct {
	Match
	// TeamType is the age group, e.g. "A-Junioren" or "Herren".
	TeamType    string
	Competition string
}

// IsYouth reports whether the match is played by a youth team.
func (m ClubMatch) IsYouth() bool { return strings.Contains(m.TeamType, "Junior") }

// The club page links each of its teams as /mannschaften/<slug>/<team id>.
var reClubTeam = regexp.MustCompile(`/mannschaften/[^/"\\]+/([0-9A-Z]{32})`)

// ClubTeams returns the IDs of the teams listed on a club's bfv.de page.
func (c *Client) ClubTeams(ctx context.Context, clubID string) ([]string, error) {
	body, err := c.get(ctx, c.BaseWeb+"/vereine/-/"+clubID)
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, m := range reClubTeam.FindAllSubmatch(body, -1) {
		if id := string(m[1]); !slices.Contains(ids, id) {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// FetchClubMatches returns the matches of all of a club's teams kicking off
// in [from, to), byes left out, ordered by age group (A-Junioren first) and
// kickoff.
func (c *Client) FetchClubMatches(ctx context.Context, clubID string, from, to time.Time) ([]ClubMatch, error) {
	teams, err := c.ClubTeams(ctx, clubID)
	if err != nil {
		return nil, err
	}
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(maxParallel)
	var (
		mu   sync.Mutex
		byID = map[string]ClubMatch{}
	)
	for _, team := range teams {
		g.Go(func() error {
			d, err := c.teamMatches(gctx, team)
			if err != nil {
				return err
			}
			mu.Lock()
			defer mu.Unlock()
			for _, wm := range d.Matches {
				m, err := wm.toMatch()
				if err != nil {
					return err
				}
				if m.IsBye() || m.Kickoff.Before(from) || !m.Kickoff.Before(to) {
					continue
				}
				byID[m.ID] = ClubMatch{Match: m, TeamType: wm.TeamType, Competition: strings.TrimSpace(wm.CompetitionName)}
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	matches := make([]ClubMatch, 0, len(byID))
	for _, m := range byID {
		matches = append(matches, m)
	}
	slices.SortFunc(matches, func(a, b ClubMatch) int {
		return cmp.Or(
			cmp.Compare(a.TeamType, b.TeamType),
			a.Kickoff.Compare(b.Kickoff),
			cmp.Compare(a.Competition, b.Competition),
			cmp.Compare(a.ID, b.ID),
		)
	})
	return matches, nil
}
