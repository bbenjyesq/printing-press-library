// Copyright 2026 Bob Benjy and contributors. Licensed under Apache-2.0. See LICENSE.
// pp:data-source live

package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newShabbatCmd(flags *rootFlags) *cobra.Command {
	var location, from string
	var weeks int
	var opt zmanimOptions
	cmd := &cobra.Command{
		Use:   "shabbat",
		Short: "Shabbat and Yom Tov candle-lighting and end times, with the parsha, from Chabad.org",
		Long: strings.Trim(`
Candle-lighting and Shabbat/holiday end times as Chabad.org publishes them
(18 minutes before sunset in most places, 40 in Jerusalem; Chabad Shabbat-end
time). Any Yom Tov inside the window is included, with "light from a
pre-existing flame" marked on second nights.

Use this command for the times themselves. For a newsletter-ready weekly
block, use 'bulletin'; for holidays and fasts over the next month, use 'ahead'.`, "\n"),
		Example: strings.Trim(`
  chabad-pp-cli shabbat --location 11213
  chabad-pp-cli shabbat --location "Los Angeles" --weeks 4
  chabad-pp-cli shabbat --location 11213 --candle-minutes 40 --json`, "\n"),
		Annotations: map[string]string{"mcp:read-only": "true", "pp:data-source": "live"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return writeDryRun(cmd.OutOrStdout(), flags, "shabbat")
			}
			if weeks < 1 || weeks > 52 {
				return usageErr(fmt.Errorf("--weeks must be between 1 and 52"))
			}
			if err := opt.validate(); err != nil {
				return err
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
			loc, err := resolveChabadLocation(ctx, flags, c, location)
			if err != nil {
				return err
			}
			res, err := fetchCandles(ctx, c, loc, start, weeks, opt)
			if err != nil {
				return classifyAPIError(cmd.OutOrStdout(), err, flags)
			}
			if !wantsHumanTable(cmd.OutOrStdout(), flags) {
				return printJSONFiltered(cmd.OutOrStdout(), res, flags)
			}
			w := cmd.OutOrStdout()
			writeHeader(w, res.Location, res.Details)
			if len(res.Groups) == 0 {
				fmt.Fprintln(w, "\nNo Shabbat or holiday times returned for this window.")
				return nil
			}
			for _, g := range res.Groups {
				fmt.Fprintln(w)
				title := g.Title
				if g.Also != "" {
					title += " · " + g.Also
				}
				fmt.Fprintln(w, bold(title))
				tw := newTabWriter(w)
				for _, e := range g.Events {
					label := e.Label
					if e.FromPreexistingFlame {
						label += " (from a pre-existing flame)"
					}
					extra := ""
					if e.ShabbatMevarchim {
						extra = "Shabbat Mevarchim"
					}
					fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\t%s\n", shortDate(e.Date), e.Hebrew, label, e.Time, extra)
				}
				_ = tw.Flush()
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&location, "location", "", locationFlagHelp)
	cmd.Flags().IntVar(&weeks, "weeks", 1, "Number of Shabbat weeks to show (holidays inside the window are added)")
	cmd.Flags().StringVar(&from, "from", "", "Start from this date, YYYY-MM-DD (default: today)")
	cmd.Flags().IntVar(&opt.CandleMinutes, "candle-minutes", 0, candleMinutesHelp)
	cmd.Flags().IntVar(&opt.ShabbatEndsMinutes, "shabbat-ends-minutes", 0, shabbatEndsMinutesHelp)
	return cmd
}

func init() {
	registerNovelCommand(func(root *cobra.Command, flags *rootFlags) {
		addNovelCommandIfAbsent(root, newShabbatCmd(flags))
	})
}
