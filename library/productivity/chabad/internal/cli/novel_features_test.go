// Copyright 2026 Bob Benjy and contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/mvanhorn/printing-press-library/library/productivity/chabad/internal/store"
)

func findEvent(t *testing.T, evs []aheadEvent, name, start string) aheadEvent {
	t.Helper()
	for _, e := range evs {
		if e.Name == name && e.Start == start {
			return e
		}
	}
	t.Fatalf("event %q starting %s not found in %+v", name, start, evs)
	return aheadEvent{}
}

func timeLabeled(e aheadEvent, label string) (aheadTime, bool) {
	for _, tm := range e.Times {
		if tm.Label == label {
			return tm, true
		}
	}
	return aheadTime{}, false
}

func TestAheadPesach(t *testing.T) {
	res, err := parseZmanimResponse(fixture(t, "zmanim_pesach.json"), false)
	if err != nil {
		t.Fatal(err)
	}
	evs := buildAheadEvents(res.Days)

	prep := findEvent(t, evs, "Erev Pesach preparations", "2027-04-20")
	if tm, ok := timeLabeled(prep, "Search for chametz (tonight)"); !ok || tm.Time != "8:09 PM" {
		t.Fatalf("bedikat chametz = %+v", prep.Times)
	}
	erev := findEvent(t, evs, "Eve of First day Passover", "2027-04-21")
	for label, want := range map[string]string{"Last time to eat chametz": "10:37 AM", "Burn chametz by": "11:46 AM", "Candle lighting": "7:23 PM"} {
		if tm, ok := timeLabeled(erev, label); !ok || tm.Time != want {
			t.Fatalf("erev pesach %s = %+v (all %+v)", label, tm, erev.Times)
		}
	}
	first := findEvent(t, evs, "Passover", "2027-04-22")
	if first.End != "2027-04-23" {
		t.Fatalf("first days end %s", first.End)
	}
	if tm, ok := timeLabeled(first, "Light candles after (from a pre-existing flame)"); !ok || tm.Time != "8:26 PM" || tm.Date != "2027-04-22" {
		t.Fatalf("second night = %+v", first.Times)
	}
	if tm, ok := timeLabeled(first, "Shabbat candle lighting (from a pre-existing flame)"); !ok || tm.Time != "7:25 PM" {
		t.Fatalf("shabbat chol hamoed candles from yom tov = %+v", first.Times)
	}
	if _, ok := timeLabeled(first, "Holiday ends"); ok {
		t.Fatalf("first days lead into Shabbat; no 'holiday ends' expected: %+v", first.Times)
	}
	chm := findEvent(t, evs, "Intermediate day Passover", "2027-04-24")
	if tm, ok := timeLabeled(chm, "Shabbat ends"); !ok || tm.Time != "8:28 PM" {
		t.Fatalf("chol hamoed shabbat ends = %+v", chm.Times)
	}
	last := findEvent(t, evs, "Passover", "2027-04-28")
	if tm, ok := timeLabeled(last, "Holiday ends"); !ok || tm.Time != "8:34 PM" || tm.Date != "2027-04-29" {
		t.Fatalf("pesach ends = %+v", last.Times)
	}
	// Negative: ordinary days produce no events.
	for _, e := range evs {
		if e.Start == "2027-04-19" || e.Start == "2027-04-30" {
			t.Fatalf("ordinary day produced event %+v", e)
		}
	}
}

func TestAheadTishaBav(t *testing.T) {
	res, err := parseZmanimResponse(fixture(t, "zmanim_9av.json"), false)
	if err != nil {
		t.Fatal(err)
	}
	evs := buildAheadEvents(res.Days)
	if len(evs) != 1 {
		t.Fatalf("expected only the fast, got %+v", evs)
	}
	fast := evs[0]
	if tm, ok := timeLabeled(fast, "Fast begins (sunset)"); !ok || tm.Date != "2027-08-11" || tm.Time != "7:59 PM" {
		t.Fatalf("9 Av begins = %+v", fast.Times)
	}
	if tm, ok := timeLabeled(fast, "Fast ends"); !ok || tm.Date != "2027-08-12" || tm.Time != "8:28 PM" {
		t.Fatalf("9 Av ends = %+v", fast.Times)
	}
}

