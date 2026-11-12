# Chabad.org Calendar & Holidays CLI Brief

## API Identity
- Domain: Jewish calendar, zmanim (halachic times), Shabbat/Yom Tov candle-lighting, holidays, weekly parsha, and Chabad daily Torah study (Chumash, Tehillim, Tanya, Rambam, Hayom Yom), as published by Chabad.org (Chabad-Lubavitch Media Center).
- Users: observant Jews and Chabad communities planning their day/week around zmanim; shluchim (Chabad emissaries) producing weekly bulletins, signage, and newsletters; developers building synagogue displays, Shabbat-mode home automation, and calendar feeds; anyone who wants Chabad-custom times (18-min candle lighting outside Jerusalem, Chabad Shabbat-end times, Alter Rebbe's opinions) rather than generic Hebcal defaults.
- Data profile: read-only, public, no auth. Time series by (location, date). Daily records with ~15 zman types, multiple opinions per type, parsha/torah-reading string, holiday name flag. Daily study is one RSS item set per date.
- Homepage: https://www.chabad.org (Zmanim page https://www.chabad.org/calendar/zmanim_cdo; Daily Study https://www.chabad.org/dailystudy/default.asp).

## Reachability Risk
- **Medium.** HTML pages (`/calendar/*.htm`, `/holidays/...`, `sitemap.xml`, `candlelighting.ics.asp`) return `403` with `cf-mitigated: challenge` (Cloudflare managed challenge) to plain stdlib HTTP. `cli-printing-press probe-reachability` → `browser_http`: Surf (Chrome TLS fingerprint) gets `200` with no clearance cookie, no browser capture needed.
- **JSON zmanim webservice is standard_http.** `GET https://www.chabad.org/webservices/zmanim/zmanim/Get_Zmanim` → `200 application/json` from both stdlib and Surf. Not disallowed by robots.txt (robots disallows `/api/` except `/api/v2/chabadorg/centers/` and `/api/v2/chabadorg/torahtexts/`, and disallows `/*candlelighting.ics.asp`).
- RSS: `https://www.chabad.org/tools/rss/dailystudy_rss.xml` → `200 text/xml` (stdlib). `https://www.chabad.org/tools/rss/zmanim.xml` → `200` but returns "No location set" without a location cookie. `/tools/rss/default.htm` → `403` (Cloudflare). Legacy `candlelighting_rss.asp` redirects to `/tools/rss/candlelighting_rss_cdo/locationid/531/locationtype/1` → `404`.
- `/api/v2/chabadorg/centers/` → `200 application/json` (Chabad center directory with coordinates).
- Only wrapper (`toolsforshlichus/chabad-org-zmanim`, npm `chabad-org-zmanim@1.0.0`, 2026-03) has 0 issues, 0 stars — no community reports of blocking, but also no proven longevity. Endpoint is undocumented and could change.
- Observed range cap: a 365-day `startdate/enddate` request returned 181 days (≈6-month cap per call; 4.7 s, 203 KB). A 31-day month returned 31 days.

## Scope decision (post-sniff)
- Ship ONLY surfaces that answer plain HTTP without a bot challenge: `Get_Locations`, `Get_Zmanim`, `Load_CandleLightingWeeks` (JSON webservices) and `dailystudy_rss.xml?tdate=` (RSS). Runtime: `standard_http`.
- Do NOT ship Cloudflare-challenged HTML pages (day view with Jewish History / Laws & Customs / Daily Thought, holiday description pages, converter.asp, Hayom Yom article text). Reaching them requires Chrome TLS impersonation to defeat the bot check; this run declines that.
- Hebrew↔Gregorian conversion, Omer, yahrzeit/birthday/bar-mitzvah anniversaries are computed locally (deterministic Hebrew-calendar math), cross-checked against the Hebrew dates the RSS feed and Load_CandleLightingWeeks return.
- Additional discovered endpoint: `GET /WebServices/RemoteCall/Get_Locations?location=<text>` → `{Suggestions:[{Title, Value, ItemType}]}` (ItemType 1 = city id, 2 = ZIP). `Load_CandleLightingWeeks?locationid&locationtype&startdate=M/D/YYYY&limit=<groups>` → Shabbat + Yom Tov groups with `GroupHeaderTitle` ("Shabbat, Torah Reading: Vayeishev" / "Passover"), `GroupSubHeaderTitle` ("Chanukah"), and per-day `WhenToLightString`, `LightFromPreexistingFlame`, `PrimaryZman{ZmanTimeString, CivilDateFormatted, JewishDateFormatted, Name, OpinionDescription}`, `JewishDate{Day, Month, MonthName, Year, DayOfOmer, IsShabbosMevorchim}`. Get_Zmanim also accepts `locationtype=3&coords=lat,lng&tzname=<IANA, / → *>&n=<name>`, `bdef=0&before=<min>`, `shabbosends=min&after=<min>`, `additional=true` (all opinions).

## Endpoint facts (from wrapper source + live probes)
- `GET /webservices/zmanim/zmanim/Get_Zmanim?locationid=<id>&locationtype=<1|2>&tdate=<M-D-YYYY>&aid=143790&startdate=<M/D/YYYY>&enddate=<M/D/YYYY>`
  - `locationtype=1` Chabad.org city ID (e.g. 531, 424023), `locationtype=2` US ZIP (e.g. 11213).
  - Wrapper sends `accept: application/json`, `x-user-agent: <UA> co_ajax/2.0`, a zmanim_cdo referer; plain curl with only `accept` and a browser UA works.
  - Response top level: `LocationName` ("Brooklyn, NY 11213"), `City`, `Coordinates{Latitude,Longitude}`, `LocationId` ("2-11213"), `LocationDetails` ("Candle Lighting is 18 mins. before sunset&nbsp; | &nbsp;-5 GMT"), `PageTitle`, `EndDate`, `GmtStartDate`/`GmtEndDate` (`/Date(ms)/`), `Footnotes`, `FootnoteOrder`, `GroupHeadings[]`, `IsAdvanced`, `IsNewLocation`, `IsDefaultLocation`, `Days[]`.
  - `Days[]`: `DisplayDate` ("11/13/2026"), `DayOfWeek` (0=Sun), `Parsha` (torah reading string, e.g. "Toldot", "Chanukah 6 - Rosh Chodesh", "Fast"), `HolidayName` ("Chanukah", "Fast - 10 Tevet", "Chanukah Vayeishev"), `IsHoliday`, `IsDstActive`, `IsDayOfDstChange`, `GmtDate`, `TimeGroups[]`.
  - `TimeGroups[]`: `Title`, `ZmanType`, `HebrewTitle` (e.g. "עלות השחר"), `EssentialZmanType`, `Items[]{ZmanType, Title, Zman ("5:11 AM"), Date (/Date(ms)/), Default, OpinionInformation, OpinionDescription, TechnicalInformation, InfoMessage}`.
  - ZmanTypes seen: AlosHashachar, EarliestTefillin, NetzHachamah, LatestShema, LatestTefillah, Chatzos, MinchahGedolah, MinchahKetanah, PlagHaminchah, CandleLighting (Fri/Erev YT only), Shkiah, Tzeis, ShabbatEndTime (Sat/YT), ChatzosNight, ShaahZmanit ("50:39 min."); wrapper also lists LastEatingChametzTime, BurnChametzTime, ShabbosEnds.
  - Hebrew date is NOT in Get_Zmanim; it appears in the RSS channel (`<hebrew_date>Kislev 01, 5787</hebrew_date>`) and on HTML calendar pages.
- `GET /tools/rss/dailystudy_rss.xml` — channel title carries the Hebrew date; items: Daily Chumash with Rashi (parsha + aliyah + verse range), Tehillim (chapters), Tehillim Hebrew, Tanya, Rambam (1 & 3 chapters, Sefer HaMitzvot), Hayom Yom. Links are `tdate=M/D/YYYY` article pages.

## Top Workflows
1. **"When is Shabbat?"** — candle-lighting and Shabbat-end time for this week (or next N weeks) at my location; the single most-asked question.
2. **Today's zmanim** — dawn → midnight for a ZIP/city, with the "next upcoming zman" highlighted (latest Shema, sunset) and minutes-until.
3. **Holiday lookup** — when is Pesach/Chanukah/Purim/Yud-Tes Kislev this year, the erev day candle-lighting, and fast start/end times.
4. **Daily study (Chitas + Rambam + Hayom Yom)** — what to learn today/any date, as text links; a Chabad-specific need no generic calendar tool covers.
5. **Bulletin / signage export** — a month or a season of Shabbat & holiday times for a location as a table, CSV, or ICS feed (shluchim weekly newsletter, shul display boards).

## Table Stakes (competitor features to absorb)
- Hebcal REST/@hebcal/core: holidays by year (major/minor/fasts/modern/Rosh Chodesh), candle lighting + havdalah, weekly parsha (Israel/diaspora), Hebrew↔Gregorian date converter, Omer count, Daf Yomi, molad, Shabbat Mevarchim, yahrzeit/birthday anniversary computation, ICS/CSV export, zmanim by ZIP/geonames/lat-lon.
- hebrewcal (PyPI) CLI: `zmanim <date> --lat --lon --tz`, date conversion.
- Classic Unix `hebcal`: year/month holiday printout, `-c` candle lighting, `-s` sedra, `-o` omer, city selection.
- chabad-org-zmanim (npm): getZmanimByZip, getZmanimByCity, getZmanimRange, hasCandleLighting, hasShabbosEnds, getSpecialZmanim (candle-lighting, Shabbat end, chametz times), raw passthrough.
- Hebcal MCP servers (community + hebcal.com/mcp): get_shabbat_times(city), holiday times.
- MyZmanim / OU zmanim APIs: by location ID or lat/lng.

## Data Layer
- Primary entities: `days` (location_id, date, display_date, dow, parsha, holiday_name, is_holiday, hebrew_date), `zmanim` (location_id, date, zman_type, title, hebrew_title, time, epoch, is_default, opinion), `locations` (location_id "2-11213", type, name, city, lat, lon, details/candle-offset, gmt-offset), `study` (date, track [chumash|tehillim|tanya|rambam1|rambam3|mitzvot|hayomyom], title, description, link).
- Sync cursor: (location_id, date range) — sync fetches in ≤180-day windows; per-location high-water mark of the latest synced date.
- FTS/search: holidays and parsha names across synced days ("when is Purim", "next Rosh Chodesh"), study descriptions (find which day Tanya covers chapter X).

## Product Thesis
- Name: **chabad-pp-cli** — Chabad.org Calendar & Holidays.
- Why it should exist: Every existing CLI/MCP uses Hebcal's generic calculation. Chabad communities follow Chabad.org's published times and customs (the Alter Rebbe's opinions, Chabad Shabbat-end time, 18-min candle-lighting with local exceptions) and learn Chitas/Rambam/Hayom Yom daily. Nothing in the terminal or agent ecosystem serves those. This CLI turns Chabad.org's undocumented zmanim webservice and daily-study feed into typed, offline-cached commands — "when is Shabbat", "next holiday", "today's Chitas" — plus season-long exports for bulletins, and an MCP surface so agents can answer halachic-time questions from the Chabad source of truth.

## Build Priorities
1. `zmanim` (by --zip / --city, --date, --from/--to range, --opinions) on Get_Zmanim, with a parsed, typed output and the raw passthrough.
2. `shabbat` (this week / --weeks N: candle-lighting, Shabbat end, parsha) and `holidays` (upcoming/year list with erev candle-lighting and end times) derived from synced days.
3. `study` (daily Chitas + Rambam + Hayom Yom from the RSS feed, any date) and `today` (one-screen dashboard: Hebrew date, parsha, next zman, study).
4. Local SQLite store + `sync` (≤180-day windows) enabling offline `next`, `search`, and `export --format ics|csv` for bulletins.
5. Location resolution: ZIP and Chabad city ID first; a city-name search if the sniff finds the site's location-lookup endpoint.
