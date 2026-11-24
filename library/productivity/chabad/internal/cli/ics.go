// Copyright 2026 Bob Benjy and contributors. Licensed under Apache-2.0. See LICENSE.
// pp:data-source live

package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/spf13/cobra"
)

type icsEvent struct {
	UID      string    `json:"uid"`
	Summary  string    `json:"summary"`
	Start    time.Time `json:"start"`
	Location string    `json:"location"`
	Note     string    `json:"description,omitempty"`
}

func icsEscape(s string) string {
	s = strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
	s = stripControl(s)
	r := strings.NewReplacer(`\`, `\\`, ";", `\;`, ",", `\,`, "\n", `\n`)
	return r.Replace(s)
}

// icsLine writes one content line, folded at 75 octets per RFC 5545 3.1
// without splitting a UTF-8 character.
func icsLine(b *strings.Builder, line string) {
	limit := 75
	for len(line) > limit {
		cut := limit
		for cut > 0 && !utf8.RuneStart(line[cut]) {
			cut--
		}
		b.WriteString(line[:cut])
		b.WriteString("\r\n ")
		line = line[cut:]
		limit = 74 // continuation lines start with a space
	}
	b.WriteString(line)
	b.WriteString("\r\n")
}

func buildICSEvents(res *candleResult, loc chabadLocation) []icsEvent {
	events := make([]icsEvent, 0)
	for _, g := range res.Groups {
		for _, e := range g.Events {
			if e.At == nil {
				continue
			}
			summary := "Candle lighting"
			if e.Kind != "candle-lighting" {
				summary = "Shabbat ends"
				if g.Type != "shabbat" {
					summary = "Holiday ends"
				}
				if strings.Contains(strings.ToLower(e.Label), "holiday ends") {
					summary = "Holiday ends"
				}
			} else if e.FromPreexistingFlame {
				summary = "Candle lighting (from a pre-existing flame)"
			}
			summary = fmt.Sprintf("%s · %s", summary, g.Title)
			h := sha256.Sum256([]byte(loc.key() + e.Date + e.Kind + e.Label))
			events = append(events, icsEvent{
				UID:      hex.EncodeToString(h[:8]) + "@chabad-pp-cli",
				Summary:  summary,
				Start:    e.At.UTC(),
				Location: res.Location,
				Note:     fmt.Sprintf("%s %s (%s). %s", e.Label, e.Time, e.Hebrew, e.Opinion),
			})
		}
	}
	return events
}

func writeICS(w io.Writer, events []icsEvent, calName string) error {
	var b strings.Builder
	b.WriteString("BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//chabad-pp-cli//Chabad.org times//EN\r\nCALSCALE:GREGORIAN\r\nMETHOD:PUBLISH\r\n")
	icsLine(&b, "X-WR-CALNAME:"+icsEscape(calName))
	stamp := time.Now().UTC().Format("20060102T150405Z")
	for _, e := range events {
		b.WriteString("BEGIN:VEVENT\r\n")
		b.WriteString("UID:" + e.UID + "\r\n")
		b.WriteString("DTSTAMP:" + stamp + "\r\n")
		b.WriteString("DTSTART:" + e.Start.Format("20060102T150405Z") + "\r\n")
		b.WriteString("DTEND:" + e.Start.Add(15*time.Minute).Format("20060102T150405Z") + "\r\n")
		icsLine(&b, "SUMMARY:"+icsEscape(e.Summary))
		icsLine(&b, "LOCATION:"+icsEscape(e.Location))
		if e.Note != "" {
			icsLine(&b, "DESCRIPTION:"+icsEscape(strings.TrimSpace(e.Note)))
		}
		b.WriteString("END:VEVENT\r\n")
	}
	b.WriteString("END:VCALENDAR\r\n")
	_, err := io.WriteString(w, b.String())
	return err
}

func newICSCmd(flags *rootFlags) *cobra.Command {
	var location, from, output string
	var weeks int
	var opt zmanimOptions
	cmd := &cobra.Command{
		Use:   "ics",
		Short: "Export Shabbat and Yom Tov candle-lighting and end times as an .ics calendar",
		Long: strings.Trim(`
Writes an iCalendar file of candle-lighting and Shabbat/holiday end times for
a location, ready to import into Google Calendar, Apple Calendar, or Outlook.
Prints to stdout unless --output is given. --json lists the events instead.`, "\n"),
		Example: strings.Trim(`
  chabad-pp-cli ics --location 11213 --weeks 26 --output shabbat.ics
  chabad-pp-cli ics --location "Los Angeles" --weeks 8 --json`, "\n"),
		Annotations: map[string]string{"mcp:read-only": "true", "pp:data-source": "live", "mcp:write-flags": "output"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				// ics normally emits iCalendar text, so a piped dry run reports
				// the machine-readable envelope rather than prose.
				if !isTerminal(cmd.OutOrStdout()) && !flags.asJSON {
					return writeEncodedJSON(cmd.OutOrStdout(), dryRunResult{DryRun: true, Success: true, Action: "ics", Would: "run ics; no changes made"}, "", "")
				}
				return writeDryRun(cmd.OutOrStdout(), flags, "ics")
			}
			if weeks < 1 || weeks > 104 {
				return usageErr(fmt.Errorf("--weeks must be between 1 and 104"))
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
			events := buildICSEvents(res, loc)
			if flags.asJSON || flags.selectFields != "" || flags.compact || flags.csv {
				return printJSONFiltered(cmd.OutOrStdout(), events, flags)
			}
			name := "Chabad.org times · " + firstNonEmpty(res.Location, loc.ID)
			if output == "" || output == "-" {
				return writeICS(cmd.OutOrStdout(), events, name)
			}
			// The path is the user's own --output choice, and an exported calendar
			// is meant to be shared/imported, so world-readable 0644 is intended.
			f, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644) // #nosec G304 G302 -- user-chosen --output path; shareable calendar file
			if err != nil {
				return fmt.Errorf("opening %s: %w", output, err)
			}
			if err := writeICS(f, events, name); err != nil {
				_ = f.Close()
				return err
			}
			if err := f.Close(); err != nil {
				return err
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "wrote %d events to %s\n", len(events), output)
			return nil
		},
	}
	cmd.Flags().StringVar(&location, "location", "", locationFlagHelp)
	cmd.Flags().IntVar(&weeks, "weeks", 12, "Number of Shabbat weeks to include (holidays in the window are added)")
	cmd.Flags().StringVar(&from, "from", "", "Start from this date, YYYY-MM-DD (default: today)")
	cmd.Flags().StringVar(&output, "output", "", "Write the .ics to this file instead of stdout")
	cmd.Flags().IntVar(&opt.CandleMinutes, "candle-minutes", 0, candleMinutesHelp)
	cmd.Flags().IntVar(&opt.ShabbatEndsMinutes, "shabbat-ends-minutes", 0, shabbatEndsMinutesHelp)
	return cmd
}

func init() {
	registerNovelCommand(func(root *cobra.Command, flags *rootFlags) {
		addNovelCommandIfAbsent(root, newICSCmd(flags))
	})
}
