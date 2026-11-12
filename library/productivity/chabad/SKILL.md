---
name: pp-chabad
description: "Chabad.org's zmanim, Shabbat and holiday times, and daily Chitas in your terminal, plus a today dashboard, next-zman countdown, and weekly bulletin. Trigger phrases: `when is candle lighting this week`, `what are today's zmanim`, `when is Pesach this year`, `what's today's Chitas`, `when is my father's yahrzeit`, `use chabad-pp-cli`, `run chabad`."
author: "Bob Benjy"
license: "Apache-2.0"
argument-hint: "<command> [args] | install cli|mcp"
allowed-tools: "Read Bash"
metadata:
  openclaw:
    requires:
      bins:
        - chabad-pp-cli
    install:
      - kind: go
        bins: [chabad-pp-cli]
        module: github.com/mvanhorn/printing-press-library/library/productivity/chabad/cmd/chabad-pp-cli
---

# Chabad.org — Printing Press CLI

## Prerequisites: Install the CLI

This skill drives the `chabad-pp-cli` binary. **You must verify the CLI is installed before invoking any command from this skill.** If it is missing, install it first:

1. Install via the Printing Press installer. It defaults binaries to `$HOME/.local/bin` on macOS/Linux and `%LOCALAPPDATA%\Programs\PrintingPress\bin` on Windows:
   ```bash
   npx -y @mvanhorn/printing-press-library install chabad --cli-only
   ```
2. Verify: `chabad-pp-cli --version`
3. Ensure the reported install directory is on `$PATH` for the agent/runtime that will invoke this skill.

If the `npx` install fails (no Node, offline, etc.), fall back to a direct Go install (requires Go 1.26.10 or newer). This installs into `$GOPATH/bin` (default `$HOME/go/bin`), so add that directory to `$PATH` instead:

```bash
go install github.com/mvanhorn/printing-press-library/library/productivity/chabad/cmd/chabad-pp-cli@latest
```

If `--version` reports "command not found" after install, the runtime cannot see the binary directory on `$PATH`. Do not proceed with skill commands until verification succeeds.

chabad-pp-cli uses the same times Chabad.org publishes, following Chabad custom (18-minute candle lighting, 40 in Jerusalem, and the Chabad Shabbat-end time), where generic calendars use Hebcal defaults. Run `today` for the whole Jewish day on one screen, `next` for a countdown to the next zman, `bulletin` for this week's newsletter block, and `ahead` for the holidays and fasts coming up. Every command works with --json for scripts and agents.

## When to Use This CLI

Use this CLI when a user or agent needs Jewish calendar facts as Chabad.org publishes them: zmanim, Shabbat and Yom Tov candle-lighting and end times, the weekly parsha, holiday dates, Hebrew dates, the Omer, yahrzeits and Hebrew birthdays, or the daily Chitas and Rambam schedule. It is the right choice for newsletters, shul display boards, Shabbat-mode automation, and quick 'when is Shabbat' answers that should follow Chabad custom.

## Anti-triggers

Do not use this CLI for:
- Do not use it for Chabad.org articles, Torah texts, Jewish History entries or Laws & Customs text; those pages are not part of this CLI
- Do not use it for Daf Yomi or Mishna Yomi; Chabad.org's feeds here do not publish them
- Do not use it as a halachic ruling; for questions of practice, ask a rabbi
- Do not use it to find or contact a Chabad center

## Unique Capabilities

### Your Jewish day at a glance
- **`today`** — One screen with the Hebrew date, parsha, holiday, today's zmanim with the next one highlighted, the Omer count, and today's Chitas and Rambam.

  _Reach for this when an agent needs everything about today's Jewish day in one structured call instead of four lookups._

  ```bash
  chabad-pp-cli today --location 11213
  ```
- **`next`** — The next halachic times and how many minutes until each, rolling into tomorrow's dawn, with an exit-code mode for automation.

  _Use it to answer 'when is the next zman' or to gate a Shabbat-mode trigger without re-implementing day rollover._

  ```bash
  chabad-pp-cli next --location 11213 --zman candle-lighting --agent
  ```

