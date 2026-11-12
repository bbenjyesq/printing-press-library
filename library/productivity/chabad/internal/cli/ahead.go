// Copyright 2026 Bob Benjy and contributors. Licensed under Apache-2.0. See LICENSE.
// pp:data-source live

package cli

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/mvanhorn/printing-press-library/library/productivity/chabad/internal/jcal"

	"github.com/spf13/cobra"
)

type aheadTime struct {
	Label string `json:"label"`
	Date  string `json:"date"`
	Time  string `json:"time"`
}

type aheadEvent struct {
	Name     string      `json:"name"`
	Category string      `json:"category"`
	Start    string      `json:"start"`
	End      string      `json:"end"`
	Hebrew   string      `json:"hebrew_date"`
	Source   string      `json:"source"`
	Times    []aheadTime `json:"times"`
}

type aheadView struct {
	Location string       `json:"location"`
	Details  string       `json:"details,omitempty"`
	From     string       `json:"from"`
	To       string       `json:"to"`
	Events   []aheadEvent `json:"events"`
}

func zmanTime(d zmanimDay, t string) (string, bool) {
	z, ok := d.find(t)
	if !ok || z.Time == "" {
		return "", false
	}
	return z.Time, true
}

// buildAheadEvents turns a window of Chabad.org day records into holiday and
// fast events with the times each needs.
func buildAheadEvents(days []zmanimDay) []aheadEvent {
	events := make([]aheadEvent, 0)
	byDate := map[string]int{}
	for i, d := range days {
		byDate[d.Date] = i
	}
	i := 0
	for i < len(days) {
		d := days[i]
		name := normalizeHolidayName(d.Holiday)
		hasChametz := false
		for _, t := range []string{"BedikatChametz", "LastEatingChametzTime", "BurnChametzTime"} {
			if _, ok := zmanTime(d, t); ok {
				hasChametz = true
			}
		}
		if name == "" && !hasChametz {
			i++
			continue
		}
		if name == "" {
			name = "Erev Pesach preparations"
		}
		// Extend the run over consecutive days with the same name.
		j := i
		for j+1 < len(days) {
			nn := normalizeHolidayName(days[j+1].Holiday)
			if nn != name {
				break
			}
			j++
		}
		run := days[i : j+1]
		ev := aheadEvent{Name: name, Category: holidayCategory(name), Start: run[0].Date, End: run[len(run)-1].Date, Hebrew: run[0].Hebrew, Source: "chabad.org", Times: make([]aheadTime, 0)}
		first, last := run[0], run[len(run)-1]
		// add records the first zman type present (detailed responses call
		// the fast's end "FastEnds", compact ones "Tzeis").
		add := func(label string, d zmanimDay, types ...string) {
			for _, t := range types {
				if tm, ok := zmanTime(d, t); ok {
					ev.Times = append(ev.Times, aheadTime{Label: label, Date: d.Date, Time: tm})
					return
				}
			}
		}
		// A Yom Tov run continues into the next day when that day is also
		// Yom Tov under another name (Shemini Atzeret -> Simchat Torah).
		nextIsYomTov := false
		if j+1 < len(days) && days[j+1].Holiday != "" && holidayCategory(days[j+1].Holiday) == "holiday" {
			nextIsYomTov = true
		}
		switch {
		case name == "Fast - 9 Av":
			if prev, ok := byDate[first.day.AddDate(0, 0, -1).Format("2006-01-02")]; ok {
				add("Fast begins (sunset)", days[prev], "Shkiah")
			}
			add("Fast ends", last, "FastEnds", "Tzeis")
		case name == "Yom Kippur":
			add("Fast and Yom Kippur end", last, "ShabbatEndTime", "FastEnds")
		case strings.HasPrefix(name, "Fast"):
			add("Fast begins (dawn)", first, "AlosHashachar")
			add("Fast ends", last, "FastEnds", "Tzeis")
		case ev.Category == "erev":
			add("Search for chametz", first, "BedikatChametz")
			add("Last time to eat chametz", first, "LastEatingChametzTime")
			add("Burn chametz by", first, "BurnChametzTime")
			add("Candle lighting", first, "CandleLighting")
		case name == "Erev Pesach preparations":
			add("Search for chametz (tonight)", first, "BedikatChametz")
			add("Last time to eat chametz", first, "LastEatingChametzTime")
			add("Burn chametz by", first, "BurnChametzTime")
		case name == "Chanukah":
			// Chanukah candles are lit at nightfall (after Shabbat ends on
			// Saturday night); Chabad.org's feed does not give a separate time.
		case ev.Category == "chol-hamoed":
			for _, d := range run {
				add("Shabbat candle lighting", d, "CandleLighting")
				add("Shabbat ends", d, "ShabbatEndTime")
			}
		default:
			// Yom Tov: candles on a Yom Tov night come after nightfall from a
			// pre-existing flame; the final day carries the end time.
			for k, d := range run {
				final := k == len(run)-1 && !nextIsYomTov
				if final {
					add("Holiday ends", d, "ShabbatEndTime")
				} else {
					add("Light candles after (from a pre-existing flame)", d, "ShabbatEndTime")
				}
				add("Shabbat candle lighting (from a pre-existing flame)", d, "CandleLighting")
			}
		}
		events = append(events, ev)
		i = j + 1
	}
	return events
}

