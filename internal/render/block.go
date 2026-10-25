package render

import (
	"bytes"
	"image/jpeg"
	"io"
	"strings"

	"github.com/go-pdf/fpdf"
)

// block is one part of a page. Pages stack blocks top to bottom, so a page
// can combine e.g. a table and its scorers.
type block interface {
	// size returns the block's width and height in millimetres; pdf has
	// the text fonts loaded.
	size(pdf *fpdf.Fpdf) (w, h float64)
	draw(pdf *fpdf.Fpdf, x, y float64)
}

// blockGap separates stacked blocks.
const blockGap = 4.0

// renderBlocks writes a single-page PDF with the blocks stacked, the page
// as wide as the widest block and cropped to them.
func renderBlocks(w io.Writer, fonts Fonts, blocks ...block) error {
	// Measure with a throwaway page, as fpdf needs one to load the fonts.
	probe, err := newPage(10, 10, fonts)
	if err != nil {
		return err
	}
	width, height := 0.0, 2*edge
	heights := make([]float64, len(blocks))
	for i, b := range blocks {
		bw, bh := b.size(probe)
		width, heights[i] = max(width, bw), bh
		if i > 0 {
			height += blockGap
		}
		height += bh
	}

	pdf, err := newPage(width+2*edge, height, fonts)
	if err != nil {
		return err
	}
	y := edge
	for i, b := range blocks {
		b.draw(pdf, edge, y)
		y += heights[i] + blockGap
	}
	return output(w, pdf)
}

// linesBlock is lines of text, as wide as the longest.
type linesBlock []string

func (l linesBlock) size(pdf *fpdf.Fpdf) (float64, float64) {
	pdf.SetFont(fontFamily, "", textFontSize)
	width := 0.0
	for _, line := range l {
		width = max(width, pdf.GetStringWidth(line))
	}
	// Keep a line's height without lines (e.g. no goals yet), so the page
	// is never empty.
	return width, float64(max(1, len(l))) * textLine
}

func (l linesBlock) draw(pdf *fpdf.Fpdf, x, y float64) {
	width, _ := l.size(pdf)
	pdf.SetCellMargin(0)
	for i, line := range l {
		pdf.SetXY(x, y+float64(i)*textLine)
		pdf.CellFormat(width, textLine, line, "", 0, "L", false, 0, "")
	}
}

// rowsBlock is bold labels in a column as wide as the longest label and
// their text next to it, wrapped to rowsWidth, as the Kader pages are set.
// A text's line breaks are kept.
type rowsBlock []Row

func (r rowsBlock) layout(pdf *fpdf.Fpdf) (labelWidth float64, wrapped [][]string, height float64) {
	pdf.SetCellMargin(0)
	pdf.SetFont(fontFamily, "B", textFontSize)
	for _, row := range r {
		labelWidth = max(labelWidth, pdf.GetStringWidth(row.Label)+labelPad)
	}
	pdf.SetFont(fontFamily, "", textFontSize)
	wrapped = make([][]string, len(r))
	for i, row := range r {
		for _, para := range strings.Split(row.Text, "\n") {
			wrapped[i] = append(wrapped[i], wrap(pdf.SplitText(para, rowsWidth-labelWidth))...)
		}
		if row.Gap && i > 0 {
			height += rowGap
		}
		height += float64(len(wrapped[i])) * textLine
	}
	return labelWidth, wrapped, height
}

func (r rowsBlock) size(pdf *fpdf.Fpdf) (float64, float64) {
	_, _, h := r.layout(pdf)
	return rowsWidth, h
}

func (r rowsBlock) draw(pdf *fpdf.Fpdf, x, y float64) {
	labelWidth, wrapped, _ := r.layout(pdf)
	for i, row := range r {
		if row.Gap && i > 0 {
			y += rowGap
		}
		pdf.SetFont(fontFamily, "B", textFontSize)
		pdf.SetXY(x, y)
		pdf.CellFormat(labelWidth, textLine, row.Label, "", 0, "L", false, 0, "")
		pdf.SetFont(fontFamily, "", textFontSize)
		for _, line := range wrapped[i] {
			pdf.SetXY(x+labelWidth, y)
			pdf.CellFormat(rowsWidth-labelWidth, textLine, line, "", 0, "L", false, 0, "")
			y += textLine
		}
	}
}

// titleBlock is a centred bold headline, shrunk to fit rowsWidth.
type titleBlock string

const titleFontSize = 20.0

func (t titleBlock) fontSize(pdf *fpdf.Fpdf) float64 {
	pdf.SetFont(fontFamily, "B", titleFontSize)
	if w := pdf.GetStringWidth(string(t)); w > rowsWidth {
		return titleFontSize * rowsWidth / w
	}
	return titleFontSize
}

// height is the line height for the font size: points to millimetres, plus
// leading.
func (t titleBlock) height(size float64) float64 { return size * 25.4 / 72 * 1.3 }

func (t titleBlock) size(pdf *fpdf.Fpdf) (float64, float64) {
	return rowsWidth, t.height(t.fontSize(pdf))
}

func (t titleBlock) draw(pdf *fpdf.Fpdf, x, y float64) {
	size := t.fontSize(pdf)
	pdf.SetFont(fontFamily, "B", size)
	pdf.SetCellMargin(0)
	pdf.SetXY(x, y)
	pdf.CellFormat(rowsWidth, t.height(size), string(t), "", 0, "C", false, 0, "")
}

// photoBlock is a JPEG photo across rowsWidth with its credit below.
type photoBlock struct {
	jpeg          []byte
	width, height int
	credit        string
}

const creditFontSize = 7.0

// newPhotoBlock returns nil for data that is not a readable JPEG, so a bad
// download leaves the photo out instead of failing the page (fpdf errors
// are sticky).
func newPhotoBlock(data []byte, credit string) block {
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width == 0 || cfg.Height == 0 {
		return nil
	}
	return photoBlock{jpeg: data, width: cfg.Width, height: cfg.Height, credit: credit}
}

func (p photoBlock) imageHeight() float64 {
	return rowsWidth * float64(p.height) / float64(p.width)
}

func (p photoBlock) size(*fpdf.Fpdf) (float64, float64) {
	h := p.imageHeight()
	if p.credit != "" {
		h += creditFontSize * 25.4 / 72 * 1.5
	}
	return rowsWidth, h
}

func (p photoBlock) draw(pdf *fpdf.Fpdf, x, y float64) {
	opts := fpdf.ImageOptions{ImageType: "JPG"}
	pdf.RegisterImageOptionsReader("photo", opts, bytes.NewReader(p.jpeg))
	pdf.ImageOptions("photo", x, y, rowsWidth, p.imageHeight(), false, opts, 0, "")
	if p.credit != "" {
		pdf.SetFont(fontFamily, "", creditFontSize)
		pdf.SetCellMargin(0)
		pdf.SetXY(x, y+p.imageHeight())
		pdf.CellFormat(rowsWidth, creditFontSize*25.4/72*1.5, "Foto: "+p.credit, "", 0, "R", false, 0, "")
	}
}