### Planning ahead
- **`bulletin`** — A copy-paste weekly block for newsletters and shul signage (--format md|plain, --days N): Shabbat times with week-over-week change, parsha, Hebrew date, Mevarchim and Rosh Chodesh notices, and upcoming holidays, for one or more locations.

  _Use it when someone needs this week's Shabbat announcement text, not raw times._

  ```bash
  chabad-pp-cli bulletin --location 11213 --location 247
  ```
- **`ahead`** — Every fast, erev Yom Tov, Yom Tov end, Rosh Chodesh, Shabbat Mevarchim and first Chanukah candle in the next N days, each with the times it needs (candle lighting, light-from-flame, fast start and end, chametz deadlines); --all adds minor and Chassidic dates.

  _Use it to plan the coming weeks around holidays and fasts with Chabad.org's exact times._

  ```bash
  chabad-pp-cli ahead --location 11213 --days 30
  ```

### Local state that compounds
- **`occasions`** — Keep a local list of family yahrzeits, Hebrew birthdays, anniversaries and bar/bat-mitzvah dates and see which fall in the coming days.

  _Use it when the question is 'whose yahrzeit or Hebrew birthday is coming up' across many people._

  ```bash
  chabad-pp-cli occasions upcoming --days 30
  ```

## Discovery Signals

This CLI was generated with browser-observed traffic context.
- Capture coverage: 3 API entries from 4 total network entries
- Protocols: rest_json (75% confidence)
- Auth signals: none
- Candidate command ideas: list_Get_Locations — Derived from observed GET /WebServices/RemoteCall/Get_Locations traffic.; list_Get_Zmanim — Derived from observed GET /webservices/zmanim/zmanim/Get_Zmanim traffic.; list_Load_CandleLightingWeeks — Derived from observed GET /webservices/Zmanim/CandleLighting/Load_CandleLightingWeeks traffic.

`--ca-cert`, `CHABAD_CA_CERT`, or `ca_cert` adds a PEM CA bundle to the system trust roots. `--tls-server-name`, `CHABAD_TLS_SERVER_NAME`, or `tls_server_name` is the name checked against the certificate. Precedence is flag, then environment, then the config file. A config file other users can change does not supply these settings. `doctor` reports the trust in use and names the setting that fixes a chain or hostname failure.

## Command Reference

Location flags: the friendly commands (`today`, `next`, `shabbat`, `bulletin`, `ahead`, `holidays`, `parsha`, `ics`) take `--location` (a ZIP, a city name, or `city:<id>`) and fall back to the default saved with `locations use`. The raw endpoint commands `zmanim` and `candles` take `--location-id` plus `--location-type` (1 = city id, 2 = ZIP).

**candles** — Shabbat and Yom Tov candle-lighting and end times from Chabad.org, grouped by week and holiday

- `chabad-pp-cli candles` — Upcoming Shabbat and holiday groups with candle-lighting, 'light from a pre-existing flame' second-night times

**locations** — Look up Chabad.org location IDs (cities and US ZIP codes) used by every zmanim and candle-lighting command

- `chabad-pp-cli locations` — Search Chabad.org locations by city name or US ZIP code. Value is the location ID; ItemType 1 = Chabad.

**zmanim** — Daily halachic times (zmanim) from Chabad.org for a location and date range

- `chabad-pp-cli zmanim` — Per-day zmanim (dawn through midnight), candle-lighting and Shabbat-end times, Torah reading


**ahead**


- `chabad-pp-cli ahead` — Every fast, erev Yom Tov, Yom Tov end, Rosh Chodesh and Erev Pesach in the next N days, each with the times it needs: candle lighting, light-from-flame, fast start and end, and the chametz deadlines.


**bulletin**


- `chabad-pp-cli bulletin` — A copy-paste weekly block for newsletters and shul signage: Shabbat times with week-over-week change, parsha, Hebrew date, Mevarchim and Rosh Chodesh notices, and upcoming holidays, for one or more locations.


**next**


