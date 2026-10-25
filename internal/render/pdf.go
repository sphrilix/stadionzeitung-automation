// Package render lays out league data (matchdays, tables, ...) as PDFs meant
// to be placed into a layout (e.g. an InDesign frame): each page is cropped
// to its content.
package render

import (
	"fmt"
	"io"
	"regexp"

	"github.com/go-pdf/fpdf"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"

	"github.com/sphrilix/stadionzeitung-automation/internal/bfv"
)

// Geometry in millimetres. Columns of a matchday box:
// date | home team (right-aligned) | "-" | guest team | result.
const (
	// edge keeps the box borders from being clipped at the page boundary.
	edge      = 0.5
	boxWidth  = 160.0
	colDate   = 20.0
	colDash   = 6.0
	colResult = 16.0
	colTeam   = (boxWidth - colDate - colDash - colResult) / 2

	headerHeight = 8.0
	rowHeight    = 6.0
	boxPadding   = 1.5
	boxGap       = 8.0

	fontFamily   = "Text"
	teamFontSize = 10.0
	minFontSize  = 6.0
)

var reScore = regexp.MustCompile(`^\s*(\d+)\s*:\s*(\d+)\s*$`)

// Fonts are TrueType files for regular and bold text. A nil font falls back
// to the embedded Go font.
type Fonts struct {
	Regular, Bold []byte
}

// Render writes a single-page PDF with one boxed table per matchday. The page
// is exactly as large as its content; the heading belongs to the layout.
func Render(w io.Writer, days []bfv.Matchday, fonts Fonts) error {
	height := 2 * edge
	for i, day := range days {
		if i > 0 {
			height += boxGap
		}
		height += boxHeight(day)
	}

	pdf, err := newPage(boxWidth+2*edge, height, fonts)
	if err != nil {
		return err
	}

	fontSize := fitFontSize(pdf, days)
	y := edge
	for _, day := range days {
		drawMatchday(pdf, edge, y, fontSize, day)
		y += boxHeight(day) + boxGap
	}
	return output(w, pdf)
}

// newPage starts a single-page PDF of the given size in millimetres with the
// text fonts loaded.
func newPage(width, height float64, fonts Fonts) (*fpdf.Fpdf, error) {
	pdf := fpdf.NewCustom(&fpdf.InitType{
		UnitStr: "mm",
		Size:    fpdf.SizeType{Wd: width, Ht: height},
	})
	pdf.SetMargins(0, 0, 0)
	pdf.SetAutoPageBreak(false, 0)
	pdf.AddUTF8FontFromBytes(fontFamily, "", orDefault(fonts.Regular, goregular.TTF))
	pdf.AddUTF8FontFromBytes(fontFamily, "B", orDefault(fonts.Bold, gobold.TTF))
	if err := pdf.Error(); err != nil {
		return nil, fmt.Errorf("loading fonts: %w", err)
	}
	pdf.AddPage()
	return pdf, nil
}

func output(w io.Writer, pdf *fpdf.Fpdf) error {
	if err := pdf.Error(); err != nil {
		return err
	}
	return pdf.Output(w)
}

func orDefault(font, fallback []byte) []byte {
	if font == nil {
		return fallback
	}
	return font
}

func boxHeight(day bfv.Matchday) float64 {
	return headerHeight + float64(len(day.Matches))*rowHeight + 2*boxPadding
}

func drawMatchday(pdf *fpdf.Fpdf, x, y, fontSize float64, day bfv.Matchday) {
	pdf.SetLineWidth(0.3)
	pdf.Rect(x, y, boxWidth, boxHeight(day), "D")
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
	var names []string
	for _, day := range days {
		for _, m := range day.Matches {
			names = append(names, m.Home, m.Guest)
		}
	}
	return fitTexts(pdf, "", colTeam, names)
}

// fitTexts returns the largest font size up to teamFontSize at which every
// text, set in style, fits a column of the given width.
func fitTexts(pdf *fpdf.Fpdf, style string, width float64, texts []string) float64 {
	pdf.SetFont(fontFamily, style, teamFontSize)
	widest := 0.0
	for _, t := range texts {
		widest = max(widest, pdf.GetStringWidth(t))
	}
	// Leave room for the cell margin fpdf adds on the aligned side.
	avail := width - 2*pdf.GetCellMargin()
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
