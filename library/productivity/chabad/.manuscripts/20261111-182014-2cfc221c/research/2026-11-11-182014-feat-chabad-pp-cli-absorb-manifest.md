# chabad-pp-cli Absorb Manifest

Shipped surfaces (plain HTTP, no bot challenge): Get_Locations, Get_Zmanim, Load_CandleLightingWeeks (JSON) and dailystudy_rss.xml?tdate= (RSS). Cloudflare-challenged HTML pages are out of scope. Calendar math (Hebrew↔Gregorian, Omer, molad, anniversaries) is computed locally and cross-checked against API Hebrew dates.

Tools surveyed: chabad-org-zmanim (npm, toolsforshlichus), zmanim-mcp-server (ariroffe72, PyPI), hebcal (classic Unix CLI + hebcal.com REST + @hebcal/core), hebrewcal (PyPI CLI), hdate / pyluach (PyPI), rzman (Ruby), community Hebcal MCP servers, Chabad.org's own calculators (Birthday, Bar/Bat Mitzvah, Yahrzeit, Date Converter, Daily Study).

## Absorbed (match or beat everything that exists)
| # | Feature | Best Source | Our Implementation | Added Value |
|---|---------|-------------|-------------------|-------------|
| 1 | Zmanim by US ZIP | chabad-org-zmanim getZmanimByZip | (generated endpoint) zmanim | --json/--select/--csv, ISO dates, typed exit codes |
| 2 | Zmanim by Chabad.org city id | chabad-org-zmanim getZmanimByCity | (generated endpoint) zmanim | Same command, --location-type 1 |
| 3 | Zmanim date range in one call | chabad-org-zmanim getZmanimRange | (generated endpoint) zmanim | --from/--to up to ~180 days, cached in SQLite |
| 4 | Raw response passthrough | chabad-org-zmanim fetchZmanimRaw | (generated endpoint) zmanim | --json returns the Chabad.org payload untouched |
| 5 | Candle-lighting / Shabbat-ends / chametz helpers | chabad-org-zmanim hasCandleLighting, hasShabbosEnds, getSpecialZmanim | chabad-pp-cli shabbat | Shabbat + Yom Tov in one view, chametz times surfaced by ahead |
| 6 | Zmanim by lat/lon + timezone | hebrewcal `zmanim --lat --lon --tz`; zmanim-mcp-server | (generated endpoint) zmanim | --location-type 3 --coords --tz with Chabad.org opinions, not a generic algorithm |
| 7 | Multiple halachic opinions per zman | KosherJava / hdate | (behavior in chabad-pp-cli zmanim) --all-opinions returns every opinion Chabad.org lists | Opinion descriptions and footnotes kept |
| 8 | Custom candle-lighting / Shabbat-end offset | zmanim-mcp-server offset; hebcal -b / -m | (behavior in chabad-pp-cli shabbat) --candle-minutes and --shabbat-ends-minutes | Server-side recalculation, also on zmanim days and candles weeks |
| 9 | City / location lookup | hebcal `cities`; Chabad.org autocomplete | (generated endpoint) locations | ZIP and city ids in one search |
| 10 | Default location | hebcal HEBCAL_CITY env | chabad-pp-cli locations use | Saved in config; every friendly command uses it |
| 11 | Shabbat candle lighting + havdalah for N weeks | hebcal -c, hebrewcal shabbat, Hebcal MCP get_shabbat_times, rzman | chabad-pp-cli shabbat | Chabad customs (18 min / Jerusalem 40, Chabad Shabbat end), parsha, Hebrew date, Mevarchim flag |
| 12 | Yom Tov candle lighting incl. second-night "from a pre-existing flame" | hebcal holiday candles | (behavior in chabad-pp-cli shabbat) holiday groups with LightFromPreexistingFlame | Exact Chabad.org wording per night |
| 13 | Shabbat/holiday groups raw | Chabad.org candle-lighting page | (generated endpoint) candles | --json, --weeks, --from |
| 14 | Weekly parsha | hebcal -s, pyluach | chabad-pp-cli parsha | Upcoming Shabbat readings incl. holiday readings |
| 15 | Holidays for a year / range (fasts, Rosh Chodesh, Chanukah days) | hebcal year listing, hebcal REST | chabad-pp-cli holidays | Chabad.org holiday names with dates, offline after first fetch |
| 16 | Gregorian → Hebrew date | hebcal converter, pyluach, hdate | chabad-pp-cli date | --after-sunset; cross-checked against Chabad.org's Hebrew date |
| 17 | Hebrew → Gregorian date | hebcal converter, pyluach | (behavior in chabad-pp-cli date) --hebrew "19 Kislev 5787" | Works offline |
| 18 | Full daily zmanim schedule | zmanim-mcp-server daily schedule | (generated endpoint) zmanim | Hebrew titles included |
| 19 | ICS calendar export | hebcal ICS | chabad-pp-cli ics | Shabbat + holiday candle times as an importable .ics |
| 20 | CSV export | hebcal REST CSV | (behavior in chabad-pp-cli shabbat) global --csv on every command | Spreadsheet-ready |
| 21 | Omer count | hebcal -o, hdate | chabad-pp-cli omer | Day, weeks+days, and the Sefirah; tonight vs today |
| 22 | Yahrzeit / Hebrew anniversary | hebcal yahrzeit API; Chabad.org Yahrzeit calculator | chabad-pp-cli yahrzeit | Next N years, after-sunset handling, Adar/Cheshvan/Kislev edge rules |
| 23 | Jewish birthday & bar/bat-mitzvah date | Chabad.org Birthday + Bar/Bat Mitzvah calculators | chabad-pp-cli birthday | Next N birthdays plus the bar (13) / bat (12) mitzvah date |
| 24 | Daily study (Chitas + Rambam + Hayom Yom) | Chabad.org Daily Study (unique) | chabad-pp-cli study | Any date or range, links, Hebrew date |
| 25 | Rosh Chodesh + molad | hebcal molad | chabad-pp-cli rosh-chodesh | Upcoming Rosh Chodesh days, molad, Shabbat Mevarchim |