- `chabad-pp-cli next` — The next halachic times and how many minutes until each, rolling into tomorrow's dawn, with an exit-code mode for automation.


**occasions**


- `chabad-pp-cli occasions` — Keep a local list of family yahrzeits, Hebrew birthdays and bar/bat-mitzvah dates and see which fall in the coming days.


**today**


- `chabad-pp-cli today` — One screen with the Hebrew date, parsha, holiday, today's zmanim with the next one highlighted, the Omer count, and today's Chitas and Rambam.




### Finding the right command

When you know what you want to do but not which command does it, ask the CLI directly:

```bash
chabad-pp-cli which "<capability in your own words>"
```

`which` resolves a natural-language capability query to the best matching command from this CLI's curated feature index. Exit code `0` means at least one match; exit code `2` means no confident match — fall back to `--help` or use a narrower query. `--json` (and other machine formats) keep that exit-2 contract and write `{"matches":[]}` on stdout so agents can inspect the envelope without treating a miss as success.

## Recipes

### This week's Shabbat for two cities

```bash
chabad-pp-cli bulletin --location 11213 --location 247
```

A newsletter-ready block for Brooklyn and Jerusalem, with each city's own candle-lighting custom.

### Chanukah week, narrowed for an agent

```bash
chabad-pp-cli zmanim --location-id 11213 --location-type 2 --from 2026-12-04 --to 2026-12-12 --agent --select DisplayDate,Parsha,HolidayName
```

Pulls nine days of Chabad.org day records and keeps only the date, Torah reading, and holiday name.

### Fire a trigger before candle lighting

```bash
chabad-pp-cli next --location 11213 --zman candle-lighting --within 30m
```

Exits 0 when candle lighting is within 30 minutes and 1 otherwise, so a cron job can start Shabbat mode.

### What's coming up this month

```bash
chabad-pp-cli ahead --location 11213 --days 30
```

Fasts, erev Yom Tov and Rosh Chodesh in the next 30 days with the times each needs.

### A yahrzeit for the next five years

```bash
chabad-pp-cli yahrzeit 2015-03-10 --after-sunset --years 5
```

Computes the Hebrew date of passing and its next five yahrzeits.

## Auth Setup

No account or API key. The CLI calls Chabad.org's public zmanim webservices and daily-study feed. Set a default location once with `chabad-pp-cli locations use <ZIP or city>`.

Run `chabad-pp-cli doctor` to verify setup.

## Agent Mode

Add `--agent` to any command. Expands to: `--json --compact --no-input --no-color`.

Global format flags share one contract on promoted, novel, sync, and `--deliver` paths:

- `--json` — one JSON document on stdout (sync progress events go to stderr)
- `--compact` — keep identity/status/timestamp fields; does not change the document vs stream shape
- `--csv` / `--plain` — tabular rows (collection envelopes unwrap to the row array). `--csv` prefixes `'` on a cell or header key that starts with `=`, `+`, `-`, `@`, tab, CR, or LF (fullwidth `＝＋－＠` too) so a spreadsheet does not run it as a formula; a plain signed number such as `-5` or `+2.5e3` is unchanged.
- `--quiet` — one identity value per row, no envelope

- **Pipeable** — JSON on stdout, errors on stderr
- **Filterable** — `--select` keeps a subset of fields. Dotted paths descend into nested structures; arrays traverse element-wise. Critical for keeping context small on verbose APIs:

  ```bash
  chabad-pp-cli locations --query example-value --agent --select Title,Value,ItemType
  ```
- **Previewable** — `--dry-run` shows the request without sending. A completed dry run exits 0 with `"success": true` and `"dry_run": true` and no HTTP `status`; validation failures still exit non-zero with `"success": false`.
- **Offline-friendly** — sync/search commands can use the local SQLite store when available
- **Non-interactive** — never prompts, every input is a flag
- **Read-only** — do not use this CLI for create, update, delete, publish, comment, upvote, invite, order, send, or other mutating requests

### Response envelope

Commands that read from the local store or the API wrap output in a provenance envelope:

```json
{
  "meta": {"source": "live" | "local", "synced_at": "...", "reason": "..."},
  "results": <data>
}
```

