// Copyright 2026 Bob Benjy and contributors. Licensed under Apache-2.0. See LICENSE.
// pp:data-source local

package cli

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mvanhorn/printing-press-library/library/productivity/chabad/internal/jcal"
	"github.com/mvanhorn/printing-press-library/library/productivity/chabad/internal/store"

	"github.com/spf13/cobra"
)

var occasionKinds = map[string]bool{"yahrzeit": true, "birthday": true, "anniversary": true}

type occasionDue struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Kind       string `json:"kind"`
	Date       string `json:"date"`
	Weekday    string `json:"weekday"`
	BeginsEven string `json:"begins_evening_of"`
	HebrewDate string `json:"hebrew_date"`
	Years      int    `json:"years"`
	DaysAway   int    `json:"days_away"`
	Milestone  string `json:"milestone,omitempty"`
	Notes      string `json:"notes,omitempty"`
}

func occasionHDate(o store.Occasion) jcal.HDate {
	return jcal.HDate{Year: o.HebrewYear, Month: o.HebrewMonth, Day: o.HebrewDay}
}

// upcomingOccasions expands saved occasions into dates within [from, from+days).
func upcomingOccasions(list []store.Occasion, from time.Time, days int) []occasionDue {
	out := make([]occasionDue, 0)
	start := jcal.FixedFromTime(from)
	end := start + days - 1
	y0, y1 := jcal.FromFixed(start).Year, jcal.FromFixed(end).Year
	for _, o := range list {
		orig := occasionHDate(o)
		rule := jcal.Birthday
		if o.Kind == "yahrzeit" {
			rule = jcal.Yahrzeit
		}
		for y := y0; y <= y1; y++ {
			if y <= orig.Year {
				continue
			}
			f := rule(orig, y)
			if f < start || f > end {
				continue
			}
			h := jcal.FromFixed(f)
			t := jcal.TimeFromFixed(f)
			due := occasionDue{
				ID: o.ID, Name: o.Name, Kind: o.Kind, Date: t.Format("2006-01-02"), Weekday: t.Weekday().String(),
				BeginsEven: jcal.TimeFromFixed(f - 1).Format("2006-01-02"), HebrewDate: h.String(),
				Years: y - orig.Year, DaysAway: f - start, Notes: o.Notes,
			}
			if o.Kind == "birthday" {
				switch due.Years {
				case 12:
					due.Milestone = "12th Hebrew birthday (bat mitzvah age)"
				case 13:
					due.Milestone = "13th Hebrew birthday (bar mitzvah age)"
				}
			}
			out = append(out, due)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Date < out[j].Date })
	return out
}

func occasionsDBPath(dbPath string) string {
	if dbPath != "" {
		return dbPath
	}
	return defaultDBPath("chabad-pp-cli")
}

func openOccasionsStore(ctx context.Context, dbPath string) (*store.Store, error) {
	db, err := store.OpenWithContext(ctx, dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening local store %s: %w", dbPath, err)
	}
	return db, nil
}

func runOccasionsUpcoming(cmd *cobra.Command, flags *rootFlags, dbPath string, days int) error {
	if days < 1 || days > 400 {
		return usageErr(fmt.Errorf("--days must be between 1 and 400"))
	}
	path := occasionsDBPath(dbPath)
	if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
		fmt.Fprintf(cmd.ErrOrStderr(), "no occasions saved yet (no local store at %s)\nadd one: chabad-pp-cli occasions add \"Name\" --kind yahrzeit --date 2015-03-10\n", path)
		if !wantsHumanTable(cmd.OutOrStdout(), flags) {
			return printJSONFiltered(cmd.OutOrStdout(), make([]occasionDue, 0), flags)
		}
		return nil
	}
	ctx, cancel := boundCtx(cmd.Context(), flags)
	defer cancel()
	db, err := openOccasionsStore(ctx, path)
	if err != nil {
		return err
	}
	defer db.Close()
	list, err := db.ListOccasions(ctx)
	if err != nil {
		return err
	}
	due := upcomingOccasions(list, todayDate(), days)
	if !wantsHumanTable(cmd.OutOrStdout(), flags) {
		return printJSONFiltered(cmd.OutOrStdout(), due, flags)
	}
	w := cmd.OutOrStdout()
	if len(list) == 0 {
		fmt.Fprintln(w, "No occasions saved yet. Add one: chabad-pp-cli occasions add \"Name\" --kind yahrzeit --date 2015-03-10")
		return nil
	}
	if len(due) == 0 {
		fmt.Fprintf(w, "Nothing in the next %d days (%d saved).\n", days, len(list))
		return nil
	}
	tw := newTabWriter(w)
	for _, d := range due {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%d years\tbegins evening of %s\t%s\n", shortDate(d.Date), bold(d.Name), d.Kind, d.HebrewDate, d.Years, shortDate(d.BeginsEven), d.Milestone)
	}
	return tw.Flush()
}

