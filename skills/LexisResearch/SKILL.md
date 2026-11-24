---
name: LexisResearch
description: >
  [OUTCOME]: A verified answer to ONE California legal-research question built on the attorney's
  own Lexis+ subscription as the source of record — the governing statutes (Deering's text
  confirmed to subdivision), the on-point cases (holdings and pin cites read from the Lexis+ copy
  with its reporter page markers, ranked binding > persuasive > distinguishable), every quotation
  matched word for word against Lexis+, and a live Shepard's signal for every case with red flags
  explained, organized into a proposition-by-proposition citation table as a self-contained HTML
  memo. California law controls.
  [TRIGGER]: "/LexisResearch," "lexis research," "research this on Lexis," "find me a case on
  Lexis for," "is there authority for X (use Lexis)," "pull cases on point from Lexis+,"
  "citation for the proposition that X, verified on Lexis."
  [ANTI-TRIGGER]: Not the CourtListener-based /legal-research skill (a separate tool). Not for
  cite-checking an existing brief (/lexis-citecheck), downloading known citations
  (/lexispull), war-gaming an issue or predicting a ruling (storm-legal), planning a motion
  (stormmotionplan), or drafting a pleading.
argument-hint: "[legal-research question, or the proposition you need a cite for]"
metadata:
  version: "1.0.0"
---

# /LexisResearch — on-point authority from Lexis+

## What this does

Takes ONE discrete California legal-research question and returns the authority that answers it:
the governing statute(s) cited to subdivision, the leading cases with holdings and pin cites, the
set ranked binding > persuasive > distinguishable, mapped proposition by proposition so the
attorney can drop a verified cite behind each point of a brief.

**Lexis+ is the source of record.** Every case and statute in the memo is retrieved from the
attorney's own Lexis+ subscription through the local `lexis-plus-pp-cli`. Holdings and pin cites
are read from the Lexis+ text with its official `[*123]` reporter page markers; quotations are
matched word for word against that text; statutory text comes from Deering's; good-law status
comes from the case's live Shepard's report. Other sources (CourtListener, web search) may be
used only as **finding aids** to surface candidate authority. Nothing is cited, quoted, or pinned
from them.

This is a standalone skill. It shares a memo style with `/legal-research` but not its sources,
steps, or files. Keep it lean: it finds and verifies authority; it does not war-game.

**Phase 4 (verification) is non-negotiable.** A fabricated or miscited authority is sanctionable
(CCP § 128.7). The point of this tool is authority the attorney can trust.

## Before you start

1. **Lexis+ access.** **First, make sure the Lexis+ session is live.** Before any Lexis+ work, tell the user in one line: "Checking your Lexis+ sign-in. If it has expired, a Chrome window will open; sign in there and it closes by itself." Then run, with a 10-minute timeout:
   
   ```bash
   bash "<this skill's folder>/scripts/lexis-session.sh" ensure
   ```
   
   - `LEXIS_SESSION_OK` (exit 0): continue.
   - `LEXIS_SESSION_LOGIN` then `LEXIS_SESSION_OK` (exit 0): the session had expired and the user signed in again; continue.
   - Exit 4 (`LEXIS_SESSION_EXPIRED`): the user did not finish signing in. Say so and stop; offer to try again.
   - Exit 1 or 3 (`LEXIS_SESSION_ERROR`): report the message plainly and stop. Lexis+ may be unreachable, or the CLI is missing.
   
   Never ask the user for their password, never type it anywhere, and never run `press-auth` or `auth login` by hand; the script does both.
   Without a live session this skill cannot deliver; do not fall back to another source for authority. Offer `/legal-research` instead.
2. **Per-matter context.** If the current directory has a `claude.md`, read it (parties, claims,
   posture, venue, judge) and use it to sharpen the question. Optional.
3. **Firm context.** Read `firm-context.md` in the working directory if present. Default:
   California law; Superior Court trial forum; plaintiff's employment and creditor-side practice;
   California Style Manual citation form (CRC 1.200), used consistently.
4. **Learnings.** Read `learnings.md` in this skill's folder and apply any `## Rules`.
5. **Pacing.** The CLI paces Lexis+ requests itself. Keep a single run to roughly 60 Lexis+
   document pulls or fewer — Lexis's terms do not allow bulk downloading. Prefer reading a few
   cases well over pulling many.

All CLI calls go through the wrapper: `bash "<this skill's folder>/scripts/lexis.sh" <args>`.
Add `--agent` for JSON output. Use timeouts of 5 minutes or more for pulls.

## Phase 0: Frame the question

1. If `$ARGUMENTS` supplies the question, use it. Otherwise ask (AskUserQuestion) for:
   - **The precise research question** (free text).
   - **The propositions to support, if any** (free text; "N/A — just tell me the law" is fine).
   - **Forum**: CA Superior Court / CA Court of Appeal / Federal district (C.D. Cal. etc.) / Not sure.
   - **Depth**: Quick (a cite or two) / Standard (governing authority mapped to propositions) /
     Thorough (plus persuasive, secondary, and contrary lines).
