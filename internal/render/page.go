package render

import "io"

// TeamPage is the Stadionzeitung's page on a team: the guest's, or one of
// the club's own teams.
type TeamPage struct {
	Name string
	// Photo is a JPEG team photo, PhotoCredit its photographer; without a
	// photo the page has none.
	Photo       []byte
	PhotoCredit string
	// Rows are the coaches, the squad by position and the top scorers.
	Rows []Row
}

// RenderTeamPage writes a team's page as one PDF: the team's name as
// headline, its photo, then its staff, squad and scorers as labelled rows.
func RenderTeamPage(w io.Writer, g TeamPage, fonts Fonts) error {
	blocks := []block{titleBlock(g.Name)}
	if p := newPhotoBlock(g.Photo, g.PhotoCredit); p != nil {
		blocks = append(blocks, p)
	}
	blocks = append(blocks, rowsBlock(g.Rows))
	return renderBlocks(w, fonts, blocks...)
}
