package render

import (
	"bytes"
	"testing"
	"time"

	"github.com/sphrilix/bfv-miner/internal/bfv"
)

func TestRender(t *testing.T) {
	kick := time.Date(2026, 10, 10, 15, 0, 0, 0, time.UTC)
	days := []bfv.Matchday{
		{Number: 15, Matches: []bfv.Match{
			{Kickoff: kick, HasTime: true, Home: "TSV 1883 Bogen", Guest: "SV Türk Gücü Straubing", Result: "0:4"},
			{Kickoff: kick, HasTime: true, Home: "1. FC Bad Kötzting", Guest: "TSV Seebach", Result: "1:1"},
		}},
		{Number: 16, Matches: []bfv.Match{
			{Kickoff: kick.AddDate(0, 0, 7), HasTime: true, Home: "1. FC Passau", Guest: "FC Amberg"},
		}},
	}
	var buf bytes.Buffer
	if err := Render(&buf, "Landesliga Mitte", days); err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(buf.Bytes(), []byte("%PDF-")) || buf.Len() < 1000 {
		t.Errorf("output does not look like a PDF (%d bytes)", buf.Len())
	}
}

func TestFormatResult(t *testing.T) {
	for in, want := range map[string]string{"": "-", "0:4": "0 - 4", " 10 : 2 ": "10 - 2", "abgesetzt": "abgesetzt"} {
		if got := formatResult(in); got != want {
			t.Errorf("formatResult(%q) = %q, want %q", in, got, want)
		}
	}
}
