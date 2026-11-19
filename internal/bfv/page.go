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
	// Starts and Ends map each matchday to its scheduled first and last
	// day (midnight, Berlin).
	Starts, Ends map[int]time.Time
	// MatchIDs are the matches the page lists. Besides the matchday's own,
	// they include matches moved there from another matchday, and matches
	// of this one moved to another date; Rescheduled marks the moved ones.
	MatchIDs    []string
	Rescheduled map[string]bool
}

// InPeriod reports whether t falls on a scheduled day of matchday n. Without
// schedule data every time does.
func (p PageInfo) InPeriod(n int, t time.Time) bool {
	start, ok1 := p.Starts[n]
	end, ok2 := p.Ends[n]
	if !ok1 || !ok2 {
		return true
	}
	y, m, d := t.In(berlin).Date()
	day := time.Date(y, m, d, 0, 0, 0, 0, berlin)
	return !day.Before(start) && !day.After(end)
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
	// A moved match carries its original kickoff, e.g.
	// "rescheduled":"$D2026-07-31T16:30:00.000Z"; others "$undefined".
	reRescheduled = regexp.MustCompile(`\\"rescheduled\\":\\"\$D`)
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
	p.Starts, p.Ends, p.Rescheduled = map[int]time.Time{}, map[int]time.Time{}, map[string]bool{}
	for _, m := range reDayNum.FindAllSubmatch(html, -1) {
		n, _ := strconv.Atoi(string(m[1]))
		p.MaxMatchday = max(p.MaxMatchday, n)
		// A period is ["17.07.","19.07.2026"]; only the last date has a year.
		if dates := reDate.FindAll(m[2], -1); len(dates) > 0 {
			if end, err := time.ParseInLocation("02.01.2006", string(dates[len(dates)-1]), berlin); err == nil {
				p.Ends[n] = end
				p.Starts[n] = periodStart(m[2], end)
			}
		}
	}
	if p.MaxMatchday < p.Matchday {
		p.MaxMatchday = p.Matchday
	}
	seen := map[string]bool{}
	locs := reMatchID.FindAllSubmatchIndex(html, -1)
	for i, loc := range locs {
		id := string(html[loc[2]:loc[3]])
		// A match's fields run up to the next match.
		end := len(html)
		if i+1 < len(locs) {
			end = locs[i+1][0]
		}
		if reRescheduled.Match(html[loc[1]:end]) {
			p.Rescheduled[id] = true
		}
		if !seen[id] {
			seen[id] = true
			p.MatchIDs = append(p.MatchIDs, id)
		}
	}
	return p, nil
}

// reDayMonth is a period's first day, given without the year: "17.07.".
var reDayMonth = regexp.MustCompile(`(\d{2})\.(\d{2})\.`)

// periodStart returns the first day of a matchday period ending on end, e.g.
// ["17.07.","19.07.2026"]; a period over New Year starts the year before.
// Without a readable first day the period is the last day alone.
func periodStart(period []byte, end time.Time) time.Time {
	m := reDayMonth.FindSubmatch(period)
	if m == nil {
		return end
	}
	day, _ := strconv.Atoi(string(m[1]))
	month, _ := strconv.Atoi(string(m[2]))
	start := time.Date(end.Year(), time.Month(month), day, 0, 0, 0, 0, berlin)
	if start.After(end) {
		start = start.AddDate(-1, 0, 0)
	}
	return start
}

// reMatchPageDay is a match page's matchday: "matchday":3.
var reMatchPageDay = regexp.MustCompile(`\\"matchday\\":(\d+)`)

// MatchMatchday returns the matchday a match belongs to, as its bfv.de page
// gives it, wherever it was moved to.
func (c *Client) MatchMatchday(ctx context.Context, matchID string) (int, error) {
	url := c.BaseWeb + "/ergebnisse/spiel/-/" + matchID
	page, err := c.get(ctx, url)
	if err != nil {
		return 0, err
	}
	m := reMatchPageDay.FindSubmatch(page)
	if m == nil {
		return 0, fmt.Errorf("%s: no matchday found", url)
	}
	return strconv.Atoi(string(m[1]))
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
