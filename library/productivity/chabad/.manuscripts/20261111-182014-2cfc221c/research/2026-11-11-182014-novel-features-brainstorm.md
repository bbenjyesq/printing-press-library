# Novel Features Brainstorm — chabad-pp-cli (first print)

## Customer model

**1. Rabbi Mendel, shaliach running a mid-size Chabad House (Brooklyn-area ZIP 11213 plus a satellite minyan in a second city)**
- Today: Thursdays opens the Chabad.org zmanim page, a Hebrew-date page, and last week's newsletter; retypes candle lighting, Shabbat ends, parsha; subtracts last week's times by hand; checks Hebrew date and Shabbat Mevorchim manually; can't easily see what falls in the next 14 days (Rosh Chodesh, fast, Yom Tov).
- Weekly ritual: Thursday-night bulletin and shul signage — upcoming Shabbat times, parsha, Hebrew date, Mevorchim/Rosh Chodesh notice, upcoming holidays/fasts, the week's Chitas pointers.
- Frustration: assembling one clean copy-pasteable weekly block from four sources, then redoing it for the second location.

**2. Sara, Brooklyn parent planning Friday around candle lighting**
- Today: bookmarks the zmanim page, sometimes asks a phone widget that uses generic (non-Chabad) times; refreshes Friday afternoons to see how long until candle lighting.
- Weekly ritual: Friday countdown to candle lighting; daily glances at latest Shema, Mincha, sunset; Saturday-night Shabbat-end check.
- Frustration: no quick "what's next and how long until it" answer.

**3. Yossi, developer building a shul display board and Shabbat-mode home automation**
- Today: wraps the undocumented Get_Zmanim JSON in a cron script, hand-parses TimeGroups, re-implements minutes-until and day-rollover logic.
- Weekly ritual: Friday cron fires pre-Shabbat triggers at candle lighting minus N minutes; display board refreshes daily.
- Frustration: reliable next-event + time-until output with machine-friendly exit codes and JSON.

**4. Leah, remote Chitas learner and keeper of family dates**
- Today: opens Daily Study each morning; keeps a notes file of family yahrzeits/birthdays/bar-bat-mitzvah dates and re-runs Chabad.org calculators one at a time each year.
- Weekly ritual: daily Chitas, weekly check of upcoming yahrzeits/birthdays, Shabbat prep.
- Frustration: the calculators are one-at-a-time and nothing remembers her list.

## Candidates (pre-cut)

| # | Name | Command | Persona | Source | Inline verdict |
|---|------|---------|---------|--------|----------------|
| 1 | Today dashboard | today | Sara, Leah | (a)(b) | Keep |
| 2 | Next zman countdown | next | Sara, Yossi | (a)(c) | Keep |
| 3 | Weekly bulletin block | bulletin | Mendel | (a)(b)(c) | Keep |
| 4 | Family occasions list | occasions | Leah, Mendel | (a)(c) | Keep |
| 5 | Shabbat compare locations | shabbat-compare | Mendel, Sara | (a)(c) | Kill: occasional; flag-level variant of shabbat |
| 6 | Holiday times card | holiday-times <name> | Mendel, Sara | (b) | Soft kill: seasonal |
| 7 | Study week sheet | study-week | Leah | (b) | Kill: thin RSS loop |
| 8 | Study find | study-find | Leah | (c) | Kill: duplicates framework search |
| 9 | Opinion spread | opinions | Sara | (b) | Kill: wrapper over zmanim --all-opinions |
| 10 | Shabbat length extremes | extremes | Sara, Mendel | (c) | Kill: monthly trivia |
| 11 | Shabbat delta | shabbat-delta | Mendel | (c) | Folded into bulletin |
| 12 | Omer tonight | omer-tonight | Sara | (b) | Folded into today |
| 13 | Display board feed | board | Yossi | (a) | Kill: today/next --json cover it |
| 14 | Date self-check | verify-dates | Yossi | (c) | Kill: belongs in doctor/dogfood |

## Survivors and kills

### Survivors
| # | Feature | Command | Score | Persona | Buildability | Long Description |
|---|---------|---------|-------|---------|--------------|------------------|
| 1 | Today dashboard | today | 10/10 | Sara, Leah | hand-code (auto) | Use this command for a one-screen overview of today. Do NOT use it for a single upcoming-time countdown or scripting; use 'next' instead. |
| 2 | Next zman countdown | next | 10/10 | Sara, Yossi | hand-code (auto) | Use this command for "what is the next halachic time and how long until it", or to script a trigger. Do NOT use it for a full-day schedule overview; use 'today' instead. |
| 3 | Weekly bulletin block | bulletin | 9/10 | Mendel | hand-code (auto) | Use this command to generate a composed weekly newsletter or signage block for one or more locations. Do NOT use it for raw candle-lighting times only; use 'shabbat' instead. |
| 4 | Family occasions list | occasions | 8/10 | Leah, Mendel | hand-code (local) | Use this command to store and list many people's recurring Hebrew-date events and see what is coming up. Do NOT use it for a one-off calculation for a single person; use 'yahrzeit' or 'birthday' instead. |

### Killed candidates
| Feature | Kill reason | Closest surviving sibling |
|---------|-------------|---------------------------|
| shabbat-compare | Occasional; a multi-location flag on shabbat | bulletin |
| holiday-times | Seasonal, filtered view of holidays/shabbat | bulletin |
| study-week | Thin loop over the RSS endpoint | today |
| study-find | Duplicates framework search | today |
| opinions | Wrapper over zmanim --all-opinions | next |
| extremes | Monthly trivia | bulletin |
| shabbat-delta | Folded into bulletin | bulletin |
| omer-tonight | 49 days/year; folded into today | today |
| board | Redundant with today/next --json | next |
| verify-dates | Maintenance check; belongs in doctor | occasions |

### Follow-up: added survivor #5 (reframed from killed holiday-times)
| # | Feature | Command | Score | Persona | Buildability | Long Description |
|---|---------|---------|-------|---------|--------------|------------------|
| 5 | Holiday-season lookahead | ahead | 7/10 | Mendel, Sara | hand-code (auto) | Use this command to see which fasts, erev-Yom-Tov days and holidays fall in the next N days, with the times each one needs. Do NOT use it for this week's Shabbat block or a newsletter; use 'bulletin' instead. Do NOT use it for the next single zman; use 'next' instead. |
Score breakdown: Domain Fit 3, User Pain 2, Build Feasibility 1, Research Backing 1. Caveat (resolved by main agent): chametz times verified present in Get_Zmanim on 2027-04-20/21 (BedikatChametz, LastEatingChametzTime, BurnChametzTime).
