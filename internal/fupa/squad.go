package fupa

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
)

// Squad is a team's players and staff.
type Squad struct {
	Players []Player
	Staff   []Staff
	// Photo is the team photo's address on FuPa's image server, empty if
	// there is none; PhotoCredit names the photographer.
	Photo, PhotoCredit string
}

// Player is a squad member.
type Player struct {
	FirstName, LastName string
	// Position is FuPa's: "Torwart", "Abwehr", "Mittelfeld" or "Angriff".
	Position string
	// Number is the shirt number, 0 if none is assigned.
	Number         int
	Captain        bool
	Matches, Goals int
}

// Name is the player's full name.
func (p Player) Name() string { return p.FirstName + " " + p.LastName }

// Staff is a coach or another team official.
type Staff struct {
	Name string
	// Role is e.g. "Trainer", "Torwart-Trainer" or "Betreuer".
	Role string
}

// Squad loads a team's squad. Deactivated players and officials are left
// out.
func (c *Client) Squad(ctx context.Context, teamSlug string) (Squad, error) {
	var r struct {
		Players []struct {
			FirstName     string   `json:"firstName"`
			LastName      string   `json:"lastName"`
			IsDeactivated bool     `json:"isDeactivated"`
			Position      string   `json:"position"`
			JerseyNumber  *int     `json:"jerseyNumber"`
			Matches       int      `json:"matches"`
			Goals         int      `json:"goals"`
			Flags         []string `json:"flags"`
		} `json:"players"`
		Coaches []struct {
			FirstName     string `json:"firstName"`
			LastName      string `json:"lastName"`
			Role          string `json:"role"`
			IsDeactivated bool   `json:"isDeactivated"`
		} `json:"coaches"`
		Info struct {
			TeamImage *struct {
				Path   string `json:"path"`
				Source string `json:"source"`
			} `json:"teamImage"`
		} `json:"info"`
	}
	if err := c.get(ctx, "/teams/"+teamSlug+"/squad", &r); err != nil {
		return Squad{}, err
	}
	var s Squad
	if img := r.Info.TeamImage; img != nil {
		s.Photo, s.PhotoCredit = img.Path, strings.TrimSpace(img.Source)
	}
	for _, p := range r.Players {
		if p.IsDeactivated {
			continue
		}
		pl := Player{
			FirstName: p.FirstName, LastName: p.LastName, Position: p.Position,
			Captain: slices.Contains(p.Flags, "captain"),
			Matches: p.Matches, Goals: p.Goals,
		}
		if p.JerseyNumber != nil {
			pl.Number = *p.JerseyNumber
		}
		s.Players = append(s.Players, pl)
	}
	for _, co := range r.Coaches {
		if !co.IsDeactivated {
			s.Staff = append(s.Staff, Staff{Name: co.FirstName + " " + co.LastName, Role: co.Role})
		}
	}
	return s, nil
}

// Numbered returns the players with a shirt number: a club's squad on FuPa
// often lists players of its other teams too, without a number. If nobody
// has a number, all players are returned.
func (s Squad) Numbered() []Player {
	var out []Player
	for _, p := range s.Players {
		if p.Number > 0 {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return s.Players
	}
	return out
}

// photoSize is the largest size FuPa's image server offers; across the
// 160 mm of a page it prints at about 300 dpi.
const photoSize = "1920x1080.jpeg"

// TeamPhoto downloads a squad's team photo as JPEG.
func (c *Client) TeamPhoto(ctx context.Context, s Squad) ([]byte, error) {
	if s.Photo == "" {
		return nil, fmt.Errorf("FuPa has no team photo")
	}
	url := strings.TrimSuffix(s.Photo, "/") + "/" + photoSize
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
		return nil, fmt.Errorf("FuPa team photo: %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}
