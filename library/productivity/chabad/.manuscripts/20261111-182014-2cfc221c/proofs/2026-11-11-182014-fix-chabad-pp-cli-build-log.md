Manifest transcendence rows: 5 planned, 5 built. Phase 3 will not pass until all 5 ship.

## Slice 1 (foundation)
- internal/jcal: Hebrew calendar math (Gregorian<->Hebrew, month lengths, molad, Rosh Chodesh, Shabbat Mevarchim, Omer, yahrzeit/birthday rules, minor + Chassidic fixed dates). Tests: Chabad.org ground-truth dates (RSS + candle JewishDate), round trip 1990-2060, Rosh Hashanah 5785-5788, molad-vs-RH consistency 5700-5900. Molad cross-checked against Hebcal for Tevet/Shevat/Adar I 5787 (exact match).
- internal/cli/chabad_api.go: location resolution (ZIP / city:<id> / name via Get_Locations / saved default), Get_Zmanim parser for BOTH response shapes (detailed TimeGroups for <=~15-day ranges; compact Zmanim[] for longer), Load_CandleLightingWeeks parser, daily-study RSS parser, chunked fetch (150-day windows).
- Instants: Chabad.org /Date()/ values are serialized in the server's US-Eastern zone (Jerusalem 4:01 PM came back as 21:01Z). Instants are computed from location wall time minus the location offset: LocationDetails "±N GMT" + per-day IsDstActive (zmanim) or CalculatedTimeZone (candles). Verified Jerusalem 14:01Z.
- RSS through the generated client needs client.HTMLResponseHeader (the client's HTML detector flags RSS <title>).
- internal/config/chabad_config.go: default location file beside config.toml.
- internal/store/chabad_migrations.go: chabad_occasions table + Add/List/Remove.
- go.mod: toolchain go1.27.4 pinned (generator emitted go 1.26.10; GO-2026-6859 in net/http, also present in local go1.27.3).

## Slice 2 (absorbed, 11 hand-built commands)
shabbat, parsha, holidays (--all adds computed Rosh Chodesh/minor/Chassidic dates), date, omer, yahrzeit, birthday, study, rosh-chodesh, ics, locations use. Generated endpoint commands cover rows 1-4, 6-7, 9, 13, 18 (promoted to `zmanim`, `locations`, `candles`).
Priority 1 review gate: holidays/yahrzeit/parsha --help examples, --dry-run JSON envelope exit 0, --json valid. PASS.

## Slice 3 (transcendence, 5/5)
today, next (--within exit 0/1), bulletin (multi-location md/plain/json, week-over-week deltas, Mevarchim + molad), ahead (fasts incl. 9 Av sunset start and FastEnds type, erev YT, second-night from-flame, chol-hamoed Shabbat, Erev Pesach chametz times, computed first Chanukah candle/Rosh Chodesh/Mevarchim), occasions (add/list/remove/upcoming, local SQLite).
Behavioral tests (novel_features_test.go): Pesach 5787 fixture asserts every time label; 9 Av fixture; computed Chanukah/Mevarchim/RC merge; nextAfter ordering/filter/empty; bulletin week + change (-7 / -6 min); occasions window + milestone; holiday-run collapse exact sequence.

## Deferred / not built
- HTML-only Chabad.org content (Jewish History, Laws & Customs, Daily Thought, holiday articles, Hayom Yom text): Cloudflare-challenged; out of scope by decision (no bot-check circumvention).
- Daf Yomi / Mishna Yomi: not Chabad.org data; not absorbed.

## Generator limitations found
- go.mod pinned go 1.26.10 which fails govulncheck (GO-2026-6859).
- 2 dead generated helpers: endpointPositionalArgs, retainCLIQueryParams.
- dogfood rewrites internal/cli/which.go without gofmt.
- defaultSyncResources empty when every endpoint has required params: `sync` is a no-op (WARN). Store-dependent novel command (occasions) populates itself from user input, so no sync path is needed.
- Client HTML detector misclassifies RSS/XML responses; required HTMLResponseHeader opt-out.
