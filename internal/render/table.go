package render

import (
	"bytes"
	"fmt"
	"image/png"
	"io"
	"strconv"
	"strings"

	"github.com/go-pdf/fpdf"

	"github.com/sphrilix/stadionzeitung-automation/internal/bfv"
)

// Columns of a table row: position | logo | team | played | won | drawn |
// lost | goals | goal difference | points.
const (
	tColPos    = 11.0
	tColLogo   = 7.0
	tColNum    = 8.0
	tColGoals  = 16.0
	tColDiff   = 13.0
	tColPoints = 10.0
	tColTeam   = boxWidth - tColPos - tColLogo - 4*tColNum - tColGoals - tColDiff - tColPoints

	tHeaderHeight = 7.0
	tRowHeight    = 6.5
	logoSize      = 5.0
)

// TableOptions tune a rendered table.
type TableOptions struct {
	// Logos maps club IDs to PNG logos; teams without one get none.
	Logos map[string][]byte
	// Highlight sets the rows of teams whose name contains it in bold, e.g.
	// the club's own teams.
	Highlight string
}

var zoneColors = map[bfv.Zone][3]int{
	bfv.ZonePromotion:         {0, 128, 60},
	bfv.ZonePromotionPlayoff:  {110, 180, 80},
	bfv.ZoneRelegationPlayoff: {240, 140, 0},
	bfv.ZoneRelegation:        {210, 30, 30},
}

// RenderTable writes a single-page PDF with a league table: one row per
// team, positions coloured by promotion and relegation zone.
func RenderTable(w io.Writer, t bfv.Table, fonts Fonts, opts TableOptions) error {
	height := 2*edge + tHeaderHeight + float64(len(t.Rows))*tRowHeight + 2*boxPadding
	pdf, err := newPage(boxWidth+2*edge, height, fonts)
	if err != nil {
		return err
	}
	logos := registerLogos(pdf, opts.Logos)

	x, y := edge, edge
	pdf.SetLineWidth(0.3)
	pdf.Rect(x, y, boxWidth, height-2*edge, "D")
	pdf.Line(x, y+tHeaderHeight, x+boxWidth, y+tHeaderHeight)

	pdf.SetFont(fontFamily, "B", 8)
	pdf.SetXY(x, y)
	pdf.CellFormat(tColPos, tHeaderHeight, "Pl.", "", 0, "C", false, 0, "")
	pdf.CellFormat(tColLogo+tColTeam, tHeaderHeight, "Verein", "", 0, "L", false, 0, "")
	for _, h := range []string{"Sp.", "G", "U", "V"} {
		pdf.CellFormat(tColNum, tHeaderHeight, h, "", 0, "C", false, 0, "")
	}
	pdf.CellFormat(tColGoals, tHeaderHeight, "Tore", "", 0, "C", false, 0, "")
	pdf.CellFormat(tColDiff, tHeaderHeight, "Diff.", "", 0, "C", false, 0, "")
	pdf.CellFormat(tColPoints, tHeaderHeight, "Pkt.", "", 0, "C", false, 0, "")

	var names []string
	for _, r := range t.Rows {
		names = append(names, r.Team)
	}
	// Bold is wider; size for it so highlighted rows fit too.
	fontSize := fitTexts(pdf, "B", tColTeam, names)

	rowY := y + tHeaderHeight + boxPadding
	pdf.SetLineWidth(0.1)
	pdf.SetDrawColor(190, 190, 190)
	for i, r := range t.Rows {
		if i > 0 {
			pdf.Line(x+1, rowY, x+boxWidth-1, rowY)
		}
		style := ""
		if opts.Highlight != "" && strings.Contains(r.Team, opts.Highlight) {
			style = "B"
		}

		pdf.SetXY(x, rowY)
		pos := strconv.Itoa(r.Position) + "."
		if c, ok := zoneColors[t.Zones.Of(i+1, len(t.Rows))]; ok {
			pdf.SetFillColor(c[0], c[1], c[2])
			pdf.Rect(x+1.5, rowY+0.75, tColPos-3, tRowHeight-1.5, "F")
			pdf.SetTextColor(255, 255, 255)
			pdf.SetFont(fontFamily, "B", fontSize)
			pdf.CellFormat(tColPos, tRowHeight, pos, "", 0, "C", false, 0, "")
			pdf.SetTextColor(0, 0, 0)
		} else {
			pdf.SetFont(fontFamily, style, fontSize)
			pdf.CellFormat(tColPos, tRowHeight, pos, "", 0, "C", false, 0, "")
		}

		if name, ok := logos[r.ClubID]; ok {
			pdf.ImageOptions(name, x+tColPos+(tColLogo-logoSize)/2, rowY+(tRowHeight-logoSize)/2,
				logoSize, logoSize, false, fpdf.ImageOptions{}, 0, "")
		}
		pdf.SetXY(x+tColPos+tColLogo, rowY)
		pdf.SetFont(fontFamily, style, fontSize)
		pdf.CellFormat(tColTeam, tRowHeight, r.Team, "", 0, "L", false, 0, "")
		for _, n := range []int{r.Played, r.Won, r.Drawn, r.Lost} {
			pdf.CellFormat(tColNum, tRowHeight, strconv.Itoa(n), "", 0, "C", false, 0, "")
		}
		goals := ""
		if r.HasGoals {
			goals = fmt.Sprintf("%d:%d", r.GoalsFor, r.GoalsAgainst)
		}
		pdf.CellFormat(tColGoals, tRowHeight, goals, "", 0, "C", false, 0, "")
		pdf.CellFormat(tColDiff, tRowHeight, strconv.Itoa(r.GoalDiff), "", 0, "C", false, 0, "")
		pdf.SetFont(fontFamily, "B", fontSize)
		pdf.CellFormat(tColPoints, tRowHeight, strconv.Itoa(r.Points), "", 0, "C", false, 0, "")
		rowY += tRowHeight
	}
	return output(w, pdf)
}

// registerLogos adds the logos to the PDF and returns their image names by
// club ID. A logo that is not a readable PNG is skipped; fpdf errors are
// sticky, so it is re-encoded in a form fpdf accepts before registering.
func registerLogos(pdf *fpdf.Fpdf, logos map[string][]byte) map[string]string {
	names := map[string]string{}
	for club, b := range logos {
		img, err := png.Decode(bytes.NewReader(b))
		if err != nil {
			continue
		}
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			continue
		}
		name := "logo-" + club
		pdf.RegisterImageOptionsReader(name, fpdf.ImageOptions{ImageType: "PNG"}, &buf)
		names[club] = name
	}
	return names
}
