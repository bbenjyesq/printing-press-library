// Copyright 2026 Bob Benjy and contributors. Licensed under Apache-2.0. See LICENSE.

// Package jcal implements the fixed arithmetic Hebrew calendar: conversion
// between Gregorian and Hebrew dates, month lengths, the molad, Rosh Chodesh,
// the Omer count, and Hebrew anniversaries (yahrzeit and birthday rules).
//
// Dates are carried as "fixed" day numbers (R.D. 1 = 1 January 1, proleptic
// Gregorian), following Dershowitz & Reingold, Calendrical Calculations.
// Month numbering follows the Torah order used by Chabad.org's own API:
// Nissan = 1 ... Elul = 6, Tishrei = 7 ... Adar (Adar I in a leap year) = 12,
// Adar II = 13.
package jcal

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Month numbers.
const (
	Nissan   = 1
	Iyar     = 2
	Sivan    = 3
	Tammuz   = 4
	Av       = 5
	Elul     = 6
	Tishrei  = 7
	Cheshvan = 8
	Kislev   = 9
	Tevet    = 10
	Shevat   = 11
	Adar     = 12 // Adar I in a leap year
	AdarII   = 13
)

// hebrewEpoch is the fixed date of 1 Tishrei AM 1 (Julian 7 October 3761 BCE).
const hebrewEpoch = -1373427

// unixEpochFixed is the fixed date of 1970-01-01.
const unixEpochFixed = 719163

// HDate is a Hebrew calendar date.
type HDate struct {
	Year  int `json:"year"`
	Month int `json:"month"`
	Day   int `json:"day"`
}

// IsLeap reports whether Hebrew year y has 13 months.
func IsLeap(y int) bool {
	return mod(7*y+1, 19) < 7
}

// MonthsInYear returns 12 or 13.
func MonthsInYear(y int) int {
	if IsLeap(y) {
		return 13
	}
	return 12
}

func mod(a, b int) int {
	m := a % b
	if m < 0 {
		m += b
	}
	return m
}

func floorDiv(a, b int) int {
	q := a / b
	if (a%b != 0) && ((a < 0) != (b < 0)) {
		q--
	}
	return q
}

// monthsBeforeYear is the number of months from the epoch to Tishrei of y.
func monthsBeforeYear(y int) int {
	return floorDiv(235*y-234, 19)
}

func elapsedDays(y int) int {
	months := monthsBeforeYear(y)
	parts := 12084 + 13753*months
	days := 29*months + floorDiv(parts, 25920)
	if mod(3*(days+1), 7) < 3 {
		days++
	}
	return days
}

func yearLengthCorrection(y int) int {
	ny0 := elapsedDays(y - 1)
	ny1 := elapsedDays(y)
	ny2 := elapsedDays(y + 1)
	switch {
	case ny2-ny1 == 356:
		return 2
	case ny1-ny0 == 382:
		return 1
	}
	return 0
}

// NewYear returns the fixed date of 1 Tishrei of year y.
func NewYear(y int) int {
	return hebrewEpoch + elapsedDays(y) + yearLengthCorrection(y)
}

// DaysInYear returns 353-355 or 383-385.
func DaysInYear(y int) int {
	return NewYear(y+1) - NewYear(y)
}

// LongCheshvan reports whether Cheshvan has 30 days in year y.
func LongCheshvan(y int) bool { return mod(DaysInYear(y), 10) == 5 }

// ShortKislev reports whether Kislev has 29 days in year y.
func ShortKislev(y int) bool { return mod(DaysInYear(y), 10) == 3 }

// DaysInMonth returns 29 or 30.
func DaysInMonth(m, y int) int {
	switch {
	case m == Iyar, m == Tammuz, m == Elul, m == Tevet, m == AdarII:
		return 29
	case m == Adar && !IsLeap(y):
		return 29
	case m == Cheshvan && !LongCheshvan(y):
		return 29
	case m == Kislev && ShortKislev(y):
		return 29
	}
	return 30
}

// Valid reports whether d names a real day.
func (d HDate) Valid() bool {
	if d.Year < 1 || d.Month < 1 || d.Month > MonthsInYear(d.Year) {
		return false
	}
	return d.Day >= 1 && d.Day <= DaysInMonth(d.Month, d.Year)
}

