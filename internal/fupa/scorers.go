package fupa

import (
	"context"
	"fmt"
)

// Scorer is a player in a league's scorer list.
type Scorer struct {
	Name, Team string
	Goals      int
}

// LeagueScorers returns a league's top scorers this season: the top n plus
// anyone tied with the last of them.
func (c *Client) LeagueScorers(ctx context.Context, competitionSlug string, n int) ([]Scorer, error) {
	var r []struct {
		Player struct {
			FirstName string `json:"firstName"`
			LastName  string `json:"lastName"`
		} `json:"player"`
		Team       teamRef `json:"team"`
		Statistics struct {
			Goals int `json:"goals"`
		} `json:"statistics"`
	}
	// Ask for more than n, so ties with the n-th are included.
	path := fmt.Sprintf("/competitions/%s/seasons/current/scorers?limit=%d", competitionSlug, n+10)
	if err := c.get(ctx, path, &r); err != nil {
		return nil, err
	}
	var out []Scorer
	for i, s := range r {
		if s.Statistics.Goals == 0 || (i >= n && s.Statistics.Goals < out[len(out)-1].Goals) {
			break
		}
		out = append(out, Scorer{
			Name:  s.Player.FirstName + " " + s.Player.LastName,
			Team:  s.Team.Name.Full,
			Goals: s.Statistics.Goals,
		})
	}
	return out, nil
}
