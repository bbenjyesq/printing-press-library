// Copyright 2026 Bob Benjy and contributors. Licensed under Apache-2.0. See LICENSE.
// pp:data-source computed

// Locally computed calendar commands: date, omer, yahrzeit, birthday,
// rosh-chodesh. They need no network.

package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/mvanhorn/printing-press-library/library/productivity/chabad/internal/jcal"

	"github.com/spf13/cobra"
)

type hebrewDateView struct {
	Gregorian   string     `json:"gregorian"`
	Weekday     string     `json:"weekday"`
	AfterSunset bool       `json:"after_sunset"`
	Hebrew      string     `json:"hebrew_date"`
	HebrewParts jcal.HDate `json:"hebrew"`
	MonthName   string     `json:"month_name"`
	LeapYear    bool       `json:"leap_year"`
	BeginsAt    string     `json:"begins_evening_of"`
	OmerDay     int        `json:"omer_day,omitempty"`
	RoshChodesh bool       `json:"rosh_chodesh"`
	Events      []string   `json:"events"`
}

func describeHebrewDate(greg time.Time, h jcal.HDate, afterSunset bool) hebrewDateView {
	f := h.Fixed()
	v := hebrewDateView{
		Gregorian:   greg.Format("2006-01-02"),
		Weekday:     greg.Weekday().String(),
		AfterSunset: afterSunset,
		Hebrew:      h.String(),
		HebrewParts: h,
		MonthName:   jcal.MonthName(h.Month, h.Year),
		LeapYear:    jcal.IsLeap(h.Year),
		BeginsAt:    jcal.TimeFromFixed(f - 1).Format("2006-01-02"),
		OmerDay:     jcal.OmerDay(h),
		Events:      make([]string, 0),
	}
	for _, rc := range jcal.RoshChodeshBetween(f, f) {
		v.RoshChodesh = true
		v.Events = append(v.Events, rc.Name)
	}
	for _, e := range jcal.EventsBetween(append(append([]jcal.Event{}, jcal.MinorEvents...), jcal.ChassidicEvents...), f, f) {
		v.Events = append(v.Events, e.Name)
	}
	return v
}

