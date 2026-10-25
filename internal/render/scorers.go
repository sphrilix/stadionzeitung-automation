package render

import (
	"fmt"
	"io"
	"strings"

	"github.com/sphrilix/stadionzeitung-automation/internal/bfv"
)

const (
	textFontSize = 10.0
	textLine     = 5.0
)

// RenderScorers writes a single-page PDF listing scorers as
// "J. Wagner (10 Tore)", one per line.
func RenderScorers(w io.Writer, scorers []bfv.Player, fonts Fonts) error {
	lines := make([]string, len(scorers))
	for i, p := range scorers {
		lines[i] = ScorerLine(p)
	}
	return RenderLines(w, lines, fonts)
}

// ScorerLine formats a scorer as in the Stadionzeitung: first name as an
// initial, e.g. "L. Canales Uhrmann (4 Tore)".
func ScorerLine(p bfv.Player) string {
	name := p.Name
	if first, rest, ok := strings.Cut(name, " "); ok && first != "" {
		name = string([]rune(first)[0]) + ". " + rest
	}
	return fmt.Sprintf("%s (%s)", name, Goals(p.Goals))
}

// Goals formats a goal count, e.g. "1 Tor" or "10 Tore".
func Goals(n int) string {
	if n == 1 {
		return "1 Tor"
	}
	return fmt.Sprintf("%d Tore", n)
}

// RenderLines writes a single-page PDF with one line of text each; the page
// is as wide as the longest line.
func RenderLines(w io.Writer, lines []string, fonts Fonts) error {
	// Measure with a throwaway page, as fpdf needs one to load the font.
	probe, err := newPage(10, 10, fonts)
	if err != nil {
		return err
	}
	probe.SetFont(fontFamily, "", textFontSize)
	width := 0.0
	for _, l := range lines {
		width = max(width, probe.GetStringWidth(l))
	}
	// Keep a page without lines (e.g. no goals yet), so the link stays valid.
	rows := max(1, len(lines))

	pdf, err := newPage(width+2*edge, float64(rows)*textLine+2*edge, fonts)
	if err != nil {
		return err
	}
	pdf.SetFont(fontFamily, "", textFontSize)
	pdf.SetCellMargin(0)
	for i, l := range lines {
		pdf.SetXY(edge, edge+float64(i)*textLine)
		pdf.CellFormat(width, textLine, l, "", 0, "L", false, 0, "")
	}
	return output(w, pdf)
}

// Row is a labelled entry of RenderRows, e.g. "Abwehr:" and its players.
type Row struct {
	Label, Text string
	// Gap adds space above the row, to separate blocks.
	Gap bool
}

const (
	rowsWidth = boxWidth
	labelPad  = 3.0
	rowGap    = 3.0
)

// RenderRows writes a single-page PDF with bold labels in a column as wide
// as the longest label and their text next to it, wrapped to the page
// width, as the Kader pages are set.
func RenderRows(w io.Writer, rows []Row, fonts Fonts) error {
	probe, err := newPage(10, 10, fonts)
	if err != nil {
		return err
	}
	probe.SetCellMargin(0)
	probe.SetFont(fontFamily, "B", textFontSize)
	labelWidth := 0.0
	for _, r := range rows {
		labelWidth = max(labelWidth, probe.GetStringWidth(r.Label)+labelPad)
	}
	textWidth := rowsWidth - labelWidth
	probe.SetFont(fontFamily, "", textFontSize)
	wrapped := make([][]string, len(rows))
	height := 2 * edge
	for i, r := range rows {
		wrapped[i] = wrap(probe.SplitText(r.Text, textWidth))
		if r.Gap && i > 0 {
			height += rowGap
		}
		height += float64(len(wrapped[i])) * textLine
	}

	pdf, err := newPage(rowsWidth+2*edge, height, fonts)
	if err != nil {
		return err
	}
	pdf.SetCellMargin(0)
	y := edge
	for i, r := range rows {
		if r.Gap && i > 0 {
			y += rowGap
		}
		pdf.SetFont(fontFamily, "B", textFontSize)
		pdf.SetXY(edge, y)
		pdf.CellFormat(labelWidth, textLine, r.Label, "", 0, "L", false, 0, "")
		pdf.SetFont(fontFamily, "", textFontSize)
		for _, line := range wrapped[i] {
			pdf.SetXY(edge+labelWidth, y)
			pdf.CellFormat(textWidth, textLine, line, "", 0, "L", false, 0, "")
			y += textLine
		}
	}
	return output(w, pdf)
}

// wrap keeps an empty text as one empty line, so its label still shows.
func wrap(lines []string) []string {
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}
