package bfv

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"time"
)

// Player is a squad member with their league statistics this season.
type Player struct {
	Name           string
	Matches, Goals int
}

type squadData struct {
	Public  bool `json:"public"`
	Players []struct {
		Player struct {
			Name string `json:"name"`
		} `json:"player"`
		Statistics struct {
			MatchesPlayed int `json:"matchesPlayed"`
			Goals         int `json:"goals"`
		} `json:"statistics"`
	} `json:"players"`
}

// ErrSquadPrivate is returned for a team whose club does not publish its
// squad on bfv.de.
var ErrSquadPrivate = errors.New("squad is not public on bfv.de")

// FetchSquad loads a team's players.
func (c *Client) FetchSquad(ctx context.Context, teamID string) ([]Player, error) {
	var r widgetResponse[squadData]
	if err := c.widget(ctx, "/team/"+teamID+"/squad", &r); err != nil {
		return nil, err
	}
	if !r.Data.Public {
		return nil, ErrSquadPrivate
	}
	players := make([]Player, 0, len(r.Data.Players))
	for _, p := range r.Data.Players {
		players = append(players, Player{Name: p.Player.Name, Matches: p.Statistics.MatchesPlayed, Goals: p.Statistics.Goals})
	}
	return players, nil
}

// Team is a team's identity on BFV.
type Team struct {
	ID, Name string
}

// NextOpponent returns the opponent in a team's next league match: the
// first one kicking off today or later, byes skipped. On a matchday that is
// the opponent of the day.
func (c *Client) NextOpponent(ctx context.Context, teamID string) (Team, error) {
	d, err := c.teamMatches(ctx, teamID)
	if err != nil {
		return Team{}, err
	}
	matches, err := d.matchesIn(d.Team.CompoundID)
	if err != nil {
		return Team{}, err
	}
	now := time.Now
	if c.Now != nil {
		now = c.Now
	}
	y, m, day := now().In(berlin).Date()
	today := time.Date(y, m, day, 0, 0, 0, 0, berlin)
	var next *Match
	for i, mt := range matches {
		if mt.IsBye() || mt.Kickoff.Before(today) {
			continue
		}
		if next == nil || mt.Kickoff.Before(next.Kickoff) {
			next = &matches[i]
		}
	}
	switch {
	case next == nil:
		return Team{}, fmt.Errorf("team %s has no league match left this season", teamID)
	case next.HomeTeamID == teamID:
		return Team{ID: next.GuestTeamID, Name: next.Guest}, nil
	default:
		return Team{ID: next.HomeTeamID, Name: next.Home}, nil
	}
}

// TopScorers returns the players with the most goals, best first: the top
// n, plus anyone tied with the last of them. Players without a goal are
// left out.
func TopScorers(players []Player, n int) []Player {
	sorted := slices.Clone(players)
	slices.SortStableFunc(sorted, func(a, b Player) int {
		if a.Goals != b.Goals {
			return cmp.Compare(b.Goals, a.Goals)
		}
		return cmp.Compare(a.Name, b.Name)
	})
	var top []Player
	for i, p := range sorted {
		if p.Goals == 0 || (i >= n && p.Goals < top[len(top)-1].Goals) {
			break
		}
		top = append(top, p)
	}
	return top
}
