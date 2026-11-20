# Memo Structure (Phase 3) — LexisResearch

Clone `assets/research-template.html`. Do not rebuild the CSS or swap the visual style. The memo
is internal attorney work product, not a filed document — write it for the lawyer who has to use
these cites, in the firm's voice: direct, practical, no fluff, get to the point.

**The structure is elastic.** Fill only what the question needs and delete unused sections. A
one-proposition lookup should produce a short memo (Short Answer + Governing Authority + a couple
verified cites + Shepard's Status); a thorough open-question memo fills everything. Do not pad a
narrow question to look comprehensive.

Every authority cited anywhere in the memo must carry its **VERIFIED / UNVERIFIED** tag (set in
Phase 4) and a pin cite read from the Lexis+ reporter page markers.

---

## Section map

1. **Header band** — `[ATTORNEY WORK PRODUCT — PRIVILEGED & CONFIDENTIAL]` and `[REQUIRES ATTORNEY
   REVIEW — NOT FOR FILING WITHOUT INDEPENDENT VERIFICATION]`. Matter name/number if known from
   `claude.md` (omit if it is a bare research question). The verification banner (filled in
   Phase 4, with the date Shepard's was pulled).

2. **Research Question** — the question stated precisely, one or two sentences, with the forum and
   the governing standard/burden if the question implicates one.

3. **Short Answer** — 2–4 sentences a partner could read in fifteen seconds: the rule that answers
   the question and the single lead authority for it. Lead with the answer. If the answer is "no
   clean California authority on point," say that here.

4. **Governing Statute(s) & Rule(s)** — the operative code section(s)/rule(s) cited to
   subdivision, with the operative text from the Deering's copy (quoted verbatim, or tightly paraphrased with the
   subdivision pinned), the elements/test, the burden allocation, and the effective date / any
   recent amendment. Each statute gets a status chip. Omit if the question is purely case-law.

5. **On-Point Authority by Proposition** — THE centerpiece. A table:
   `Proposition | Best Authority (canonical cite + pin) | What it holds / supports | Status |
   Notes (binding vs. persuasive; backup cite)`. One lead authority per proposition; add a backup
   only where it genuinely helps. This is the "drop a verified cite behind each point" deliverable.
   If a proposition has no clean California support, say so in the row rather than stretching a
   case.

6. **Persuasive & Secondary Authority** — sister-state / federal persuasive authority and
   secondary sources (Witkin, CEB, Rutter, law review) that frame or bolster the answer. Label
   each as persuasive/secondary — never dressed up as binding. Include only at Standard/Thorough
   depth; delete for a Quick lookup.

7. **Contrary / Cautionary Authority** — authority cutting the other way, any district split or
   tension, and every Shepard's red flag from Phase 4. Mark each item **live** (a real
   obstacle the attorney must deal with) or **remote** (noted for completeness). Honest counsel:
   if the weight of authority is against the proposition, say so plainly.

8. **Authorities Relied On** — every case and statute in one at-a-glance audit table:
   `Authority | Cite (canonical, with pin) | Proposition | Status (VERIFIED/UNVERIFIED) |
   Shepard's signal`. This is what the attorney scans before relying on the memo. (For a very short
   memo this can be folded into Section 5; keep it separate once there is more than a handful of
   authorities.)

9. **Shepard's Status** — one row per cited case: full reporter cite, its Shepard's signal, the
   red flag(s) with a plain-language bottom line (or "no Warning-level treatment"), and the
   red-flag memo file name. Then a cut-and-paste list of every cited case with full and parallel
   cites. Prefaced with the date the signals were pulled and the reminder that Caution/Questioned
   treatment is listed but not analyzed.

10. **Gaps / Run Down Next** — coverage gaps (including any Lexis+ search that returned nothing
    usable), any UNVERIFIED authority, propositions left uncited, and the single question that
    would most resolve the remaining uncertainty.

11. **Footer** — the disclaimer (AI-assisted research on the attorney's Lexis+ subscription; text,
    pins, quotes, and Shepard's signals pulled on the stated date; red flags summarized, Caution
    not analyzed; not a substitute for the attorney's independent review and judgment; California
    law controls) and the date.

---

## Optional alternate deliverable

If the matter calls for a filed-quality research memorandum rather than an HTML memo, the same
content can be emitted as a `.docx` legal memo via the `docx` skill, or formatted on firm pleading
paper via the `benjy-pleading-engine` family. Default to the HTML memo unless the user asks for
Word.
