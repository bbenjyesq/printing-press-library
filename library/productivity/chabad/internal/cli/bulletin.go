// Copyright 2026 Bob Benjy and contributors. Licensed under Apache-2.0. See LICENSE.
// pp:data-source live

package cli

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/mvanhorn/printing-press-library/library/productivity/chabad/internal/jcal"

	"github.com/spf13/cobra"
)

type bulletinTime struct {
	Date          string `json:"date"`
	Hebrew        string `json:"hebrew_date"`
	Label         string `json:"label"`
	Time          string `json:"time"`
	ChangeMinutes *int   `json:"change_minutes,omitempty"`
}

type bulletinLocation struct {
	Location       string         `json:"location"`
	Details        string         `json:"details,omitempty"`
	CandleLighting *bulletinTime  `json:"candle_lighting,omitempty"`
	ShabbatEnds    *bulletinTime  `json:"shabbat_ends,omitempty"`
	HolidayTimes   []bulletinTime `json:"holiday_times"`
	Error          string         `json:"error,omitempty"`
}

type bulletinMevarchim struct {
	Month       string   `json:"month"`
	RoshChodesh []string `json:"rosh_chodesh"`
	Molad       string   `json:"molad"`
}

type bulletinView struct {
	ShabbatDate   string             `json:"shabbat_date"`
	HebrewDate    string             `json:"hebrew_date"`
	Parsha        string             `json:"parsha,omitempty"`
	Also          string             `json:"also,omitempty"`
	Mevarchim     *bulletinMevarchim `json:"shabbat_mevarchim,omitempty"`
	Locations     []bulletinLocation `json:"locations"`
	ComingUp      []aheadEvent       `json:"coming_up"`
	FetchFailures []fetchFailure     `json:"fetch_failures,omitempty"`
}

// weekShabbat finds this week's Shabbat group (the first whose events end on
// or after ref) and the Shabbat group before it.
func weekShabbat(groups []candleGroup, ref string, now time.Time) (cur, prev *candleGroup, holidays []candleGroup) {
	for i := range groups {
		g := &groups[i]
		if len(g.Events) == 0 {
			continue
		}
		lastEv := g.Events[len(g.Events)-1]
		// A group is past once its last date is before ref, or (on Saturday
		// night) once its end time has already passed.
		past := lastEv.Date < ref || (!now.IsZero() && lastEv.At != nil && lastEv.At.Before(now))
		isShabbat := g.Type == "shabbat"
		switch {
		case past && isShabbat:
			prev = g
		case !past && isShabbat && cur == nil:
			cur = g
		case !past && !isShabbat:
			holidays = append(holidays, *g)
		}
	}
	return cur, prev, holidays
}

func eventOf(g *candleGroup, kind string) *candleEvent {
	if g == nil {
		return nil
	}
	for i := range g.Events {
		if g.Events[i].Kind == kind {
			return &g.Events[i]
		}
	}
	return nil
}

// weekChange is the clock-time change between two events a week apart.
func weekChange(cur, prev *candleEvent) *int {
	if cur == nil || prev == nil || cur.At == nil || prev.At == nil {
		return nil
	}
	diff := cur.At.Sub(*prev.At) - 7*24*time.Hour
	n := int(diff.Round(time.Minute) / time.Minute)
	if n < -120 || n > 120 {
		return nil
	}
	return &n
}