func newDateCmd(flags *rootFlags) *cobra.Command {
	var hebrew string
	var afterSunset bool
	cmd := &cobra.Command{
		Use:   "date [YYYY-MM-DD]",
		Short: "Convert between Gregorian and Hebrew dates (offline)",
		Long: strings.Trim(`
Convert a Gregorian date to its Hebrew date, or a Hebrew date (--hebrew) to
the Gregorian day it falls on. The Hebrew day begins at nightfall, so pass
--after-sunset for an event that happened after sunset. Also reports the Omer
day, Rosh Chodesh, and any minor or Chassidic date on that day.`, "\n"),
		Example: strings.Trim(`
  chabad-pp-cli date
  chabad-pp-cli date 2026-12-25
  chabad-pp-cli date 1994-06-12 --after-sunset
  chabad-pp-cli date --hebrew "19 Kislev 5787"`, "\n"),
		Annotations: map[string]string{"mcp:read-only": "true", "pp:data-source": "computed"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return writeDryRun(cmd.OutOrStdout(), flags, "date")
			}
			if len(args) > 1 {
				return usageErr(fmt.Errorf("pass at most one date"))
			}
			var view hebrewDateView
			if hebrew != "" {
				if len(args) == 1 {
					return usageErr(fmt.Errorf("pass either a Gregorian date or --hebrew, not both"))
				}
				h, err := jcal.Parse(hebrew, jcal.FromTime(todayDate(), false).Year)
				if err != nil {
					return usageErr(err)
				}
				view = describeHebrewDate(h.Time(), h, false)
			} else {
				arg := ""
				if len(args) == 1 {
					arg = args[0]
				}
				g, err := parseDateArg(arg)
				if err != nil {
					return err
				}
				h := jcal.FromTime(g, afterSunset)
				view = describeHebrewDate(g, h, afterSunset)
			}
			if !wantsHumanTable(cmd.OutOrStdout(), flags) {
				return printJSONFiltered(cmd.OutOrStdout(), view, flags)
			}
			w := cmd.OutOrStdout()
			when := longDate(view.Gregorian)
			if view.AfterSunset {
				when += " (after sunset)"
			}
			fmt.Fprintf(w, "%s\n%s\n", bold(view.Hebrew), when)
			if hebrew != "" {
				fmt.Fprintf(w, "Begins at nightfall on %s\n", longDate(view.BeginsAt))
			}
			if view.OmerDay > 0 {
				fmt.Fprintf(w, "Omer: day %d (%s)\n", view.OmerDay, jcal.OmerSefirah(view.OmerDay))
			}
			for _, e := range view.Events {
				fmt.Fprintln(w, "• "+e)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&hebrew, "hebrew", "", "Convert a Hebrew date instead, e.g. \"19 Kislev 5787\"")
	cmd.Flags().BoolVar(&afterSunset, "after-sunset", false, "The Gregorian date's event was after sunset (use the next Hebrew day)")
	return cmd
}

type omerView struct {
	Date         string `json:"date"`
	HebrewDate   string `json:"hebrew_date"`
	InOmer       bool   `json:"in_omer"`
	TodayDay     int    `json:"today_day,omitempty"`
	TonightDay   int    `json:"tonight_day,omitempty"`
	TonightCount string `json:"tonight_count,omitempty"`
	Sefirah      string `json:"tonight_sefirah,omitempty"`
	NextStart    string `json:"next_starts_evening_of,omitempty"`
}

func newOmerCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "omer [YYYY-MM-DD]",
		Short: "Sefirat HaOmer: tonight's count and its sefirah (offline)",
		Long: strings.Trim(`
The Omer is counted at night for 49 days from the second night of Pesach to
Shavuot. This prints the count for the evening of the given date (default:
today) and the day already counted last night. Outside the Omer it says when
counting next begins.`, "\n"),
		Example: strings.Trim(`
  chabad-pp-cli omer
  chabad-pp-cli omer 2027-05-05 --json`, "\n"),
		Annotations: map[string]string{"mcp:read-only": "true", "pp:data-source": "computed"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return writeDryRun(cmd.OutOrStdout(), flags, "omer")
			}
			if len(args) > 1 {
				return usageErr(fmt.Errorf("pass at most one date"))
			}
			arg := ""
			if len(args) == 1 {
				arg = args[0]
			}
			g, err := parseDateArg(arg)
			if err != nil {
				return err
			}
			today := jcal.FromTime(g, false)
			tonight := jcal.FromTime(g, true)
			v := omerView{Date: g.Format("2006-01-02"), HebrewDate: today.String(), TodayDay: jcal.OmerDay(today), TonightDay: jcal.OmerDay(tonight)}
			v.InOmer = v.TodayDay > 0 || v.TonightDay > 0
			if v.TonightDay > 0 {
				v.TonightCount = jcal.OmerPhrase(v.TonightDay)
				v.Sefirah = jcal.OmerSefirah(v.TonightDay)
			}
			if !v.InOmer {
				y := today.Year
				start := jcal.HDate{Year: y, Month: jcal.Nissan, Day: 16}.Fixed()
				if start-1 < jcal.FixedFromTime(g) {
					start = jcal.HDate{Year: y + 1, Month: jcal.Nissan, Day: 16}.Fixed()
				}
				v.NextStart = jcal.TimeFromFixed(start - 1).Format("2006-01-02")
			}
			if !wantsHumanTable(cmd.OutOrStdout(), flags) {
				return printJSONFiltered(cmd.OutOrStdout(), v, flags)
			}
			w := cmd.OutOrStdout()
			if !v.InOmer {
				fmt.Fprintf(w, "Not during the Omer. Counting begins the evening of %s.\n", longDate(v.NextStart))
				return nil
			}
			if v.TonightDay > 0 {
				fmt.Fprintf(w, "Tonight (%s evening): %s\n%s\n", shortDate(v.Date), bold("Today is "+v.TonightCount+" of the Omer"), v.Sefirah)
			} else {
				fmt.Fprintln(w, "The Omer count is complete; Shavuot begins tonight.")
			}
			if v.TodayDay > 0 {
				fmt.Fprintf(w, "Last night was day %d.\n", v.TodayDay)
			}
			return nil
		},
	}
	return cmd
}

type anniversaryRow struct {
	HebrewYear   int    `json:"hebrew_year"`
	Date         string `json:"date"`
	Weekday      string `json:"weekday"`
	BeginsEvenOf string `json:"begins_evening_of"`
	HebrewDate   string `json:"hebrew_date"`
	Years        int    `json:"years"`
}

