package fupa

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Match is a fixture of a team.
type Match struct {
	Slug       string
	Kickoff    time.Time
	Home, Away Team
}

type matchJSON struct {
	Slug     string    `json:"slug"`
	Kickoff  time.Time `json:"kickoff"`
	HomeTeam teamRef   `json:"homeTeam"`
	AwayTeam teamRef   `json:"awayTeam"`
	Category string    `json:"category"`
}

// NextLeagueMatch returns the team's first league match kicking off today
// or later (Berlin time): on a matchday, the match of the day.
func (c *Client) NextLeagueMatch(ctx context.Context, teamSlug string) (Match, error) {
	var r []matchJSON
	// "current" lists the upcoming matches, starting with the latest played.
	if err := c.get(ctx, "/teams/"+teamSlug+"/matches?flavor=current&limit=60", &r); err != nil {
		return Match{}, err
	}
	y, m, d := c.now().In(berlin).Date()
	today := time.Date(y, m, d, 0, 0, 0, 0, berlin)
	var next *matchJSON
	for i, mj := range r {
		if mj.Category != "league" || mj.Kickoff.Before(today) {
			continue
		}
		if next == nil || mj.Kickoff.Before(next.Kickoff) {
			next = &r[i]
		}
	}
	if next == nil {
		return Match{}, fmt.Errorf("FuPa lists no upcoming league match for %s", teamSlug)
	}
	return Match{
		Slug:    next.Slug,
		Kickoff: next.Kickoff,
		Home:    Team{Slug: next.HomeTeam.Slug, Name: next.HomeTeam.Name.Full},
		Away:    Team{Slug: next.AwayTeam.Slug, Name: next.AwayTeam.Name.Full},
	}, nil
}

// Opponent returns the team that teamSlug (with season) plays in m.
func (m Match) Opponent(teamSlug string) Team {
	if m.Home.Slug == teamSlug {
		return m.Away
	}
	return m.Home
}

// Referee is a match official with their club.
type Referee struct {
	Name, Club string
}

// ErrNoReferee is returned while FuPa has no referee for a match yet.
var ErrNoReferee = errors.New("FuPa has no referee for the match yet")

// Referee returns the referee of a match. FuPa lists only the referee, not
// the assistants.
func (c *Client) Referee(ctx context.Context, matchSlug string) (Referee, error) {
	var r struct {
		Referee *struct {
			Name string `json:"name"`
			Club *struct {
				Name string `json:"name"`
			} `json:"club"`
		} `json:"referee"`
	}
	if err := c.get(ctx, "/matches/"+matchSlug, &r); err != nil {
		return Referee{}, err
	}
	if r.Referee == nil || r.Referee.Name == "" {
		return Referee{}, ErrNoReferee
	}
	ref := Referee{Name: r.Referee.Name}
	if r.Referee.Club != nil {
		ref.Club = strings.TrimSpace(r.Referee.Club.Name)
	}
	return ref, nil
}
