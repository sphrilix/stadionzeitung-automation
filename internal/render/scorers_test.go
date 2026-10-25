package render

import (
	"bytes"
	"testing"

	"github.com/sphrilix/stadionzeitung-automation/internal/bfv"
)

func TestScorerLine(t *testing.T) {
	for p, want := range map[bfv.Player]string{
		{Name: "Joris Wagner", Goals: 10}:        "J. Wagner (10 Tore)",
		{Name: "Luca Canales Uhrmann", Goals: 4}: "L. Canales Uhrmann (4 Tore)",
		{Name: "Ömer Yıldız", Goals: 1}:          "Ö. Yıldız (1 Tor)",
		{Name: "Pelé", Goals: 2}:                 "Pelé (2 Tore)",
	} {
		if got := ScorerLine(p); got != want {
			t.Errorf("ScorerLine(%+v) = %q, want %q", p, got, want)
		}
	}
}

func TestRenderScorers(t *testing.T) {
	for _, scorers := range [][]bfv.Player{{{Name: "Joris Wagner", Goals: 10}, {Name: "Manuel Mader", Goals: 8}}, nil} {
		var buf bytes.Buffer
		if err := RenderScorers(&buf, scorers, Fonts{}); err != nil {
			t.Fatal(err)
		}
		if !bytes.HasPrefix(buf.Bytes(), []byte("%PDF-")) {
			t.Errorf("%d scorers: output is not a PDF", len(scorers))
		}
	}
}
