package bfv

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// PageInfo is the matchday structure extracted from a bfv.de competition page.
//
// bfv.de obfuscates team names, dates and results with a per-request font, so
// only the plain-text structure (matchday numbers and match IDs) is taken from
// the page; the readable details come from the widget API.
type PageInfo struct {
	LeagueName string
	// Matchday is the matchday the page shows; for the default page this is
	// BFV's current matchday, i.e. the last one played.
	Matchday    int
	MaxMatchday int
	// Ends maps each matchday to its scheduled last day (midnight, Berlin).
	Ends     map[int]time.Time
	MatchIDs []string
}

// RunningOrNext returns the matchday in progress at now or, between
// matchdays, the next one. A matchday counts as running through its last
// scheduled day; after the season it returns the final matchday. Without
// schedule data it falls back to the matchday shown on the page.
func (p PageInfo) RunningOrNext(now time.Time) int {
	if len(p.Ends) == 0 {
		return p.Matchday
	}
	y, m, d := now.In(berlin).Date()
	today := time.Date(y, m, d, 0, 0, 0, 0, berlin)
	for n := 1; n <= p.MaxMatchday; n++ {
		if end, ok := p.Ends[n]; ok && !end.Before(today) {
			return n
		}
	}
	return p.MaxMatchday
}

// The page embeds its data as escaped JSON inside Next.js flight scripts,
// hence the literal backslashes before every quote.
var (
	reTitle    = regexp.MustCompile(`<title>([^|<]+)`)
	reMatchday = regexp.MustCompile(`\\"matchDay\\":\\"(\d+)\\"`)
	reDayNum   = regexp.MustCompile(`\{\\"number\\":(\d+),\\"period\\":\[([^\]]*)\]`)
	reDate     = regexp.MustCompile(`\d{2}\.\d{2}\.\d{4}`)
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
	p.Ends = map[int]time.Time{}
	for _, m := range reDayNum.FindAllSubmatch(html, -1) {
		n, _ := strconv.Atoi(string(m[1]))
		p.MaxMatchday = max(p.MaxMatchday, n)
		// A period is ["17.07.","19.07.2026"]; only the last date has a year.
		if dates := reDate.FindAll(m[2], -1); len(dates) > 0 {
			if end, err := time.ParseInLocation("02.01.2006", string(dates[len(dates)-1]), berlin); err == nil {
				p.Ends[n] = end
			}
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
