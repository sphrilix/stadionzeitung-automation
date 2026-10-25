package bfv

import (
	"context"
	"fmt"
	"sort"
)

// Season is a team's league fixtures for the whole season.
type Season struct {
	Team, League string
	Matches      []SeasonMatch
}

// SeasonMatch is a fixture of a Season. Byes are included, as the
// Stadionzeitung lists them ("SPIELFREI").
type SeasonMatch struct {
	// Matchday is 0 if the match is on no matchday page.
	Matchday int
	Match
}

// FetchSeason loads a team's season fixtures in the league it plays in.
//
// The widget API has no matchday numbers and lists rescheduled matches by
// their new date, so the numbers come from the competition's matchday pages.
func (c *Client) FetchSeason(ctx context.Context, teamID string) (Season, error) {
	d, err := c.teamMatches(ctx, teamID)
	if err != nil {
		return Season{}, err
	}
	compoundID := d.Team.CompoundID
	if compoundID == "" {
		return Season{}, fmt.Errorf("team %s plays in no league", teamID)
	}
	matches, err := d.matchesIn(compoundID)
	if err != nil {
		return Season{}, err
	}

	first, err := c.FetchPage(ctx, compoundID, 0)
	if err != nil {
		return Season{}, err
	}
	numbers := make([]int, first.MaxMatchday)
	for i := range numbers {
		numbers[i] = i + 1
	}
	pages, err := c.fetchPages(ctx, compoundID, numbers, first)
	if err != nil {
		return Season{}, err
	}
	matchday := map[string]int{}
	for i, p := range pages {
		for _, id := range p.MatchIDs {
			matchday[id] = numbers[i]
		}
	}

	s := Season{Team: d.Team.Name, League: d.Team.CompetitionName}
	for _, m := range matches {
		s.Matches = append(s.Matches, SeasonMatch{Matchday: matchday[m.ID], Match: m})
	}
	sort.SliceStable(s.Matches, func(a, b int) bool {
		ma, mb := s.Matches[a], s.Matches[b]
		if ma.Matchday != mb.Matchday {
			return ma.Matchday < mb.Matchday
		}
		return ma.Kickoff.Before(mb.Kickoff)
	})
	return s, nil
}

// TeamInfo is a team's name and the league it plays in.
type TeamInfo struct {
	Name string
	// CompoundID is the league's competition id, League its name.
	CompoundID, League string
}

// FetchTeamInfo returns a team's name and league, so a team's bfv.de page
// is enough to find its league's matchdays and table.
func (c *Client) FetchTeamInfo(ctx context.Context, teamID string) (TeamInfo, error) {
	d, err := c.teamMatches(ctx, teamID)
	if err != nil {
		return TeamInfo{}, err
	}
	if d.Team.CompoundID == "" {
		return TeamInfo{}, fmt.Errorf("team %s plays in no league", teamID)
	}
	return TeamInfo{Name: d.Team.Name, CompoundID: d.Team.CompoundID, League: d.Team.CompetitionName}, nil
}
