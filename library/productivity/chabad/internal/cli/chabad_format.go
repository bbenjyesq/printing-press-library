// Copyright 2026 Bob Benjy and contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"fmt"
	"io"
	"strings"
	"time"
)

// shortDate renders "2026-12-04" as "Fri Dec 4".
func shortDate(iso string) string {
	t, err := time.Parse("2006-01-02", iso)
	if err != nil {
		return iso
	}
	return t.Format("Mon Jan 2")
}

// longDate renders "2026-12-04" as "Friday, December 4, 2026".
func longDate(iso string) string {
	t, err := time.Parse("2006-01-02", iso)
	if err != nil {
		return iso
	}
	return t.Format("Monday, January 2, 2006")
}

// humanDuration renders a positive duration as "2h 13m" / "45m".
func humanDuration(d time.Duration) string {
	if d < 0 {
		d = -d
	}
	mins := int(d.Round(time.Minute) / time.Minute)
	days := mins / (24 * 60)
	mins -= days * 24 * 60
	h, m := mins/60, mins%60
	switch {
	case days > 0:
		return fmt.Sprintf("%dd %dh %dm", days, h, m)
	case h > 0:
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}

// signedMinutes renders a minute delta as "+2 min" / "-1 min" / "no change".
func signedMinutes(n int) string {
	switch {
	case n > 0:
		return fmt.Sprintf("+%d min", n)
	case n < 0:
		return fmt.Sprintf("%d min", n)
	}
	return "no change"
}

func writeHeader(w io.Writer, title, sub string) {
	fmt.Fprintln(w, bold(title))
	if strings.TrimSpace(sub) != "" {
		fmt.Fprintln(w, sub)
	}
}