2. Restate it in one sentence as the RESEARCH QUESTION and derive a kebab-case `question-slug`.
   Ask a clarifying question only if ambiguity would change the research.
3. Tell the user the pipeline is running (Lexis+ research → Lexis+ verify). One line.

## Phase 1: Research

Read `references/research-method.md` and follow it. In summary:

1. **Statutes first.** Identify the governing section(s) and pull them with `statute get` (Deering's,
   annotated). Read the operative subdivision with `citecheck text "<statute cite>"`. Use the
   Deering's annotations ("Notes of Decisions") to find the leading cases construing the section.
2. **Cases on point.** Find candidates through Lexis+ first: Deering's annotations, the Lexis+
   search box (`lookup start --terms` / `lookup result`), and the authorities cited inside cases
   already pulled. CourtListener or web search may supplement discovery only. For every candidate,
   confirm it exists in Lexis+ (`cite check`), then read the relevant pages of the Lexis+ text
   (`citecheck text --pages`) to confirm the holding and capture the pin. No snippet citations.
3. **Map to propositions.** One best authority per proposition (plus a backup where useful).
4. **Contrary & cautionary.** Look for authority cutting the other way, splits, and Shepard's
   signals. If the research does not support the hoped-for proposition, say so.

## Phase 2: Rank & organize

Sort every authority: **binding CA on point > persuasive CA/sister-state/federal >
distinguishable > dicta/secondary.** Resolve one lead cite per proposition. Separate the contrary
line. Identify gaps.

## Phase 3: Assemble the memo (HTML)

Read `references/report-structure.md` and clone `assets/research-template.html` — do not rebuild
its CSS. The structure is **elastic**: fill only what the question needs. Write to
`lexis-research/{question-slug}-memo.html` in the working directory (create the folder if needed).

## Phase 4: Verify every authority against Lexis+ (do not skip)

Read `references/lexis-verification.md` and follow it exactly. In summary:

1. **Existence.** `cite check` every cited authority. Anything Lexis+ does not recognize is
   corrected or **cut**.
2. **Holding and pin.** For each load-bearing case, read the cited pages with
   `citecheck text "<cite>" --pages <pin>` and confirm the proposition is on that page, in the
   majority opinion (not a dissent, concurrence, or Lexis editorial headnote).
3. **Quotations.** Run every quotation through `citecheck quote` with its pin. Any un-signaled
   difference is fixed to match Lexis+ or the quote becomes a paraphrase.
4. **Shepard's.** Run `case redflags` on every cited case. A red-flagged case is dropped from
   load-bearing use or its limits are stated, with a reason drawn from the deciding court's own
   language. Caution/Questioned signals are listed in the Shepard's Status table.
5. **Statutes.** Confirm the cited subdivision's text and currency against the Deering's copy.
6. **Tag** every authority VERIFIED / UNVERIFIED and fill the banner truthfully.

## Phase 5: Self-check

Re-read the memo cold against `references/quality-checklist.md` and fix anything that fails. For a
high-stakes question you MAY give the checklist to one `general-purpose` agent with clean context.

## Output

1. Final deliverable: `lexis-research/{question-slug}-memo.html`. The Lexis+ Word copies of every
   authority are in `~/Downloads/Lexis Downloads` (or `--out` if one was given) — mention the folder.
2. Open or present the memo.
3. In chat, a tight summary: the short answer in one or two lines; the best cite for the headline
   proposition; the verification tally; any red flag; and any gap or contrary authority.

## Citation discipline & legal guardrails (mandatory)

- **Lexis+ or silence.** No authority is named unless it was retrieved from Lexis+. If Lexis+
  cannot retrieve it, it is cut, not hedged.
- **Quotation lockdown.** No quoted language unless it passed `citecheck quote` against the
  Lexis+ text, with a pin.
- **Holding, not headnote.** Never cite a case for language that appears only in a Lexis headnote,
  case summary, or other editorial material; `citecheck text` labels those blocks.
- **Label every authority** VERIFIED or UNVERIFIED; cite statutes to subdivision and cases to pin.
- **Shepard's is a snapshot.** Report the date signals were pulled; red flags are summarized, and
  Caution/Questioned treatment is not analyzed. Tell the attorney to re-check before filing.
- **Attorney review required.** Every memo carries **[ATTORNEY WORK PRODUCT — REQUIRES ATTORNEY
  REVIEW]** and must not be filed or relied on without the attorney's own judgment.
- **Unpublished opinions (CRC 8.1115).** Do not build on an uncitable unpublished California
  opinion; flag it if it is the only thing on point.
- **The subscription is the attorney's.** Use it only for this research; never share documents
  pulled from Lexis+ outside the attorney's own machine.

## Learnings

After delivering, ask: "Did this research hit the mark? Anything to refine for next time?" Log
durable feedback to `learnings.md` (`## Rules` for always-apply, `## Observations` for patterns).
Prune past ~50 lines.
