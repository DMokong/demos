# Round 1 — the token-burn claim, corrected and explained

`file:///Users/dustincheng/projects/demos/talk/thread.html` · 1 annotation (1 highlight, 0 draw marks)

Edited on **rung 3** of §1a — the bundled document itself. No source path was named and I did not
generate this page in this conversation. The anchor resolved on **step 1** of §4 (selector +
offsets exact: `container_text_length` 527 matched, `slice(94,173)` equalled `quoted_text`
character for character).

## Applied

- **[h1 · fact-check]** *"Opus 5 burning roughly twice the output tokens of Opus 4.8 on comparable
  tasks."* → replaced with a `<mark data-redline="proposed">` reading **"roughly 65% more output
  tokens per call than a frontier baseline"**. One hunk, inside the anchored range. The `title`
  attribute carries the full rationale so it survives out of context.

## Assumptions

- **Corrected the number, not the paragraph.** Your comment supplied a *causal explanation*
  (legacy prompts cause over-verification) rather than a replacement figure. Rewriting the
  paragraph to make that argument would have gone well past the anchor, so I corrected the claim
  within the marked range and put the explanation here and in the `title`. If you want the
  migration-artifact point made in the prose, mark the paragraph `expand` next round.
- **The phrase occurs twice in the file** — once in the prose and once inside the state block as
  `annotations[0].anchor.quoted_text`. I replaced only the prose occurrence; the annotation record
  is byte-identical, per §10.1. Worth knowing, because a naive find-and-replace on this document
  silently corrupts its own provenance.

## Replies

None — this was a `fact-check`, not a `question`.

## Fact-check findings

- **[h1 · fact-check]** Claim: *"Opus 5 burning roughly twice the output tokens of Opus 4.8 on
  comparable tasks."*

  **Verdict: overstated, and confounded.** Two separate problems.

  **1. The number is wrong, or at least unsupported.** The "2×" figure is single-source — a
  Substack post and tech-ish.com — and no independent party has measured it. The one genuinely
  independent measurement that exists is CodeRabbit's code-review bench: **≈9,500 output tokens per
  call against a baseline of ≈5,800 — about +65%, not +100%.** They also report reading ~50% more
  input. Two caveats that matter and are stated in the mark's `title`: CodeRabbit measured against
  "the frontier models in our baseline," **not Opus 4.8 specifically**, and it is one task domain
  (code review), not "comparable tasks" generally. So the corrected figure is better-evidenced than
  the original but is not a like-for-like replacement — I have not silently swapped one precise
  claim for another.

  **2. Your causal point is confirmed, and it is the more important half.** Anthropic's own
  prompting guide for Opus 5 states that the model **verifies its own work unprompted**, that
  carried-over verification instructions ("include a final verification step", "use a subagent to
  verify") cause **over-verification**, and that removing them **reduces wasted tokens with no loss
  in quality** — the migration is described as *mostly deletion*. So an unknown fraction of any
  observed token increase is people running Opus 5 on Opus 4.8-era scaffolding, asking it to
  duplicate work it now does internally. That is a property of the prompt, not the model.

  **Why this matters beyond this sentence:** it means "Opus 5 burns more tokens" and "Opus 5 costs
  more per task" are different claims, and the second does not follow from the first if the burn is
  migration debt. Your own outline §5 already contains the mechanism — *"❌ Double-check your answer
  / verify with a subagent → over-verification, wasted tokens, no quality gain"* — so the talk
  supplies the explanation for the observation two sections earlier. Worth connecting explicitly.

  Action: correction applied wrapped in `<mark data-redline="proposed">` at the anchored range.
  Awaiting your call.

  Sources: [CodeRabbit — Claude Opus 5 Benchmarks for AI Code Review](https://www.coderabbit.ai/blog/opus-5-model-review) ·
  [Anthropic — Prompting Claude Opus 5](https://platform.claude.com/docs/en/build-with-claude/prompt-engineering/prompting-claude-opus-5)

## Verification

- **Diff: 1 hunk, traced to h1.** Verified programmatically: with the proposed `<mark>` removed and
  the original phrase restored, the document is byte-identical to `snapshot`. Nothing else moved.
- **Anchor: 1/1 exact, on §4 step 1.** No fallback rung, no ambiguity.
- **Proposed corrections: 1** `<mark data-redline="proposed">` present, with its rationale in the
  `title` attribute.
- **Annotation record intact.** `annotations[0]` is unmodified; only `round` (1 → 2) and
  `responses` changed in the state block. Exactly one `<script>` in the file — the inert state
  block — so the document still round-trips through the snapshot pipeline.
- **Widths: reasoned from CSS, not rendered.** The edit adds an inline `<mark>` inside an existing
  `<p>`; `body` is `max-width: 42rem` with fluid padding and nothing in the document carries a fixed
  width or `nowrap`, so the added text reflows rather than overflowing at either 860px or 390px.
  The replacement is *shorter* than the original phrase, so it cannot introduce a wrap that was not
  already possible. I did not open a browser to confirm.
- **Not checked:** whether the Substack/tech-ish "2×" figure was itself measured against Opus 4.8
  under matched prompts — neither post documents its methodology, which is part of why it does not
  survive as a claim.
