package render

import (
	"io"
	"math"

	"github.com/sphrilix/stadionzeitung-automation/internal/bfv"
)

// Columns of a club match row: date and kickoff | home team (right-aligned)
// | "-" | guest team | result.
const (
	cColWhen   = 27.0
	cColResult = 14.0
	cColTeam   = (boxWidth - cColWhen - colDash - cColResult) / 2

	cGroupHeight = 6.0
	cRowHeight   = 5.5
	cGroupGap    = 2.0

	// clubPageHeight gives the youth pages the A-series proportions of the
	// programme's pages, so that fitted proportionally the PDF fills a
	// page.
	clubPageHeight = boxWidth * math.Sqrt2
	// cMaxStretch caps how much rows grow on a page with few matches; the
	// rest of the page goes to the gaps between the groups, up to cMaxGap
	// each, and what is still left stays at the bottom.
	cMaxStretch = 2.0
	cMaxGap     = 2 * cGroupHeight * cMaxStretch
	// cMaxFontSize caps the text on stretched rows.
	cMaxFontSize = 14.0
)

// RenderClubMatches writes a full page with a club's matches, one block per
// age group and competition, e.g. "A-Junioren · U 19 (A-Jun.) BOL
// Niederbayern". The rows stretch to fill the page, or shrink when there
// are many matches. Without results the result column shows nothing, for a
// preview of upcoming matches.
func RenderClubMatches(w io.Writer, matches []bfv.ClubMatch, results bool, fonts Fonts) error {
	groups := groupClubMatches(matches)
	l := layoutClubPage(groups)
	scale, rowH, groupH, gap := l.scale, l.row, l.group, l.gap

	pdf, err := newPage(boxWidth+2*edge, clubPageHeight, fonts)
	if err != nil {
		return err
	}

	x, y := edge, edge
	pdf.SetLineWidth(0.3)
	pdf.Rect(x, y, boxWidth, clubPageHeight-2*edge, "D")

	var names []string
	for _, m := range matches {
		names = append(names, m.Home, m.Guest)
	}
	fontSize := fitTextsUpTo(pdf, "", cColTeam, names, min(cMaxFontSize, teamFontSize*scale))
	// The date column is fixed too: keep the date from overflowing it.
	fontSize = fitTextsUpTo(pdf, "", cColWhen-1, []string{"00.00.00 00:00"}, fontSize)

	rowY := y + boxPadding
	for i, g := range groups {
		if i > 0 {
			rowY += gap
		}
		first := g[0]
		pdf.SetFillColor(230, 230, 230)
		pdf.Rect(x+1, rowY, boxWidth-2, groupH, "F")
		heading := first.TeamType + " · " + first.Competition
		pdf.SetFont(fontFamily, "B", fitTextsUpTo(pdf, "B", boxWidth-2, []string{heading}, fontSize*0.9))
		pdf.SetXY(x, rowY)
		pdf.CellFormat(boxWidth, groupH, heading, "", 0, "C", false, 0, "")
		rowY += groupH

		for _, m := range g {
			when := m.Kickoff.Format("02.01.06")
			if m.HasTime {
				when += " " + m.Kickoff.Format("15:04")
			}
			result := ""
			if results {
				result = formatResult(m.Result)
			}
			pdf.SetFont(fontFamily, "", fontSize)
			pdf.SetXY(x+1, rowY)
			pdf.CellFormat(cColWhen-1, rowH, when, "", 0, "L", false, 0, "")
			pdf.CellFormat(cColTeam, rowH, m.Home, "", 0, "R", false, 0, "")
			pdf.CellFormat(colDash, rowH, "-", "", 0, "C", false, 0, "")
			pdf.CellFormat(cColTeam, rowH, m.Guest, "", 0, "L", false, 0, "")
			pdf.CellFormat(cColResult, rowH, result, "", 0, "C", false, 0, "")
			rowY += rowH
		}
	}
	return output(w, pdf)
}

// ClubMatchesFit reports whether matches fit a youth page at their natural
// size, without shrinking the rows.
func ClubMatchesFit(matches []bfv.ClubMatch) bool {
	return layoutClubPage(groupClubMatches(matches)).scale >= 1
}

// clubLayout is the scale and heights in millimetres a youth page is set
// with.
type clubLayout struct {
	scale, row, group, gap float64
}

// layoutClubPage scales rows, headers and gaps alike to fill the page; past
// cMaxStretch only the gaps grow, up to cMaxGap.
func layoutClubPage(groups [][]bfv.ClubMatch) clubLayout {
	rows := 0
	for _, g := range groups {
		rows += len(g)
	}
	gaps := float64(max(0, len(groups)-1))
	avail := clubPageHeight - 2*edge - 2*boxPadding
	natural := gaps*cGroupGap + float64(len(groups))*cGroupHeight + float64(rows)*cRowHeight

	scale := cMaxStretch
	if natural > 0 {
		scale = min(cMaxStretch, avail/natural)
	}
	l := clubLayout{scale, cRowHeight * scale, cGroupHeight * scale, cGroupGap * scale}
	if gaps > 0 {
		l.gap = min(cMaxGap, l.gap+(avail-natural*scale)/gaps)
	}
	return l
}

// groupClubMatches groups matches by age group and competition, keeping the
// order in which the groups first appear.
func groupClubMatches(matches []bfv.ClubMatch) [][]bfv.ClubMatch {
	index := map[[2]string]int{}
	var groups [][]bfv.ClubMatch
	for _, m := range matches {
		key := [2]string{m.TeamType, m.Competition}
		i, ok := index[key]
		if !ok {
			i = len(groups)
			index[key] = i
			groups = append(groups, nil)
		}
		groups[i] = append(groups[i], m)
	}
	return groups
}
