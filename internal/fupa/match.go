package fupa

import (
	"context"
	"errors"
	"fmt"
	"net/url"
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
	// Slug identifies the referee's FuPa profile.
	Slug, Name, Club string
}

type refereeJSON struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
	Club *struct {
		Name string `json:"name"`
	} `json:"club"`
}

func (r refereeJSON) referee() Referee {
	ref := Referee{Slug: r.Slug, Name: strings.TrimSpace(r.Name)}
	if r.Club != nil {
		ref.Club = strings.TrimSpace(r.Club.Name)
	}
	return ref
}

// ErrNoReferee is returned while FuPa has no referee for a match yet.
var ErrNoReferee = errors.New("FuPa has no referee for the match yet")

// Referee returns the referee of a match. FuPa lists only the referee, not
// the assistants.
func (c *Client) Referee(ctx context.Context, matchSlug string) (Referee, error) {
	var r struct {
		Referee *refereeJSON `json:"referee"`
	}
	if err := c.get(ctx, "/matches/"+matchSlug, &r); err != nil {
		return Referee{}, err
	}
	if r.Referee == nil || r.Referee.Name == "" {
		return Referee{}, ErrNoReferee
	}
	return r.Referee.referee(), nil
}

// RefereeGroup returns the slug of the referee group ("Schiedsrichter-
// gruppe") a referee belongs to, e.g. "isar-rott".
func (c *Client) RefereeGroup(ctx context.Context, refereeSlug string) (string, error) {
	var r struct {
		Group *struct {
			Slug string `json:"slug"`
		} `json:"group"`
	}
	if err := c.get(ctx, "/referees/"+refereeSlug, &r); err != nil {
		return "", err
	}
	if r.Group == nil || r.Group.Slug == "" {
		return "", fmt.Errorf("FuPa names no referee group for %s", refereeSlug)
	}
	return r.Group.Slug, nil
}

// groupPage is the most referees FuPa returns per request.
const groupPage = 100

// GroupReferees returns the referees of a referee group with their clubs.
func (c *Client) GroupReferees(ctx context.Context, groupSlug string) ([]Referee, error) {
	var out []Referee
	for offset := 0; ; offset += groupPage {
		var r struct {
			Referees []refereeJSON `json:"referees"`
		}
		path := fmt.Sprintf("/referee-groups/%s/referees?limit=%d&offset=%d", url.PathEscape(groupSlug), groupPage, offset)
		if err := c.get(ctx, path, &r); err != nil {
			return nil, err
		}
		for _, rj := range r.Referees {
			out = append(out, rj.referee())
		}
		if len(r.Referees) < groupPage {
			return out, nil
		}
	}
}
