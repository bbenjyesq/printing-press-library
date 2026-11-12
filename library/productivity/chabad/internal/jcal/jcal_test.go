// Copyright 2026 Bob Benjy and contributors. Licensed under Apache-2.0. See LICENSE.

package jcal

import (
	"testing"
	"time"
)

func day(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

// Ground truth: Hebrew dates Chabad.org itself returned for these Gregorian
// days (daily-study RSS channel titles and candle-lighting JewishDate fields).
func TestFromTimeMatchesChabadOrg(t *testing.T) {
	cases := []struct {
		greg string
		want string
	}{
		{"2026-11-11", "Kislev 1, 5787"},
		{"2026-11-13", "Kislev 3, 5787"},
		{"2026-12-10", "Kislev 30, 5787"},
		{"2026-12-11", "Tevet 1, 5787"},
		{"2026-12-25", "Tevet 15, 5787"},
		{"2027-03-23", "Adar II 14, 5787"},
		{"2027-04-15", "Nissan 8, 5787"},
		{"2000-01-01", "Tevet 23, 5760"},
	}
	for _, c := range cases {
		got := FromTime(day(c.greg), false).String()
		if got != c.want {
			t.Errorf("FromTime(%s) = %q, want %q", c.greg, got, c.want)
		}
	}
}

func TestAfterSunsetAdvancesOneDay(t *testing.T) {
	got := FromTime(day("2026-12-10"), true).String()
	if got != "Tevet 1, 5787" {
		t.Fatalf("after sunset on 2026-12-10 = %q, want Tevet 1, 5787", got)
	}
}

func TestRoundTrip(t *testing.T) {
	start := FixedFromTime(day("1990-01-01"))
	end := FixedFromTime(day("2060-12-31"))
	for f := start; f <= end; f++ {
		h := FromFixed(f)
		if !h.Valid() {
			t.Fatalf("FromFixed(%d) produced invalid %+v", f, h)
		}
		if back := h.Fixed(); back != f {
			t.Fatalf("round trip %d -> %+v -> %d", f, h, back)
		}
	}
}

func TestNewYearDates(t *testing.T) {
	cases := map[int]string{
		5785: "2024-10-03",
		5786: "2025-09-23",
		5787: "2026-09-12",
		5788: "2027-10-02",
	}
	for y, want := range cases {
		got := TimeFromFixed(NewYear(y)).Format("2006-01-02")
		if got != want {
			t.Errorf("Rosh Hashanah %d = %s, want %s", y, got, want)
		}
	}
}

func TestLeapYearsAndMonthNames(t *testing.T) {
	if !IsLeap(5787) || IsLeap(5786) || !IsLeap(5784) {
		t.Fatalf("leap year rule wrong")
	}
	if MonthName(Adar, 5787) != "Adar I" || MonthName(Adar, 5786) != "Adar" || MonthName(AdarII, 5787) != "Adar II" {
		t.Fatalf("Adar naming wrong")
	}
}

func TestParse(t *testing.T) {
	cases := []struct {
		in   string
		want HDate
	}{
		{"19 Kislev 5787", HDate{5787, Kislev, 19}},
		{"Kislev 19, 5787", HDate{5787, Kislev, 19}},
		{"14 Adar II 5787", HDate{5787, AdarII, 14}},
		{"14 adar 2 5787", HDate{5787, AdarII, 14}},
		{"14 Adar II 5786", HDate{5786, Adar, 14}}, // common year: Adar II collapses to Adar
		{"10 Teves 5786", HDate{5786, Tevet, 10}},
		{"3 Tammuz", HDate{5786, Tammuz, 3}},
	}
	for _, c := range cases {
		got, err := Parse(c.in, 5786)
		if err != nil {
			t.Errorf("Parse(%q) error %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("Parse(%q) = %+v, want %+v", c.in, got, c.want)
		}
	}
	for _, bad := range []string{"30 Tevet 5787", "Smarch 3 5787", "", "31 Kislev 5787"} {
		if _, err := Parse(bad, 5786); err == nil {
			t.Errorf("Parse(%q) expected error", bad)
		}
	}
}

func TestRoshChodeshAndMevarchim(t *testing.T) {
	// Tevet 5787: two-day Rosh Chodesh Dec 10-11 2026 (Chanukah 6-7), Mevarchim Shabbat Dec 5 (Chabad.org flags it).
	rc := RoshChodesh(Tevet, 5787)
	if len(rc) != 2 || TimeFromFixed(rc[0]).Format("2006-01-02") != "2026-12-10" || TimeFromFixed(rc[1]).Format("2006-01-02") != "2026-12-11" {
		t.Fatalf("Rosh Chodesh Tevet 5787 = %v", rc)
	}
	if got := TimeFromFixed(ShabbatMevarchim(Tevet, 5787)).Format("2006-01-02"); got != "2026-12-05" {
		t.Fatalf("Shabbat Mevarchim Tevet 5787 = %s, want 2026-12-05", got)
	}
	if RoshChodesh(Tishrei, 5787) != nil || ShabbatMevarchim(Tishrei, 5787) != 0 {
		t.Fatalf("Tishrei must have no Rosh Chodesh / Mevarchim")
	}
	// Kislev 5787 follows a 29-day Cheshvan? Check single vs double day is consistent with month length.
	for m := Nissan; m <= MonthsInYear(5787); m++ {
		if m == Tishrei {
			continue
		}
		rc := RoshChodesh(m, 5787)
		prevLen := DaysInMonth(FromFixed(HDate{5787, m, 1}.Fixed()-1).Month, FromFixed(HDate{5787, m, 1}.Fixed()-1).Year)
		if (prevLen == 30) != (len(rc) == 2) {
			t.Errorf("month %d: prev length %d but %d RC days", m, prevLen, len(rc))
		}
	}
}

func TestMoladPrecedesRoshHashanah(t *testing.T) {
	// Rosh Hashanah is the molad day or postponed by at most two days.
	for y := 5700; y <= 5900; y++ {
		m := MoladOf(Tishrei, y)
		moladHebrewDay := m.Fixed
		if m.Hour >= 18 {
			moladHebrewDay++ // after 6 PM it is already the next Hebrew day
		}
		diff := NewYear(y) - moladHebrewDay
		if diff < 0 || diff > 2 {
			t.Fatalf("year %d: molad %s is %d days from Rosh Hashanah", y, m, diff)
		}
	}
}

func TestOmer(t *testing.T) {
	d16 := HDate{5787, Nissan, 16}
	if OmerDay(d16) != 1 {
		t.Fatalf("16 Nissan should be Omer day 1")
	}
	if OmerDay(HDate{5787, Sivan, 5}) != 49 || OmerDay(HDate{5787, Sivan, 6}) != 0 || OmerDay(HDate{5787, Nissan, 15}) != 0 {
		t.Fatalf("Omer bounds wrong")
	}
	if OmerSefirah(1) != "Chesed sheb'Chesed" || OmerSefirah(33) != "Hod sheb'Hod" {
		t.Fatalf("sefirah naming wrong: %s / %s", OmerSefirah(1), OmerSefirah(33))
	}
	if OmerPhrase(17) != "17 days, which are 2 weeks and 3 days" || OmerPhrase(7) != "7 days, which are 1 week" {
		t.Fatalf("omer phrase wrong: %s / %s", OmerPhrase(17), OmerPhrase(7))
	}
}

func TestYahrzeitRules(t *testing.T) {
	// Adar II passing, observed in plain Adar of a common year.
	got := FromFixed(Yahrzeit(HDate{5784, AdarII, 10}, 5786))
	if got != (HDate{5786, Adar, 10}) {
		t.Fatalf("Adar II yahrzeit in common year = %+v", got)
	}
	// Adar (common-year) passing observed in Adar I of a leap year.
	got = FromFixed(Yahrzeit(HDate{5786, Adar, 7}, 5787))
	if got != (HDate{5787, Adar, 7}) {
		t.Fatalf("Adar yahrzeit in leap year = %+v", got)
	}
	// Ordinary date stays put.
	got = FromFixed(Yahrzeit(HDate{5780, Tammuz, 3}, 5787))
	if got != (HDate{5787, Tammuz, 3}) {
		t.Fatalf("ordinary yahrzeit = %+v", got)
	}
}

func TestBirthdayRules(t *testing.T) {
	// Born in Adar of a common year: celebrated in Adar II of a leap year.
	got := FromFixed(Birthday(HDate{5786, Adar, 20}, 5787))
	if got != (HDate{5787, AdarII, 20}) {
		t.Fatalf("Adar birthday in leap year = %+v", got)
	}
	// Born in Adar I of a leap year: celebrated in Adar of a common year.
	got = FromFixed(Birthday(HDate{5787, Adar, 5}, 5788))
	if got != (HDate{5788, Adar, 5}) {
		t.Fatalf("Adar I birthday in common year = %+v", got)
	}
}

func TestEventsBetween(t *testing.T) {
	from := FixedFromTime(day("2026-11-01"))
	to := FixedFromTime(day("2026-12-31"))
	evs := EventsBetween(ChassidicEvents, from, to)
	found := false
	for _, e := range evs {
		if e.Name == "Yud-Tes Kislev, Rosh Hashanah of Chassidus" {
			found = true
			if TimeFromFixed(e.Fixed).Format("2006-01-02") != "2026-11-29" {
				t.Fatalf("19 Kislev 5787 = %s", TimeFromFixed(e.Fixed).Format("2006-01-02"))
			}
		}
	}
	if !found {
		t.Fatalf("19 Kislev missing from %+v", evs)
	}
	rc := RoshChodeshBetween(from, to)
	if len(rc) == 0 || rc[0].Name != "Rosh Chodesh Kislev" {
		t.Fatalf("Rosh Chodesh list = %+v", rc)
	}
	// Purim Katan only in leap years.
	if got := EventsBetween(MinorEvents, FixedFromTime(day("2026-02-01")), FixedFromTime(day("2026-03-31"))); len(got) > 0 {
		for _, e := range got {
			if e.Name == "Purim Katan" {
				t.Fatalf("Purim Katan listed in common year 5786")
			}
		}
	}
}

func TestRoshChodeshBetweenLongWindow(t *testing.T) {
	rc := RoshChodeshBetween(FixedFromTime(day("2027-01-01")), FixedFromTime(day("2029-12-31")))
	last := TimeFromFixed(rc[len(rc)-1].Fixed).Format("2006-01-02")
	if last < "2029-12-01" {
		t.Fatalf("long window stopped early at %s", last)
	}
}
