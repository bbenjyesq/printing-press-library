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

type holidayRow struct {
	Name        string `json:"name"`
	Category    string `json:"category"`
	Start       string `json:"start"`
	End         string `json:"end"`
	Days        int    `json:"days"`
	HebrewStart string `json:"hebrew_start"`
	Source      string `json:"source"`
}

type holidaysView struct {
	Location string       `json:"location"`
	From     string       `json:"from"`
	To       string       `json:"to"`
	Note     string       `json:"note,omitempty"`
	Holidays []holidayRow `json:"holidays"`
}

// normalizeHolidayName folds "Chanukah Vayeishev" (Chanukah on a Shabbat)
// into "Chanukah" so multi-day runs collapse.
func normalizeHolidayName(name string) string {
	if strings.HasPrefix(name, "Chanukah ") {
		return "Chanukah"
	}
	return name
}

func holidayCategory(name string) string {
	switch {
	case strings.HasPrefix(name, "Fast"), name == "Yom Kippur":
		return "fast"
	case strings.HasPrefix(name, "Eve of"):
		return "erev"
	case strings.HasPrefix(name, "Intermediate"):
		return "chol-hamoed"
	}
	return "holiday"
}

// collapseHolidayRuns merges consecutive days that share a holiday name.
func collapseHolidayRuns(days []zmanimDay) []holidayRow {
	out := make([]holidayRow, 0)
	for _, d := range days {
		if d.Holiday == "" {
			continue
		}
		name := normalizeHolidayName(d.Holiday)
		if n := len(out); n > 0 && out[n-1].Name == name {
			prev, _ := time.Parse("2006-01-02", out[n-1].End)
			cur, _ := time.Parse("2006-01-02", d.Date)
			if cur.Sub(prev) == 24*time.Hour {
				out[n-1].End = d.Date
				out[n-1].Days++
				continue
			}
		}
		out = append(out, holidayRow{Name: name, Category: holidayCategory(name), Start: d.Date, End: d.Date, Days: 1, HebrewStart: d.Hebrew, Source: "chabad.org"})
	}
	return out
}

func computedHolidayRows(from, to time.Time) []holidayRow {
	f, t := jcal.FixedFromTime(from), jcal.FixedFromTime(to)
	occ := jcal.RoshChodeshBetween(f, t)
	occ = append(occ, jcal.EventsBetween(jcal.MinorEvents, f, t)...)
	occ = append(occ, jcal.EventsBetween(jcal.ChassidicEvents, f, t)...)
	rows := make([]holidayRow, 0, len(occ))
	for _, o := range occ {
		d := jcal.TimeFromFixed(o.Fixed).Format("2006-01-02")
		rows = append(rows, holidayRow{Name: o.Name, Category: o.Category, Start: d, End: d, Days: 1, HebrewStart: o.Hebrew.String(), Source: "computed"})
	}
	return mergeRoshChodesh(rows)
}

// mergeRoshChodesh joins the two days of a two-day Rosh Chodesh into one row.
func mergeRoshChodesh(rows []holidayRow) []holidayRow {
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].Start < rows[j].Start })
	out := make([]holidayRow, 0, len(rows))
	for _, r := range rows {
		if n := len(out); n > 0 && r.Category == "rosh-chodesh" && out[n-1].Name == r.Name {
			out[n-1].End = r.End
			out[n-1].Days++
			continue
		}
		out = append(out, r)
	}
	return out
}