// Fixed converts a Hebrew date to a fixed day number. Out-of-range days roll
// forward into the next month (30 Cheshvan in a short year is 1 Kislev).
func (d HDate) Fixed() int {
	days := NewYear(d.Year) + d.Day - 1
	last := MonthsInYear(d.Year)
	if d.Month < Tishrei {
		for m := Tishrei; m <= last; m++ {
			days += DaysInMonth(m, d.Year)
		}
		for m := Nissan; m < d.Month; m++ {
			days += DaysInMonth(m, d.Year)
		}
	} else {
		for m := Tishrei; m < d.Month; m++ {
			days += DaysInMonth(m, d.Year)
		}
	}
	return days
}

// FromFixed converts a fixed day number to a Hebrew date.
func FromFixed(fixed int) HDate {
	approx := floorDiv((fixed-hebrewEpoch)*98496, 35975351) + 1
	y := approx - 1
	for NewYear(y+1) <= fixed {
		y++
	}
	for NewYear(y) > fixed {
		y--
	}
	start := Tishrei
	if fixed >= (HDate{Year: y, Month: Nissan, Day: 1}).Fixed() {
		start = Nissan
	}
	m := start
	for {
		end := HDate{Year: y, Month: m, Day: DaysInMonth(m, y)}.Fixed()
		if fixed <= end {
			break
		}
		m++
		if m > MonthsInYear(y) {
			m = Nissan
		}
	}
	day := fixed - HDate{Year: y, Month: m, Day: 1}.Fixed() + 1
	return HDate{Year: y, Month: m, Day: day}
}

// FixedFromTime converts the calendar date of t (in t's own location) to a
// fixed day number. The clock time is ignored.
func FixedFromTime(t time.Time) int {
	u := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	return int(u.Unix()/86400) + unixEpochFixed
}

// TimeFromFixed returns midnight UTC of the Gregorian date of fixed.
func TimeFromFixed(fixed int) time.Time {
	return time.Unix(int64(fixed-unixEpochFixed)*86400, 0).UTC()
}

// FromTime converts a Gregorian calendar date to a Hebrew date. When
// afterSunset is true the Hebrew date of the following evening is returned,
// since the Hebrew day begins at nightfall.
func FromTime(t time.Time, afterSunset bool) HDate {
	f := FixedFromTime(t)
	if afterSunset {
		f++
	}
	return FromFixed(f)
}

// Time returns midnight UTC of the Gregorian date on which d falls (the
// daytime portion; the Hebrew day began the previous evening).
func (d HDate) Time() time.Time { return TimeFromFixed(d.Fixed()) }

// Weekday returns the Gregorian weekday of the daytime portion of d.
func (d HDate) Weekday() time.Weekday { return d.Time().Weekday() }

var monthNames = map[int]string{
	Nissan: "Nissan", Iyar: "Iyar", Sivan: "Sivan", Tammuz: "Tammuz", Av: "Av", Elul: "Elul",
	Tishrei: "Tishrei", Cheshvan: "Cheshvan", Kislev: "Kislev", Tevet: "Tevet", Shevat: "Shevat",
	Adar: "Adar", AdarII: "Adar II",
}

// MonthName returns Chabad.org's transliteration of month m in year y
// ("Adar I" for month 12 of a leap year).
func MonthName(m, y int) string {
	if m == Adar && IsLeap(y) {
		return "Adar I"
	}
	return monthNames[m]
}

// String formats d the way Chabad.org does: "Kislev 19, 5787".
func (d HDate) String() string {
	return fmt.Sprintf("%s %d, %d", MonthName(d.Month, d.Year), d.Day, d.Year)
}

var monthAliases = map[string]int{
	"nissan": Nissan, "nisan": Nissan,
	"iyar": Iyar, "iyyar": Iyar, "iyar1": Iyar,
	"sivan":  Sivan,
	"tammuz": Tammuz, "tamuz": Tammuz,
	"av": Av, "menachem av": Av, "menachemav": Av,
	"elul":    Elul,
	"tishrei": Tishrei, "tishri": Tishrei,
	"cheshvan": Cheshvan, "heshvan": Cheshvan, "marcheshvan": Cheshvan, "mar cheshvan": Cheshvan, "marheshvan": Cheshvan,
	"kislev": Kislev, "kislew": Kislev,
	"tevet": Tevet, "teves": Tevet, "tebeth": Tevet,
	"shevat": Shevat, "shvat": Shevat, "sh'vat": Shevat, "shevet": Shevat,
	"adar": Adar, "adar i": Adar, "adar 1": Adar, "adar aleph": Adar, "adar rishon": Adar,
	"adar ii": AdarII, "adar 2": AdarII, "adar beit": AdarII, "adar bet": AdarII, "adar sheni": AdarII,
}