func newNovelOccasionsCmd(flags *rootFlags) *cobra.Command {
	var dbPath string
	var days int

	cmd := &cobra.Command{
		Use:   "occasions",
		Short: "Keep a local list of family yahrzeits, Hebrew birthdays and bar/bat-mitzvah dates and see which fall in the coming days.",
		Long: strings.Trim(`
A saved, local list of people's Hebrew-date occasions: yahrzeits, Hebrew
birthdays (with bar/bat-mitzvah milestones) and anniversaries. Bare
'occasions' shows what falls in the next --days (default 30); use the
subcommands to add, list, and remove entries. Data stays in the CLI's local
SQLite store; nothing is sent anywhere.

Use this command to store and list many people's recurring Hebrew-date events
and see what is coming up. Do NOT use it for a one-off calculation for a single
person; use 'yahrzeit' or 'birthday' instead.`, "\n"),
		Example: strings.Trim(`
  chabad-pp-cli occasions upcoming --days 30
  chabad-pp-cli occasions add "Zeide Moshe" --kind yahrzeit --date 2015-03-10 --after-sunset
  chabad-pp-cli occasions list --json`, "\n"),
		Annotations: map[string]string{"mcp:read-only": "true", "pp:data-source": "local"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return writeDryRun(cmd.OutOrStdout(), flags, "occasions")
			}
			if len(args) > 0 {
				return usageErr(fmt.Errorf("unknown occasions subcommand %q (try add, list, remove, upcoming)", args[0]))
			}
			return runOccasionsUpcoming(cmd, flags, dbPath, days)
		},
	}
	cmd.PersistentFlags().StringVar(&dbPath, "db", "", "Local store path (default: the CLI's data.db)")
	cmd.Flags().IntVar(&days, "days", 30, "How many days ahead to look")
	cmd.AddCommand(newOccasionsUpcomingCmd(flags, &dbPath))
	cmd.AddCommand(newOccasionsAddCmd(flags, &dbPath))
	cmd.AddCommand(newOccasionsListCmd(flags, &dbPath))
	cmd.AddCommand(newOccasionsRemoveCmd(flags, &dbPath))
	return cmd
}

func newOccasionsUpcomingCmd(flags *rootFlags, dbPath *string) *cobra.Command {
	var days int
	cmd := &cobra.Command{
		Use:         "upcoming",
		Short:       "Saved yahrzeits, Hebrew birthdays and anniversaries that fall in the next N days",
		Example:     "  chabad-pp-cli occasions upcoming --days 30",
		Annotations: map[string]string{"mcp:read-only": "true", "pp:data-source": "local", "pp:happy-args": "--days=30"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return writeDryRun(cmd.OutOrStdout(), flags, "occasions upcoming")
			}
			return runOccasionsUpcoming(cmd, flags, *dbPath, days)
		},
	}
	cmd.Flags().IntVar(&days, "days", 30, "How many days ahead to look")
	return cmd
}