Parse `.results` for data and `.meta.source` to know whether it's live or local. A human-readable `N results (live)` summary is printed to stderr only when stdout is a terminal AND no machine-format flag (`--json`, `--csv`, `--compact`, `--quiet`, `--plain`, `--select`) is set — piped/agent consumers and explicit-format runs get pure JSON on stdout.

## Paths and state

Agents should treat the CLI's path resolver as part of the runtime contract:

- Use `--home <dir>` for one invocation, or set `CHABAD_HOME=<dir>` to relocate all four path kinds under one root.
- Use per-kind env vars only when a specific kind must diverge: `CHABAD_CONFIG_DIR`, `CHABAD_DATA_DIR`, `CHABAD_STATE_DIR`, `CHABAD_CACHE_DIR`.
- Resolution order is per-kind env var, `--home`, `CHABAD_HOME`, XDG (`XDG_CONFIG_HOME`, `XDG_DATA_HOME`, `XDG_STATE_HOME`, `XDG_CACHE_HOME`), then platform defaults.
- `config` contains settings like `config.toml` and profiles. `data` contains durable local data such as `data.db`. `state` contains persisted queries, jobs, and `teach.log`. `cache` contains regenerable HTTP/cache files.
- Run `chabad-pp-cli doctor --fail-on warn` to surface path warnings. `agent-context` exposes a schema v4 `paths` block for agents that need the resolved dirs.
- For MCP, pass relocation through the MCP host config. The MCP binary does not inherit CLI flags:

  ```json
  {
    "mcpServers": {
      "chabad": {
        "command": "chabad-pp-mcp",
        "env": {
          "CHABAD_HOME": "/srv/chabad"
        }
      }
    }
  }
  ```

Fleet precedence: an inherited per-kind env var overrides an explicit `--home` for that kind. Use `CHABAD_HOME` or per-kind vars as durable fleet levers, and use `--home` only for a single invocation. Relocation is not reversible by unsetting env vars; move files manually before clearing `CHABAD_HOME`, or `doctor` will not find files left under the former root.

Local data is unencrypted and relies on OS account permissions plus disk encryption. Reads in the default `--data-source auto` mode are written to the SQLite store; GET responses are cached for five minutes under the cache dir. `local-data purge` previews what it would erase (store files, response cache, learn journal) and deletes only with `--yes`; it is not exposed over MCP. `CHABAD_NO_LOCAL_DATA=1` disables the response cache and store writes, turns `auto` reads into live reads, and blocks `sync` and `workflow archive`; `--data-source local` still reads an existing store, and `CHABAD_DATA_SOURCE` sets the `--data-source` default.

## Automatic learning

This CLI ships a self-capturing learning loop. The CLI does its own bookkeeping: every invocation is journaled locally, a failed flag followed by a corrected retry auto-derives a `flag_alias` candidate, and a `teach` on a query family without a playbook auto-synthesizes a `playbook_candidate` from the session's journal. Your job is judgment only: `recall` first, act on surfaced candidates, `teach` the final answer, `playbook amend` when you observe a correction. You never record failures by hand.

### Step 1: `recall` before any discovery

Before list/search/drill commands on a new user question, pass the question as an argv or MCP tool argument to `recall --agent`. Do not interpolate user-controlled text into a shell command line.

Quoted `recall "<question>"` breaks on an apostrophe, which is ordinary English. A quoted heredoc breaks when a body line equals the delimiter, and that delimiter is published in these docs. Write the question with a non-shell file-writing tool, then read it back as data:

```bash
# Write the question verbatim with your file-writing tool (no shell involved).
# Command substitution on a file only ever yields data — the shell never
# parses the file's bytes as syntax.
QUERY=$(cat /path/to/question.txt)
chabad-pp-cli recall "$QUERY" --agent
```

Prefer MCP: pass the question as the tool's query argument. `"$QUERY"` after a file read is argv-safe; putting the question itself in the command text is not.

The response envelope:

