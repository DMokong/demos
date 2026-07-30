# Round 1 — two questions answered, nothing edited

Session s-20260730-313937 · 8 annotations (1 highlight, 7 draw marks)

## Applied

None. **No byte of the document changed** — `result.html` is byte-identical to
`snapshot.html` (md5 `78fb48c1…` for both).

That is not incompleteness, and it is worth stating plainly rather than burying: of
the eight marks, two are questions (which never produce an edit), and six carry no
instruction at all — five pen strokes and one rect whose only words are also a
question. Nothing in this round licensed a change to the text. Details for every
mark are below; two of them are one word away from becoming edits next round.

## Assumptions

- **[h2 · labelled `instruct`]** Treated as a **question**, not an instruction. The
  intent field says `instruct`, but the comment — *"Why do we have to state this?
  Does it make the argument better?"* — contains no instruction, only two questions.
  The two readings differ sharply in blast radius: answering costs nothing, while
  acting on a guess would delete or rewrite your entire thesis paragraph. I took the
  smaller reading and answered under *Replies*. **The alternative:** if you meant
  "cut it," say so and it goes in one line next round — I have already worked out
  which sentence I would cut and why.
- **[d1 + d2 · pen]** Two red strokes forming an **X over the table's `Owner`
  column** — the `<th>Owner</th>` header plus all three cells (`platform`,
  `product eng`, `product eng`). Neither stroke carries a note, so there is no way
  to tell "delete this column" from "these owners are wrong" or "who really owns
  INP?". **Left untouched**, because the conservative reading is the only safe one
  here and the destructive reading fights the article itself: the paragraph directly
  above ends *"a budget with no owner is a wish"*, and your closing line is *"they
  can tell you who signed off."* Deleting the owners would cut the evidence for both.
  Tell me which of the three readings you meant.
- **[d8 + d9 + d10 · pen]** A green smiley (two eyes, one curve) beside the `<h1>`
  in the header. Read as approval of the headline, not an instruction. No edit.
- **Other pending rounds, deliberately skipped.** `./inbox` held four pending rounds;
  I took this one as newest by `created_at` (`2026-07-30T03:57:05Z`) and left
  `s-20260729-dff0fd/round-1` (18:59:40Z), `s-20260729-2a0b58/round-1` (18:59:38Z),
  and `s-20260729-372474/round-2` (18:59:37Z) untouched.

## Replies

- **[h2 · question]** *"This is a draft of an argument I keep making in review
  meetings, written down so I can stop making it out loud. The claim is simple:
  latency is a product decision, and if you do not decide it deliberately, it gets
  decided for you by whoever merged last."*

  The paragraph is doing two jobs, and they are not equally worth keeping. Sentence
  two is your thesis — everything downstream (the three numbers, the owner column,
  the enforcement mechanism) is an argument for it, so it has to stay somewhere near
  the top. Sentence one is framing: it tells the reader this is a rehearsed argument
  being written down. That one is costing you something. It pre-apologises for the
  piece before the piece has made its case, and it is off-voice — everywhere else you
  are flatly declarative (*"a budget with no owner is a wish"*), which is the register
  that makes this readable. Your lede has also already done the framing work, and
  better, by ending on the audit-versus-design-discussion turn.

  **Recommendation: cut sentence one, keep sentence two.** So the answer to "does it
  make the argument better" is: the claim does, the preamble does not. Say the word and
  that is the round-2 edit — one sentence deleted, nothing else touched.

- **[d4 + d5 · rect + note]** *"where did this quote come from ?"* — boxing the
  blockquote, *"A performance budget is not a diet. It is a currency: it lets you say
  yes to the expensive feature you actually want by saying no to the four you did not
  notice you were buying."*

  Nothing in the document attributes it: no `<cite>`, no source, no footnote. I searched
  for the exact wording and found no published origin, and it matches your own voice
  closely enough (the "four you did not notice" cadence is yours) that I read it as
  your line rather than a quotation you are half-remembering.

  One thing you should know before publishing it unattributed, though: **the "currency"
  half of the metaphor is prior art in this exact field.** Addy Osmani's performance-
  budgeting writing frames budgets as *"a currency to spend and trade on user-experience"*,
  and there is a post titled literally "Web Performance Budgets as currency." The
  diet-versus-currency contrast looks like yours; the currency framing is not. So:
  if it is your line, it reads better as prose than as a pull quote — a quote block with
  no attribution invites exactly the question you just asked. Either fold it into the
  preceding paragraph as that section's closing claim, or keep the block and nod to the
  prior art. If you were quoting someone, Osmani is who you are reaching for, and it
  needs a real citation.

  No edit made either way — this was a question.

## Fact-check findings

None — no annotation in this round carried the `fact-check` intent. (The provenance
research for the blockquote is reported under *Replies*, since that mark was a question.)

Worth one line, since your own footer raises it: **"HTTP/2, standardised in 2012"**
is still in the document and still wrong (RFC 7540 was published May 2015), and your
draft notes say *"Check the HTTP/2 date before this goes anywhere."* No annotation
points at it this round, so **I did not touch it** — flagging it, not fixing it. Mark it
and it gets a wrapped correction next round.

## Verification

- **Diff: 0 hunks.** `cmp snapshot.html result.html` reports no difference; both files
  md5 `78fb48c1c25bd4a82dd8cbc2b36fc594`. Every unannotated byte is therefore trivially
  byte-identical, and no hunk needs tracing to an annotation because there are none.
- **Anchor resolution: 1/1 exact, on the first rule.** h2's `#post > p:nth-of-type(2)`
  resolved directly — `container_text_length` 253 matches the element's real
  `textContent` length, and `slice(0, 253)` equals `quoted_text` character for
  character. No fallback ladder needed, no unresolved anchors.
- **Widths: 860px (the annotated width) and 390px — no overflow at either.** Reasoned
  from the CSS rather than rendered, since the output is byte-identical to the input and
  no edit exists that could introduce a regression. At 390px, `.wrap` is fluid
  (`max-width: 720px`, 20px side padding → 350px of content) and nothing in the document
  carries a fixed width, a `min-width`, or `white-space: nowrap`. The only real overflow
  candidate is the metrics table, which is `width: 100%` with `border-collapse: collapse`
  and 6px cell padding, so its longest cells (`< 200 ms p75`, `product eng`) wrap rather
  than push the table wide. No images to scale.
- **Proposed corrections: 0** `<mark data-redline="proposed">` elements, correctly —
  nothing was corrected, so nothing needed wrapping.
- **Not checked:** the two width readings were derived from the stylesheet, not measured
  in a browser. Whether the X on the `Owner` column means delete, dispute, or query is
  unresolved by design — that is a question for you, not something I could verify.
