# Phase 4.85 agentic output review — chabad-pp-cli
status: WARN (3 warnings, all fixed in-session)
Samples: today, next, bulletin, ahead passed live-check; occasions upcoming skipped (local-write family).
1. ahead clipped multi-day events at the window edge (Chanukah end 12-10 with --days 30). FIXED: fetch a 10-day margin and filter on overlap (eventsOverlapping), true start/end kept; test TestEventsOverlappingKeepsTrueEnds.
2. "First Chanukah candle tonight" shown weeks ahead. FIXED: label now "Erev Chanukah: first candle at nightfall".
3. bulletin coming_up included today's event. FIXED: coming_up starts tomorrow and excludes already-running events.
