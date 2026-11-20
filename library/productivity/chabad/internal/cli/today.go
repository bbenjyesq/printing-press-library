// Copyright 2026 Bob Benjy and contributors. Licensed under Apache-2.0. See LICENSE.
// pp:data-source live

package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/mvanhorn/printing-press-library/library/productivity/chabad/internal/jcal"

	"github.com/spf13/cobra"
)

type nextZman struct {
	Type         string    `json:"type"`
	Slug         string    `json:"slug"`
	Title        string    `json:"title"`
	Date         string    `json:"date"`
	Time         string    `json:"time"`
	At           time.Time `json:"at"`
	MinutesUntil int       `json:"minutes_until"`
	In           string    `json:"in"`
}

type omerToday struct {
	TonightDay int    `json:"tonight_day"`
	Count      string `json:"tonight_count"`
	Sefirah    string `json:"tonight_sefirah"`
}

type fetchFailure struct {
	Source string `json:"source"`
	Error  string `json:"error"`
}

type todayView struct {
	Date          string         `json:"date"`
	Weekday       string         `json:"weekday"`
	HebrewDate    string         `json:"hebrew_date"`
	Location      string         `json:"location"`
	Details       string         `json:"details,omitempty"`
	TorahReading  string         `json:"torah_reading,omitempty"`
	Holiday       string         `json:"holiday,omitempty"`
	Observances   []string       `json:"observances"`
	Omer          *omerToday     `json:"omer,omitempty"`
	Zmanim        []zmanEntry    `json:"zmanim"`
	Next          *nextZman      `json:"next,omitempty"`
	Study         []studyItem    `json:"study"`
	FetchFailures []fetchFailure `json:"fetch_failures,omitempty"`
}

func newNovelTodayCmd(flags *rootFlags) *cobra.Command {
	var flagLocation, flagDate string
	var opt zmanimOptions

	cmd := &cobra.Command{
		Use:   "today",
		Short: "One screen with the Hebrew date, parsha, holiday, today's zmanim with the next one highlighted, the Omer count",
		Long: strings.Trim(`
Everything about the Jewish day on one screen: the Hebrew date, Torah reading,
holiday, Rosh Chodesh and Chassidic dates, today's zmanim from Chabad.org with
the next one highlighted, tonight's Omer count, and today's Chitas, Hayom Yom
and Rambam from Chabad.org's daily study.

Use this command for a one-screen overview of today. Do NOT use it for a
single upcoming-time countdown or scripting; use 'next' instead.`, "\n"),
		Example:     "  chabad-pp-cli today --location 11213",
		Annotations: map[string]string{"mcp:read-only": "true", "pp:data-source": "live"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return writeDryRun(cmd.OutOrStdout(), flags, "today")
			}
			if err := opt.validate(); err != nil {
				return err
			}
			date, err := parseDateArg(flagDate)
			if err != nil {
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
			res, err := fetchZmanim(ctx, c, loc, date, date, opt)
			if err != nil {
				return classifyAPIError(cmd.OutOrStdout(), err, flags)
			}
			h := jcal.FromTime(date, false)
			view := todayView{
				Date:        date.Format("2006-01-02"),
				Weekday:     date.Weekday().String(),
				HebrewDate:  h.String(),
				Location:    res.Location,
				Details:     res.Details,
				Observances: describeHebrewDate(date, h, false).Events,
				Zmanim:      make([]zmanEntry, 0),
				Study:       make([]studyItem, 0),
			}
			if len(res.Days) > 0 {
				d := res.Days[0]
				view.TorahReading, view.Holiday, view.Zmanim = d.Reading, d.Holiday, d.Zmanim
				// A countdown only makes sense for today, not a --date in the past or future.
				if date.Equal(todayDate()) {
					if nz := nextAfter(res.Days, time.Now(), "", 1); len(nz) > 0 {
						view.Next = &nz[0]
					}
				}
			}
			if n := jcal.OmerDay(jcal.FromTime(date, true)); n > 0 {
				view.Omer = &omerToday{TonightDay: n, Count: jcal.OmerPhrase(n), Sefirah: jcal.OmerSefirah(n)}
			}
			study, err := fetchStudy(ctx, c, date)
			if err != nil {
				view.FetchFailures = append(view.FetchFailures, fetchFailure{Source: "daily-study", Error: err.Error()})
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: daily study unavailable: %v\n", err)
			} else {
				view.Study = study.Items
				if study.Hebrew != "" {
					view.HebrewDate = study.Hebrew
				}
			}
			if !wantsHumanTable(cmd.OutOrStdout(), flags) {
				return printJSONFiltered(cmd.OutOrStdout(), view, flags)
			}
			renderToday(cmd, view)
			return nil
		},
	}
	cmd.Flags().StringVar(&flagLocation, "location", "", locationFlagHelp)
	cmd.Flags().StringVar(&flagDate, "date", "", "Show this date instead of today, YYYY-MM-DD")
	cmd.Flags().IntVar(&opt.CandleMinutes, "candle-minutes", 0, candleMinutesHelp)
	cmd.Flags().IntVar(&opt.ShabbatEndsMinutes, "shabbat-ends-minutes", 0, shabbatEndsMinutesHelp)
	return cmd
}