func computedAheadEvents(from, to time.Time, includeAll bool) []aheadEvent {
	f, t := jcal.FixedFromTime(from), jcal.FixedFromTime(to)
	occ := jcal.RoshChodeshBetween(f, t)
	// Chanukah: the first candle is lit on the evening of 24 Kislev.
	for y := jcal.FromFixed(f).Year; y <= jcal.FromFixed(t).Year; y++ {
		eve := jcal.HDate{Year: y, Month: jcal.Kislev, Day: 24}.Fixed()
		if eve >= f && eve <= t {
			name := "Erev Chanukah: first candle at nightfall"
			switch jcal.TimeFromFixed(eve).Weekday() {
			case time.Friday:
				// On Friday the menorah is lit before Shabbat candle lighting.
				name = "Erev Chanukah: first candle, lit before Shabbat candle lighting"
			case time.Saturday:
				name = "Erev Chanukah: first candle after Shabbat ends"
			}
			occ = append(occ, jcal.Occurrence{Name: name, Category: "chanukah", Fixed: eve, Hebrew: jcal.FromFixed(eve)})
		}
	}
	// Shabbat Mevarchim for months starting in (or just after) the window.
	h := jcal.FromFixed(f)
	m, y := h.Month, h.Year
	for i := 0; i < 8; i++ {
		if sm := jcal.ShabbatMevarchim(m, y); sm >= f && sm <= t {
			occ = append(occ, jcal.Occurrence{Name: "Shabbat Mevarchim " + jcal.MonthName(m, y), Category: "mevarchim", Fixed: sm, Hebrew: jcal.FromFixed(sm)})
		}
		m, y = jcal.NextMonth(m, y)
	}
	if includeAll {
		occ = append(occ, jcal.EventsBetween(jcal.MinorEvents, f, t)...)
		occ = append(occ, jcal.EventsBetween(jcal.ChassidicEvents, f, t)...)
	}
	out := make([]aheadEvent, 0, len(occ))
	for _, o := range occ {
		d := jcal.TimeFromFixed(o.Fixed).Format("2006-01-02")
		out = append(out, aheadEvent{Name: o.Name, Category: o.Category, Start: d, End: d, Hebrew: o.Hebrew.String(), Source: "computed", Times: make([]aheadTime, 0)})
	}
	return out
}

// aheadMarginDays is how far past the window events are fetched so that a
// holiday straddling the window edge reports its true end date.
const aheadMarginDays = 10

// eventsOverlapping keeps events whose [Start, End] overlaps [from, to],
// leaving their dates unclipped.
func eventsOverlapping(evs []aheadEvent, from, to time.Time) []aheadEvent {
	f, t := from.Format("2006-01-02"), to.Format("2006-01-02")
	out := make([]aheadEvent, 0, len(evs))
	for _, e := range evs {
		if e.End >= f && e.Start <= t {
			out = append(out, e)
		}
	}
	return out
}

func mergeAheadRoshChodesh(evs []aheadEvent) []aheadEvent {
	sort.SliceStable(evs, func(i, j int) bool { return evs[i].Start < evs[j].Start })
	out := make([]aheadEvent, 0, len(evs))
	for _, e := range evs {
		if e.Category == "rosh-chodesh" {
			merged := false
			for k := len(out) - 1; k >= 0 && k >= len(out)-3; k-- {
				if out[k].Name == e.Name {
					out[k].End = e.End
					merged = true
					break
				}
			}
			if merged {
				continue
			}
		}
		out = append(out, e)
	}
	return out
}

