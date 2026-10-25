package render

import (
	"fmt"
	"io"
	"strings"

	"github.com/sphrilix/stadionzeitung-automation/internal/bfv"
)

const (
	scorerFontSize = 10.0
	scorerLine     = 5.0
)

// RenderScorers writes a single-page PDF listing scorers as
// "J. Wagner (10 Tore)", one per line; the page is as wide as the longest.
func RenderScorers(w io.Writer, scorers []bfv.Player, fonts Fonts) error {
	lines := make([]string, len(scorers))
	for i, p := range scorers {
		lines[i] = ScorerLine(p)
	}
	// Measure with a throwaway page, as fpdf needs one to load the font.
	probe, err := newPage(10, 10, fonts)
	if err != nil {
		return err
	}
	probe.SetFont(fontFamily, "", scorerFontSize)
	width := 0.0
	for _, l := range lines {
		width = max(width, probe.GetStringWidth(l))
	}
	// Keep a page for a team without goals yet, so the link stays valid.
	rows := max(1, len(lines))

	pdf, err := newPage(width+2*edge, float64(rows)*scorerLine+2*edge, fonts)
	if err != nil {
		return err
	}
	pdf.SetFont(fontFamily, "", scorerFontSize)
	pdf.SetCellMargin(0)
	for i, l := range lines {
		pdf.SetXY(edge, edge+float64(i)*scorerLine)
		pdf.CellFormat(width, scorerLine, l, "", 0, "L", false, 0, "")
	}
	return output(w, pdf)
}

// ScorerLine formats a scorer as in the Stadionzeitung: first name as an
// initial, e.g. "L. Canales Uhrmann (4 Tore)".
func ScorerLine(p bfv.Player) string {
	name := p.Name
	if first, rest, ok := strings.Cut(name, " "); ok && first != "" {
		name = string([]rune(first)[0]) + ". " + rest
	}
	unit := "Tore"
	if p.Goals == 1 {
		unit = "Tor"
	}
	return fmt.Sprintf("%s (%d %s)", name, p.Goals, unit)
}