// ParseMonth resolves a Hebrew month name (several common spellings).
func ParseMonth(s string) (int, bool) {
	k := strings.ToLower(strings.TrimSpace(s))
	k = strings.Join(strings.Fields(k), " ")
	m, ok := monthAliases[k]
	return m, ok
}

// Parse reads a Hebrew date such as "19 Kislev 5787", "Kislev 19, 5787",
// "Adar II 14 5787" or "19 Kislev" (year defaults to defaultYear).
func Parse(s string, defaultYear int) (HDate, error) {
	clean := strings.NewReplacer(",", " ", "-", " ", "/", " ").Replace(s)
	fields := strings.Fields(clean)
	var nums []int
	var words []string
	for _, f := range fields {
		if n, err := strconv.Atoi(f); err == nil {
			nums = append(nums, n)
			continue
		}
		words = append(words, f)
	}
	monthText := strings.Join(words, " ")
	// "Adar 1 5787" style: a lone 1/2 right after "adar" belongs to the month.
	if strings.EqualFold(monthText, "adar") && len(nums) == 3 && (nums[0] == 1 || nums[0] == 2 || nums[1] == 1 || nums[1] == 2) {
		// ambiguous ordering; prefer interpreting the small number adjacent to the name
		for i, f := range fields {
			if strings.EqualFold(f, "adar") && i+1 < len(fields) && (fields[i+1] == "1" || fields[i+1] == "2") {
				monthText = "adar " + fields[i+1]
				nums = removeFirst(nums, atoi(fields[i+1]))
				break
			}
		}
	}
	m, ok := ParseMonth(monthText)
	if !ok {
		return HDate{}, fmt.Errorf("unknown Hebrew month %q (try e.g. \"19 Kislev 5787\")", monthText)
	}
	var day, year int
	switch len(nums) {
	case 1:
		day, year = nums[0], defaultYear
	case 2:
		day, year = nums[0], nums[1]
		if day > 31 && year <= 31 {
			day, year = year, day
		}
	default:
		return HDate{}, fmt.Errorf("cannot read Hebrew date %q (expected day, month, and optional year)", s)
	}
	if m == AdarII && !IsLeap(year) {
		m = Adar
	}
	d := HDate{Year: year, Month: m, Day: day}
	if year < 1 || day < 1 || day > 30 {
		return HDate{}, fmt.Errorf("invalid Hebrew date %q", s)
	}
	if m > MonthsInYear(year) {
		return HDate{}, fmt.Errorf("%d is not a leap year, so it has no %s", year, monthNames[m])
	}
	if day > DaysInMonth(m, year) {
		return HDate{}, fmt.Errorf("%s %d has only %d days", MonthName(m, year), year, DaysInMonth(m, year))
	}
	return d, nil
}

func atoi(s string) int { n, _ := strconv.Atoi(s); return n }

func removeFirst(xs []int, v int) []int {
	for i, x := range xs {
		if x == v {
			return append(append([]int{}, xs[:i]...), xs[i+1:]...)
		}
	}
	return xs
}

// Molad is the mean lunar conjunction announced on Shabbat Mevarchim.
type Molad struct {
	// Weekday is the civil weekday of the molad's clock time.
	Weekday time.Weekday `json:"weekday"`
	// Hour, Minute and Chalakim are civil clock values (0-23, 0-59, 0-17).
	Hour     int `json:"hour"`
	Minute   int `json:"minute"`
	Chalakim int `json:"chalakim"`
	// Fixed is the fixed date of the civil day on which the molad falls.
	Fixed int `json:"fixed"`
}

const (
	partsPerHour = 1080
	partsPerDay  = 24 * partsPerHour
	lunarMonth   = 29*partsPerDay + 12*partsPerHour + 793
	moladBaharad = 1*partsPerDay + 5*partsPerHour + 204 // Monday, 5h 204p after 6 PM Sunday
)

