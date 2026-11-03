package bfv

import (
	"context"
	"errors"
	"fmt"
	"html"
	"regexp"
	"strings"

	"golang.org/x/image/font/sfnt"
)

// Officials are the referee and the assistant referees of a match.
type Officials struct {
	Referee    string
	Assistants []string
}

// ErrNoReferee is returned while BFV has no referee for a match yet.
var ErrNoReferee = errors.New("BFV has no referee for the match yet")

// The match page lists the officials in its footer, each name in a
// value-text span set in the footer's obfuscation font.
var (
	reFooterFont = regexp.MustCompile(`StageMatchOverviewFooter_container[^"]*" style="--value-text-font:font-([0-9a-z]+)"`)
	reOfficials  = regexp.MustCompile(`StageMatchOverviewFooter_(referee|assistants)__[^"]*">(.*?)</div>`)
	reValueText  = regexp.MustCompile(`<span class="value-text">([^<]*)</span>`)
)

// MatchOfficials returns the referee and assistants of a match, as its
// bfv.de page lists them.
func (c *Client) MatchOfficials(ctx context.Context, matchID string) (Officials, error) {
	url := c.BaseWeb + "/ergebnisse/spiel/-/" + matchID
	page, err := c.get(ctx, url)
	if err != nil {
		return Officials{}, err
	}
	m := reFooterFont.FindSubmatch(page)
	if m == nil {
		return Officials{}, fmt.Errorf("%s: no match details found", url)
	}
	d, err := c.fetchDecoder(ctx, string(m[1]))
	if err != nil {
		return Officials{}, err
	}
	var o Officials
	// The footer appears twice, for wide and narrow screens; each role is
	// read from its first appearance.
	seen := map[string]bool{}
	for _, sec := range reOfficials.FindAllSubmatch(page, -1) {
		role := string(sec[1])
		if seen[role] {
			continue
		}
		seen[role] = true
		for _, v := range reValueText.FindAllSubmatch(sec[2], -1) {
			name, err := d.decode(html.UnescapeString(string(v[1])))
			if err != nil {
				return Officials{}, fmt.Errorf("%s: %w", url, err)
			}
			if name = strings.Join(strings.Fields(name), " "); name == "" {
				continue
			}
			if role == "referee" {
				o.Referee = name
			} else {
				o.Assistants = append(o.Assistants, name)
			}
		}
	}
	if o.Referee == "" {
		return Officials{}, ErrNoReferee
	}
	return o, nil
}

// decoder reads text obfuscated with a bfv.de font. The font draws each
// character of the page with the glyph of another one, and names that
// glyph after the character it shows (Adobe glyph names, e.g. "adieresis").
type decoder struct {
	font *sfnt.Font
	buf  sfnt.Buffer
}

func (c *Client) fetchDecoder(ctx context.Context, fontID string) (*decoder, error) {
	b, err := c.get(ctx, c.BaseFont+fontID+"/type/font")
	if err != nil {
		return nil, err
	}
	f, err := sfnt.Parse(b)
	if err != nil {
		return nil, fmt.Errorf("font %s: %w", fontID, err)
	}
	return &decoder{font: f}, nil
}

func (d *decoder) decode(s string) (string, error) {
	var b strings.Builder
	for _, r := range s {
		gi, err := d.font.GlyphIndex(&d.buf, r)
		if err != nil {
			return "", err
		}
		if gi == 0 {
			// Not in the font, so not obfuscated, e.g. the spaces.
			b.WriteRune(r)
			continue
		}
		name, err := d.font.GlyphName(&d.buf, gi)
		if err != nil {
			return "", err
		}
		t, ok := glyphText(name)
		if !ok {
			return "", fmt.Errorf("unknown glyph %q", name)
		}
		b.WriteString(t)
	}
	return b.String(), nil
}