```json
{
  "query": "...",
  "normalized": "<normalized form>",
  "query_entities": ["..."],
  "found": true | false,
  "match_score": 0.0,
  "results": [
    { "resource_id": "...", "resource_type": "...", "venue": "...",
      "confidence": 2, "entity_match": "exact|partial|unknown",
      "source": "taught|preseed|pattern", "warnings": ["..."] }
  ],
  "mismatches": [ /* only when --debug-mismatches */ ],
  "warnings": [ /* top-level */ ],
  "candidates": [
    { "id": 12, "class": "flag_alias | playbook_candidate",
      "summary": "...", "sightings": 3, "last_seen": "...",
      "rationale": "...",
      "next_action": ["<trial command>", "chabad-pp-cli learnings confirm 12"] }
  ],
  "playbook": {
    "query_family": "...",
    "playbook": {
      "steps": [ { "cmd": "<command with {slot} substitution>", "purpose": "..." } ],
      "entity_slots": ["$ENTITY"],
      "expected_tool_calls": 3
    },
    "slots_resolved": { "$ENTITY": { "token": "<live token>", "canonical": "<canonical>" } },
    "notes": "<workarounds + gotchas for this query family>"
  },
  "notes": "<duplicate surface for non-playbook callers>"
}
```

Empty-store short-circuit: if the store has no learnings, playbooks, or candidates yet (recall finds nothing and `learnings list` and `learnings candidates` are both empty), skip recall for the rest of this session instead of taxing every query; resume recall-first once something has been taught.

### Step 2: decision tree

Read `candidates`, `playbook`, `notes`, `results[0]`, and warnings in that order:

```
if Candidates present (warnings include "candidates_present"):
    -> candidates are try-then-confirm, never facts. Follow each candidate's
       two-step next_action verbatim: run the trial command first, then run
       `learnings confirm <id>` only after the trial verified the behavior.
       Reject a wrong candidate with `learnings reject <id>`.
    -> NEVER re-teach something recall surfaced as a candidate; confirm or
       reject that candidate instead of teaching a duplicate.
    -> candidates ride alongside playbooks and resource hits, not instead of
       them; continue with the branches below after acting on them.

if Playbook present:
    -> READ Playbook.notes verbatim FIRST (workarounds + gotchas the CLI surface doesn't expose)
    -> replay Playbook.steps in order, substituting Playbook.slots_resolved entries
       for the entity slot tokens. If a step's slot is unresolved, fall back to
       discovery for that step only.
    -> the Playbook's expected_tool_calls is a budget; if you find yourself running
       materially more, record the divergence via `chabad-pp-cli playbook amend`
       at end-of-session.

elif Notes present (no Playbook):
    -> read Notes verbatim before any discovery step; they carry known gotchas
       for this query family even when no structured choreography exists yet.

elif Found AND Results[0].EntityMatch == "exact" AND Results[0].Confidence >= 2:
    -> skip discovery; fetch live data for Results[*].ResourceID in parallel

elif Found AND Results[0].EntityMatch == "partial":
    -> candidate hint, NOT a hit; read the resource title to validate before trusting

elif (any row in Mismatches[] when --debug-mismatches was passed):
    -> treat as cold start; the stored learning is for a different entity
       (different canonical resolved from query_entities)

else:  // Found == false, no playbook, no notes
    -> cold start; run discovery normally; teach the answer afterward (Step 4).
       If the family has no playbook yet, that teach auto-synthesizes a
       playbook candidate from this session's journal - you do not need to
       record one by hand.
```

Playbook and Notes are orthogonal to the per-resource path. A recall response can carry both a Playbook AND a `Results[]` hit - use both: the Playbook tells you which choreography to run; the resource hits short-circuit specific steps. Default to skipping `mismatches`; pass `--debug-mismatches` only when investigating cold-start surprises.

Candidate judgment details: `learnings confirm <id>` prints the candidate's full payload before materializing it - check that the printed payload matches the behavior you verified. `learnings reject <id>` tombstones the derivation signature so the same candidate does not resurface. The envelope carries only the few candidates worth acting on now; `chabad-pp-cli learnings candidates` lists the full open set.

