package main

import (
	"fmt"
	"testing"

	"github.com/sphrilix/stadionzeitung-automation/internal/fupa"
)

func TestSquadRows(t *testing.T) {
	s := fupa.Squad{
		Staff: []fupa.Staff{
			{Name: "Matthias Süß", Role: "Trainer"}, {Name: "Pascal Keese", Role: "Torwart-Trainer"},
			{Name: "Dominik Schwarz", Role: "Trainer"}, {Name: "Alexander Geiger", Role: "Sportdirektor"},
		},
		Players: []fupa.Player{
			{FirstName: "Stefan", LastName: "Schulz", Position: "Torwart", Number: 21},
			{FirstName: "Christoph", LastName: "Obermüller", Position: "Torwart", Number: 1, Captain: true},
			{FirstName: "Michael", LastName: "Hellauer", Position: "Torwart"},
			{FirstName: "Joris", LastName: "Wagner", Position: "Angriff", Number: 9},
			{FirstName: "Neu", LastName: "Ling", Position: "Libero", Number: 5},
		},
	}
	got := fmt.Sprint(squadRows(s, func(string) bool { return true }, false))
	want := "[{Trainer: Matthias Süß, Dominik Schwarz false} {TW-Trainer: Pascal Keese false} {Sportdirektor: Alexander Geiger false} " +
		"{Tor: Christoph Obermüller (K), Stefan Schulz true} {Angriff: Joris Wagner false} {Weitere: Neu Ling false}]"
	if got != want {
		t.Errorf("squadRows =\n%s\nwant\n%s", got, want)
	}
	// Only coaches, and every player.
	got = fmt.Sprint(squadRows(s, func(r string) bool { return r == "Trainer" }, true)[:2])
	want = "[{Trainer: Matthias Süß, Dominik Schwarz false} {Tor: Michael Hellauer, Christoph Obermüller (K), Stefan Schulz true}]"
	if got != want {
		t.Errorf("squadRows(Trainer, all) =\n%s\nwant\n%s", got, want)
	}
}
