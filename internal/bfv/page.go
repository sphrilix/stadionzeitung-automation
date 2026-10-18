package bfv

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// PageInfo is the matchday structure extracted from a bfv.de competition page.
//
// bfv.de obfuscates team names, dates and results with a per-request font, so
// only the plain-text structure (matchday numbers and match IDs) is taken from
// the page; the readable details come from the widget API.
type PageInfo struct {
	LeagueName string
	// Matchday is the matchday the page shows; for the default page this is
	// BFV's current matchday.
	Matchday    int
	MaxMatchday int
	MatchIDs    []string
}

// The page embeds its data as escaped JSON inside Next.js flight scripts,
// hence the literal backslashes before every quote.
var (
	reTitle    = regexp.MustCompile(`<title>([^|<]+)`)
	reMatchday = regexp.MustCompile(`\\"matchDay\\":\\"(\d+)\\"`)
	reDayNum   = regexp.MustCompile(`\{\\"number\\":(\d+),\\"period\\"`)
	reMatchID  = regexp.MustCompile(`\{\\"id\\":\\"([0-9A-Z]{32})\\",\\"status\\"`)
)

func parsePage(html []byte) (PageInfo, error) {
	var p PageInfo
	if m := reTitle.FindSubmatch(html); m != nil {
		p.LeagueName = strings.TrimSpace(string(m[1]))
	}
	m := reMatchday.FindSubmatch(html)
	if m == nil {
		return p, errors.New("no matchday found on page (not a league competition?)")
	}
	p.Matchday, _ = strconv.Atoi(string(m[1]))
	for _, m := range reDayNum.FindAllSubmatch(html, -1) {
		if n, _ := strconv.Atoi(string(m[1])); n > p.MaxMatchday {
			p.MaxMatchday = n
		}
	}
	if p.MaxMatchday < p.Matchday {
		p.MaxMatchday = p.Matchday
	}
	seen := map[string]bool{}
	for _, m := range reMatchID.FindAllSubmatch(html, -1) {
		id := string(m[1])
		if !seen[id] {
			seen[id] = true
			p.MatchIDs = append(p.MatchIDs, id)
		}
	}
	return p, nil
}

// FetchPage loads the bfv.de page of a competition for matchday n; n == 0
// loads the default page, which shows the current matchday.
func (c *Client) FetchPage(ctx context.Context, compoundID string, n int) (PageInfo, error) {
	url := fmt.Sprintf("%s/ergebnisse/wettbewerb/-/%s", c.BaseWeb, compoundID)
	if n > 0 {
		url += fmt.Sprintf("/spieltag/%d", n)
	}
	body, err := c.get(ctx, url)
	if err != nil {
		return PageInfo{}, err
	}
	p, err := parsePage(body)
	if err != nil {
		return p, fmt.Errorf("%s: %w", url, err)
	}
	return p, nil
}
