package bfv

import (
	"context"
	"testing"
)

func TestFetchTable(t *testing.T) {
	c := serve(t, map[string]string{
		"/widget/competition/" + testCID + "/table": `{"state":0,"data":{
			"competitionName":"Test Liga",
			"configuration":{"promotionTeamCount":1,"promotionPlayoffTeamCount":0,"relegationPlayoffTeamCount":0,"relegationTeamCount":1},
			"table":[
			{"position":1,"matches":2,"matchesWon":2,"matchesDrawn":0,"matchesLost":0,"goalsDiff":4,"points":6,"team":{"permanentId":"A","name":"Alpha"}},
			{"position":2,"matches":1,"matchesWon":0,"matchesDrawn":0,"matchesLost":1,"goalsDiff":-2,"points":0,"team":{"permanentId":"B","name":"Bravo"}},
			{"position":3,"matches":1,"matchesWon":0,"matchesDrawn":0,"matchesLost":1,"goalsDiff":-3,"points":0,"team":{"permanentId":"C","name":"Charlie"}}]}}`,
		"/widget/team/A/matches": `{"state":0,"data":{"matches":[
			{"matchId":"M1","compoundId":"` + testCID + `","result":"3:1","homeTeamPermanentId":"A","homeClubId":"CA","guestTeamPermanentId":"C","guestClubId":"CC"},
			{"matchId":"M2","compoundId":"` + testCID + `","result":"2:0","homeTeamPermanentId":"A","homeClubId":"CA","guestTeamPermanentId":"B","guestClubId":"CB","guestLogoPrivate":true},
			{"matchId":"F","compoundId":"OTHER","result":"9:0","homeTeamPermanentId":"A","guestTeamPermanentId":"Z"}]}}`,
		"/widget/team/B/matches": `{"state":0,"data":{"matches":[
			{"matchId":"M2","compoundId":"` + testCID + `","result":"2:0","homeTeamPermanentId":"A","homeClubId":"CA","guestTeamPermanentId":"B","guestClubId":"CB","guestLogoPrivate":true},
			{"matchId":"M3","compoundId":"` + testCID + `","result":"","homeTeamPermanentId":"B","homeClubId":"CB","homeLogoPrivate":true,"guestTeamPermanentId":"C","guestClubId":"CC"}]}}`,
		"/widget/team/C/matches": `{"state":0,"data":{"matches":[
			{"matchId":"M1","compoundId":"` + testCID + `","result":"3:1","homeTeamPermanentId":"A","homeClubId":"CA","guestTeamPermanentId":"C","guestClubId":"CC"}]}}`,
	})
	tbl, err := c.FetchTable(context.Background(), testCID)
	if err != nil {
		t.Fatal(err)
	}
	if tbl.League != "Test Liga" || tbl.Zones != (Zones{Promotion: 1, Relegation: 1}) || len(tbl.Rows) != 3 {
		t.Fatalf("Table = %+v", tbl)
	}
	a, b, ch := tbl.Rows[0], tbl.Rows[1], tbl.Rows[2]
	// The friendly against Z does not count.
	if a.Team != "Alpha" || !a.HasGoals || a.GoalsFor != 5 || a.GoalsAgainst != 1 || a.Points != 6 || a.ClubID != "CA" {
		t.Errorf("Alpha = %+v, want 5:1 goals, 6 points, club CA", a)
	}
	// Bravo's logo is private.
	if !b.HasGoals || b.GoalsFor != 0 || b.GoalsAgainst != 2 || b.ClubID != "" {
		t.Errorf("Bravo = %+v, want 0:2 goals and no club logo", b)
	}
	// 1:3 does not add up to the published difference of -3 (say, a match
	// awarded by the association), so the goals are not shown.
	if ch.HasGoals || ch.GoalDiff != -3 {
		t.Errorf("Charlie = %+v, want no goals, difference -3", ch)
	}
}

func TestZonesOf(t *testing.T) {
	z := Zones{Promotion: 1, PromotionPlayoff: 1, RelegationPlayoff: 3, Relegation: 2}
	want := map[int]Zone{1: ZonePromotion, 2: ZonePromotionPlayoff, 3: ZoneNone, 13: ZoneNone,
		14: ZoneRelegationPlayoff, 16: ZoneRelegationPlayoff, 17: ZoneRelegation, 18: ZoneRelegation}
	for pos, zone := range want {
		if got := z.Of(pos, 18); got != zone {
			t.Errorf("Of(%d, 18) = %v, want %v", pos, got, zone)
		}
	}
}
