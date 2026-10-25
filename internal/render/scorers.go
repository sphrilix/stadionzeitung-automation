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
	return renderBlocks(w, fonts, linesBlock(lines))
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
	return renderBlocks(w, fonts, rowsBlock(rows))
}

// wrap keeps an empty text as one empty line, so its label still shows.
func wrap(lines []string) []string {
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}
