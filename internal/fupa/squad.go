package fupa

import (
	"context"
	"slices"
)

// Squad is a team's players and staff.
type Squad struct {
	Players []Player
	Staff   []Staff
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
	}
	if err := c.get(ctx, "/teams/"+teamSlug+"/squad", &r); err != nil {
		return Squad{}, err
	}
	var s Squad
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