type anniversaryView struct {
	Kind        string           `json:"kind"`
	Original    hebrewDateView   `json:"original"`
	Upcoming    []anniversaryRow `json:"upcoming"`
	BarMitzvah  *anniversaryRow  `json:"bar_mitzvah,omitempty"`
	BatMitzvah  *anniversaryRow  `json:"bat_mitzvah,omitempty"`
	Explanation string           `json:"rule,omitempty"`
}

func anniversaryFromInputs(args []string, hebrew string, afterSunset bool) (jcal.HDate, time.Time, bool, error) {
	if len(args) > 1 {
		return jcal.HDate{}, time.Time{}, false, usageErr(fmt.Errorf("pass one date"))
	}
	if hebrew != "" {
		if len(args) == 1 {
			return jcal.HDate{}, time.Time{}, false, usageErr(fmt.Errorf("pass either a Gregorian date or --hebrew, not both"))
		}
		h, err := jcal.Parse(hebrew, jcal.FromTime(todayDate(), false).Year)
		if err != nil {
			return jcal.HDate{}, time.Time{}, false, usageErr(err)
		}
		return h, h.Time(), false, nil
	}
	if len(args) == 0 {
		return jcal.HDate{}, time.Time{}, false, usageErr(fmt.Errorf("pass the date (YYYY-MM-DD) or --hebrew \"<day> <month> <year>\""))
	}
	g, err := parseDateArg(args[0])
	if err != nil {
		return jcal.HDate{}, time.Time{}, false, err
	}
	return jcal.FromTime(g, afterSunset), g, afterSunset, nil
}

func anniversaryRowFor(fixed int, origYear int) anniversaryRow {
	h := jcal.FromFixed(fixed)
	t := jcal.TimeFromFixed(fixed)
	return anniversaryRow{
		HebrewYear:   h.Year,
		Date:         t.Format("2006-01-02"),
		Weekday:      t.Weekday().String(),
		BeginsEvenOf: jcal.TimeFromFixed(fixed - 1).Format("2006-01-02"),
		HebrewDate:   h.String(),
		Years:        h.Year - origYear,
	}
}

func upcomingAnniversaries(orig jcal.HDate, n int, fn func(jcal.HDate, int) int) []anniversaryRow {
	out := make([]anniversaryRow, 0, n)
	today := jcal.FixedFromTime(todayDate())
	y := jcal.FromFixed(today).Year
	if y <= orig.Year {
		y = orig.Year + 1
	}
	for len(out) < n && y < orig.Year+1000 {
		f := fn(orig, y)
		if f >= today {
			out = append(out, anniversaryRowFor(f, orig.Year))
		}
		y++
	}
	return out
}

func printAnniversaries(cmd *cobra.Command, flags *rootFlags, v anniversaryView, label string) error {
	if !wantsHumanTable(cmd.OutOrStdout(), flags) {
		return printJSONFiltered(cmd.OutOrStdout(), v, flags)
	}
	w := cmd.OutOrStdout()
	when := longDate(v.Original.Gregorian)
	if v.Original.AfterSunset {
		when += " (after sunset)"
	}
	fmt.Fprintf(w, "%s: %s — %s\n", label, bold(v.Original.Hebrew), when)
	if v.Explanation != "" {
		fmt.Fprintln(w, v.Explanation)
	}
	fmt.Fprintln(w)
	tw := newTabWriter(w)
	for _, r := range v.Upcoming {
		fmt.Fprintf(tw, "%s\t%s\t%s\tbegins evening of %s\n", longDate(r.Date), r.HebrewDate, fmt.Sprintf("%d years", r.Years), shortDate(r.BeginsEvenOf))
	}
	if err := tw.Flush(); err != nil {
		return err
	}
	if v.BarMitzvah != nil {
		fmt.Fprintf(w, "\nBar mitzvah (13): %s, %s\n", longDate(v.BarMitzvah.Date), v.BarMitzvah.HebrewDate)
	}
	if v.BatMitzvah != nil {
		fmt.Fprintf(w, "Bat mitzvah (12): %s, %s\n", longDate(v.BatMitzvah.Date), v.BatMitzvah.HebrewDate)
	}
	return nil
}

