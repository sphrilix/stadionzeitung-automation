// Package render lays out league matchdays as a printable PDF.
package render

import (
	"fmt"
	"io"
	"regexp"

	"github.com/go-pdf/fpdf"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"

	"github.com/sphrilix/bfv-miner/internal/bfv"
)

// Page geometry in millimetres (A4 portrait). Columns of a matchday box:
// date | home team (right-aligned) | "-" | guest team | result.
const (
	boxWidth  = 160.0
	colDate   = 20.0
	colDash   = 6.0
	colResult = 16.0
	colTeam   = (boxWidth - colDate - colDash - colResult) / 2

	headerHeight = 8.0
	rowHeight    = 6.0
	boxPadding   = 1.5
	boxGap       = 8.0

	fontFamily   = "Go"
	teamFontSize = 10.0
	minFontSize  = 6.0
)

var reScore = regexp.MustCompile(`^\s*(\d+)\s*:\s*(\d+)\s*$`)

// Render writes a PDF headed "Aktuelle Spieltage in der <league>" with one
// boxed table per matchday.
func Render(w io.Writer, league string, days []bfv.Matchday) error {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.AddUTF8FontFromBytes(fontFamily, "", goregular.TTF)
	pdf.AddUTF8FontFromBytes(fontFamily, "B", gobold.TTF)
	pdf.SetAutoPageBreak(false, 0)
	pdf.AddPage()

	pageW, pageH := pdf.GetPageSize()
	_, top, _, _ := pdf.GetMargins()
	left := (pageW - boxWidth) / 2

	pdf.SetFont(fontFamily, "B", 14)
	pdf.SetXY(left, top)
	pdf.CellFormat(boxWidth, 10, "Aktuelle Spieltage in der "+league, "", 1, "C", false, 0, "")
	pdf.SetY(pdf.GetY() + 4)

	fontSize := fitFontSize(pdf, days)
	for _, day := range days {
		height := headerHeight + float64(len(day.Matches))*rowHeight + 2*boxPadding
		if pdf.GetY()+height > pageH-top {
			pdf.AddPage()
			pdf.SetY(top)
		}
		y := pdf.GetY()
		drawMatchday(pdf, left, y, fontSize, day)
		pdf.SetY(y + height + boxGap)
	}

	if err := pdf.Error(); err != nil {
		return err
	}
	return pdf.Output(w)
}

func drawMatchday(pdf *fpdf.Fpdf, x, y, fontSize float64, day bfv.Matchday) {
	height := headerHeight + float64(len(day.Matches))*rowHeight + 2*boxPadding
	pdf.SetLineWidth(0.3)
	pdf.Rect(x, y, boxWidth, height, "D")
	pdf.Line(x, y+headerHeight, x+boxWidth, y+headerHeight)

	pdf.SetFont(fontFamily, "B", 12)
	pdf.SetXY(x, y)
	pdf.CellFormat(boxWidth, headerHeight, fmt.Sprintf("%d. Spieltag", day.Number), "", 0, "C", false, 0, "")

	pdf.SetFont(fontFamily, "", fontSize)
	rowY := y + headerHeight + boxPadding
	lastDate := ""
	for _, m := range day.Matches {
		date := ""
		if !m.Kickoff.IsZero() {
			date = m.Kickoff.Format("02.01.06")
		}
		shownDate := date
		if date == lastDate {
			shownDate = ""
		}
		lastDate = date

		pdf.SetXY(x+1, rowY)
		pdf.CellFormat(colDate-1, rowHeight, shownDate, "", 0, "L", false, 0, "")
		pdf.CellFormat(colTeam, rowHeight, m.Home, "", 0, "R", false, 0, "")
		pdf.CellFormat(colDash, rowHeight, "-", "", 0, "C", false, 0, "")
		pdf.CellFormat(colTeam, rowHeight, m.Guest, "", 0, "L", false, 0, "")
		pdf.CellFormat(colResult, rowHeight, formatResult(m.Result), "", 0, "C", false, 0, "")
		rowY += rowHeight
	}
}

// fitFontSize returns the largest font size up to teamFontSize at which every
// team name fits its column.
func fitFontSize(pdf *fpdf.Fpdf, days []bfv.Matchday) float64 {
	pdf.SetFont(fontFamily, "", teamFontSize)
	widest := 0.0
	for _, day := range days {
		for _, m := range day.Matches {
			widest = max(widest, pdf.GetStringWidth(m.Home), pdf.GetStringWidth(m.Guest))
		}
	}
	// Leave room for the cell margin fpdf adds on the aligned side.
	avail := colTeam - 2*pdf.GetCellMargin()
	if widest <= avail {
		return teamFontSize
	}
	return max(minFontSize, teamFontSize*avail/widest)
}

// formatResult turns "0:4" into "0 - 4", shows "-" for unplayed matches and
// passes anything else (e.g. "abgesetzt") through.
func formatResult(r string) string {
	if r == "" {
		return "-"
	}
	if m := reScore.FindStringSubmatch(r); m != nil {
		return m[1] + " - " + m[2]
	}
	return r
}
