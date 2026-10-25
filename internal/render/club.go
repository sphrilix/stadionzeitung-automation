package render

import (
	"io"

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
)

// RenderClubMatches writes a single-page PDF with a club's matches, one
// block per age group and competition, e.g. "A-Junioren · U 19 (A-Jun.) BOL
// Niederbayern". Without results the result column shows nothing, for a
// preview of upcoming matches.
func RenderClubMatches(w io.Writer, matches []bfv.ClubMatch, results bool, fonts Fonts) error {
	groups := groupClubMatches(matches)
	height := 2*edge + 2*boxPadding
	for i, g := range groups {
		if i > 0 {
			height += cGroupGap
		}
		height += cGroupHeight + float64(len(g))*cRowHeight
	}
	pdf, err := newPage(boxWidth+2*edge, height, fonts)
	if err != nil {
		return err
	}

	x, y := edge, edge
	pdf.SetLineWidth(0.3)
	pdf.Rect(x, y, boxWidth, height-2*edge, "D")

	var names []string
	for _, m := range matches {
		names = append(names, m.Home, m.Guest)
	}
	fontSize := fitTexts(pdf, "", cColTeam, names)

	rowY := y + boxPadding
	for i, g := range groups {
		if i > 0 {
			rowY += cGroupGap
		}
		first := g[0]
		pdf.SetFillColor(230, 230, 230)
		pdf.Rect(x+1, rowY, boxWidth-2, cGroupHeight, "F")
		pdf.SetFont(fontFamily, "B", min(fontSize, 9))
		pdf.SetXY(x, rowY)
		pdf.CellFormat(boxWidth, cGroupHeight, first.TeamType+" · "+first.Competition, "", 0, "C", false, 0, "")
		rowY += cGroupHeight

		pdf.SetFont(fontFamily, "", fontSize)
		for _, m := range g {
			when := m.Kickoff.Format("02.01.06")
			if m.HasTime {
				when += " " + m.Kickoff.Format("15:04")
			}
			result := ""
			if results {
				result = formatResult(m.Result)
			}
			pdf.SetXY(x+1, rowY)
			pdf.CellFormat(cColWhen-1, cRowHeight, when, "", 0, "L", false, 0, "")
			pdf.CellFormat(cColTeam, cRowHeight, m.Home, "", 0, "R", false, 0, "")
			pdf.CellFormat(colDash, cRowHeight, "-", "", 0, "C", false, 0, "")
			pdf.CellFormat(cColTeam, cRowHeight, m.Guest, "", 0, "L", false, 0, "")
			pdf.CellFormat(cColResult, cRowHeight, result, "", 0, "C", false, 0, "")
			rowY += cRowHeight
		}
	}
	return output(w, pdf)
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