Graceful degradation: if `learnings confirm` is an unknown command, you are driving an older binary - ignore the candidates guidance and follow the rest of the protocol.

### Step 3: always read `warnings`

- `low_confidence`: row exists at `confidence<2`. Treat as a hint, not a skip-discovery hit.
- `resource_not_in_store`: the local store doesn't have the resource the learning points at. The match validator couldn't classify entities — direct-fetch and re-evaluate.
- `cross_alias_match` (per-result): the row was taught under a different alias and matched the live query's canonical via `entity_lookups` (e.g., a "USA" teach satisfying a "United States" recall). Trust the resource_id.
- `similar_shape_different_entity:<canonical>` (top-level): a structurally matching row exists but its canonical entity differs from the live query's. Treated as cold start; the warning carries the conflicting canonical as a hint, but the row is NOT promoted into Results.
- `ambiguous_alias` (top-level): a single query entity resolved to multiple canonicals (e.g., "Cards" → Arizona Cardinals + St. Louis Cardinals). Surface the ambiguity from context before committing to a resource.
- `candidates_present` (top-level): the envelope carries a `candidates` section. Handle it via the candidates branch in Step 2 before anything else.
- `lookup_refresh_available` (top-level): an entity in the query has no lookup row yet, but synced data could provide one. Run `chabad-pp-cli sync` to refresh entity lookups.
- Top-level `no_learnings_for_query_family`: the table had no rows above the Jaccard floor. Pure cold start.

### Step 4: `teach &` after finalizing your response - always

Teaching is unconditional. After resolving a query the store could not answer, background-teach the final resource mapping - no call-count threshold, no judging whether it was "worth" learning. The teach is the anchor of the loop: it triggers playbook synthesis for a family without a playbook, and same-referent phrasings fold into one family so near-duplicate teaches do not fragment the store. Fire it after assembling your user-facing response but BEFORE emitting it, with a shell `&` so the call returns immediately. Pass the query the same way as recall — argv/MCP, or file-then-`$QUERY`. Do not splice the question into the command text:

```bash
QUERY=$(cat /path/to/question.txt)
chabad-pp-cli teach --query "$QUERY" --resource-type <type> --resource <id1> --resource <id2>
# (append shell `&` to background it)
```

Silent on success. Errors only land in `teach.log` under the resolved state dir. Teach the **most specific** resource - if the user asked a broad question and you walked through parent records to find the specific answer, teach the leaf id, not the parent. The CLI uses seeded `entity_lookups` for cross-alias resolution at recall time, so a teach under one alias (e.g., "Niners") satisfies future queries under another alias (e.g., "49ers", "San Francisco") automatically.

PII rule: teach the structural question with identifiers stripped - never include names, emails, phone numbers, account ids, or other personal identifiers in taught queries or notes. The CLI scans teach queries for obvious email/phone shapes and warns, but does not block; strip before teaching rather than relying on the warning.

### Step 5: playbooks - optional flags, automatic synthesis

You do not need to decide whether a session "deserves" a playbook: a teach on a family without one auto-synthesizes a `playbook_candidate` from the session's journal, and the next session judges it via confirm/reject. Attach explicit playbook flags only when you already hold choreography worth recording verbatim - workarounds the CLI didn't surface (silently-dropped flags, undocumented params, pagination tricks, payload gotchas). Prefer the **integrated one-call form** - record the resource learning and the playbook in the same `teach` invocation:

```bash
# Common case: record both the resource learning AND the playbook in one call.
QUERY=$(cat /path/to/question.txt)
chabad-pp-cli teach \
  --query "$QUERY" \
  --resource <id> \
  --playbook-file ~/playbooks/<shape>.json \
  --playbook-notes-file ~/playbooks/<shape>-notes.md
# (append shell `&` to background it)

# Alternate: playbook-only (no resource to record alongside).
QUERY=$(cat /path/to/question.txt)
chabad-pp-cli teach-playbook \
  --query "$QUERY" \
  --playbook-file ~/playbooks/<shape>.json \
  --notes-file ~/playbooks/<shape>-notes.md
```