// monthsFromTishrei returns how many months month m is after Tishrei of y.
func monthsFromTishrei(m, y int) int {
	if m >= Tishrei {
		return m - Tishrei
	}
	return m + MonthsInYear(y) - Tishrei
}

// MoladOf returns the molad for month m of year y.
func MoladOf(m, y int) Molad {
	months := monthsBeforeYear(y) + monthsFromTishrei(m, y)
	parts := int64(moladBaharad) + int64(months)*int64(lunarMonth)
	days := parts / partsPerDay // days since the (Sunday 6 PM) start of the epoch week
	rem := parts % partsPerDay
	hoursFrom6pm := int(rem / partsPerHour)
	remParts := int(rem % partsPerHour)
	civilHour := (hoursFrom6pm + 18) % 24
	// Day index k is the Hebrew day whose daytime is fixed date
	// hebrewEpoch-1+k (index 1 is Monday 1 Tishrei AM 1). Hebrew days begin at
	// 18:00 of the previous civil day, so the first six hours belong to the
	// civil day before. BaHaRaD itself is Sunday 11:11 PM, fixed hebrewEpoch-1.
	civilFixed := hebrewEpoch - 2 + int(days)
	if hoursFrom6pm >= 6 {
		civilFixed++
	}
	return Molad{
		Weekday:  TimeFromFixed(civilFixed).Weekday(),
		Hour:     civilHour,
		Minute:   remParts / 18,
		Chalakim: remParts % 18,
		Fixed:    civilFixed,
	}
}

// String formats a molad like "Tuesday, 3:47 PM and 5 chalakim".
func (m Molad) String() string {
	h := m.Hour % 12
	if h == 0 {
		h = 12
	}
	ampm := "AM"
	if m.Hour >= 12 {
		ampm = "PM"
	}
	return fmt.Sprintf("%s %s, %d:%02d %s and %d chalakim", m.Weekday, TimeFromFixed(m.Fixed).Format("Jan 2, 2006"), h, m.Minute, ampm, m.Chalakim)
}

// RoshChodesh returns the fixed dates of Rosh Chodesh for month m of year y:
// two days (30th of the previous month and the 1st) when the previous month
// has 30 days, otherwise one. Tishrei has no Rosh Chodesh (it is Rosh
// Hashanah) and returns nil.
func RoshChodesh(m, y int) []int {
	if m == Tishrei {
		return nil
	}
	first := HDate{Year: y, Month: m, Day: 1}.Fixed()
	prev := FromFixed(first - 1)
	if prev.Day == 30 {
		return []int{first - 1, first}
	}
	return []int{first}
}

// ShabbatMevarchim returns the fixed date of the Shabbat on which the coming
// month m of year y is blessed: the last Shabbat before the first day of Rosh
// Chodesh. Tishrei is not blessed and returns 0.
func ShabbatMevarchim(m, y int) int {
	rc := RoshChodesh(m, y)
	if len(rc) == 0 {
		return 0
	}
	d := rc[0] - 1
	for TimeFromFixed(d).Weekday() != time.Saturday {
		d--
	}
	return d
}

// NextMonth returns the month after m in year y (rolling over the year).
func NextMonth(m, y int) (int, int) {
	switch {
	case m == Elul:
		return Tishrei, y + 1
	case m == MonthsInYear(y):
		return Nissan, y
	}
	return m + 1, y
}

var sefirot = []string{"Chesed", "Gevurah", "Tiferet", "Netzach", "Hod", "Yesod", "Malchut"}

// OmerDay returns the day of the Omer (1-49) whose count is said on the
// evening that begins Hebrew date d, or 0 outside the Omer.
func OmerDay(d HDate) int {
	start := HDate{Year: d.Year, Month: Nissan, Day: 16}.Fixed()
	n := d.Fixed() - start + 1
	if n < 1 || n > 49 {
		return 0
	}
	return n
}

// OmerSefirah returns the attribute pair for Omer day n, e.g. "Gevurah
// sheb'Chesed" for day 2.
func OmerSefirah(n int) string {
	if n < 1 || n > 49 {
		return ""
	}
	week := (n - 1) / 7
	day := (n - 1) % 7
	return fmt.Sprintf("%s sheb'%s", sefirot[day], sefirot[week])
}

