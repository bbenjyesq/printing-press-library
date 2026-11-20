// Copyright 2026 Bob Benjy and contributors. Licensed under Apache-2.0. See LICENSE.
// pp:data-source live

package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

type parshaRow struct {
	Date    string `json:"date"`
	Weekday string `json:"weekday"`
	Hebrew  string `json:"hebrew_date"`
	Reading string `json:"reading"`
	Holiday string `json:"holiday,omitempty"`
}

type parshaView struct {
	Location string      `json:"location"`
	Note     string      `json:"note,omitempty"`
	Readings []parshaRow `json:"readings"`
}

func newParshaCmd(flags *rootFlags) *cobra.Command {
	var location, from string
	var weeks int
	var weekdays bool
	cmd := &cobra.Command{
		Use:   "parsha",
		Short: "Upcoming Shabbat Torah readings (Israel or Diaspora schedule by location)",
		Long: strings.Trim(`
The weekly parsha for upcoming Shabbatot as Chabad.org lists it, including
holiday readings. The schedule depends on location: Israel and the Diaspora
diverge after some Yom Tovim. Without a location (and no saved default) the
Diaspora schedule for Crown Heights (ZIP 11213) is used.

Pass --weekdays to include Monday/Thursday and holiday readings too.`, "\n"),
		Example: strings.Trim(`
  chabad-pp-cli parsha
  chabad-pp-cli parsha --weeks 6 --location city:247
  chabad-pp-cli parsha --weekdays --json`, "\n"),
		Annotations: map[string]string{"mcp:read-only": "true", "pp:data-source": "live"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return writeDryRun(cmd.OutOrStdout(), flags, "parsha")
			}
			if weeks < 1 || weeks > 26 {
				return usageErr(fmt.Errorf("--weeks must be between 1 and 26"))
			}
			start, err := parseDateArg(from)
			if err != nil {
				return err
			}
			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			view := parshaView{Readings: make([]parshaRow, 0)}
			loc, usedDefault, err := resolveLocationOrDiaspora(ctx, flags, c, location)
			if err != nil {
				return err
			}
			if usedDefault {
				view.Note = "no location given; showing the Diaspora schedule (Crown Heights). Pass --location for Israel."
			}
			// Cover `weeks` Saturdays from start.
			end := start.AddDate(0, 0, weeks*7-1)
			for end.Weekday() != 6 { // ensure the last Saturday is included
				end = end.AddDate(0, 0, 1)
			}
			res, err := fetchZmanim(ctx, c, loc, start, end, zmanimOptions{})
			if err != nil {
				return classifyAPIError(cmd.OutOrStdout(), err, flags)
			}
			view.Location = res.Location
			for _, d := range res.Days {
				if d.Reading == "" {
					continue
				}
				if !weekdays && d.Weekday != "Saturday" {
					continue
				}
				view.Readings = append(view.Readings, parshaRow{Date: d.Date, Weekday: d.Weekday, Hebrew: d.Hebrew, Reading: d.Reading, Holiday: d.Holiday})
			}
			if !wantsHumanTable(cmd.OutOrStdout(), flags) {
				return printJSONFiltered(cmd.OutOrStdout(), view, flags)
			}
			w := cmd.OutOrStdout()
			if view.Note != "" {
				fmt.Fprintln(cmd.ErrOrStderr(), view.Note)
			}
			if len(view.Readings) == 0 {
				fmt.Fprintln(w, "No readings returned for this window.")
				return nil
			}
			tw := newTabWriter(w)
			for _, r := range view.Readings {
				extra := ""
				if r.Holiday != "" && r.Holiday != r.Reading {
					extra = r.Holiday
				}
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", shortDate(r.Date), r.Hebrew, bold(r.Reading), extra)
			}
			return tw.Flush()
		},
	}
	cmd.Flags().StringVar(&location, "location", "", locationFlagHelp)
	cmd.Flags().IntVar(&weeks, "weeks", 1, "Number of upcoming Shabbatot to list")
	cmd.Flags().StringVar(&from, "from", "", "Start from this date, YYYY-MM-DD (default: today)")
	cmd.Flags().BoolVar(&weekdays, "weekdays", false, "Also list Monday/Thursday and holiday readings")
	return cmd
}

func init() {
	registerNovelCommand(func(root *cobra.Command, flags *rootFlags) {
		addNovelCommandIfAbsent(root, newParshaCmd(flags))
	})
}
