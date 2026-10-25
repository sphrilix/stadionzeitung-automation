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
	CompetitionName string `json:"competitionName"`
	Configuration   struct {
		PromotionTeamCount         int `json:"promotionTeamCount"`
		PromotionPlayoffTeamCount  int `json:"promotionPlayoffTeamCount"`
		RelegationPlayoffTeamCount int `json:"relegationPlayoffTeamCount"`
		RelegationTeamCount        int `json:"relegationTeamCount"`
	} `json:"configuration"`
	Table []struct {
		Position     int `json:"position"`
		Matches      int `json:"matches"`
		MatchesWon   int `json:"matchesWon"`
		MatchesDrawn int `json:"matchesDrawn"`
		MatchesLost  int `json:"matchesLost"`
		GoalsDiff    int `json:"goalsDiff"`
		Points       int `json:"points"`
		Team         struct {
			PermanentID string `json:"permanentId"`
			Name        string `json:"name"`
		} `json:"team"`
	} `json:"table"`
}

type teamMatchesData struct {
	Team struct {
		Name            string `json:"name"`
		CompoundID      string `json:"compoundId"`
		CompetitionName string `json:"competitionName"`
	} `json:"team"`
	Matches []widgetMatch `json:"matches"`
}

type widgetMatch struct {
	MatchID         string `json:"matchId"`
	CompoundID      string `json:"compoundId"`
	CompetitionName string `json:"competitionName"`
	TeamType        string `json:"teamType"`
	KickoffDate     string `json:"kickoffDate"`
	KickoffTime     string `json:"kickoffTime"`
	HomeTeamName    string `json:"homeTeamName"`
	GuestTeamName   string `json:"guestTeamName"`
	Result          string `json:"result"`

	HomeTeamPermanentID  string `json:"homeTeamPermanentId"`
	GuestTeamPermanentID string `json:"guestTeamPermanentId"`
	HomeClubID           string `json:"homeClubId"`
	GuestClubID          string `json:"guestClubId"`
	HomeLogoPrivate      bool   `json:"homeLogoPrivate"`
	GuestLogoPrivate     bool   `json:"guestLogoPrivate"`
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

func (c *Client) table(ctx context.Context, compoundID string) (tableData, error) {
	var r widgetResponse[tableData]
	err := c.widget(ctx, "/competition/"+compoundID+"/table", &r)
	return r.Data, err
}

// TeamIDs returns the permanent IDs of all teams in a competition's table.
func (c *Client) TeamIDs(ctx context.Context, compoundID string) ([]string, error) {
	t, err := c.table(ctx, compoundID)
	if err != nil {
		return nil, err
	}
	return t.teamIDs(), nil
}

func (t tableData) teamIDs() []string {
	ids := make([]string, 0, len(t.Table))
	for _, row := range t.Table {
		ids = append(ids, row.Team.PermanentID)
	}
	return ids
}

// TeamMatches returns a team's season fixtures within the given competition,
// including byes.
func (c *Client) TeamMatches(ctx context.Context, teamID, compoundID string) ([]Match, error) {
	d, err := c.teamMatches(ctx, teamID)
	if err != nil {
		return nil, err
	}
	return d.matchesIn(compoundID)
}

func (c *Client) teamMatches(ctx context.Context, teamID string) (teamMatchesData, error) {
	var r widgetResponse[teamMatchesData]
	err := c.widget(ctx, "/team/"+teamID+"/matches", &r)
	return r.Data, err
}

func (d teamMatchesData) matchesIn(compoundID string) ([]Match, error) {
	var out []Match
	for _, wm := range d.Matches {
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

		HomeTeamID:  wm.HomeTeamPermanentID,
		GuestTeamID: wm.GuestTeamPermanentID,
		HomeClubID:  wm.HomeClubID,
		GuestClubID: wm.GuestClubID,

		HomeLogoPrivate:  wm.HomeLogoPrivate,
		GuestLogoPrivate: wm.GuestLogoPrivate,
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