func newNovelBulletinCmd(flags *rootFlags) *cobra.Command {
	var locations []string
	var format string
	var days int
	var opt zmanimOptions

	cmd := &cobra.Command{
		Use:   "bulletin",
		Short: "A copy-paste weekly block for newsletters and shul signage: Shabbat times with week-over-week change, parsha",
		Long: strings.Trim(`
Builds this week's Shabbat block for newsletters and shul signage: the parsha
and Hebrew date, candle lighting and Shabbat end for each location with the
change from last week, a Shabbat Mevarchim notice (Rosh Chodesh days and the
molad) when it applies, any Yom Tov times this week, and what is coming up in
the next --days (holidays, fasts, Rosh Chodesh). Repeat --location for several
cities (each keeps its own candle-lighting custom). --format md (default on a
terminal) or plain prints the block even when piped; --json gives the data.

Use this command to generate a composed weekly newsletter or signage block for
one or more locations. Do NOT use it for raw candle-lighting times only; use
'shabbat' instead.`, "\n"),
		Example: strings.Trim(`
  chabad-pp-cli bulletin --location 11213 --location 247
  chabad-pp-cli bulletin --location 11213 --format plain > shabbat.txt
  chabad-pp-cli bulletin --location 11213 --json`, "\n"),
		Annotations: map[string]string{"mcp:read-only": "true", "pp:data-source": "live"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return writeDryRun(cmd.OutOrStdout(), flags, "bulletin")
			}
			if format != "" && format != "md" && format != "plain" {
				return usageErr(fmt.Errorf("--format must be md or plain"))
			}
			if days < 1 || days > 60 {
				return usageErr(fmt.Errorf("--days must be between 1 and 60"))
			}
			if err := opt.validate(); err != nil {
				return err
			}
			if len(locations) > 8 {
				return usageErr(fmt.Errorf("at most 8 locations per bulletin"))
			}
			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			inputs := locations
			if len(inputs) == 0 {
				inputs = []string{""} // saved default
			}
			today := todayDate()
			ref := today.Format("2006-01-02")
			view := bulletinView{Locations: make([]bulletinLocation, 0, len(inputs)), ComingUp: make([]aheadEvent, 0)}
			var firstLoc *chabadLocation
			for _, in := range inputs {
				loc, err := resolveChabadLocation(ctx, flags, c, in)
				if err != nil {
					if len(inputs) == 1 {
						return err
					}
					view.FetchFailures = append(view.FetchFailures, fetchFailure{Source: in, Error: err.Error()})
					continue
				}
				res, err := fetchCandles(ctx, c, loc, today.AddDate(0, 0, -7), 3, opt)
				if err != nil {
					if len(inputs) == 1 {
						return classifyAPIError(cmd.OutOrStdout(), err, flags)
					}
					view.FetchFailures = append(view.FetchFailures, fetchFailure{Source: firstNonEmpty(loc.Name, loc.ID), Error: err.Error()})
					continue
				}
				if firstLoc == nil {
					l := loc
					firstLoc = &l
				}
				cur, prev, hol := weekShabbat(res.Groups, ref, time.Now())
				bl := bulletinLocation{Location: res.Location, Details: res.Details, HolidayTimes: make([]bulletinTime, 0)}
				if cl := eventOf(cur, "candle-lighting"); cl != nil {
					bl.CandleLighting = &bulletinTime{Date: cl.Date, Hebrew: cl.Hebrew, Label: cl.Label, Time: cl.Time, ChangeMinutes: weekChange(cl, eventOf(prev, "candle-lighting"))}
				}
				if se := eventOf(cur, "ends"); se != nil {
					bl.ShabbatEnds = &bulletinTime{Date: se.Date, Hebrew: se.Hebrew, Label: se.Label, Time: se.Time, ChangeMinutes: weekChange(se, eventOf(prev, "ends"))}
					if view.ShabbatDate == "" {
						view.ShabbatDate, view.HebrewDate = se.Date, se.Hebrew
						if cur != nil {
							view.Parsha, view.Also = cur.Parsha, cur.Also
						}
						if se.ShabbatMevarchim {
							view.Mevarchim = mevarchimFor(se.Date)
						}
					}
				}
				weekEnd := today.AddDate(0, 0, 7).Format("2006-01-02")
				for _, g := range hol {
					for _, e := range g.Events {
						if e.Date <= weekEnd {
							label := g.Title + ": " + e.Label
							if e.FromPreexistingFlame {
								label += " (from a pre-existing flame)"
							}
							bl.HolidayTimes = append(bl.HolidayTimes, bulletinTime{Date: e.Date, Hebrew: e.Hebrew, Label: label, Time: e.Time})
						}
					}
				}
				view.Locations = append(view.Locations, bl)
			}
			if len(view.Locations) == 0 {
				return apiErr(fmt.Errorf("no location could be loaded: %v", view.FetchFailures))
			}
			if firstLoc != nil {
				// "Coming up" starts tomorrow; fetch a margin so events keep true ends.
				start, end := today.AddDate(0, 0, 1), today.AddDate(0, 0, days)
				res, err := fetchZmanim(ctx, c, *firstLoc, start.AddDate(0, 0, -2), end.AddDate(0, 0, aheadMarginDays), opt)
				if err != nil {
					view.FetchFailures = append(view.FetchFailures, fetchFailure{Source: "coming-up", Error: err.Error()})
				} else {
					evs := buildAheadEvents(res.Days)
					evs = append(evs, computedAheadEvents(start.AddDate(0, 0, -2), end.AddDate(0, 0, aheadMarginDays), false)...)
					upcoming := make([]aheadEvent, 0)
					for _, e := range eventsOverlapping(mergeAheadRoshChodesh(evs), start, end) {
						if e.Start >= start.Format("2006-01-02") { // already-running events are not "coming up"
							upcoming = append(upcoming, e)
						}
					}
					view.ComingUp = upcoming
				}
			}
			for _, f := range view.FetchFailures {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: %s: %s\n", f.Source, f.Error)
			}
			if format == "" && !wantsHumanTable(cmd.OutOrStdout(), flags) {
				return printJSONFiltered(cmd.OutOrStdout(), view, flags)
			}
			if flags.asJSON {
				return printJSONFiltered(cmd.OutOrStdout(), view, flags)
			}
			renderBulletin(cmd.OutOrStdout(), view, format != "plain")
			return nil
		},
	}
	cmd.Flags().StringArrayVar(&locations, "location", nil, "Location (repeatable): ZIP, city name, or city:<id>; defaults to the saved location")
	cmd.Flags().StringVar(&format, "format", "", "Text format: md (markdown) or plain; prints text even when piped")
	cmd.Flags().IntVar(&days, "days", 14, "How many days of upcoming holidays and fasts to include")
	cmd.Flags().IntVar(&opt.CandleMinutes, "candle-minutes", 0, candleMinutesHelp)
	cmd.Flags().IntVar(&opt.ShabbatEndsMinutes, "shabbat-ends-minutes", 0, shabbatEndsMinutesHelp)
	return cmd
}