func newHolidaysCmd(flags *rootFlags) *cobra.Command {
	var location, from, to string
	var year, hebrewYear, days int
	var all bool
	cmd := &cobra.Command{
		Use:   "holidays",
		Short: "Jewish holidays and fasts by date, as Chabad.org lists them (add --all for Rosh Chodesh and Chassidic dates)",
		Long: strings.Trim(`
Holidays and fasts for a window (default: the next 12 months) with
Chabad.org's names and dates. Multi-day holidays are collapsed into one row.
Israel and the Diaspora differ for second-day Yom Tov, so pass a location
(or save one); without it the Diaspora calendar is used.

--all adds days Chabad.org's zmanim feed does not flag, computed locally:
Rosh Chodesh, Tu B'Shvat, Purim Katan, Shushan Purim, Pesach Sheni, Lag
BaOmer, Tu B'Av, Hoshana Rabbah, and the Chabad calendar's Chassidic dates
(Yud-Tes Kislev, Yud Shevat, Gimmel Tammuz, Chai Elul, and others).`, "\n"),
		Example: strings.Trim(`
  chabad-pp-cli holidays
  chabad-pp-cli holidays --year 2027 --location city:247
  chabad-pp-cli holidays --hebrew-year 5787 --all --json`, "\n"),
		Annotations: map[string]string{"mcp:read-only": "true", "pp:data-source": "live"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return writeDryRun(cmd.OutOrStdout(), flags, "holidays")
			}
			start, end, err := holidayWindow(from, to, year, hebrewYear, days)
			if err != nil {
				return err
			}
			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			view := holidaysView{From: start.Format("2006-01-02"), To: end.Format("2006-01-02"), Holidays: make([]holidayRow, 0)}
			loc, usedDefault, err := resolveLocationOrDiaspora(ctx, flags, c, location)
			if err != nil {
				return err
			}
			if usedDefault {
				view.Note = "no location given; showing Diaspora dates. Pass --location city:247 (Jerusalem) for Israel."
			}
			res, err := fetchZmanim(ctx, c, loc, start, end, zmanimOptions{})
			if err != nil {
				return classifyAPIError(cmd.OutOrStdout(), err, flags)
			}
			view.Location = res.Location
			view.Holidays = collapseHolidayRuns(res.Days)
			if all {
				view.Holidays = append(view.Holidays, computedHolidayRows(start, end)...)
				sort.SliceStable(view.Holidays, func(i, j int) bool { return view.Holidays[i].Start < view.Holidays[j].Start })
			}
			if !wantsHumanTable(cmd.OutOrStdout(), flags) {
				return printJSONFiltered(cmd.OutOrStdout(), view, flags)
			}
			w := cmd.OutOrStdout()
			if view.Note != "" {
				fmt.Fprintln(cmd.ErrOrStderr(), view.Note)
			}
			writeHeader(w, fmt.Sprintf("Holidays %s to %s", view.From, view.To), view.Location)
			if len(view.Holidays) == 0 {
				fmt.Fprintln(w, "No holidays in this window.")
				return nil
			}
			tw := newTabWriter(w)
			for _, h := range view.Holidays {
				when := shortDate(h.Start)
				if h.End != h.Start {
					when += " – " + shortDate(h.End)
				}
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", when, h.HebrewStart, h.Name, h.Category)
			}
			return tw.Flush()
		},
	}
	cmd.Flags().StringVar(&location, "location", "", locationFlagHelp)
	cmd.Flags().IntVar(&year, "year", 0, "Gregorian year (e.g. 2027)")
	cmd.Flags().IntVar(&hebrewYear, "hebrew-year", 0, "Hebrew year (e.g. 5787), Rosh Hashanah to Elul")
	cmd.Flags().StringVar(&from, "from", "", "Window start, YYYY-MM-DD")
	cmd.Flags().StringVar(&to, "to", "", "Window end, YYYY-MM-DD")
	cmd.Flags().IntVar(&days, "days", 365, "Window length in days when no year or range is given")
	cmd.Flags().BoolVar(&all, "all", false, "Also list Rosh Chodesh, minor days, and Chassidic dates (computed locally)")
	return cmd
}

func holidayWindow(from, to string, year, hebrewYear, days int) (time.Time, time.Time, error) {
	switch {
	case year != 0 && hebrewYear != 0:
		return time.Time{}, time.Time{}, usageErr(fmt.Errorf("use either --year or --hebrew-year, not both"))
	case year != 0:
		if year < 1900 || year > 2200 {
			return time.Time{}, time.Time{}, usageErr(fmt.Errorf("--year must be between 1900 and 2200"))
		}
		return time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(year, 12, 31, 0, 0, 0, 0, time.UTC), nil
	case hebrewYear != 0:
		if hebrewYear < 5660 || hebrewYear > 5960 {
			return time.Time{}, time.Time{}, usageErr(fmt.Errorf("--hebrew-year must be between 5660 and 5960"))
		}
		return jcal.TimeFromFixed(jcal.NewYear(hebrewYear)), jcal.TimeFromFixed(jcal.NewYear(hebrewYear+1) - 1), nil
	}
	start, err := parseDateArg(from)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	var end time.Time
	if to != "" {
		if end, err = parseDateArg(to); err != nil {
			return time.Time{}, time.Time{}, err
		}
	} else {
		if days < 1 || days > 1100 {
			return time.Time{}, time.Time{}, usageErr(fmt.Errorf("--days must be between 1 and 1100"))
		}
		end = start.AddDate(0, 0, days-1)
	}
	if end.Before(start) {
		return time.Time{}, time.Time{}, usageErr(fmt.Errorf("--to is before --from"))
	}
	if end.Sub(start) > 1100*24*time.Hour {
		return time.Time{}, time.Time{}, usageErr(fmt.Errorf("window is limited to about three years"))
	}
	return start, end, nil
}

func init() {
	registerNovelCommand(func(root *cobra.Command, flags *rootFlags) {
		addNovelCommandIfAbsent(root, newHolidaysCmd(flags))
	})
}