func renderToday(cmd *cobra.Command, v todayView) {
	w := cmd.OutOrStdout()
	fmt.Fprintf(w, "%s · %s\n", bold(v.HebrewDate), longDate(v.Date))
	fmt.Fprintln(w, v.Location)
	var notes []string
	if v.Holiday != "" {
		notes = append(notes, v.Holiday)
	}
	if v.TorahReading != "" && v.TorahReading != v.Holiday {
		notes = append(notes, "Torah reading: "+v.TorahReading)
	}
	notes = append(notes, v.Observances...)
	for _, n := range notes {
		fmt.Fprintln(w, "• "+n)
	}
	if v.Omer != nil {
		fmt.Fprintf(w, "• Omer tonight: %s (%s)\n", v.Omer.Count, v.Omer.Sefirah)
	}
	fmt.Fprintln(w)
	tw := newTabWriter(w)
	for _, z := range v.Zmanim {
		marker := " "
		if v.Next != nil && z.Type == v.Next.Type && z.Time == v.Next.Time {
			marker = "→"
		}
		fmt.Fprintf(tw, "%s %s\t%s\n", marker, z.Title, z.Time)
	}
	_ = tw.Flush()
	if v.Next != nil {
		fmt.Fprintf(w, "\nNext: %s at %s (in %s)\n", v.Next.Title, v.Next.Time, v.Next.In)
	}
	if len(v.Study) > 0 {
		fmt.Fprintln(w, "\n"+bold("Daily study"))
		tw = newTabWriter(w)
		for _, s := range v.Study {
			fmt.Fprintf(tw, "  %s\t%s\n", s.Track, s.Lesson)
		}
		_ = tw.Flush()
	}
}

// nextAfter returns the next n zmanim after now across days (optionally only
// one zman type), skipping durations and entries without a known instant.
func nextAfter(days []zmanimDay, now time.Time, onlyType string, n int) []nextZman {
	out := make([]nextZman, 0, n)
	cands := make([]nextZman, 0)
	for _, d := range days {
		for _, z := range d.Zmanim {
			if z.At == nil || z.Type == "ShaahZmanit" || !z.At.After(now) {
				continue
			}
			if onlyType != "" && !strings.EqualFold(z.Type, onlyType) &&
				!(onlyType == "ShabbatEndTime" && z.Type == "ShabbosEnds") &&
				!(onlyType == "Tzeis" && z.Type == "FastEnds") {
				continue
			}
			until := z.At.Sub(now)
			cands = append(cands, nextZman{
				Type: z.Type, Slug: z.Slug, Title: z.Title, Date: d.Date, Time: z.Time,
				At: *z.At, MinutesUntil: int(until.Round(time.Minute) / time.Minute), In: humanDuration(until),
			})
		}
	}
	for i := 1; i < len(cands); i++ {
		for j := i; j > 0 && cands[j].At.Before(cands[j-1].At); j-- {
			cands[j], cands[j-1] = cands[j-1], cands[j]
		}
	}
	for _, c := range cands {
		if len(out) >= n {
			break
		}
		out = append(out, c)
	}
	return out
}
