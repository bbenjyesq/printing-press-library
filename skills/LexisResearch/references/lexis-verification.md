# Lexis+ Verification Protocol (Phase 4)

Run this on the fully assembled draft, before Phase 5 and before delivery. Every authority in the
memo is confirmed against the Lexis+ copy, every quotation is matched word for word, and every
case carries a dated Shepard's signal.

All commands run through `bash "<skill folder>/scripts/lexis.sh" <args>`; add `--agent` for JSON.

---

## Step 1 — Existence: `cite check`

List every case and statute cited anywhere in the draft (`brief cites <memo.html-as-text>` can
extract them from a text copy of the draft) and check them in one call:

```bash
lexis.sh cite check "27 Cal.3d 167" "17 Cal.5th 46" "Cal Lab Code 1102.5" --agent
```

For each result:
- **Recognized, and the document title matches the case name used** → continue to Step 2.
- **Recognized, but the title names a different case** → the hallucination signal (transposed
  numbers or a wrong name). Find the real case or cut the cite.
- **`error (not checked)`** → the lookup failed (timeout, Lexis+ error); exit 5. That says nothing about the citation: retry it, and if it still fails, tag it UNVERIFIED and say it was not checked. If the error is an expired session (exit 4), run `scripts/lexis-session.sh login` as SKILL.md says and rerun.
- **Not recognized** → correct it (parallel cite, typo) or cut it and substitute verified
  authority, or soften the proposition. A proposition left uncited is fine; a phantom cite is not.

Record each resolution (corrected / cut / substituted) for the banner.

## Step 2 — Holding and pin for each LOAD-BEARING authority

Load-bearing = anything in the Short Answer, anything carrying a proposition in the centerpiece
table, and the dispositive statute. For each case:

```bash
lexis.sh citecheck text "<cite>" --pages <pin or range>
```

Confirm:
1. The proposition is actually on the pinned page(s). Fix the pin if it is on a neighboring page.
2. The words come from the **majority opinion**, not a concurrence, dissent, footnote quoted from
   another case, or Lexis editorial material (headnotes, case summary, core terms). Editorial
   material is never cited as the court's holding.
3. It is a holding, not dicta, as far as the opinion shows. If it is dicta, say so in the memo.

If the case does not support the proposition as stated, fix the characterization or move the
case to the contrary/cautionary column.

## Step 3 — Quotations: `citecheck quote`

Every quotation in the memo goes through the quote checker with its pin:

```bash
lexis.sh citecheck quote "27 Cal.3d 167" --quote "<exact text as the memo has it>" --pin 178 --agent
```

For several at once, use `--input quotes.json` with an array of `{"id","quote","pin"}`.

- Brackets and ellipses are treated as signaled changes. Every other difference is a defect:
  correct the quotation to match Lexis+ exactly, or turn it into a paraphrase.
- If the checker reports a different page, correct the pin.
- A pin matches only when it covers every page of the quotation. If the checker says the pin covers only part of it, use the full range it reports (e.g. `713-714`).
- If it reports that the words come from a dissent, concurrence, or editorial material, either
  attribute them correctly or drop the quote.

## Step 4 — Shepard's: `case redflags`

Run on every cited case (not only load-bearing ones):

```bash
lexis.sh case redflags "<cite>" "<cite>" --agent
```

For each case, record its `shepards_signal` and the date pulled.

- **Red flag (Warning).** Read each flag's `court_language`. Decide what the case may still be
  cited for. If the red flag reaches the proposition the memo relies on, drop the case from
  load-bearing use and substitute; otherwise keep it and state the limit in the memo. Finish the
  Word red-flag memo the tool wrote with a bottom line and one reason per flag:

  ```bash
  lexis.sh case redflag-memo "<cite>" --overall "<bottom line>" --reason "<reason for flag 1>" [--reason ...]
  ```

  If `court_language` is empty for a flag, write the reason from the treatment label and
  `lexis_summary`, say so, and do not invent one.
- **Caution (yellow) / Questioned (orange).** List in the Shepard's Status table without
  analysis, and tell the attorney to read the report in Lexis+ before relying on the case.
- **No negative signal.** Say "no Warning-level treatment," not "good law."

## Step 5 — Statutes

For each cited section, confirm against the Deering's copy pulled in Phase 1 (`citecheck text`
on the statute cite):
1. The cited subdivision says what the memo claims. Quote operative language verbatim.
2. Note the effective date and any recent amendment; if the tool reported picking among versions,
   say which version the memo relies on.
3. Tag VERIFIED, or UNVERIFIED if the text could not be confirmed.

A Rule of Court fetched from `courts.ca.gov` (see research-method Step 1.6) is tagged VERIFIED with
the source named, since Lexis+ did not supply it.

## Step 6 — Tag and banner

- **Tag every authority** VERIFIED (retrieved from Lexis+, holding and pin confirmed, quotes
  matched) or UNVERIFIED (could not be confirmed — must be corrected or cut; nothing ships as
  authority while UNVERIFIED).
- **Fill the banner truthfully:** `N citations checked · X unverified/cut · Y corrected ·
  Z red-flagged · C Caution/Questioned`, with the date Shepard's was pulled.
- **Fill the Shepard's Status table** (one row per case) and the cut-and-paste cite list.

---

## California citation rules (apply throughout)

- **Form.** CRC 1.200 permits the California Style Manual or the Bluebook, used consistently.
  Default to the California Style Manual for state-court work.
- **Reporters.** Supreme Court to **Cal.** (Cal.5th), Court of Appeal to **Cal.App.**
  (Cal.App.5th), with the parallel **Cal.Rptr.** or **P.** cite where Lexis+ shows one. Give pin
  pages from the official reporter page markers.
- **Unpublished opinions (CRC 8.1115).** Generally must not be cited or relied on, except law of
  the case, res judicata/collateral estoppel, or a related criminal/disciplinary matter. Flag it if
  it is the only thing on point.
- **Federal forum.** Same discipline; use the federal analogs (FRCP, Rule 11) and the controlling
  Ninth Circuit / district authority, and confirm which sovereign's law governs the point. The
  CLI's search is pre-filtered to California; federal authority may need the finding-aid route
  and must still be pulled from Lexis+ before it is cited.