func TestAheadComputedChanukahAndMevarchim(t *testing.T) {
	from := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 12, 15, 0, 0, 0, 0, time.UTC)
	evs := mergeAheadRoshChodesh(computedAheadEvents(from, to, false))
	findEvent(t, evs, "Erev Chanukah: first candle, lit before Shabbat candle lighting", "2026-12-04")
	findEvent(t, evs, "Shabbat Mevarchim Tevet", "2026-12-05")
	rc := findEvent(t, evs, "Rosh Chodesh Tevet", "2026-12-10")
	if rc.End != "2026-12-11" {
		t.Fatalf("two-day Rosh Chodesh not merged: %+v", rc)
	}
	for _, e := range evs {
		if e.Category == "chassidic" {
			t.Fatalf("chassidic dates must need --all: %+v", e)
		}
	}
}

func TestNextAfterRollsOverAndFilters(t *testing.T) {
	res, err := parseZmanimResponse(fixture(t, "zmanim_detailed.json"), false)
	if err != nil {
		t.Fatal(err)
	}
	// 2026-12-04 21:00Z is 4:00 PM in Brooklyn, 11 minutes before candle lighting.
	now := time.Date(2026, 12, 4, 21, 0, 0, 0, time.UTC)
	got := nextAfter(res.Days, now, "", 3)
	if len(got) != 3 || got[0].Type != "CandleLighting" || got[0].MinutesUntil != 11 {
		t.Fatalf("next = %+v", got)
	}
	for i := 1; i < len(got); i++ {
		if got[i].At.Before(got[i-1].At) {
			t.Fatalf("not sorted: %+v", got)
		}
	}
	only := nextAfter(res.Days, now, "ShabbatEndTime", 5)
	if len(only) != 1 || only[0].Date != "2026-12-05" {
		t.Fatalf("filtered next = %+v", only)
	}
	// Past the last zman of the window: nothing, not stale data.
	if late := nextAfter(res.Days, now.AddDate(0, 0, 5), "", 3); len(late) != 0 {
		t.Fatalf("expected no upcoming zmanim, got %+v", late)
	}
}

func TestBulletinWeekShabbatAndChange(t *testing.T) {
	res, err := parseCandleResponse(fixture(t, "candles_weeks.json"))
	if err != nil {
		t.Fatal(err)
	}
	cur, prev, hol := weekShabbat(res.Groups, "2026-11-11", time.Time{})
	if cur == nil || cur.Parsha != "Toldot" || prev == nil || prev.Parsha != "Chayei Sarah" || len(hol) != 0 {
		t.Fatalf("cur=%+v prev=%+v hol=%+v", cur, prev, hol)
	}
	ch := weekChange(eventOf(cur, "candle-lighting"), eventOf(prev, "candle-lighting"))
	if ch == nil || *ch != -7 {
		t.Fatalf("candle change = %v", ch)
	}
	ends := weekChange(eventOf(cur, "ends"), eventOf(prev, "ends"))
	if ends == nil || *ends != -6 {
		t.Fatalf("ends change = %v", ends)
	}
	// On Shabbat itself, this week's Shabbat is still the current one.
	cur2, _, _ := weekShabbat(res.Groups, "2026-11-14", time.Time{})
	if cur2 == nil || cur2.Parsha != "Toldot" {
		t.Fatalf("saturday ref picked %+v", cur2)
	}
	if mv := mevarchimFor("2026-12-05"); mv == nil || mv.Month != "Tevet" || len(mv.RoshChodesh) != 2 {
		t.Fatalf("mevarchim = %+v", mv)
	}
}

func TestUpcomingOccasions(t *testing.T) {
	list := []store.Occasion{
		{ID: 1, Name: "Zeide", Kind: "yahrzeit", HebrewYear: 5776, HebrewMonth: 9, HebrewDay: 8},
		{ID: 2, Name: "Levi", Kind: "birthday", HebrewYear: 5774, HebrewMonth: 9, HebrewDay: 19},
		{ID: 3, Name: "Far away", Kind: "birthday", HebrewYear: 5770, HebrewMonth: 4, HebrewDay: 3},
	}
	from := time.Date(2026, 11, 11, 0, 0, 0, 0, time.UTC)
	got := upcomingOccasions(list, from, 30)
	if len(got) != 2 {
		t.Fatalf("expected 2 within 30 days, got %+v", got)
	}
	if got[0].Name != "Zeide" || got[0].Date != "2026-11-18" || got[0].BeginsEven != "2026-11-17" || got[0].Years != 11 {
		t.Fatalf("yahrzeit = %+v", got[0])
	}
	if got[1].Name != "Levi" || got[1].Date != "2026-11-29" || !strings.Contains(got[1].Milestone, "13th") {
		t.Fatalf("birthday = %+v", got[1])
	}
	if none := upcomingOccasions(nil, from, 30); len(none) != 0 || none == nil {
		t.Fatalf("empty list should give empty non-nil slice, got %#v", none)
	}
}

