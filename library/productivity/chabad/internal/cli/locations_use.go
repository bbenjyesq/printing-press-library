// Copyright 2026 Bob Benjy and contributors. Licensed under Apache-2.0. See LICENSE.
// pp:data-source live

package cli

import (
	"fmt"
	"strings"

	"github.com/mvanhorn/printing-press-library/library/productivity/chabad/internal/config"

	"github.com/spf13/cobra"
)

type locationUseView struct {
	Location chabadLocation `json:"location"`
	Saved    bool           `json:"saved"`
	Path     string         `json:"path,omitempty"`
	Note     string         `json:"note,omitempty"`
}

func newLocationsUseCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "use [location]",
		Short: "Save a default location so other commands can skip --location",
		Long: strings.Trim(`
Save a default Chabad.org location. Every command that takes --location
(shabbat, today, next, bulletin, ahead, ics, ...) uses it when --location is
omitted. Pass a US ZIP, a city name, or city:<Chabad.org city id>. With no
argument, prints the current default.`, "\n"),
		Example: strings.Trim(`
  chabad-pp-cli locations use 11213
  chabad-pp-cli locations use "Los Angeles"
  chabad-pp-cli locations use city:247
  chabad-pp-cli locations use`, "\n"),
		Annotations: map[string]string{"mcp:local-write": "true", "pp:data-source": "live"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return writeDryRun(cmd.OutOrStdout(), flags, "locations use")
			}
			if len(args) > 1 {
				return usageErr(fmt.Errorf("pass one location (quote names with spaces)"))
			}
			if len(args) == 0 {
				saved, ok, err := config.LoadDefaultLocation(flags.configPath)
				if err != nil {
					return configErr(err)
				}
				path, _ := config.DefaultLocationPath(flags.configPath)
				view := locationUseView{Saved: ok, Path: path}
				if ok {
					view.Location = chabadLocation{ID: saved.ID, Type: saved.Type, Name: saved.Name}
				} else {
					view.Note = "no default location saved; run 'chabad-pp-cli locations use 11213'"
				}
				if !wantsHumanTable(cmd.OutOrStdout(), flags) {
					return printJSONFiltered(cmd.OutOrStdout(), view, flags)
				}
				if !ok {
					fmt.Fprintln(cmd.OutOrStdout(), view.Note)
					return nil
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Default location: %s (%s)\n", firstNonEmpty(saved.Name, saved.ID), locationKindLabel(saved.Type, saved.ID))
				return nil
			}
			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			loc, err := resolveChabadLocation(ctx, flags, c, args[0])
			if err != nil {
				return err
			}
			// Validate the id against Chabad.org and pick up its display name.
			today := todayDate()
			res, err := fetchZmanim(ctx, c, loc, today, today, zmanimOptions{})
			if err != nil {
				return classifyAPIError(cmd.OutOrStdout(), err, flags)
			}
			if res.Location != "" {
				loc.Name = res.Location
			}
			if loc.Name == "" {
				return notFoundErr(fmt.Errorf("Chabad.org did not recognize location %q", args[0]))
			}
			path, err := config.SaveDefaultLocation(flags.configPath, config.SavedLocation{ID: loc.ID, Type: loc.Type, Name: loc.Name})
			if err != nil {
				return configErr(fmt.Errorf("saving default location: %w", err))
			}
			view := locationUseView{Location: loc, Saved: true, Path: path}
			if !wantsHumanTable(cmd.OutOrStdout(), flags) {
				return printJSONFiltered(cmd.OutOrStdout(), view, flags)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Default location set to %s (%s)\n", loc.Name, locationKindLabel(loc.Type, loc.ID))
			return nil
		},
	}
	return cmd
}

func locationKindLabel(t int, id string) string {
	if t == 2 {
		return "ZIP " + id
	}
	return "Chabad.org city id " + id
}

func init() {
	registerNovelCommand(func(root *cobra.Command, flags *rootFlags) {
		if locCmd, _, err := root.Find([]string{"locations"}); err == nil && locCmd != nil && locCmd != root {
			addNovelCommandIfAbsent(locCmd, newLocationsUseCmd(flags))
		}
	})
}
