package render

import (
	"io"
	"strconv"

	"github.com/go-pdf/fpdf"

	"github.com/sphrilix/stadionzeitung-automation/internal/bfv"
)

// Columns of a season row: matchday | date | kickoff | home team
// (right-aligned) | "-" | guest team | result.
const (
	sColDay    = 10.0
	sColDate   = 20.0
	sColTime   = 14.0
	sColResult = 16.0
	sColTeam   = (boxWidth - sColDay - sColDate - sColTime - colDash - sColResult) / 2

	sRowHeight = 5.5
)

// RenderSeason writes a single-page PDF with a team's season fixtures, one
// row per matchday, byes included.
func RenderSeason(w io.Writer, s bfv.Season, fonts Fonts) error {
	height := 2*edge + tHeaderHeight + float64(len(s.Matches))*sRowHeight + 2*boxPadding
	pdf, err := newPage(boxWidth+2*edge, height, fonts)
	if err != nil {
		return err
	}

	x, y := edge, edge
	pdf.SetLineWidth(0.3)
	pdf.Rect(x, y, boxWidth, height-2*edge, "D")
	pdf.Line(x, y+tHeaderHeight, x+boxWidth, y+tHeaderHeight)

	pdf.SetFont(fontFamily, "B", 8)
	pdf.SetXY(x, y)
	pdf.CellFormat(sColDay, tHeaderHeight, "SpT.", "", 0, "C", false, 0, "")
	pdf.CellFormat(sColDate, tHeaderHeight, "Datum", "", 0, "C", false, 0, "")
	pdf.CellFormat(sColTime, tHeaderHeight, "Anstoß", "", 0, "C", false, 0, "")
	pdf.CellFormat(2*sColTeam+colDash, tHeaderHeight, "Spielpaarung", "", 0, "C", false, 0, "")
	pdf.CellFormat(sColResult, tHeaderHeight, "Ergebnis", "", 0, "C", false, 0, "")

	var names []string
	for _, m := range s.Matches {
		names = append(names, m.Home, m.Guest)
	}
	fontSize := fitTexts(pdf, "", sColTeam, names)

	rowY := y + tHeaderHeight + boxPadding
	pdf.SetLineWidth(0.1)
	pdf.SetDrawColor(190, 190, 190)
	pdf.SetFont(fontFamily, "", fontSize)
	for i, m := range s.Matches {
		if i > 0 {
			pdf.Line(x+1, rowY, x+boxWidth-1, rowY)
		}
		drawSeasonRow(pdf, x, rowY, m)
		rowY += sRowHeight
	}
	return output(w, pdf)
}

func drawSeasonRow(pdf *fpdf.Fpdf, x, y float64, m bfv.SeasonMatch) {
	day, date, kickoff, result := "", "", "", formatResult(m.Result)
	if m.Matchday > 0 {
		day = strconv.Itoa(m.Matchday)
	}
	// A bye has a date on BFV but no match, so the layout leaves it blank.
	if m.IsBye() {
		result = ""
	} else if !m.Kickoff.IsZero() {
		date = m.Kickoff.Format("02.01.2006")
		if m.HasTime {
			kickoff = m.Kickoff.Format("15:04")
		}
	}
	pdf.SetXY(x, y)
	pdf.CellFormat(sColDay, sRowHeight, day, "", 0, "C", false, 0, "")
	pdf.CellFormat(sColDate, sRowHeight, date, "", 0, "C", false, 0, "")
	pdf.CellFormat(sColTime, sRowHeight, kickoff, "", 0, "C", false, 0, "")
	pdf.CellFormat(sColTeam, sRowHeight, m.Home, "", 0, "R", false, 0, "")
	pdf.CellFormat(colDash, sRowHeight, "-", "", 0, "C", false, 0, "")
	pdf.CellFormat(sColTeam, sRowHeight, m.Guest, "", 0, "L", false, 0, "")
	pdf.CellFormat(sColResult, sRowHeight, result, "", 0, "C", false, 0, "")
}
