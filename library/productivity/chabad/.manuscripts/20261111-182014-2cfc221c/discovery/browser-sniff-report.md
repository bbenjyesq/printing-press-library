# chabad.org Browser-Sniff Report

## 1. User Goal Flow
- Goal: "Find when Shabbat starts/ends and today's zmanim for my city, plus the Hebrew date, holidays, and daily study for any date."
- Steps completed:
  1. Opened Zmanim page (defaulted to Dallas via geo, locationid 139) → page load triggers Get_Zmanim.
  2. Typed "Los Ange" in the Change Location box → `GET /WebServices/RemoteCall/Get_Locations?location=Los%20Ange`.
  3. Selected "Los Angeles (LA)" (city id 1481) → navigation to `/calendar/zmanim_cdo/.../locationid/1481/locationtype/1/tdate/11-11-2026/...` + Get_Zmanim.
  4. Inspected the advanced-options form (Shabbat-begins minutes, Shabbat-ends minutes, visual shkiah adjustment, custom coordinates + time zone).
  5. Read `location_tool.js` (w2.chabad.org, plain HTTP) → found `Load_CandleLightingWeeks` and the AcceptableParameters list.
  6. Visited Jewish Calendar day view, month view, Date Converter, Jewish Holidays index (all server-rendered HTML; no data XHR).
- Secondary flows: Shabbat/Yom Tov candle-lighting weeks (Pesach window), daily study RSS for arbitrary dates.
- Coverage: 6 of 6 planned steps.

## 2. Pages & Interactions
- `/calendar/zmanim_cdo/aid/143790/jewish/Zmanim-Halachic-Times.htm` — typed partial city, picked suggestion, inspected advanced form.
- `/calendar/view/day.htm` — read SSR content (zmanim, holiday, Torah reading, Jewish History, Laws & Customs, Daily Study, Daily Thought).
- `/calendar/view/month_cdo/jewish/Jewish-Calendar.htm` — SSR grid; only XHR was `/webservices/contentloader/Load_Control?controlName=Modules!LiveEvents`.
- `/calendar/converter_cdo/aid/6225/...` — GET form to `/calendar/converter.asp` (`convertfrom=gregorian|hebrew&month&day&year`); Hebrew month numbering Nissan=1 … Elul=6, Tishrei=7 … Adar/Adar I=12, Adar II=13.
- `/holidays/default_cdo/jewish/Jewish-Holidays.htm` — SSR; per-year listing at `/holidays/default_cdo/year/<YYYY>/jewish/holidays-<YYYY>.htm`.

## 3. Browser-Sniff Configuration
- Backend: agent-browser 0.41.2 headless attempted first → stuck on Cloudflare "Just a moment..." (challenge-only). browser-use installed build is the new Python-harness style without `open/eval` CLI commands. User chose chrome-MCP from the recovery menu; capture ran in a fresh tab in the user's Chrome, closed afterwards.
- Response bodies: captured by an in-page fetch/XHR interceptor for Get_Locations; all four shipped endpoints were then re-fetched by plain stdlib-HTTP curl (Chrome UA + `Accept`) for the enriched capture (`browser-sniff-capture.json`).
- Pacing: ~1 request/s on direct probes; no 429s.
- Proxy-envelope: not detected. GraphQL BFF: not detected.

## 4. Endpoints Discovered (shippable — all plain HTTP, no challenge)
| Method | Path | Status | Content-Type |
|---|---|---|---|
| GET | /WebServices/RemoteCall/Get_Locations?location= | 200 | application/json |
| GET | /webservices/zmanim/zmanim/Get_Zmanim?locationid&locationtype&tdate&startdate&enddate&aid[&bdef&before&shabbosends&after&sadjdef&sadj&coords&tzname&n&additional&seconds] | 200 | application/json |
| GET | /webservices/Zmanim/CandleLighting/Load_CandleLightingWeeks?locationid&locationtype&startdate&limit[&aid] | 200 | application/json |
| GET | /tools/rss/dailystudy_rss.xml?tdate=M/D/YYYY | 200 | text/xml |

