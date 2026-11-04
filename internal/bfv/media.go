package bfv

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// DefaultBaseMedia serves the team photos bfv.de shows on a team's page.
const DefaultBaseMedia = "https://service.media.fussball.de/api/uimg/cont/-/ID/"

// ErrNoTeamPhoto is returned for a team without a photo on bfv.de.
var ErrNoTeamPhoto = errors.New("BFV has no team photo")

// teamPhotoSize is the largest size of a team photo, 1024 px wide.
const teamPhotoSize = "/TYP/50/SZ/0"

// TeamPhoto downloads a team's photo as JPEG. The media server redirects
// to the photo, or to a PNG placeholder if the team has none.
func (c *Client) TeamPhoto(ctx context.Context, teamID string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseMedia+teamID+teamPhotoSize, nil)
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
		return nil, fmt.Errorf("BFV team photo: %s", resp.Status)
	}
	if strings.Contains(resp.Request.URL.Path, "placeholder") || resp.Header.Get("Content-Type") != "image/jpeg" {
		return nil, ErrNoTeamPhoto
	}
	return io.ReadAll(resp.Body)
}

// Scorer is a player in a league's scorer list.
type Scorer struct {
	Name, Team string
	Goals      int
}

// LeagueScorers returns a competition's top scorers this season: the top n
// plus anyone tied with the last of them.
func (c *Client) LeagueScorers(ctx context.Context, compoundID string, n int) ([]Scorer, error) {
	var r widgetResponse[struct {
		Scorers []struct {
			Name string `json:"name"`
			Team struct {
				Name string `json:"name"`
			} `json:"team"`
			Goals int `json:"goals"`
		} `json:"scorers"`
	}]
	if err := c.widget(ctx, "/competition/"+compoundID+"/topscorer", &r); err != nil {
		return nil, err
	}
	var out []Scorer
	for i, s := range r.Data.Scorers {
		if s.Goals == 0 || (i >= n && s.Goals < out[len(out)-1].Goals) {
			break
		}
		out = append(out, Scorer{Name: strings.TrimSpace(s.Name), Team: strings.TrimSpace(s.Team.Name), Goals: s.Goals})
	}
	return out, nil
}