func newOccasionsAddCmd(flags *rootFlags, dbPath *string) *cobra.Command {
	var kind, date, hebrew, notes string
	var afterSunset bool
	cmd := &cobra.Command{
		Use:   "add [name]",
		Short: "Save a person's yahrzeit, Hebrew birthday, or anniversary",
		Example: strings.Trim(`
  chabad-pp-cli occasions add "Zeide Moshe" --kind yahrzeit --date 2015-03-10 --after-sunset
  chabad-pp-cli occasions add "Chaya" --kind birthday --hebrew "11 Nissan 5772"`, "\n"),
		Annotations: map[string]string{"mcp:local-write": "true", "pp:data-source": "local"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return writeDryRun(cmd.OutOrStdout(), flags, "occasions add")
			}
			if len(args) != 1 || strings.TrimSpace(args[0]) == "" {
				return usageErr(fmt.Errorf("pass the person's name (quote names with spaces)"))
			}
			if !occasionKinds[kind] {
				return usageErr(fmt.Errorf("--kind must be yahrzeit, birthday, or anniversary"))
			}
			if (date == "") == (hebrew == "") {
				return usageErr(fmt.Errorf("pass exactly one of --date YYYY-MM-DD or --hebrew \"<day> <month> <year>\""))
			}
			var h jcal.HDate
			greg := ""
			if hebrew != "" {
				parsed, err := jcal.Parse(hebrew, 0)
				if err != nil || parsed.Year == 0 {
					return usageErr(fmt.Errorf("--hebrew needs a full date with year, e.g. \"3 Tammuz 5754\""))
				}
				h = parsed
				greg = h.Time().Format("2006-01-02")
			} else {
				g, err := parseDateArg(date)
				if err != nil {
					return err
				}
				h = jcal.FromTime(g, afterSunset)
				greg = g.Format("2006-01-02")
			}
			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()
			path := occasionsDBPath(*dbPath)
			db, err := openOccasionsStore(ctx, path)
			if err != nil {
				return err
			}
			defer db.Close()
			o := store.Occasion{Name: strings.TrimSpace(args[0]), Kind: kind, HebrewYear: h.Year, HebrewMonth: h.Month, HebrewDay: h.Day, Gregorian: greg, AfterSunset: afterSunset, Notes: notes}
			id, err := db.AddOccasion(ctx, o)
			if err != nil {
				return err
			}
			o.ID = id
			view := map[string]any{"id": id, "name": o.Name, "kind": kind, "hebrew_date": h.String(), "gregorian_date": greg, "after_sunset": afterSunset}
			if !wantsHumanTable(cmd.OutOrStdout(), flags) {
				return printJSONFiltered(cmd.OutOrStdout(), view, flags)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Saved #%d: %s — %s on %s\n", id, o.Name, kind, h.String())
			return nil
		},
	}
	cmd.Flags().StringVar(&kind, "kind", "yahrzeit", "yahrzeit, birthday, or anniversary")
	cmd.Flags().StringVar(&date, "date", "", "Gregorian date of the original event, YYYY-MM-DD")
	cmd.Flags().StringVar(&hebrew, "hebrew", "", "Hebrew date of the original event instead, e.g. \"3 Tammuz 5754\"")
	cmd.Flags().BoolVar(&afterSunset, "after-sunset", false, "The original event was after sunset")
	cmd.Flags().StringVar(&notes, "notes", "", "Optional note (relationship, shul, etc.)")
	return cmd
}

func newOccasionsListCmd(flags *rootFlags, dbPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "list",
		Short:       "List every saved occasion",
		Example:     "  chabad-pp-cli occasions list --json",
		Annotations: map[string]string{"mcp:read-only": "true", "pp:data-source": "local"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return writeDryRun(cmd.OutOrStdout(), flags, "occasions list")
			}
			path := occasionsDBPath(*dbPath)
			if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
				fmt.Fprintf(cmd.ErrOrStderr(), "no occasions saved yet (no local store at %s)\n", path)
				if !wantsHumanTable(cmd.OutOrStdout(), flags) {
					return printJSONFiltered(cmd.OutOrStdout(), make([]store.Occasion, 0), flags)
				}
				return nil
			}
			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()
			db, err := openOccasionsStore(ctx, path)
			if err != nil {
				return err
			}
			defer db.Close()
			list, err := db.ListOccasions(ctx)
			if err != nil {
				return err
			}
			if !wantsHumanTable(cmd.OutOrStdout(), flags) {
				return printJSONFiltered(cmd.OutOrStdout(), list, flags)
			}
			if len(list) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No occasions saved yet.")
				return nil
			}
			tw := newTabWriter(cmd.OutOrStdout())
			for _, o := range list {
				fmt.Fprintf(tw, "#%d\t%s\t%s\t%s\t%s\n", o.ID, bold(o.Name), o.Kind, occasionHDate(o).String(), o.Notes)
			}
			return tw.Flush()
		},
	}
	return cmd
}

func newOccasionsRemoveCmd(flags *rootFlags, dbPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "remove [id]",
		Short:       "Remove a saved occasion by id (see 'occasions list')",
		Example:     "  chabad-pp-cli occasions remove 3",
		Annotations: map[string]string{"pp:data-source": "local"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return writeDryRun(cmd.OutOrStdout(), flags, "occasions remove")
			}
			if len(args) != 1 {
				return usageErr(fmt.Errorf("pass the occasion id"))
			}
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil || id < 1 {
				return usageErr(fmt.Errorf("invalid id %q", args[0]))
			}
			path := occasionsDBPath(*dbPath)
			if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
				return notFoundErr(fmt.Errorf("no occasion #%d (nothing saved yet)", id))
			}
			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()
			db, err := openOccasionsStore(ctx, path)
			if err != nil {
				return err
			}
			defer db.Close()
			ok, err := db.RemoveOccasion(ctx, id)
			if err != nil {
				return err
			}
			if !ok {
				return notFoundErr(fmt.Errorf("no occasion #%d", id))
			}
			if !wantsHumanTable(cmd.OutOrStdout(), flags) {
				return printJSONFiltered(cmd.OutOrStdout(), map[string]any{"id": id, "removed": true}, flags)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Removed occasion #%d\n", id)
			return nil
		},
	}
	return cmd
}
