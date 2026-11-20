# Chabad.org CLI

**Chabad.org's zmanim, Shabbat and holiday times, and daily Chitas in your terminal, plus a today dashboard, next-zman countdown, and weekly bulletin.**

chabad-pp-cli uses the same times Chabad.org publishes, following Chabad custom (18-minute candle lighting, 40 in Jerusalem, and the Chabad Shabbat-end time), where generic calendars use Hebcal defaults. Run `today` for the whole Jewish day on one screen, `next` for a countdown to the next zman, `bulletin` for this week's newsletter block, and `ahead` for the holidays and fasts coming up. Every command works with --json for scripts and agents.

Learn more at [Chabad.org](https://www.chabad.org).

Created by [@bbenjyesq](https://github.com/bbenjyesq) (Bob Benjy).

## Install

The recommended path installs both the `chabad-pp-cli` binary and the `pp-chabad` agent skill (Claude Code, Codex, Cursor, Gemini CLI, GitHub Copilot, and other agents supported by the upstream [`skills`](https://github.com/vercel-labs/skills) CLI) in one shot:

```bash
npx -y @mvanhorn/printing-press-library install chabad
```

For CLI only (no skill):

```bash
npx -y @mvanhorn/printing-press-library install chabad --cli-only
```

For skill only — installs the skill into the same agents as the default command above, but skips the CLI binary (use this to update or reinstall just the skill):

```bash
npx -y @mvanhorn/printing-press-library install chabad --skill-only
```

To constrain the skill install to one or more specific agents (repeatable — agent names match the [`skills`](https://github.com/vercel-labs/skills) CLI):

```bash
npx -y @mvanhorn/printing-press-library install chabad --agent claude-code
npx -y @mvanhorn/printing-press-library install chabad --agent claude-code --agent codex
```

### Without Node (Go fallback)

If `npx` isn't available (no Node, offline), install the CLI directly via Go (requires Go 1.26.10 or newer):

```bash
go install github.com/mvanhorn/printing-press-library/library/productivity/chabad/cmd/chabad-pp-cli@latest
```

This installs the CLI only — no skill.

### Pre-built binary

Download a pre-built binary for your platform from the [latest release](https://github.com/mvanhorn/printing-press-library/releases/tag/chabad-current). On macOS, clear the Gatekeeper quarantine: `xattr -d com.apple.quarantine <binary>`. On Unix, mark it executable: `chmod +x <binary>`.

<!-- pp-hermes-install-anchor -->
## Install for Hermes

Install the CLI binary first. The installer writes binaries to a per-user managed bin directory by default: `$HOME/.local/bin` on macOS/Linux and `%LOCALAPPDATA%\Programs\PrintingPress\bin` on Windows.

```bash
npx -y @mvanhorn/printing-press-library install chabad --cli-only
```

Then install the focused Hermes skill.

From the Hermes CLI:

```bash
hermes skills install mvanhorn/printing-press-library/cli-skills/pp-chabad --force
```

Inside a Hermes chat session:

```bash
/skills install mvanhorn/printing-press-library/cli-skills/pp-chabad --force
```

Restart the Hermes session or gateway if the newly installed skill is not visible immediately.

## Install for OpenClaw
Install both the CLI binary and the focused OpenClaw skill. The installer defaults binaries to a per-user bin directory (`$HOME/.local/bin` on macOS/Linux, `%LOCALAPPDATA%\Programs\PrintingPress\bin` on Windows):

```bash
npx -y @mvanhorn/printing-press-library install chabad --agent openclaw
```

Restart the OpenClaw session or gateway if the newly installed skill is not visible immediately.

## Use with Claude Desktop

This CLI ships an [MCPB](https://github.com/modelcontextprotocol/mcpb) bundle — Claude Desktop's standard format for one-click MCP extension installs (no JSON config required).

To install:

1. Download the `.mcpb` for your platform from the [latest release](https://github.com/mvanhorn/printing-press-library/releases/tag/chabad-current).
2. Double-click the `.mcpb` file. Claude Desktop opens and walks you through the install.

Requires Claude Desktop 1.0.0 or later. Pre-built bundles ship for macOS Apple Silicon (`darwin-arm64`) and Windows (`amd64`, `arm64`); for other platforms, use the manual config below.

<details>
<summary>Manual JSON config (advanced)</summary>

If you can't use the MCPB bundle (older Claude Desktop, unsupported platform), install the MCP binary and configure it manually.


```bash
go install github.com/mvanhorn/printing-press-library/library/productivity/chabad/cmd/chabad-pp-mcp@latest
```

Add to your Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json` on macOS, `%APPDATA%\Claude\claude_desktop_config.json` on Windows):

```json
{
  "mcpServers": {
    "chabad": {
      "command": "chabad-pp-mcp"
    }
  }
}
```

</details>

### Remote MCP over HTTP

`chabad-pp-mcp --transport http` (or `PP_MCP_TRANSPORT=http`) serves streamable HTTP at `http://127.0.0.1:7777/mcp`. It refuses to start until `CHABAD_MCP_HTTP_TOKEN` is set to at least 32 characters, and every request must send `Authorization: Bearer <token>`. The command below prints 64 hex characters and meets that minimum. Plain HTTP is allowed only on a loopback `--addr`; any other bind, including `:7777` (every interface), also needs `--tls-cert` and `--tls-key`. Requests with no `Origin` header are accepted, as are loopback origins (`localhost`, `127.0.0.1`, or `[::1]`, any port). Any other `Origin` is refused unless it is listed in `CHABAD_MCP_HTTP_ALLOWED_ORIGINS`, a comma-separated list of `scheme://host[:port]` values with no path. That list is not a CORS policy; the server does not emit CORS headers.

```bash
export CHABAD_MCP_HTTP_TOKEN="$(openssl rand -hex 32)"
chabad-pp-mcp --transport http
```

Set `CHABAD_MCP_READ_ONLY=1` (exactly `1`) to drop mirrored commands that are not marked read-only and make the raw execute tool refuse state-changing calls. Commands marked local-write, which write only this CLI's own learn store, stay available.

## Authentication

No account or API key. The CLI calls Chabad.org's public zmanim webservices and daily-study feed. Set a default location once with `chabad-pp-cli locations use <ZIP or city>`.

## Quick Start

```bash
# Check the install without touching the network
chabad-pp-cli doctor --dry-run

# Find your Chabad.org location ID (a ZIP or a city ID)
chabad-pp-cli locations --query Brooklyn

# This week's candle lighting, Shabbat end, and parsha
chabad-pp-cli shabbat --location 11213

# The whole Jewish day on one screen
chabad-pp-cli today --location 11213

# Today's Chumash, Tehillim, Tanya, Hayom Yom and Rambam
chabad-pp-cli study

```

## Unique Features

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

## Usage

Run `chabad-pp-cli --help` for the full command reference and flag list.

## Paths & environment variables

This CLI separates local files into four path kinds:

| Kind | Contents |
|------|----------|
| `config` | User-editable settings such as `config.toml` and saved profiles |
| `data` | Durable local data such as `data.db` |
| `state` | Runtime state such as persisted queries, jobs, and `teach.log` |
| `cache` | Regenerable HTTP/cache files |

Each kind resolves independently. The ladder is:

1. Per-kind env var: `CHABAD_CONFIG_DIR`, `CHABAD_DATA_DIR`, `CHABAD_STATE_DIR`, or `CHABAD_CACHE_DIR`
2. `--home <dir>` for this invocation
3. `CHABAD_HOME` for a flat relocated root
4. XDG env vars: `XDG_CONFIG_HOME`, `XDG_DATA_HOME`, `XDG_STATE_HOME`, `XDG_CACHE_HOME`
5. Platform defaults matching existing installs

For containers and agent sandboxes, prefer a single relocated root:

```bash
export CHABAD_HOME=/srv/chabad
chabad-pp-cli doctor
```

Under `CHABAD_HOME=/srv/chabad`, the four dirs resolve to `/srv/chabad/config`, `/srv/chabad/data`, `/srv/chabad/state`, and `/srv/chabad/cache`.

On Windows, set the same variable in PowerShell:

```powershell
$env:CHABAD_HOME = "C:\srv\chabad"
chabad-pp-cli doctor
```

Under `$env:CHABAD_HOME = "C:\srv\chabad"`, the four dirs resolve to `C:\srv\chabad\config`, `C:\srv\chabad\data`, `C:\srv\chabad\state`, and `C:\srv\chabad\cache`.

MCP servers do not receive CLI flags from the host. Put relocation in the host `env` block:

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

Precedence matters in fleets: an ambient per-kind variable such as `CHABAD_DATA_DIR` overrides an explicit `--home` for that kind. Use `CHABAD_HOME` or the per-kind variables for durable fleet relocation; treat `--home` as the weaker per-invocation lever.

Relocation is one-way. Unsetting `CHABAD_HOME` does not move files back to platform defaults, and `doctor` cannot find files left under a former root. Move the files manually before unsetting relocation variables.

Existing installs keep working because the platform-default rung matches the legacy layout. Run `chabad-pp-cli doctor --fail-on warn` to check path warnings in automation.

### Local data

Reads are saved locally by default: in `--data-source auto` mode every successful read is also written to the SQLite store (`data.db`, or `data-<hash>.db` per credential) in the data directory. GET responses are cached for five minutes under `http/` in the cache directory. The learn journal lives under `learn/` in the state directory. `chabad-pp-cli doctor` prints the resolved paths, every store file, and the response-cache size.

None of it is encrypted. New files are created owner-only; beyond that the data relies on OS account permissions and full-disk encryption (FileVault, BitLocker, LUKS).

- **Erase it:** `chabad-pp-cli local-data purge` lists the store files, response cache, and learn journal it would delete; add `--yes` to delete them, or `--only store|cache|journal` to narrow it. Config and credentials are kept.
- **Run without a local copy:** set `CHABAD_NO_LOCAL_DATA=1`. Reads go to the API (`--data-source live`; `--data-source local` still reads an existing store), the response cache is off (`--no-cache`), nothing new is written to the store, and `sync` and `workflow archive` refuse to run. Set it in an MCP host's `env` block and every command the server runs inherits it.
- **Choose the read source without flags:** `CHABAD_DATA_SOURCE=auto|live|local` sets the `--data-source` default; an explicit flag wins.

## Commands

### candles

Shabbat and Yom Tov candle-lighting and end times from Chabad.org, grouped by week and holiday

- **`chabad-pp-cli candles`** - Upcoming Shabbat and holiday groups with candle-lighting, 'light from a pre-existing flame' second-night times, Shabbat/holiday end times, Hebrew dates, and Torah readings.

### locations

Look up Chabad.org location IDs (cities and US ZIP codes) used by every zmanim and candle-lighting command

- **`chabad-pp-cli locations`** - Search Chabad.org locations by city name or US ZIP code. Value is the location ID; ItemType 1 = Chabad.org city ID, 2 = US ZIP code.

### zmanim

Daily halachic times (zmanim) from Chabad.org for a location and date range

- **`chabad-pp-cli zmanim`** - Per-day zmanim (dawn through midnight), candle-lighting and Shabbat-end times, Torah reading, and holiday name for a location over a date range (up to ~180 days per call).


### ahead


- **`chabad-pp-cli ahead`** - Every fast, erev Yom Tov, Yom Tov end, Rosh Chodesh and Erev Pesach in the next N days, each with the times it needs: candle lighting, light-from-flame, fast start and end, and the chametz deadlines.


### bulletin


- **`chabad-pp-cli bulletin`** - A copy-paste weekly block for newsletters and shul signage: Shabbat times with week-over-week change, parsha, Hebrew date, Mevarchim and Rosh Chodesh notices, and upcoming holidays, for one or more locations.


### next


- **`chabad-pp-cli next`** - The next halachic times and how many minutes until each, rolling into tomorrow's dawn, with an exit-code mode for automation.


### occasions


- **`chabad-pp-cli occasions`** - Keep a local list of family yahrzeits, Hebrew birthdays and bar/bat-mitzvah dates and see which fall in the coming days.


### today


- **`chabad-pp-cli today`** - One screen with the Hebrew date, parsha, holiday, today's zmanim with the next one highlighted, the Omer count, and today's Chitas and Rambam.




### Self-learning loop

This CLI caches per-question discovery so repeat queries skip the walk and structurally similar queries get answered via entity substitution. The loop also self-captures: every invocation is journaled locally, and failed-flag corrections plus fresh teaches surface as candidates on the next `recall` for confirm/reject judgment. Agents call `recall` before discovery and fire `teach &` after answering. See the `## Automatic learning` section in `SKILL.md` for the full protocol.

- **`chabad-pp-cli recall <query>`** - Look up cached resources for a query before running discovery
- **`chabad-pp-cli teach`** - Record a query -> resource mapping (silent on success, safe to background with `&`)
- **`chabad-pp-cli learnings list`** - Inspect taught rows
- **`chabad-pp-cli learnings forget <query>`** - Undo a teach
- **`chabad-pp-cli learnings candidates`** - List auto-captured candidates awaiting confirm/reject
- **`chabad-pp-cli learnings stats`** - Local loop metrics: recall hit rate, teach-to-reuse, playbook resolution, candidate counts
- **`chabad-pp-cli teach-pattern`** - Install a query/resource template up front
- **`chabad-pp-cli teach-lookup`** - Add an entity mapping (e.g. country code, team alias) for pattern substitution

Pass `--no-learn` or set `CHABAD_NO_LEARN=true` to disable the loop for deterministic flows.

The local store's schema version stamp is one-way: once this version of `chabad-pp-cli` opens the database, older binaries refuse it with a version error — upgrade the binary rather than downgrading.

## Output Formats

```bash
# Human-readable table (default in terminal, JSON when piped)
chabad-pp-cli locations --query Brooklyn

# JSON for scripting and agents
chabad-pp-cli locations --query Brooklyn --json
# Filter to specific fields
chabad-pp-cli locations --query Brooklyn --json --select Title,Value,ItemType

# Dry run — show the request without sending
chabad-pp-cli locations --query Brooklyn --dry-run

# Agent mode — JSON + compact + no prompts in one flag
chabad-pp-cli locations --query Brooklyn --agent
```

`--csv` prefixes a leading `'` on cells and header keys that start with `=`, `+`, `-`, `@`, a tab, or a line break (including fullwidth `＝＋－＠`) so a spreadsheet does not run them as formulas. Plain signed numbers such as `-5` and `+2.5e3` are left unchanged.

## Agent Usage

This CLI is designed for AI agent consumption:

- **Non-interactive** - never prompts, every input is a flag
- **Pipeable** - `--json` output to stdout, errors to stderr
- **Filterable** - `--select <field>[,<field>...]` returns only fields you need
- **Previewable** - `--dry-run` shows the request without sending
- **Read-only by default** - this CLI does not create, update, delete, publish, send, or mutate remote resources
- **Offline-friendly** - sync/search commands can use the local SQLite store when available
- **Agent-safe by default** - no colors or formatting unless `--human-friendly` is set

Exit codes: `0` success, `2` usage error, `3` not found, `5` API error, `7` rate limited or blocked, `10` config error.

## Health Check

```bash
chabad-pp-cli doctor
```

Verifies configuration and connectivity to the API.

## Configuration

Run `chabad-pp-cli doctor` to see the resolved config, data, state, and cache directories. The platform-default config path is `~/.config/chabad-pp-cli/config.toml` on macOS/Linux and `%USERPROFILE%\.config\chabad-pp-cli\config.toml` on Windows; `--home`, `CHABAD_HOME`, and per-kind env vars can relocate it.

Static request headers can be configured under `headers`; per-command header overrides take precedence.

`--ca-cert`, `CHABAD_CA_CERT`, or `ca_cert` adds a PEM CA bundle to the system trust roots. `--tls-server-name`, `CHABAD_TLS_SERVER_NAME`, or `tls_server_name` is the name checked against the certificate when the dialed address does not match it. Precedence is flag, then environment, then the config file. A config file other users can change does not supply these settings. Certificate verification failures are not retried.

## Troubleshooting
**Not found errors (exit code 3)**
- Check the resource ID is correct
- Run the `list` command to see available items

### API-specific
- **No location given and no default set** — Pass --location 11213 (a ZIP or city name), or save one with: chabad-pp-cli locations use 11213
- **A city name matched the wrong place** — Run: chabad-pp-cli locations --query "<city>" and reuse the exact Value it returns (or city:<Value> for a city id) as your location
- **zmanim returns fewer days than requested** — Chabad.org caps one call at about 180 days; split long ranges into several --from/--to calls (holidays and ahead do this for you)
- **HTTP 500 or an XML error body from a webservice** — The service needs Accept: application/json, which the CLI sends; if you call it yourself, add that header

## Discovery Signals

This CLI was generated with browser-captured traffic analysis.
- Target observed: https://www.chabad.org/calendar/zmanim_cdo/aid/143790/jewish/Zmanim-Halachic-Times.htm
- Capture coverage: 3 API entries from 4 total network entries
- Reachability: standard_http (65% confidence)
- Protocols: rest_json (75% confidence)
- Auth signals: none
- Candidate command ideas: list_Get_Locations — Derived from observed GET /WebServices/RemoteCall/Get_Locations traffic.; list_Get_Zmanim — Derived from observed GET /webservices/zmanim/zmanim/Get_Zmanim traffic.; list_Load_CandleLightingWeeks — Derived from observed GET /webservices/Zmanim/CandleLighting/Load_CandleLightingWeeks traffic.

---

## Sources & Inspiration

This CLI was built by studying these projects and resources:

- [**hebcal**](https://github.com/hebcal/hebcal) — Go (189 stars)
- [**pyluach**](https://github.com/simlist/pyluach) — Python (72 stars)
- [**py-libhdate**](https://github.com/py-libhdate/py-libhdate) — Python (42 stars)
- [**zmanim-mcp-server**](https://github.com/ariroffe72/zmanim-mcp-server) — Python (2 stars)
- [**chabad-org-zmanim**](https://github.com/toolsforshlichus/chabad-org-zmanim) — TypeScript
- [**rzman**](https://github.com/hummusonrails/rzman) — Ruby

Generated by [CLI Printing Press](https://github.com/mvanhorn/cli-printing-press)