// OmerPhrase describes day n as "17 days, which are 2 weeks and 3 days".
func OmerPhrase(n int) string {
	if n < 1 || n > 49 {
		return ""
	}
	if n < 7 {
		if n == 1 {
			return "1 day"
		}
		return fmt.Sprintf("%d days", n)
	}
	weeks, days := n/7, n%7
	ws := "weeks"
	if weeks == 1 {
		ws = "week"
	}
	if days == 0 {
		return fmt.Sprintf("%d days, which are %d %s", n, weeks, ws)
	}
	ds := "days"
	if days == 1 {
		ds = "day"
	}
	return fmt.Sprintf("%d days, which are %d %s and %d %s", n, weeks, ws, days, ds)
}

// Yahrzeit returns the fixed date (daytime) of the yahrzeit in Hebrew year
// year for a passing on Hebrew date death. Rules (Dershowitz & Reingold,
// matching common practice):
//   - 30 Cheshvan when the year after the passing had no 30 Cheshvan, and the
//     analogous 30 Kislev case, fall on the day before the 1st of the next month.
//   - A passing in Adar II is observed in the last Adar of the year.
//   - A passing on 30 Adar I is observed on 30 Shevat in a common year.
//   - A passing in Adar of a common year is observed in Adar I of a leap year
//     (some also observe Adar II; Chabad custom observes Adar I).
func Yahrzeit(death HDate, year int) int {
	switch {
	case death.Month == Cheshvan && death.Day == 30 && !LongCheshvan(death.Year+1):
		return HDate{Year: year, Month: Kislev, Day: 1}.Fixed() - 1
	case death.Month == Kislev && death.Day == 30 && ShortKislev(death.Year+1):
		return HDate{Year: year, Month: Tevet, Day: 1}.Fixed() - 1
	case death.Month == AdarII:
		return HDate{Year: year, Month: MonthsInYear(year), Day: death.Day}.Fixed()
	case death.Month == Adar && death.Day == 30 && !IsLeap(year):
		return HDate{Year: year, Month: Shevat, Day: 30}.Fixed()
	}
	return HDate{Year: year, Month: death.Month, Day: 1}.Fixed() + death.Day - 1
}

// Birthday returns the fixed date of the Hebrew birthday in year year for a
// birth on Hebrew date birth. A birth in the last month of the year (Adar of a
// common year, Adar II of a leap year) is celebrated in the last month of the
// target year (Adar II in a leap year). A 30th that does not exist rolls to
// the 1st of the next month.
func Birthday(birth HDate, year int) int {
	if birth.Month == MonthsInYear(birth.Year) {
		return HDate{Year: year, Month: MonthsInYear(year), Day: 1}.Fixed() + birth.Day - 1
	}
	return HDate{Year: year, Month: birth.Month, Day: 1}.Fixed() + birth.Day - 1
}

// Event is a fixed-date observance computed locally.
type Event struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Month    int    `json:"-"`
	Day      int    `json:"-"`
	// LeapOnly limits the event to leap years (Purim Katan).
	LeapOnly bool `json:"-"`
}

// MinorEvents are fixed-date days that Chabad.org's zmanim feed does not flag
// as holidays.
var MinorEvents = []Event{
	{Name: "Tu B'Shvat", Category: "minor", Month: Shevat, Day: 15},
	{Name: "Purim Katan", Category: "minor", Month: Adar, Day: 14, LeapOnly: true},
	{Name: "Shushan Purim", Category: "minor", Month: AdarII, Day: 15},
	{Name: "Pesach Sheni", Category: "minor", Month: Iyar, Day: 14},
	{Name: "Lag BaOmer", Category: "minor", Month: Iyar, Day: 18},
	{Name: "Tu B'Av", Category: "minor", Month: Av, Day: 15},
	{Name: "Hoshana Rabbah", Category: "minor", Month: Tishrei, Day: 21},
}

