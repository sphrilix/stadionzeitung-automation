package bfv

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"golang.org/x/image/font/gofont/goregular"
)

// matchPageHTML mimics the footer of a bfv.de match page, wide and narrow.
// The real page writes each name in private-use characters that only its
// font maps to glyphs; the Go font used in the tests maps the plain ones,
// which the decoder treats alike.
func matchPageHTML(referee string, assistants ...string) string {
	value := func(s string) string {
		return fmt.Sprintf(`<span class="Paragraph_paragraph__XHf1x"><span class="value-text">%s</span></span>`, s)
	}
	footer := `<div class="StageMatchOverviewFooter_referee__CVDfG StageMatchOverviewFooter_border__BCWER"><span class="Paragraph_bold__UYq2g">Schiedsrichter</span>` + value(referee) + `</div>` +
		`<div class="StageMatchOverviewFooter_assistants__fvBS2 StageMatchOverviewFooter_border__BCWER"><span class="Paragraph_bold__UYq2g">Assistenten</span>`
	for _, a := range assistants {
		footer += value(a)
	}
	footer += `</div><div class="StageMatchOverviewFooter_spectators__5uA60">` + value("150") + `</div>`
	return `<div class="StageMatchOverviewFooter_container__PzuIS hide-below-lg" style="--value-text-font:font-testfont">` + footer + `</div>` +
		`<div class="StageMatchOverviewFooter_collapsibleContainer__LnsTA hide-above-lg" style="--value-text-font:font-testfont">` + footer + `</div>`
}

func TestMatchOfficials(t *testing.T) {
	c := serve(t, map[string]string{
		"/ergebnisse/spiel/-/M1":   matchPageHTML(" Marijo Kraljić ", "Moritz Breitruck", "Dominik Von Maffei"),
		"/ergebnisse/spiel/-/M2":   matchPageHTML("Lothar Ostheimer", " ", "  "),
		"/ergebnisse/spiel/-/M3":   matchPageHTML(" ", " ", "  "),
		"/font/testfont/type/font": string(goregular.TTF),
	})
	o, err := c.MatchOfficials(context.Background(), "M1")
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%q", o); got != `{"Marijo Kraljić" ["Moritz Breitruck" "Dominik Von Maffei"]}` {
		t.Errorf("MatchOfficials(M1) = %s", got)
	}
	// Assistants are not always appointed.
	if o, err := c.MatchOfficials(context.Background(), "M2"); err != nil || o.Referee != "Lothar Ostheimer" || len(o.Assistants) != 0 {
		t.Errorf("MatchOfficials(M2) = %q, %v", o, err)
	}
	if _, err := c.MatchOfficials(context.Background(), "M3"); !errors.Is(err, ErrNoReferee) {
		t.Errorf("MatchOfficials(M3): err = %v, want ErrNoReferee", err)
	}
}

func TestGlyphText(t *testing.T) {
	for name, want := range map[string]string{
		"M": "M", "adieresis": "ä", "germandbls": "ß", "cacute": "ć", "Lslash": "Ł",
		"one": "1", "hyphen": "-", "backslash": `\`, "uni0326": "̦",
	} {
		if got, ok := glyphText(name); !ok || got != want {
			t.Errorf("glyphText(%q) = %q, %v; want %q", name, got, ok, want)
		}
	}
	if _, ok := glyphText("notaglyph"); ok {
		t.Error("glyphText(notaglyph): want not ok")
	}
}
