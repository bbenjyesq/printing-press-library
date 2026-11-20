# chabad-pp-cli shipcheck

## Final result (loop 5 — small spec/generator-file fixes between loops)
- shipcheck: PASS 7/7 legs (verify, validate-narrative, dogfood, workflow-verify, apify-audit, verify-skill, scorecard)
- verify: PASS, 100% (47/47), 0 critical; data pipeline PASS (`sync --resources candles`)
- scorecard: 92/100 Grade A (from 89 on loop 1); sample output probe 4/4
- verify-skill: all checks passed; validate-narrative --strict --full-examples: PASS

## Blockers found and fixes
1. verify FAIL — "no safe sync resource for the data-pipeline probe". Every endpoint required a location, so `sync` was a no-op.
   - Spec: `candles` gets defaults locationid=11213 / locationtype=2 (Crown Heights), `id_field: GroupHeaderArticleId`, `pagination: {type: none}` (both location endpoints), and public param names `weeks`/`candle_minutes` with `url_name: limit`/`before` (the profiler had read `before` as a pagination cursor and `limit` as a page size; a cursor sent as `before=` would have changed candle-lighting minutes).
   - `generate --force` without `--base` kept stale function bodies (empty `defaultSyncResources`), so generator-owned files were refreshed from a clean `--validate=false` generation; hand-written files untouched.
2. scorecard HOLD — "manifest evidence unavailable": `.printing-press.json` was never written because the generator's post-gen govulncheck gate pins Go 1.26.10 (GO-2026-6859) and fails every run. Manifest/CHANGELOG/manifest.json/spec.yaml/.printing-press-patches taken from the clean generation. Working tree pins `toolchain go1.27.4` (recorded in .printing-press-patches/go-toolchain-pin.json); `govulncheck ./...` is clean on it.
3. verify-skill FAIL — static scanner could not see `--location`/`--candle-minutes` declared through helper functions; flags are now declared inline in each command. Troubleshooting text no longer names the stale `locations search` form.

## Behavioral checks of every novel feature (live, chabad.org)
- today --location 11213: Hebrew date Kislev 1 5787, Rosh Chodesh, zmanim with next highlighted, daily study. OK
- next --location 11213 --zman candle-lighting --within 30m: next Friday 4:21 PM, exit 1 (not within). Jerusalem instants correct (+2). OK
- bulletin --location 11213 --location city:247: Toldot, -7/-6 and -5/-4 min deltas, coming-up list. OK
- ahead --location 11213 --days 45: first Chanukah candle, Chanukah, Mevarchim Tevet, 2-day RC, 10 Tevet fast times. OK
- occasions add/list/upcoming/remove against a temp DB: yahrzeit + 13th-birthday milestone, typed not-found exit 3. OK

## Ship recommendation: ship
