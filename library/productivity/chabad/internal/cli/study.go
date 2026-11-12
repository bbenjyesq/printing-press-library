// Copyright 2026 Bob Benjy and contributors. Licensed under Apache-2.0. See LICENSE.
// pp:data-source live

package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func filterStudyTracks(day *studyDay, tracks map[string]bool) {
	if len(tracks) == 0 {
		return
	}
	kept := make([]studyItem, 0, len(day.Items))
	for _, it := range day.Items {
		if tracks[it.Track] {
			kept = append(kept, it)
		}
	}
	day.Items = kept
}

func parseTrackFlag(s string) (map[string]bool, error) {
	out := map[string]bool{}
	if strings.TrimSpace(s) == "" {
		return out, nil
	}
	valid := map[string]bool{}
	for _, t := range studyTrackNames {
		valid[t] = true
	}
	aliases := map[string]string{"chitas": "", "rambam": "rambam-3", "mitzvot": "sefer-hamitzvot", "hayomyom": "hayom-yom", "psalms": "tehillim"}
	for _, raw := range strings.Split(s, ",") {
		t := strings.ToLower(strings.TrimSpace(raw))
		if t == "" {
			continue
		}
		if t == "chitas" {
			out["chumash"], out["tehillim"], out["tanya"] = true, true, true
			continue
		}
		if a, ok := aliases[t]; ok && a != "" {
			t = a
		}
		if !valid[t] {
			return nil, usageErr(fmt.Errorf("unknown --track %q; choose from %s, or chitas", raw, strings.Join(studyTrackNames, ", ")))
		}
		out[t] = true
	}
	return out, nil
}

func newStudyCmd(flags *rootFlags) *cobra.Command {
	var days int
	var track string
	cmd := &cobra.Command{
		Use:   "study [YYYY-MM-DD]",
		Short: "Chabad.org daily study: Chumash, Tehillim, Tanya (Chitas), Hayom Yom, Sefer HaMitzvot, Rambam",
		Long: strings.Trim(`
The daily-study schedule Chabad.org publishes for any date: Chumash with Rashi
(parsha, aliyah, verses), Tehillim (monthly cycle), Tanya, Hayom Yom, Sefer
HaMitzvot, and Rambam (1-chapter and 3-chapter tracks), each with a link to
the text on Chabad.org, plus the Hebrew date. Filter with --track (chitas =
Chumash + Tehillim + Tanya).`, "\n"),
		Example: strings.Trim(`
  chabad-pp-cli study
  chabad-pp-cli study 2026-12-25 --track chitas
  chabad-pp-cli study --days 7 --track rambam-3 --json`, "\n"),
		Annotations: map[string]string{"mcp:read-only": "true", "pp:data-source": "live"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return writeDryRun(cmd.OutOrStdout(), flags, "study")
			}
			if len(args) > 1 {
				return usageErr(fmt.Errorf("pass at most one date"))
			}
			if days < 1 || days > 31 {
				return usageErr(fmt.Errorf("--days must be between 1 and 31"))
			}
			tracks, err := parseTrackFlag(track)
			if err != nil {
				return err
			}
			arg := ""
			if len(args) == 1 {
				arg = args[0]
			}
			start, err := parseDateArg(arg)
			if err != nil {
				return err
			}
			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			out := make([]*studyDay, 0, days)
			for i := 0; i < days; i++ {
				d, err := fetchStudy(ctx, c, start.AddDate(0, 0, i))
				if err != nil {
					return classifyAPIError(cmd.OutOrStdout(), err, flags)
				}
				filterStudyTracks(d, tracks)
				out = append(out, d)
			}
			if !wantsHumanTable(cmd.OutOrStdout(), flags) {
				if days == 1 {
					return printJSONFiltered(cmd.OutOrStdout(), out[0], flags)
				}
				return printJSONFiltered(cmd.OutOrStdout(), out, flags)
			}
			w := cmd.OutOrStdout()
			for i, d := range out {
				if i > 0 {
					fmt.Fprintln(w)
				}
				writeHeader(w, fmt.Sprintf("%s · %s", longDate(d.Date), d.Hebrew), "")
				tw := newTabWriter(w)
				for _, it := range d.Items {
					fmt.Fprintf(tw, "  %s\t%s\n", it.Track, it.Lesson)
				}
				_ = tw.Flush()
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&days, "days", 1, "Number of consecutive days to list (max 31)")
	cmd.Flags().StringVar(&track, "track", "", "Comma-separated tracks: chumash, tehillim, tanya, hayom-yom, sefer-hamitzvot, rambam-1, rambam-3, or chitas")
	return cmd
}

func init() {
	registerNovelCommand(func(root *cobra.Command, flags *rootFlags) {
		addNovelCommandIfAbsent(root, newStudyCmd(flags))
	})
}
