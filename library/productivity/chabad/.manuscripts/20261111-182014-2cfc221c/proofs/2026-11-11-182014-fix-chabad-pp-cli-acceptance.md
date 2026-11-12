Acceptance Report: chabad
  Level: Full Dogfood (live, Chabad.org, no auth)
  Tests: 160/160 passed (loop 2; loop 1 was 159/160)
  Failures (loop 1):
    - ics dry_run_json: expected a JSON envelope on piped --dry-run, got prose
  Fixes applied: 1
    - CLI fix: ics emits the JSON dry-run envelope when stdout is not a terminal
  Printing Press issues: 4 (see build log): go 1.26.10 pin fails govulncheck; --force without --base keeps stale generated bodies; client HTML detector flags RSS; profiler reads `before`/`limit` wire names as pagination.
  Gate: PASS
