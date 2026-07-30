# Round 1 — three annotations, none actionable

`sample:sample/article.html` · 3 annotations (3 highlights, 0 draw marks)

Edited on **rung 3** of §1a — the bundled document itself. No explicit path was named
and I did not generate this page in this conversation, so rungs 1 and 2 did not apply.
All three anchors resolved on **step 1** of §4 (selector + offsets exact).

## Applied

None. **The document is byte-identical** outside the state block.

Not a shortfall to skim past: none of the three comments contains an instruction, a
question, or a claim. Acting on any of them would mean inventing intent the author never
expressed, against a paragraph-sized anchor. Each is listed below with what it referenced.

## Assumptions

- **[h1 · fact-check]** Comment is `"the quick brown fox"` — a pangram, not a claim.
  It anchors the **entire lede paragraph** (419 chars), which contains no verifiable
  factual assertion in any case; it is a statement of the author's experience. Nothing to
  verify, so nothing was verified and nothing was marked. Left `open`.
- **[h3 · instruct]** Comment is `"asdasdasd"`. `instruct` means "apply exactly that
  change to exactly that range", and no change is specified. The anchor is the **entire**
  494-character paragraph, so a guess here would rewrite a whole paragraph on no
  instruction. Left `open`.
- **[h2 · question]** Comment is **empty** (`""`). There is no question to answer.
  Left `open`.
- **All three were left without a `responses` entry, deliberately.** §9 reserves entries
  for annotations acted on. Writing `{"kind":"replied"}` with an explanation of why there
  was nothing to reply to would render in redline's gutter as **answered** — telling you
  the round is settled when it is not. Leaving them out keeps all three showing as **open**,
  which is the truth.

## Replies

- **[h2 · question]** "Lab numbers are reproducible and field numbers are true. You need
  both, and teams that only have one of them tend to argue past each other for a quarter
  before noticing."

  No reply is possible — the comment is empty, so the annotation records *where* you wanted
  to ask something but not *what*. The anchor is intact and the highlight is still live, so
  typing the question next round costs nothing; it will land on exactly this sentence.

## Fact-check findings

None performed — see **[h1]** above; its comment supplied no claim.

One thing worth flagging, because the pairing looks accidental rather than intended:

- The **fact-check** annotation sits on the lede, which has nothing checkable in it.
- The paragraph that *does* contain a false claim — *"HTTP/2, standardised in 2012"*
  (RFC 7540 was published **May 2015**) — is annotated **h3 · instruct**, not `fact-check`,
  and its comment is `"asdasdasd"`. Your own footer already says *"Check the HTTP/2 date
  before this goes anywhere."*

  I did **not** correct it. Its annotation's intent is `instruct` and no instruction was
  given, and §10.4 forbids letting a fact-check correction in unwrapped and unrequested.
  Re-mark that paragraph as `fact-check` and it gets a `<mark data-redline="proposed">`
  correction next round.

## Verification

- **Diff: 0 hunks.** Everything outside the `redline-state` block is byte-identical between
  the round-1 bundle and the round-2 output (compared programmatically, not by eye). No hunk
  needs tracing to an annotation because there are none.
- **Anchors: 3/3 exact, all on §4 step 1.** For each, `container_text_length` matched the
  element's real `textContent` length and `slice(start_offset, end_offset)` equalled
  `quoted_text` character for character. No fallback rung used, no unresolved anchors, no
  ambiguity.
- **Widths: not applicable, and not checked.** Zero bytes of document changed, so no edit
  exists that could introduce overflow at 860px or 390px. Reporting this as "checked" would
  be a false claim.
- **Proposed corrections: 0** `<mark data-redline="proposed">` elements, correctly — nothing
  was corrected, so nothing needed wrapping.
- **State block:** `round` bumped 1 → 2, `responses` deliberately left `{}`, every other
  field preserved verbatim.
- **Not checked:** whether the three comments are placeholders from a smoke test or genuinely
  intended text. I read them as written and did not assume.
