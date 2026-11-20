// Copyright 2026 Bob Benjy and contributors. Licensed under Apache-2.0. See LICENSE.
// pp:data-source live

package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/mvanhorn/printing-press-library/library/productivity/chabad/internal/cliutil"

	"github.com/spf13/cobra"
)

type nextView struct {
	Location string     `json:"location"`
	Now      time.Time  `json:"now"`
	Filter   string     `json:"zman,omitempty"`
	Within   string     `json:"within,omitempty"`
	InWindow *bool      `json:"in_window,omitempty"`
	Next     []nextZman `json:"next"`
	Note     string     `json:"note,omitempty"`
}

func newNovelNextCmd(flags *rootFlags) *cobra.Command {
	var flagLocation, flagZman, flagWithin string
	var count int
	var opt zmanimOptions

	cmd := &cobra.Command{
		Use:   "next",
		Short: "The next halachic times and how many minutes until each, rolling into tomorrow's dawn",
		Long: strings.Trim(`
The next zmanim at a location and how long until each, rolling over into
tomorrow when today's are past. Filter to one zman with --zman (candle-lighting,
shabbat-ends, sunset, latest-shema, plag, nightfall, dawn, ...); with a filter
the search looks up to nine days ahead, so --zman candle-lighting finds the
coming Friday.

--within <duration> turns this into a trigger for scripts and cron: the
command exits 0 when the next matching zman is within the window and exits 1
when it is not (the JSON is printed either way).

Use this command for "what is the next halachic time and how long until it",
or to script a trigger. Do NOT use it for a full-day schedule overview; use
'today' instead.`, "\n"),
		Example: strings.Trim(`
  chabad-pp-cli next --location 11213 --zman candle-lighting --agent
  chabad-pp-cli next --location 11213 --count 5
  chabad-pp-cli next --location 11213 --zman candle-lighting --within 30m`, "\n"),
		Annotations: map[string]string{"mcp:read-only": "true", "pp:data-source": "live", "pp:typed-exit-codes": "0,1"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return writeDryRun(cmd.OutOrStdout(), flags, "next")
			}
			if count < 1 || count > 50 {
				return usageErr(fmt.Errorf("--count must be between 1 and 50"))
			}
			if err := opt.validate(); err != nil {
				return err
			}
			onlyType := ""
			if flagZman != "" {
				t, ok := resolveZmanType(flagZman)
				if !ok {
					return usageErr(fmt.Errorf("unknown --zman %q; try candle-lighting, shabbat-ends, sunset, nightfall, dawn, sunrise, latest-shema, midday, plag", flagZman))
				}
				onlyType = t
			}
			var within time.Duration
			if flagWithin != "" {
				d, err := cliutil.ParseDurationLoose(flagWithin)
				if err != nil || d <= 0 {
					return usageErr(fmt.Errorf("invalid --within %q: use a duration like 30m, 2h, or 1d", flagWithin))
				}
				within = d
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
			// Short ranges return per-day DST flags, which instants need.
			today := todayDate()
			end := today.AddDate(0, 0, 1)
			if onlyType != "" {
				end = today.AddDate(0, 0, 9)
			}
			res, err := fetchZmanim(ctx, c, loc, today.AddDate(0, 0, -1), end, opt)
			if err != nil {
				return classifyAPIError(cmd.OutOrStdout(), err, flags)
			}
			now := time.Now().UTC().Truncate(time.Second)
			view := nextView{Location: res.Location, Now: now, Filter: onlyType, Next: nextAfter(res.Days, now, onlyType, count)}
			if len(view.Next) == 0 {
				view.Note = "no upcoming zmanim found in the fetched window"
			}
			var notWithin bool
			if within > 0 {
				in := len(view.Next) > 0 && view.Next[0].At.Sub(now) <= within
				view.Within = flagWithin
				view.InWindow = &in
				notWithin = !in
			}
			if !wantsHumanTable(cmd.OutOrStdout(), flags) {
				if err := printJSONFiltered(cmd.OutOrStdout(), view, flags); err != nil {
					return err
				}
			} else {
				w := cmd.OutOrStdout()
				fmt.Fprintln(w, bold(res.Location))
				if len(view.Next) == 0 {
					fmt.Fprintln(w, view.Note)
				}
				tw := newTabWriter(w)
				for _, z := range view.Next {
					fmt.Fprintf(tw, "%s\t%s\t%s\tin %s\n", z.Title, shortDate(z.Date), z.Time, z.In)
				}
				_ = tw.Flush()
				if view.InWindow != nil {
					fmt.Fprintf(w, "within %s: %v\n", flagWithin, *view.InWindow)
				}
			}
			if notWithin {
				return &cliError{code: 1, err: fmt.Errorf("next %s is not within %s", firstNonEmpty(flagZman, "zman"), flagWithin)}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&flagLocation, "location", "", locationFlagHelp)
	cmd.Flags().StringVar(&flagZman, "zman", "", "Only this zman: candle-lighting, shabbat-ends, sunset, nightfall, dawn, sunrise, latest-shema, midday, plag, ...")
	cmd.Flags().IntVar(&count, "count", 3, "How many upcoming zmanim to list")
	cmd.Flags().StringVar(&flagWithin, "within", "", "Exit 0 only if the next matching zman is within this duration (e.g. 30m, 2h); otherwise exit 1")
	cmd.Flags().IntVar(&opt.CandleMinutes, "candle-minutes", 0, candleMinutesHelp)
	cmd.Flags().IntVar(&opt.ShabbatEndsMinutes, "shabbat-ends-minutes", 0, shabbatEndsMinutesHelp)
	return cmd
}