func yahrzeitRuleNote(h jcal.HDate) string {
	switch {
	case h.Month == jcal.Adar && !jcal.IsLeap(h.Year):
		return "Passing in Adar of a common year: observed in Adar I in leap years (Chabad custom; some also observe Adar II)."
	case h.Month == jcal.AdarII:
		return "Passing in Adar II: observed in Adar in common years and Adar II in leap years."
	case (h.Month == jcal.Cheshvan || h.Month == jcal.Kislev) && h.Day == 30:
		return "Passing on the 30th of a month that is sometimes 29 days: in such years it is observed on the 29th."
	}
	return ""
}

func newYahrzeitCmd(flags *rootFlags) *cobra.Command {
	var hebrew string
	var afterSunset bool
	var years int
	cmd := &cobra.Command{
		Use:   "yahrzeit [YYYY-MM-DD]",
		Short: "Yahrzeit dates for the coming years from a date of passing (offline)",
		Long: strings.Trim(`
Computes the Hebrew date of passing and the Gregorian dates of the next
yahrzeits. Each yahrzeit begins at nightfall the evening before. Pass
--after-sunset if the passing was after sunset. Adar and 30-Cheshvan/Kislev
edge cases follow the common rules (Chabad custom for Adar: Adar I).

For a saved list of many people's dates, use 'occasions'.`, "\n"),
		Example: strings.Trim(`
  chabad-pp-cli yahrzeit 2015-03-10 --after-sunset --years 5
  chabad-pp-cli yahrzeit --hebrew "3 Tammuz 5754"`, "\n"),
		Annotations: map[string]string{"mcp:read-only": "true", "pp:data-source": "computed"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && hebrew == "" && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return writeDryRun(cmd.OutOrStdout(), flags, "yahrzeit")
			}
			if years < 1 || years > 50 {
				return usageErr(fmt.Errorf("--years must be between 1 and 50"))
			}
			h, g, as, err := anniversaryFromInputs(args, hebrew, afterSunset)
			if err != nil {
				return err
			}
			v := anniversaryView{
				Kind:        "yahrzeit",
				Original:    describeHebrewDate(g, h, as),
				Upcoming:    upcomingAnniversaries(h, years, jcal.Yahrzeit),
				Explanation: yahrzeitRuleNote(h),
			}
			return printAnniversaries(cmd, flags, v, "Passing")
		},
	}
	cmd.Flags().StringVar(&hebrew, "hebrew", "", "Hebrew date of passing instead, e.g. \"3 Tammuz 5754\"")
	cmd.Flags().BoolVar(&afterSunset, "after-sunset", false, "The passing was after sunset")
	cmd.Flags().IntVar(&years, "years", 5, "How many upcoming yahrzeits to list")
	return cmd
}

func newBirthdayCmd(flags *rootFlags) *cobra.Command {
	var hebrew string
	var afterSunset bool
	var years int
	cmd := &cobra.Command{
		Use:   "birthday [YYYY-MM-DD]",
		Short: "Hebrew birthday, upcoming Hebrew birthdays, and bar/bat-mitzvah dates (offline)",
		Long: strings.Trim(`
Computes the Hebrew birth date, the next Hebrew birthdays, and the bar mitzvah
(13) and bat mitzvah (12) dates. Pass --after-sunset if the birth was after
sunset. Born in Adar of a common year: the birthday is in Adar II in leap years.

For a saved list of many people's dates, use 'occasions'.`, "\n"),
		Example: strings.Trim(`
  chabad-pp-cli birthday 2014-03-02
  chabad-pp-cli birthday 2014-03-02 --after-sunset --years 3 --json`, "\n"),
		Annotations: map[string]string{"mcp:read-only": "true", "pp:data-source": "computed"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && hebrew == "" && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return writeDryRun(cmd.OutOrStdout(), flags, "birthday")
			}
			if years < 1 || years > 50 {
				return usageErr(fmt.Errorf("--years must be between 1 and 50"))
			}
			h, g, as, err := anniversaryFromInputs(args, hebrew, afterSunset)
			if err != nil {
				return err
			}
			bar := anniversaryRowFor(jcal.Birthday(h, h.Year+13), h.Year)
			bat := anniversaryRowFor(jcal.Birthday(h, h.Year+12), h.Year)
			v := anniversaryView{
				Kind:       "birthday",
				Original:   describeHebrewDate(g, h, as),
				Upcoming:   upcomingAnniversaries(h, years, jcal.Birthday),
				BarMitzvah: &bar,
				BatMitzvah: &bat,
			}
			return printAnniversaries(cmd, flags, v, "Born")
		},
	}
	cmd.Flags().StringVar(&hebrew, "hebrew", "", "Hebrew birth date instead, e.g. \"11 Nissan 5762\"")
	cmd.Flags().BoolVar(&afterSunset, "after-sunset", false, "The birth was after sunset")
	cmd.Flags().IntVar(&years, "years", 3, "How many upcoming Hebrew birthdays to list")
	return cmd
}