Out of scope (not shippable without bot-check circumvention): Today in Jewish History, Laws & Customs, Daily Thought, holiday description articles, Hayom Yom full text. Not chabad.org data, so not absorbed: Daf Yomi, Mishna Yomi.

## Transcendence (only possible with our approach)
| # | Feature | Command | Buildability | Score | Why Only We Can Do This | Long Description |
|---|---------|---------|--------------|-------|-------------------------|------------------|
| 1 | Today dashboard | today | hand-code | 10/10 | Joins Get_Zmanim (times, parsha, holiday), the daily-study RSS (Hebrew date + Chitas/Rambam), and local Omer math into one screen with the next zman highlighted | Use this command for a one-screen overview of today. Do NOT use it for a single upcoming-time countdown or scripting; use 'next' instead. |
| 2 | Next zman countdown | next | hand-code | 10/10 | Flattens today's and tomorrow's zmanim in the location's time zone, picks the next one, and reports minutes-until; --within exits non-zero for cron triggers | Use this command for "what is the next halachic time and how long until it", or to script a trigger. Do NOT use it for a full-day schedule overview; use 'today' instead. |
| 3 | Weekly bulletin block | bulletin | hand-code | 9/10 | Joins Shabbat/holiday groups (with Mevarchim flag), parsha, Hebrew dates, local Rosh Chodesh/molad math and week-over-week minute deltas into one copy-paste block for one or more locations | Use this command to generate a composed weekly newsletter or signage block for one or more locations. Do NOT use it for raw candle-lighting times only; use 'shabbat' instead. |
| 4 | Family occasions list | occasions | hand-code | 8/10 | Local SQLite list of people's Hebrew-date events (yahrzeit, birthday, bar/bat mitzvah) expanded to upcoming Gregorian dates | Use this command to store and list many people's recurring Hebrew-date events and see what is coming up. Do NOT use it for a one-off calculation for a single person; use 'yahrzeit' or 'birthday' instead. |
| 5 | Holiday-season lookahead | ahead | hand-code | 7/10 | Scans the next N days of Chabad.org day records for fasts, erev Yom Tov, Yom Tov end, Rosh Chodesh, Mevarchim and Erev Pesach and attaches the times each needs (candle lighting, light-from-flame, fast start/end, bedikat/last-eating/burn chametz — verified returned by Get_Zmanim on 2027-04-20/21) | Use this command to see which fasts, erev-Yom-Tov days and holidays fall in the next N days, with the times each one needs. Do NOT use it for this week's Shabbat block or a newsletter; use 'bulletin' instead. Do NOT use it for the next single zman; use 'next' instead. |

Stubs: none.