// mevarchimFor describes the month blessed on the Shabbat of date iso.
func mevarchimFor(iso string) *bulletinMevarchim {
	t, err := time.Parse("2006-01-02", iso)
	if err != nil {
		return nil
	}
	h := jcal.FromTime(t, false)
	m, y := jcal.NextMonth(h.Month, h.Year)
	if m == jcal.Tishrei {
		return nil
	}
	mv := &bulletinMevarchim{Month: jcal.MonthName(m, y), Molad: jcal.MoladOf(m, y).String(), RoshChodesh: make([]string, 0, 2)}
	for _, f := range jcal.RoshChodesh(m, y) {
		mv.RoshChodesh = append(mv.RoshChodesh, jcal.TimeFromFixed(f).Format("2006-01-02"))
	}
	return mv
}

func renderBulletin(w io.Writer, v bulletinView, md bool) {
	b := func(s string) string {
		if md {
			return "**" + s + "**"
		}
		return s
	}
	title := "Shabbat"
	if v.Parsha != "" {
		title = "Shabbat Parshat " + v.Parsha
	}
	line := b(title)
	if v.ShabbatDate != "" {
		line += fmt.Sprintf(" — %s (%s)", v.HebrewDate, longDate(v.ShabbatDate))
	}
	if v.Also != "" {
		line += " · " + v.Also
	}
	fmt.Fprintln(w, line)
	if v.Mevarchim != nil {
		rc := make([]string, len(v.Mevarchim.RoshChodesh))
		for i, d := range v.Mevarchim.RoshChodesh {
			rc[i] = shortDate(d)
		}
		fmt.Fprintf(w, "Shabbat Mevarchim %s: Rosh Chodesh %s. Molad: %s\n", v.Mevarchim.Month, strings.Join(rc, " & "), v.Mevarchim.Molad)
	}
	fmt.Fprintln(w)
	change := func(t *bulletinTime) string {
		if t == nil {
			return "—"
		}
		s := t.Time
		if t.ChangeMinutes != nil {
			s += " (" + signedMinutes(*t.ChangeMinutes) + ")"
		}
		return s
	}
	if md {
		fmt.Fprintln(w, "| Location | Candle lighting | Shabbat ends |")
		fmt.Fprintln(w, "|---|---|---|")
		for _, l := range v.Locations {
			fmt.Fprintf(w, "| %s | %s | %s |\n", l.Location, change(l.CandleLighting), change(l.ShabbatEnds))
		}
	} else {
		for _, l := range v.Locations {
			fmt.Fprintf(w, "%s: candle lighting %s; Shabbat ends %s\n", l.Location, change(l.CandleLighting), change(l.ShabbatEnds))
		}
	}
	for _, l := range v.Locations {
		if len(l.HolidayTimes) == 0 {
			continue
		}
		fmt.Fprintf(w, "\n%s (%s)\n", b("Yom Tov times"), l.Location)
		for _, t := range l.HolidayTimes {
			fmt.Fprintf(w, "- %s %s: %s\n", shortDate(t.Date), t.Label, t.Time)
		}
	}
	if len(v.ComingUp) > 0 {
		fmt.Fprintf(w, "\n%s\n", b("Coming up"))
		for _, e := range v.ComingUp {
			when := shortDate(e.Start)
			if e.End != e.Start {
				when += " – " + shortDate(e.End)
			}
			extra := ""
			if len(e.Times) > 0 {
				parts := make([]string, 0, len(e.Times))
				for _, t := range e.Times {
					parts = append(parts, fmt.Sprintf("%s %s", t.Label, t.Time))
				}
				extra = " — " + strings.Join(parts, "; ")
			}
			fmt.Fprintf(w, "- %s: %s%s\n", when, e.Name, extra)
		}
	}
}
