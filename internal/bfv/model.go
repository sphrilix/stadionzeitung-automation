// Package bfv fetches league fixtures and results from the Bayerischer
// Fußball-Verband (bfv.de).
package bfv

import "time"

// Match is a single fixture of a league.
type Match struct {
	ID      string
	Kickoff time.Time
	// HasTime is false when BFV published only the date (e.g. "SPIELFREI" or
	// fixtures without a fixed kickoff yet).
	HasTime bool
	Home    string
	Guest   string
	// Result is the score as published by BFV, e.g. "0:4"; empty when the
	// match has not been played.
	Result string
}

// byeTeam is the placeholder opponent BFV uses for a team without a match.
const byeTeam = "SPIELFREI"

// IsBye reports whether m is a placeholder for a team that has no match.
func (m Match) IsBye() bool { return m.Home == byeTeam || m.Guest == byeTeam }

// Matchday is one round ("Spieltag") of a league.
type Matchday struct {
	Number  int
	Matches []Match
}
