package bfv

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

const (
	DefaultBaseWeb    = "https://www.bfv.de"
	DefaultBaseWidget = "https://widget-prod.bfv.de/api/service/widget/v1"
	DefaultBaseLogo   = "https://app.bfv.de/export.media/-/action/getLogo/format/7/id/"

	userAgent = "Mozilla/5.0 (X11; Linux x86_64) stadionzeitung-automation"
	// maxParallel bounds concurrent requests against BFV.
	maxParallel = 4
)

// Client talks to bfv.de and the BFV widget API.
type Client struct {
	HTTP       *http.Client
	BaseWeb    string
	BaseWidget string
	// BaseLogo is prefixed to a club ID to get its logo.
	BaseLogo string
	// Now returns the current time; nil means time.Now.
	Now func() time.Time
}

// NewClient returns a Client for the production BFV endpoints.
func NewClient() *Client {
	return &Client{HTTP: http.DefaultClient, BaseWeb: DefaultBaseWeb, BaseWidget: DefaultBaseWidget, BaseLogo: DefaultBaseLogo}
}

func (c *Client) get(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// Window is a league's current matchday together with its neighbours.
type Window struct {
	League  string
	Current int
	// Days holds the matchdays before, at and after Current; those outside
	// the season are omitted.
	Days []Matchday
}

// FetchWindow loads the matchdays around the current one: the matchday
// running today or, between matchdays, the next one. A positive current
// overrides that choice.
func (c *Client) FetchWindow(ctx context.Context, compoundID string, current int) (Window, error) {
	first, err := c.FetchPage(ctx, compoundID, current)
	if err != nil {
		return Window{}, err
	}
	if current <= 0 {
		now := time.Now
		if c.Now != nil {
			now = c.Now
		}
		current = first.RunningOrNext(now())
	} else {
		current = first.Matchday
	}

	var numbers []int
	for n := current - 1; n <= current+1; n++ {
		if n >= 1 && n <= first.MaxMatchday {
			numbers = append(numbers, n)
		}
	}

	teams, err := c.TeamIDs(ctx, compoundID)
	if err != nil {
		return Window{}, err
	}

	pages, err := c.fetchPages(ctx, compoundID, numbers, first)
	if err != nil {
		return Window{}, err
	}
	matches, err := c.leagueMatches(ctx, compoundID, teams)
	if err != nil {
		return Window{}, err
	}

	days := make([]Matchday, len(numbers))
	for i, n := range numbers {
		days[i] = Matchday{Number: n}
		var missing []string
		for _, id := range pages[i].MatchIDs {
			m, ok := matches[id]
			if !ok {
				missing = append(missing, id)
				continue
			}
			if m.IsBye() {
				continue
			}
			days[i].Matches = append(days[i].Matches, m)
		}
		if len(missing) > 0 {
			return Window{}, fmt.Errorf("matchday %d: no details for matches %s", n, strings.Join(missing, ", "))
		}
		sort.SliceStable(days[i].Matches, func(a, b int) bool {
			return days[i].Matches[a].Kickoff.Before(days[i].Matches[b].Kickoff)
		})
	}
	return Window{League: first.LeagueName, Current: current, Days: days}, nil
}

// leagueMatches collects every match of a competition, byes included, keyed
// by match ID. BFV has no per-competition list, so it merges the season
// fixtures of all teams.
func (c *Client) leagueMatches(ctx context.Context, compoundID string, teams []string) (map[string]Match, error) {
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(maxParallel)
	var (
		mu      sync.Mutex
		matches = map[string]Match{}
	)
	for _, team := range teams {
		g.Go(func() error {
			ms, err := c.TeamMatches(gctx, team, compoundID)
			mu.Lock()
			defer mu.Unlock()
			for _, m := range ms {
				matches[m.ID] = m
			}
			return err
		})
	}
	return matches, g.Wait()
}

// fetchPages loads the pages of the given matchdays, reusing first (an
// already loaded page) where it matches.
func (c *Client) fetchPages(ctx context.Context, compoundID string, numbers []int, first PageInfo) ([]PageInfo, error) {
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(maxParallel)
	pages := make([]PageInfo, len(numbers))
	for i, n := range numbers {
		if n == first.Matchday {
			pages[i] = first
			continue
		}
		g.Go(func() error {
			p, err := c.FetchPage(gctx, compoundID, n)
			pages[i] = p
			return err
		})
	}
	return pages, g.Wait()
}
