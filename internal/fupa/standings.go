package fupa

import (
	"context"
	"net/url"
)

// Standing is one row of a league table, with the team's club logo.
type Standing struct {
	Rank, Matches, Points int
	Team                  string
	// Logo is the logo's address on FuPa's image server, empty if none.
	Logo string
}

// Standings returns a league's current table.
func (c *Client) Standings(ctx context.Context, competitionSlug string) ([]Standing, error) {
	var r struct {
		Standings []struct {
			Rank    int `json:"rank"`
			Matches int `json:"matches"`
			Points  int `json:"points"`
			Team    struct {
				teamRef
				Image *struct {
					Path string `json:"path"`
				} `json:"image"`
			} `json:"team"`
		} `json:"standings"`
	}
	if err := c.get(ctx, "/standings?competition="+url.QueryEscape(competitionSlug), &r); err != nil {
		return nil, err
	}
	out := make([]Standing, len(r.Standings))
	for i, s := range r.Standings {
		out[i] = Standing{Rank: s.Rank, Matches: s.Matches, Points: s.Points, Team: s.Team.Name.Full}
		if s.Team.Image != nil {
			out[i].Logo = s.Team.Image.Path
		}
	}
	return out, nil
}

// logoSize is a size FuPa's image server offers logos in; the tables
// print them at a few millimetres.
const logoSize = "200x200.png"

// Logo downloads a club logo as PNG.
func (c *Client) Logo(ctx context.Context, path string) ([]byte, error) {
	return c.image(ctx, path, logoSize)
}