func newNovelAheadCmd(flags *rootFlags) *cobra.Command {
	var flagLocation string
	var days int
	var all bool
	var opt zmanimOptions

	cmd := &cobra.Command{
		Use:   "ahead",
		Short: "Every fast, erev Yom Tov, Yom Tov end, Rosh Chodesh and Erev Pesach in the next N days, each with the times it needs",
		Long: strings.Trim(`
Looks ahead N days (default 30) and lists every fast, erev Yom Tov, Yom Tov,
Rosh Chodesh, Shabbat Mevarchim and the first Chanukah candle, each with the
Chabad.org times it needs: candle lighting, second-night "light after (from a
pre-existing flame)", holiday end, fast start and end, and the Erev Pesach
chametz deadlines. --all adds minor and Chassidic dates.

Use this command to see which fasts, erev-Yom-Tov days and holidays fall in the
next N days, with the times each one needs. Do NOT use it for this week's
Shabbat block or a newsletter; use 'bulletin' instead. Do NOT use it for the
next single zman; use 'next' instead.`, "\n"),
		Example: strings.Trim(`
  chabad-pp-cli ahead --location 11213 --days 30
  chabad-pp-cli ahead --location city:247 --days 60 --all --json`, "\n"),
		Annotations: map[string]string{"mcp:read-only": "true", "pp:data-source": "live"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return writeDryRun(cmd.OutOrStdout(), flags, "ahead")
			}
			if days < 1 || days > 180 {
				return usageErr(fmt.Errorf("--days must be between 1 and 180"))
			}
			if err := opt.validate(); err != nil {
				return err
			}
			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			loc, err := resolveChabadLocation(ctx, flags, c, flagLocation)
			if err != nil {
				return err
			}
			from := todayDate()
			to := from.AddDate(0, 0, days-1)
			// Fetch a margin on both sides so events that straddle the window
			// keep their true start and end, then keep anything overlapping it.
			res, err := fetchZmanim(ctx, c, loc, from.AddDate(0, 0, -2), to.AddDate(0, 0, aheadMarginDays), opt)
			if err != nil {
				return classifyAPIError(cmd.OutOrStdout(), err, flags)
			}
			evs := buildAheadEvents(res.Days)
			evs = append(evs, computedAheadEvents(from.AddDate(0, 0, -2), to.AddDate(0, 0, aheadMarginDays), all)...)
			kept := eventsOverlapping(mergeAheadRoshChodesh(evs), from, to)
			view := aheadView{Location: res.Location, Details: res.Details, From: from.Format("2006-01-02"), To: to.Format("2006-01-02"), Events: kept}
			if !wantsHumanTable(cmd.OutOrStdout(), flags) {
				return printJSONFiltered(cmd.OutOrStdout(), view, flags)
			}
			w := cmd.OutOrStdout()
			writeHeader(w, fmt.Sprintf("Next %d days · %s", days, view.Location), view.Details)
			if len(view.Events) == 0 {
				fmt.Fprintln(w, "\nNothing coming up in this window.")
				return nil
			}
			for _, e := range view.Events {
				when := shortDate(e.Start)
				if e.End != e.Start {
					when += " – " + shortDate(e.End)
				}
				fmt.Fprintf(w, "\n%s  %s  (%s)\n", bold(when), e.Name, e.Hebrew)
				for _, t := range e.Times {
					fmt.Fprintf(w, "    %s: %s %s\n", t.Label, shortDate(t.Date), t.Time)
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&flagLocation, "location", "", locationFlagHelp)
	cmd.Flags().IntVar(&days, "days", 30, "How many days ahead to look (max 180)")
	cmd.Flags().BoolVar(&all, "all", false, "Also include minor and Chassidic dates")
	cmd.Flags().IntVar(&opt.CandleMinutes, "candle-minutes", 0, candleMinutesHelp)
	cmd.Flags().IntVar(&opt.ShabbatEndsMinutes, "shabbat-ends-minutes", 0, shabbatEndsMinutesHelp)
	return cmd
}