// ChassidicEvents are the Chabad calendar's Chassidic dates.
var ChassidicEvents = []Event{
	{Name: "Yahrzeit of Rebbetzin Chana Schneerson", Category: "chassidic", Month: Tishrei, Day: 6},
	{Name: "Yahrzeit of the Rebbe Maharash", Category: "chassidic", Month: Tishrei, Day: 13},
	{Name: "Birthday of the Rebbe Rashab", Category: "chassidic", Month: Cheshvan, Day: 20},
	{Name: "Birthday and yahrzeit of the Mitteler Rebbe", Category: "chassidic", Month: Kislev, Day: 9},
	{Name: "Liberation of the Mitteler Rebbe (Yud Kislev)", Category: "chassidic", Month: Kislev, Day: 10},
	{Name: "Wedding anniversary of the Rebbe and Rebbetzin", Category: "chassidic", Month: Kislev, Day: 14},
	{Name: "Yud-Tes Kislev, Rosh Hashanah of Chassidus", Category: "chassidic", Month: Kislev, Day: 19},
	{Name: "Didan Notzach (Hei Teves)", Category: "chassidic", Month: Tevet, Day: 5},
	{Name: "Yahrzeit of the Alter Rebbe", Category: "chassidic", Month: Tevet, Day: 24},
	{Name: "Yud Shevat", Category: "chassidic", Month: Shevat, Day: 10},
	{Name: "Yahrzeit of Rebbetzin Chaya Mushka", Category: "chassidic", Month: Shevat, Day: 22},
	{Name: "Birthday of Rebbetzin Chaya Mushka", Category: "chassidic", Month: AdarII, Day: 25},
	{Name: "Yahrzeit of the Rebbe Rashab", Category: "chassidic", Month: Nissan, Day: 2},
	{Name: "The Rebbe's birthday (Yud-Alef Nissan)", Category: "chassidic", Month: Nissan, Day: 11},
	{Name: "Yahrzeit of the Tzemach Tzedek", Category: "chassidic", Month: Nissan, Day: 13},
	{Name: "Birthday of the Rebbe Maharash", Category: "chassidic", Month: Iyar, Day: 2},
	{Name: "Gimmel Tammuz", Category: "chassidic", Month: Tammuz, Day: 3},
	{Name: "Yud-Beis Tammuz, liberation of the Previous Rebbe", Category: "chassidic", Month: Tammuz, Day: 12},
	{Name: "Yahrzeit of Rabbi Levi Yitzchak Schneerson", Category: "chassidic", Month: Av, Day: 20},
	{Name: "Chai Elul", Category: "chassidic", Month: Elul, Day: 18},
}

// Occurrence is a computed event on a specific date.
type Occurrence struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Fixed    int    `json:"-"`
	Hebrew   HDate  `json:"hebrew"`
}

// EventsBetween lists the given fixed-date events whose daytime falls in
// [from, to] (fixed dates, inclusive), in date order.
func EventsBetween(events []Event, from, to int) []Occurrence {
	out := make([]Occurrence, 0)
	startYear := FromFixed(from).Year
	endYear := FromFixed(to).Year
	for y := startYear; y <= endYear; y++ {
		for _, e := range events {
			m := e.Month
			if e.LeapOnly && !IsLeap(y) {
				continue
			}
			if m == AdarII && !IsLeap(y) {
				m = Adar
			}
			d := HDate{Year: y, Month: m, Day: e.Day}
			if e.Day > DaysInMonth(m, y) {
				continue
			}
			f := d.Fixed()
			if f < from || f > to {
				continue
			}
			out = append(out, Occurrence{Name: e.Name, Category: e.Category, Fixed: f, Hebrew: d})
		}
	}
	sortOccurrences(out)
	return out
}

// RoshChodeshBetween lists Rosh Chodesh days in [from, to].
func RoshChodeshBetween(from, to int) []Occurrence {
	out := make([]Occurrence, 0)
	h := FromFixed(from)
	m, y := h.Month, h.Year
	// Walk month by month from the month containing from until a month starts
	// after to; a two-day Rosh Chodesh starting in from's month is included.
	for {
		for _, f := range RoshChodesh(m, y) {
			if f >= from && f <= to {
				out = append(out, Occurrence{Name: "Rosh Chodesh " + MonthName(m, y), Category: "rosh-chodesh", Fixed: f, Hebrew: FromFixed(f)})
			}
		}
		first := HDate{Year: y, Month: m, Day: 1}.Fixed()
		if first > to {
			break
		}
		m, y = NextMonth(m, y)
	}
	sortOccurrences(out)
	return out
}

func sortOccurrences(xs []Occurrence) {
	sort.SliceStable(xs, func(i, j int) bool { return xs[i].Fixed < xs[j].Fixed })
}