type roshChodeshRow struct {
	Month            string   `json:"month"`
	HebrewYear       int      `json:"hebrew_year"`
	Days             []string `json:"days"`
	Molad            string   `json:"molad"`
	ShabbatMevarchim string   `json:"shabbat_mevarchim,omitempty"`
}

func newRoshChodeshCmd(flags *rootFlags) *cobra.Command {
	var months int
	var from string
	cmd := &cobra.Command{
		Use:   "rosh-chodesh",
		Short: "Upcoming Rosh Chodesh days, the molad, and Shabbat Mevarchim (offline)",
		Example: strings.Trim(`
  chabad-pp-cli rosh-chodesh
  chabad-pp-cli rosh-chodesh --months 6 --json`, "\n"),
		Annotations: map[string]string{"mcp:read-only": "true", "pp:data-source": "computed"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return writeDryRun(cmd.OutOrStdout(), flags, "rosh-chodesh")
			}
			if months < 1 || months > 24 {
				return usageErr(fmt.Errorf("--months must be between 1 and 24"))
			}
			start, err := parseDateArg(from)
			if err != nil {
				return err
			}
			startF := jcal.FixedFromTime(start)
			h := jcal.FromFixed(startF)
			m, y := h.Month, h.Year
			rows := make([]roshChodeshRow, 0, months)
			// Start with the current month so a Rosh Chodesh already under way
			// (or today) is listed, then advance.
			for i := 0; len(rows) < months && i < 40; i, m, y = i+1, nextMonthM(m, y), nextMonthY(m, y) {
				if m == jcal.Tishrei {
					continue // Rosh Hashanah, not Rosh Chodesh
				}
				rc := jcal.RoshChodesh(m, y)
				if len(rc) == 0 || rc[len(rc)-1] < startF {
					continue
				}
				row := roshChodeshRow{Month: jcal.MonthName(m, y), HebrewYear: y, Days: make([]string, 0, 2), Molad: jcal.MoladOf(m, y).String()}
				for _, f := range rc {
					row.Days = append(row.Days, jcal.TimeFromFixed(f).Format("2006-01-02"))
				}
				if sm := jcal.ShabbatMevarchim(m, y); sm != 0 {
					row.ShabbatMevarchim = jcal.TimeFromFixed(sm).Format("2006-01-02")
				}
				rows = append(rows, row)
			}
			if !wantsHumanTable(cmd.OutOrStdout(), flags) {
				return printJSONFiltered(cmd.OutOrStdout(), rows, flags)
			}
			tw := newTabWriter(cmd.OutOrStdout())
			for _, r := range rows {
				days := make([]string, len(r.Days))
				for i, d := range r.Days {
					days[i] = shortDate(d)
				}
				fmt.Fprintf(tw, "%s %d\t%s\tMevarchim %s\tMolad %s\n", bold(r.Month), r.HebrewYear, strings.Join(days, " & "), shortDate(r.ShabbatMevarchim), r.Molad)
			}
			return tw.Flush()
		},
	}
	cmd.Flags().IntVar(&months, "months", 3, "How many upcoming months to list")
	cmd.Flags().StringVar(&from, "from", "", "Start from this date, YYYY-MM-DD (default: today)")
	return cmd
}

func init() {
	registerNovelCommand(func(root *cobra.Command, flags *rootFlags) {
		addNovelCommandIfAbsent(root, newDateCmd(flags))
		addNovelCommandIfAbsent(root, newOmerCmd(flags))
		addNovelCommandIfAbsent(root, newYahrzeitCmd(flags))
		addNovelCommandIfAbsent(root, newBirthdayCmd(flags))
		addNovelCommandIfAbsent(root, newRoshChodeshCmd(flags))
	})
}

func nextMonthM(m, y int) int { nm, _ := jcal.NextMonth(m, y); return nm }
func nextMonthY(m, y int) int { _, ny := jcal.NextMonth(m, y); return ny }