func TestHolidayRunsCollapse(t *testing.T) {
	res, err := parseZmanimResponse(fixture(t, "zmanim_pesach.json"), false)
	if err != nil {
		t.Fatal(err)
	}
	rows := collapseHolidayRuns(res.Days)
	names := make([]string, 0, len(rows))
	for _, r := range rows {
		names = append(names, r.Name+":"+r.Start+".."+r.End)
	}
	want := "Eve of First day Passover:2027-04-21..2027-04-21|Passover:2027-04-22..2027-04-23|Intermediate day Passover:2027-04-24..2027-04-26|Eve of Seventh day Passover:2027-04-27..2027-04-27|Passover:2027-04-28..2027-04-29"
	if strings.Join(names, "|") != want {
		t.Fatalf("runs = %s", strings.Join(names, "|"))
	}
}

func TestEventsOverlappingKeepsTrueEnds(t *testing.T) {
	evs := []aheadEvent{
		{Name: "Chanukah", Start: "2026-12-05", End: "2026-12-12"},
		{Name: "Past", Start: "2026-11-01", End: "2026-11-02"},
		{Name: "Later", Start: "2027-01-01", End: "2027-01-01"},
	}
	got := eventsOverlapping(evs, time.Date(2026, 11, 11, 0, 0, 0, 0, time.UTC), time.Date(2026, 12, 10, 0, 0, 0, 0, time.UTC))
	if len(got) != 1 || got[0].End != "2026-12-12" {
		t.Fatalf("overlap = %+v", got)
	}
}

func TestReviewFixes(t *testing.T) {
	// Night-side zman printed after midnight belongs to the next day.
	day := time.Date(2027, 6, 20, 0, 0, 0, 0, time.UTC)
	at, ok := absoluteZman(day, 120, "Tzeis", "12:30 AM")
	if !ok || at.Format("2006-01-02 15:04") != "2027-06-20 22:30" {
		t.Fatalf("after-midnight nightfall = %v", at)
	}
	if at2, _ := absoluteZman(day, 120, "AlosHashachar", "3:30 AM"); at2.Format("2006-01-02") != "2027-06-20" {
		t.Fatalf("dawn must stay on its own day: %v", at2)
	}
	// ICS folding: no line over 75 octets, multi-byte runes intact.
	var b strings.Builder
	icsLine(&b, "DESCRIPTION:"+strings.Repeat("הדלקת נרות ", 20))
	for _, ln := range strings.Split(strings.TrimSuffix(b.String(), "\r\n"), "\r\n") {
		if len(ln) > 75 {
			t.Fatalf("unfolded line of %d octets", len(ln))
		}
	}
	if strings.Contains(b.String(), "�") {
		t.Fatalf("folding split a rune")
	}
	// Control characters from remote text are dropped.
	if got := stripHTML("Brooklyn&#27;]52;c;x\rATTACH:evil"); strings.ContainsAny(got, "\x1b\r") {
		t.Fatalf("control chars survived: %q", got)
	}
	if got := icsEscape("a\rb"); got != `a\nb` {
		t.Fatalf("icsEscape CR = %q", got)
	}
	// Occasions across a window that reaches two Hebrew years ahead.
	list := []store.Occasion{{ID: 1, Name: "X", Kind: "birthday", HebrewYear: 5780, HebrewMonth: 7, HebrewDay: 20}}
	got := upcomingOccasions(list, time.Date(2027, 9, 30, 0, 0, 0, 0, time.UTC), 400)
	foundY2 := false
	for _, d := range got {
		if strings.HasSuffix(d.HebrewDate, "5789") {
			foundY2 = true
		}
	}
	if !foundY2 {
		t.Fatalf("missing 5789 occurrence: %+v", got)
	}
}
