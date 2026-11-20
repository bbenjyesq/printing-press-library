# Research Method (Phase 1) — Lexis+ first

Work the question in this order: statutes → cases → proposition map → contrary line. Everything
here feeds the draft, which is then verified in Phase 4. Nothing is authority until it clears
verification against Lexis+.

All commands run through the wrapper: `bash "<skill folder>/scripts/lexis.sh" <args>`.
Add `--agent` for JSON. Retrieved Word copies land in `~/Downloads/Lexis Downloads` and are reused
on later calls, so reading a case a second time does not cost another Lexis+ pull.

---

## Step 1 — Pin the governing statute(s)

1. Identify the operative code(s): CCP, Evidence Code, Civil Code, Government Code (incl. FEHA,
   Gov. Code § 12940 et seq.), Labor Code, Corporations Code, Commercial Code, etc.
2. Pull the section from Lexis+ (Deering's, annotated):

   ```bash
   lexis.sh statute get "Cal Lab Code 1102.5" --agent
   ```

   The CLI accepts California Style Manual forms too ("Lab. Code, § 1102.5"); `brief cites -`
   normalizes text on stdin if a form is not recognized. When a section has more than one version,
   the tool picks the version in effect today and says so — record that.
3. Read the operative subdivision as text:

   ```bash
   lexis.sh citecheck text "Cal Lab Code 1102.5" --agent
   ```

   Capture the exact subdivision, the amendment history note, and the effective date. Do not rely
   on recall for statutory text or subdivision lettering.
4. **Mine the annotations.** The Deering's "Notes of Decisions" list the cases construing the
   section, grouped by topic. These are usually the fastest route to the leading cases.
5. Record the burden allocation and any elements or test the statute sets up.
6. **California Rules of Court** may not be available through the CLI. If a rule is load-bearing
   and Lexis+ cannot retrieve it, fetch it from `courts.ca.gov` and tag it as sourced there — the
   one permitted non-Lexis primary source, and it must be named as such in the memo.

## Step 2 — Find the cases on point

**Authority hierarchy:** California Supreme Court > published Court of Appeal (note the district) >
persuasive (sister-state high courts, federal decisions applying California law, Court of Appeal
dicta) > secondary (Witkin, CEB, Rutter — used to find primary authority, cited sparingly).

**Discovery routes, in order:**

1. **Deering's annotations** from Step 1.
2. **Lexis+ search box.** Start a search and read its result:

   ```bash
   lexis.sh lookup start --terms "whistleblower retaliation contributing factor 1102.6" --agent
   lexis.sh lookup result <searchId> --agent
   ```

   The search is pre-filtered to California. A citation returns a single-document hit; a terms
   search returns whatever the result reader can parse. If the result is unusable for terms,
   say so in the Gaps section and move to the next route — do not guess at results.
3. **Citation chain.** Once one good case is pulled, read its text (`citecheck text`) and follow
   the California authorities it relies on.
4. **Finding aids (optional).** CourtListener search or web search may surface candidates,
   especially very recent opinions. A candidate found this way is only a lead: it must be pulled
   from Lexis+ before it can appear in the memo.

**Confirm before you rely.** For each candidate:

```bash
lexis.sh cite check "27 Cal.3d 167" --agent            # Lexis+ recognizes it, and what it is
lexis.sh citecheck text "27 Cal.3d 167" --agent        # writes <name>.citecheck.txt, full text
lexis.sh citecheck text "27 Cal.3d 167" --pages 175-176   # judge-written text on those pages only
```

Read the passage that supports the proposition. Confirm it is in the **majority opinion** — the
text output labels majority, concurrence, dissent, and Lexis editorial blocks — and capture the
**pin cite** from the `[*123]` page markers. A case that says something adjacent to the point is
not on point; say so.

## Step 3 — Map authority to propositions (the centerpiece)

For each proposition the attorney flagged (or each point the question implies), attach the single
best on-point authority, plus one backup only where it genuinely helps.

- One lead cite per proposition; no string-citing for volume.
- Label persuasive-only or distinguishable support as such.
- If a proposition has no clean California support, say so. That is a finding.

## Step 4 — Contrary and cautionary authority

Actively look for authority cutting the other way, district splits, statutory exceptions the
leading cases assume away, and negative Shepard's treatment. A quick early read is worth it:

```bash
lexis.sh case redflags "23 Cal.3d 880" --no-memo --agent
```

The full Shepard's pass happens in Phase 4.

## Depth calibration

- **Quick** — one or two cases and the statute. Short memo.
- **Standard** — governing authority mapped to the propositions, contrary line noted. Default.
- **Thorough** — add persuasive, secondary, analogous lines, and fuller contrary/split analysis.

## Optional parallelization

For a broad question with independent sub-issues, you MAY spawn 1–2 `general-purpose` agents (one
sub-issue each), each told to use only `scripts/lexis.sh` for authority and to return pin cites
from the Lexis+ text. Keep the combined run within the ~60-pull ceiling.