Observed but NOT shipped (Cloudflare managed challenge to stdlib HTTP; reachable only with Chrome TLS impersonation, which this run declines to ship as bot-check circumvention):
| GET | /calendar/view/day.asp?tdate= | 403 stdlib / 200 Surf | text/html |
| GET | /calendar/converter.asp?convertfrom=&month=&day=&year= | 403 / 200 Surf | text/html |
| GET | /holidays/default_cdo/year/<YYYY>/jewish/holidays-<YYYY>.htm | 403 / 200 Surf | text/html |
| GET | /dailystudy/hayomyom.asp?tdate= (and other study article pages) | 403 / 200 Surf | text/html |
| GET | /calendar/candlelighting.ics.asp | 403 | (robots.txt Disallow) |

Parameter facts:
- `locationtype`: 1 = Chabad.org city ID (e.g. 1481 LA, 247 Jerusalem, 139 Dallas), 2 = US ZIP, 3 = custom coordinates (`coords=lat,lng`, `tzname=<IANA with / → * and + → ~>`, `n=<display name>`).
- Custom times: `bdef=0&before=<min>` (candle-lighting minutes before sunset), `shabbosends=min&after=<min>` (Shabbat-end minutes after sunset), `sadjdef=0&sadj=<min>` (visual sunset adjustment). Verified: before=40/after=72 → "Candle Lighting is 40 mins. before sunset | Shabbat Ends 72min. after sunset".
- `additional=true` → `IsAdvanced: true` and multiple opinions per zman (e.g. 4 dawn opinions, 7 nightfall opinions). `seconds=true` accepted.
- Get_Zmanim date range: ≤ ~181 days per call (365-day request returned 181 days).
- Load_CandleLightingWeeks: `limit` counts Shabbat/holiday groups; Yom Tov groups are inserted (Pesach window: "Passover", "Shabbat, Intermediate Days of Passover", "Passover" with "Light Holiday Candles after" + `LightFromPreexistingFlame:true`, "Holiday Ends").
- Daily-study RSS honors any `tdate` (verified 1/1/2000, 12/25/2026, 3/23/2027); channel title carries the Hebrew date ("Tevet 15, 5787").

## 5. Traffic Analysis
- Protocols: rest_json (3 endpoints) + RSS XML (1 feed).
- Auth signals: none. Analyzer rejected `limit` as an auth candidate (correct).
- Protection signals: Cloudflare managed challenge (`cf-mitigated: challenge`) on HTML pages only; webservices and RSS are not challenged.
- Reachability: `standard_http` (analysis confidence 0.65; `probe-reachability` on Get_Zmanim → standard_http, both stdlib and Surf 200).
- Warnings: none (no empty_response_shapes).

## 6. Coverage Analysis
- Covered: location search, zmanim (daily + range + custom opinions/coords), Shabbat & Yom Tov candle-lighting/end times with Hebrew dates and Torah portions, holiday names per day, daily study (Chitas + Rambam + Hayom Yom pointer) with Hebrew date for any date.
- Not shippable without bot-check circumvention: Jewish History, Laws & Customs, Daily Thought, holiday descriptions, the converter page. Hebrew↔Gregorian conversion can instead be computed locally (deterministic calendar math) and cross-checked against the RSS Hebrew date.

## 7. Response Samples
- See `direct-response-get-locations.json`, `direct-response-get-zmanim.json`, `direct-response-candlelighting-weeks.json`, `direct-response-dailystudy.xml` in this directory, and `research/chabad-browser-sniff-spec-samples/`.

## 8. Rate Limiting Events
- None. ~25 direct requests over the session at ≤1 req/s.

## 9. Authentication Context
- No authenticated session used. Capture tab in the user's Chrome was anonymous chabad.org browsing; no cookies or headers were stored.
