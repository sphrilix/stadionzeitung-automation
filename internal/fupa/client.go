// Package fupa reads squads, scorers and referees from FuPa (fupa.net),
// which publishes what BFV does not: positions, shirt numbers, staff,
// league top scorers and referees.
//
// FuPa's terms of use forbid reusing its data without written permission,
// so callers must only use this package with that permission.
package fupa

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
	_ "time/tzdata" // kickoff times are compared in Europe/Berlin
)

const (
	DefaultBase = "https://api.fupa.net/v1"
	// userAgent identifies the tool, so FuPa can tell its requests apart.
	userAgent = "stadionzeitung-automation (+https://github.com/sphrilix/stadionzeitung-automation)"
)

// Client talks to the FuPa API.
type Client struct {
	HTTP *http.Client
	Base string
	// Now returns the current time; nil means time.Now.
	Now func() time.Time
}

// NewClient returns a Client for the production FuPa API.
func NewClient() *Client {
	return &Client{HTTP: http.DefaultClient, Base: DefaultBase}
}

func (c *Client) get(ctx context.Context, path string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.Base+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("FuPa %s: %s", path, resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("decoding FuPa %s: %w", path, err)
	}
	return nil
}

func (c *Client) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

var berlin = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		panic(err)
	}
	return loc
}()

// reTeamSlug matches a team slug in a fupa.net URL or on its own, e.g.
// "fc-sturm-hauzenberg-m1" or "…/team/fc-sturm-hauzenberg-m1-2026-27".
var reTeamSlug = regexp.MustCompile(`^(?:.*/team/)?([a-z0-9-]+)(?:[/?#].*)?$`)

// ParseTeam extracts the team slug from a fupa.net team URL or a bare slug.
// A slug without season ("…-m1") always means the current season.
func ParseTeam(arg string) (string, error) {
	m := reTeamSlug.FindStringSubmatch(strings.TrimSpace(arg))
	if m == nil {
		return "", fmt.Errorf("no FuPa team in %q (expected something like https://www.fupa.net/team/fc-sturm-hauzenberg-m1)", arg)
	}
	return m[1], nil
}

type name struct {
	Full string `json:"full"`
}

type teamRef struct {
	Slug string `json:"slug"`
	Name name   `json:"name"`
}

type competitionRef struct {
	Slug       string `json:"slug"`
	MiddleName string `json:"middleName"`
}

// Team is a team in one season.
type Team struct {
	// Slug includes the season, e.g. "fc-sturm-hauzenberg-m1-2026-27".
	Slug, Name string
	// Competition is the slug of the league it plays in.
	Competition string
}

// Team resolves a team slug, with or without season, to the team's
// current season.
func (c *Client) Team(ctx context.Context, slug string) (Team, error) {
	var r struct {
		teamRef
		Competition competitionRef `json:"competition"`
	}
	// FuPa redirects a slug without season to the current one.
	if err := c.get(ctx, "/teams/"+slug, &r); err != nil {
		return Team{}, err
	}
	return Team{Slug: r.Slug, Name: r.Name.Full, Competition: r.Competition.Slug}, nil
}
