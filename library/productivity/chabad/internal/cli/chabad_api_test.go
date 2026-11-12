// Copyright 2026 Bob Benjy and contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "chabad", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return b
}

func TestParseZmanimDetailed(t *testing.T) {
	res, err := parseZmanimResponse(fixture(t, "zmanim_detailed.json"), false)
	if err != nil {
		t.Fatal(err)
	}
	if res.Location != "Brooklyn, NY 11213" || len(res.Days) != 3 {
		t.Fatalf("location/days = %q/%d", res.Location, len(res.Days))
	}
	fri := res.Days[0]
	if fri.Date != "2026-12-04" || fri.Weekday != "Friday" || fri.Hebrew != "Kislev 24, 5787" {
		t.Fatalf("friday = %+v", fri)
	}
	cl, ok := fri.find("CandleLighting")
	if !ok || cl.Time != "4:11 PM" || cl.Slug != "candle-lighting" {
		t.Fatalf("candle lighting = %+v ok=%v", cl, ok)
	}
	if cl.At == nil || cl.At.UTC().Format("2006-01-02 15:04") != "2026-12-04 21:11" {
		t.Fatalf("candle lighting instant = %v", cl.At)
	}
	sat := res.Days[1]
	if sat.Reading != "Vayeishev" || sat.Holiday != "Chanukah Vayeishev" || !sat.IsHoliday {
		t.Fatalf("shabbat chanukah = %+v", sat)
	}
	if _, ok := sat.find("CandleLighting"); ok {
		t.Fatalf("Saturday should have no candle lighting")
	}
	// one entry per type when not asking for all opinions
	seen := map[string]bool{}
	for _, z := range fri.Zmanim {
		if seen[z.Type] {
			t.Fatalf("duplicate zman type %s", z.Type)
		}
		seen[z.Type] = true
	}
	// midnight after 12 AM rolls into the next day
	if mid, ok := fri.find("ChatzosNight"); ok && mid.At != nil && mid.At.Before(*cl.At) {
		t.Fatalf("midnight %v before candle lighting %v", mid.At, cl.At)
	}
}

func TestParseZmanimCompactShape(t *testing.T) {
	res, err := parseZmanimResponse(fixture(t, "zmanim_compact.json"), false)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Days) != 3 {
		t.Fatalf("days = %d", len(res.Days))
	}
	fast := res.Days[1]
	if fast.Holiday != "Fast - 10 Tevet" {
		t.Fatalf("fast day = %+v", fast)
	}
	dawn, ok1 := fast.find("AlosHashachar")
	night, ok2 := fast.find("Tzeis")
	if !ok1 || !ok2 || dawn.Time != "5:42 AM" || night.Time != "5:03 PM" {
		t.Fatalf("fast times dawn=%+v night=%+v", dawn, night)
	}
	if dawn.Title == "" || dawn.Slug != "dawn" {
		t.Fatalf("compact entries need titles/slugs: %+v", dawn)
	}
}

func TestParseCandlesPesach(t *testing.T) {
	res, err := parseCandleResponse(fixture(t, "candles_pesach.json"))
	if err != nil {
		t.Fatal(err)
	}
	if res.CandleOffset != 18 || len(res.Groups) < 3 {
		t.Fatalf("offset/groups = %d/%d", res.CandleOffset, len(res.Groups))
	}
	if res.Groups[0].Parsha != "Metzora" || res.Groups[0].Type != "shabbat" {
		t.Fatalf("first group = %+v", res.Groups[0])
	}
	var pesach *candleGroup
	for i := range res.Groups {
		if res.Groups[i].Title == "Passover" {
			pesach = &res.Groups[i]
			break
		}
	}
	if pesach == nil || len(pesach.Events) < 2 {
		t.Fatalf("passover group missing: %+v", res.Groups)
	}
	second := pesach.Events[1]
	if !second.FromPreexistingFlame || second.Kind != "candle-lighting" || second.Date != "2027-04-22" {
		t.Fatalf("second-night candles = %+v", second)
	}
	if pesach.Events[0].At == nil {
		t.Fatalf("candle instants should be parsed")
	}
}

func TestParseStudyRSS(t *testing.T) {
	day, err := parseStudyRSS(fixture(t, "study.xml"), time.Date(2026, 12, 25, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if day.Hebrew != "Tevet 15, 5787" {
		t.Fatalf("hebrew = %q", day.Hebrew)
	}
	tracks := map[string]studyItem{}
	for _, it := range day.Items {
		tracks[it.Track] = it
	}
	for _, want := range studyTrackNames {
		if _, ok := tracks[want]; !ok {
			t.Fatalf("missing track %s in %+v", want, day.Items)
		}
	}
	if tracks["tehillim"].Lesson != "Chapters 77-78" {
		t.Fatalf("tehillim lesson = %q", tracks["tehillim"].Lesson)
	}
	if len(day.Items) != len(studyTrackNames) {
		t.Fatalf("hebrew tehillim duplicate not dropped: %d items", len(day.Items))
	}
}

func TestResolveZmanType(t *testing.T) {
	for in, want := range map[string]string{"candle lighting": "CandleLighting", "havdalah": "ShabbatEndTime", "Shkiah": "Shkiah", "tzeis": "Tzeis", "sunrise": "NetzHachamah"} {
		got, ok := resolveZmanType(in)
		if !ok || got != want {
			t.Errorf("resolveZmanType(%q) = %q, %v", in, got, ok)
		}
	}
	if _, ok := resolveZmanType("lunch"); ok {
		t.Fatalf("unknown zman resolved")
	}
}

func TestStripHTML(t *testing.T) {
	got := stripHTML("Candle Lighting is 18 mins. before sunset&nbsp; | &nbsp;-5 GMT")
	if got != "Candle Lighting is 18 mins. before sunset | -5 GMT" {
		t.Fatalf("stripHTML = %q", got)
	}
}