Playbook files are JSON with `steps`, `entity_slots`, `expected_tool_calls`. Notes files are markdown carrying the gotchas verbatim. File-free callers (MCP-only agents) pass the same content inline: `--playbook-json` and `--playbook-notes` on the integrated `teach` form, `--playbook-json` and `--notes` on `teach-playbook`. On the integrated `teach` form, the playbook flags are optional - omit them entirely for a resource-only teach. On the standalone `teach-playbook` form, at least one of the playbook and notes flags must be set; both empty is rejected. Playbooks are keyed on the structural query family (entities stripped) so a recipe taught from one entity-shaped query applies to every other query of the same shape, with `slots_resolved` binding the live query's canonical at recall time.

When you DO find a playbook on a future recall, treat it as ground truth: replay the steps with `slots_resolved` substitutions, skip the discovery that the choreography already documents, and read `notes` before any step.

### Step 6: `playbook amend &` when your debug response identifies a correction

If your debug-protocol response identifies a concrete correction the notes or playbook should know — a workaround, an undocumented endpoint shape, a stale field name, observed schema drift, an empty-payload fallback — fire `playbook amend` BEFORE emitting your user-facing response. Same fire-and-forget posture as `teach`. Pass the query and note as argv/MCP arguments, or write each with a non-shell file tool and read them back (`QUERY=$(cat ...)`, `NOTE=$(cat ...)`). Do not interpolate either string into the command text:

```bash
QUERY=$(cat /path/to/question.txt)
NOTE=$(cat /path/to/note.txt)
chabad-pp-cli playbook amend \
  --query "$QUERY" \
  --add-note "$NOTE"
# (append shell `&` to background it)
```

What counts as worth amending: a behavior you OBSERVED this session that future-you would benefit from knowing. Examples worth amending:

- A workaround for a CLI surface that silently drops or misorders a flag.
- An undocumented endpoint shape (response wrapped in `{meta, results}`, payload nested two levels deeper than the docs claim).
- Observed schema drift (a field renamed, an index that shifted between seasons, a category label that the API now returns lower-cased).

What does NOT belong in notes:

- The year-specific or entity-specific answer to the user's question. That's the response, not a learning.
- Per-team / per-athlete / per-row data the playbook already retrieves at runtime.
- Statements that paraphrase what the existing notes already say.

The amend command appends to the family's existing notes with a timestamped marker (`[amend YYYY-MM-DDTHH:MMZ]: <text>`). Multiple amends accumulate; the audit trail is visible. If no playbook exists yet for the family, amend creates a notes-only one (so cold-start corrections still land).

#### PII discipline for amend notes

`playbook amend` notes are designed to potentially flow upstream as shared knowledge in future versions of the Printing Press. Keep them clean of user-identifying content so the upstream-contribution path stays open without retroactive scrubbing:

- **Do NOT embed** paths to user filesystems, personal API keys or tokens, user email addresses, user GitHub handles, or specific query histories tied to a single user.
- **Acceptable**: endpoint shapes, undocumented field names, API gotchas, observed schema drift, workarounds for CLI surfaces, generalizable pagination or retry tactics.

If a correction is only meaningful with user-specific context, it belongs in a personal note, not in the playbook amend.

### Measuring the loop

`chabad-pp-cli learnings stats` reports recall hit rate, teach-to-reuse, playbook resolution rate, and candidate confirm/reject counts from the local `learn_events` table. Rates are null until they have a denominator; everything stays on this machine. Use it to check whether the loop is earning its keep for this CLI.

### Disabling learning

- `--no-learn` on a single command short-circuits both `recall` and the `teach` write path. Use for deterministic agent flows or tests that must not be affected by accumulated learnings.
- `CHABAD_NO_LEARN=true` in the environment globally disables the pipeline.

## Agent Feedback

When you (or the agent) notice something off about this CLI, record it:

```
chabad-pp-cli feedback "the --since flag is inclusive but docs say exclusive"
chabad-pp-cli feedback --stdin < notes.txt
chabad-pp-cli feedback list --json --limit 10
```

