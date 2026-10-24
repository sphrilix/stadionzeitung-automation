package bfv

import (
	"context"
	"regexp"
	"strconv"
)

// Table is a league's current standings.
type Table struct {
	League string
	Rows   []Standing
	Zones  Zones
}

// Zones counts the teams at the top and bottom of the table that are
// promoted, relegated or go into a playoff.
type Zones struct {
	Promotion, PromotionPlayoff   int
	RelegationPlayoff, Relegation int
}

// Zone names the part of the table a position falls into.
type Zone int

const (
	ZoneNone Zone = iota
	ZonePromotion
	ZonePromotionPlayoff
	ZoneRelegationPlayoff
	ZoneRelegation
)

// Of returns the zone of position (1-based) in a table of n teams.
func (z Zones) Of(position, n int) Zone {
	switch {
	case position <= z.Promotion:
		return ZonePromotion
	case position <= z.Promotion+z.PromotionPlayoff:
		return ZonePromotionPlayoff
	case position > n-z.Relegation:
		return ZoneRelegation
	case position > n-z.Relegation-z.RelegationPlayoff:
		return ZoneRelegationPlayoff
	}
	return ZoneNone
}

// Standing is one row of a league table.
type Standing struct {
	Position int
	Team     string
	TeamID   string
	// ClubID identifies the club's logo; empty if BFV keeps it private.
	ClubID string

	Played, Won, Drawn, Lost int
	// GoalsFor and GoalsAgainst are summed from the published results, as
	// the table itself carries only the difference. HasGoals is false when
	// they do not add up to GoalDiff (e.g. a match decided by the
	// association), so only GoalDiff is reliable.
	GoalsFor, GoalsAgainst int
	HasGoals               bool
	GoalDiff, Points       int
}

var reScore = regexp.MustCompile(`^(\d+):(\d+)$`)

// FetchTable loads a competition's table.
func (c *Client) FetchTable(ctx context.Context, compoundID string) (Table, error) {
	td, err := c.table(ctx, compoundID)
	if err != nil {
		return Table{}, err
	}
	matches, err := c.leagueMatches(ctx, compoundID, td.teamIDs())
	if err != nil {
		return Table{}, err
	}

	type goals struct{ scored, conceded int }
	byTeam := map[string]*goals{}
	clubs := map[string]string{}
	add := func(team, club string, private bool, scored, conceded int) {
		if byTeam[team] == nil {
			byTeam[team] = &goals{}
		}
		byTeam[team].scored += scored
		byTeam[team].conceded += conceded
		if club != "" && !private {
			clubs[team] = club
		}
	}
	for _, m := range matches {
		home, guest := -1, -1
		if s := reScore.FindStringSubmatch(m.Result); s != nil {
			home, _ = strconv.Atoi(s[1])
			guest, _ = strconv.Atoi(s[2])
		}
		if home < 0 {
			// Unplayed, but still tells which club fields the team.
			add(m.HomeTeamID, m.HomeClubID, m.HomeLogoPrivate, 0, 0)
			add(m.GuestTeamID, m.GuestClubID, m.GuestLogoPrivate, 0, 0)
			continue
		}
		add(m.HomeTeamID, m.HomeClubID, m.HomeLogoPrivate, home, guest)
		add(m.GuestTeamID, m.GuestClubID, m.GuestLogoPrivate, guest, home)
	}

	t := Table{
		League: td.CompetitionName,
		Zones: Zones{
			Promotion:         td.Configuration.PromotionTeamCount,
			PromotionPlayoff:  td.Configuration.PromotionPlayoffTeamCount,
			RelegationPlayoff: td.Configuration.RelegationPlayoffTeamCount,
			Relegation:        td.Configuration.RelegationTeamCount,
		},
	}
	for _, row := range td.Table {
		s := Standing{
			Position: row.Position,
			Team:     row.Team.Name,
			TeamID:   row.Team.PermanentID,
			ClubID:   clubs[row.Team.PermanentID],
			Played:   row.Matches,
			Won:      row.MatchesWon,
			Drawn:    row.MatchesDrawn,
			Lost:     row.MatchesLost,
			GoalDiff: row.GoalsDiff,
			Points:   row.Points,
		}
		if g := byTeam[s.TeamID]; g != nil && g.scored-g.conceded == s.GoalDiff {
			s.GoalsFor, s.GoalsAgainst, s.HasGoals = g.scored, g.conceded, true
		}
		t.Rows = append(t.Rows, s)
	}
	return t, nil
}

// ClubLogo returns a club's logo as PNG.
func (c *Client) ClubLogo(ctx context.Context, clubID string) ([]byte, error) {
	return c.get(ctx, c.BaseLogo+clubID)
}
