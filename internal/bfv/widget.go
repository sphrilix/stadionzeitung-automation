package bfv

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	_ "time/tzdata" // kickoff times are Europe/Berlin regardless of the host
)

type widgetResponse[T any] struct {
	State   int    `json:"state"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

type tableData struct {
	Table []struct {
		Team struct {
			PermanentID string `json:"permanentId"`
			Name        string `json:"name"`
		} `json:"team"`
	} `json:"table"`
}

type teamMatchesData struct {
	Matches []widgetMatch `json:"matches"`
}

type widgetMatch struct {
	MatchID       string `json:"matchId"`
	CompoundID    string `json:"compoundId"`
	KickoffDate   string `json:"kickoffDate"`
	KickoffTime   string `json:"kickoffTime"`
	HomeTeamName  string `json:"homeTeamName"`
	GuestTeamName string `json:"guestTeamName"`
	Result        string `json:"result"`
}

var berlin = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		panic(err)
	}
	return loc
}()

func (c *Client) widget(ctx context.Context, path string, v any) error {
	body, err := c.get(ctx, c.BaseWidget+path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("decoding %s: %w", path, err)
	}
	return nil
}

// TeamIDs returns the permanent IDs of all teams in a competition's table.
func (c *Client) TeamIDs(ctx context.Context, compoundID string) ([]string, error) {
	var r widgetResponse[tableData]
	if err := c.widget(ctx, "/competition/"+compoundID+"/table", &r); err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(r.Data.Table))
	for _, row := range r.Data.Table {
		ids = append(ids, row.Team.PermanentID)
	}
	return ids, nil
}

// TeamMatches returns a team's season fixtures within the given competition,
// including byes.
func (c *Client) TeamMatches(ctx context.Context, teamID, compoundID string) ([]Match, error) {
	var r widgetResponse[teamMatchesData]
	if err := c.widget(ctx, "/team/"+teamID+"/matches", &r); err != nil {
		return nil, err
	}
	var out []Match
	for _, wm := range r.Data.Matches {
		if wm.CompoundID != compoundID {
			continue
		}
		m, err := wm.toMatch()
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, nil
}

func (wm widgetMatch) toMatch() (Match, error) {
	m := Match{
		ID:     wm.MatchID,
		Home:   strings.TrimSpace(wm.HomeTeamName),
		Guest:  strings.TrimSpace(wm.GuestTeamName),
		Result: strings.TrimSpace(wm.Result),
	}
	if wm.KickoffDate == "" {
		return m, nil
	}
	layout, value := "02.01.2006", wm.KickoffDate
	if wm.KickoffTime != "" {
		layout, value = layout+" 15:04", value+" "+wm.KickoffTime
		m.HasTime = true
	}
	t, err := time.ParseInLocation(layout, value, berlin)
	if err != nil {
		return m, fmt.Errorf("match %s: %w", wm.MatchID, err)
	}
	m.Kickoff = t
	return m, nil
}