Entries are stored locally as `feedback.jsonl` under the resolved data dir. They are never POSTed unless `CHABAD_FEEDBACK_ENDPOINT` is set AND either `--send` is passed or `CHABAD_FEEDBACK_AUTO_SEND=true`. Default behavior is local-only.

Write what *surprised* you, not a bug report. Short, specific, one line: that is the part that compounds.

## Output Delivery

Every command accepts `--deliver <sink>`. The output goes to the named sink in addition to (or instead of) stdout, so agents can route command results without hand-piping. Three sinks are supported:

| Sink | Effect |
|------|--------|
| `stdout` | Default; write to stdout only |
| `file:<path>` | Atomically write output to `<path>` as a private 0600 file (exclusive randomly named temp in the destination directory, then rename; a planted symlink is not followed). Binary-response commands write decoded payload bytes (not the base64 JSON envelope) and print a small JSON receipt on stdout; `--json`/`--csv` do not refuse when this sink is set. |
| `webhook:<url>` | POST the output body to the URL (`application/json`). Redirects are not followed; a 3xx response fails the delivery and the body is not sent to the `Location`. |

Unknown schemes are refused with a structured error naming the supported set. Webhook failures, including redirects, return non-zero and log the URL and HTTP status on stderr.

## Named Profiles

A profile is a saved set of flag values, reused across invocations. Use it when a scheduled or recurring agent reuses the same saved flags while providing different input each run.

```
chabad-pp-cli profile save briefing --json
chabad-pp-cli --profile briefing locations --query example-value
chabad-pp-cli profile list --json
chabad-pp-cli profile show briefing
chabad-pp-cli profile delete briefing --yes
```

Explicit flags always win over profile values; profile values win over defaults. `agent-context` lists all available profiles under `available_profiles` so introspecting agents discover them at runtime.

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 2 | Usage error (wrong arguments) |
| 3 | Resource not found |
| 5 | API error (upstream issue) |
| 7 | Rate limited or blocked by a firewall or bot challenge (back off; do not retry immediately) |
| 10 | Config error |
| 1 | `next --within`: the next matching zman is not inside the window (not an error; the JSON is still printed) |

## Argument Parsing

Parse `$ARGUMENTS`:

1. **Empty, `help`, or `--help`** → show `chabad-pp-cli --help` output
2. **Starts with `install`** → ends with `mcp` → MCP installation; otherwise → see Prerequisites above
3. **Anything else** → Direct Use (execute as CLI command with `--agent`)

## MCP Server Installation

1. Install the MCP server:
   ```bash
   go install github.com/mvanhorn/printing-press-library/library/productivity/chabad/cmd/chabad-pp-mcp@latest
   ```
2. Register with Claude Code:
   ```bash
   claude mcp add chabad-pp-mcp -- chabad-pp-mcp
   ```
3. Verify: `claude mcp list`

For a remote host, `chabad-pp-mcp --transport http` requires `CHABAD_MCP_HTTP_TOKEN` set to at least 32 characters (callers send `Authorization: Bearer <token>`; `openssl rand -hex 32` prints 64 hex characters and meets that minimum), binds `127.0.0.1:7777` by default, and needs `--tls-cert`/`--tls-key` for any non-loopback `--addr`. Requests with no `Origin` header are accepted. Loopback origins are accepted; any other `Origin` is refused unless listed in `CHABAD_MCP_HTTP_ALLOWED_ORIGINS` (comma-separated `scheme://host[:port]` values). `CHABAD_MCP_READ_ONLY=1` drops mirrored write commands (local-write commands that only touch the CLI's own learn store stay) and makes the raw execute tool refuse state-changing calls.

## Direct Use

1. Check if installed: `which chabad-pp-cli`
   If not found, offer to install (see Prerequisites at the top of this skill).
2. Match the user query to the best command from the Unique Capabilities and Command Reference above.
3. Execute with the `--agent` flag:
   ```bash
   chabad-pp-cli <command> [subcommand] [args] --agent
   ```
4. If ambiguous, drill into subcommand help: `chabad-pp-cli <command> --help`.
