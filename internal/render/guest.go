package render

import "io"

// GuestPage is the Stadionzeitung's page on the visiting team.
type GuestPage struct {
	Name string
	// Photo is a JPEG team photo, PhotoCredit its photographer; without a
	// photo the page has none.
	Photo       []byte
	PhotoCredit string
	// Rows are the coaches, the squad by position and the top scorers.
	Rows []Row
}

// RenderGuestPage writes the guest page as one PDF: the team's name as
// headline, its photo, then its coaches, squad and scorers as labelled rows.
func RenderGuestPage(w io.Writer, g GuestPage, fonts Fonts) error {
	blocks := []block{titleBlock(g.Name)}
	if p := newPhotoBlock(g.Photo, g.PhotoCredit); p != nil {
		blocks = append(blocks, p)
	}
	blocks = append(blocks, rowsBlock(g.Rows))
	return renderBlocks(w, fonts, blocks...)
}
