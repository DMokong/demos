# Round 1 — tighten the intro, fact-check the HTTP/2 date, lead with the numbers

Session s-20260729-372474 · 4 annotations (3 highlights, 1 draw mark)

## Applied
- **[h1 · instruct]** "This is a draft of an argument I keep making in review meetings, written down so I can stop making it out loud. The claim is simple: latency is a product decision, and if you do not decide it deliberately, it gets decided for you by whoever merged last." → cut from four sentence-length clauses to two sentences, same claim, no new content: "This is a draft of an argument I keep making in review meetings, written down so I can stop repeating it. Latency is a product decision: decide it deliberately, or it gets decided for you by whoever merged last."
- **[d1 · arrow]** Moved the `<h2 id="s-numbers">The three numbers worth arguing about</h2>` heading + its 3 following blocks (`<p>`, `<ul>`, `<p>`, `<table>`) above the `<h2 id="s-where">Where the budget actually goes</h2>` heading. Markup preserved verbatim; only position changed. The note named the unit explicitly ("heading, list, and table"), so scope was not ambiguous.

## Assumptions
None.

## Replies
- **[h2 · question]** "Every product team I have worked with has a performance budget."
  It works, and I'd keep it. For an engineering audience the risk with a claim this broad is usually that it reads like a keynote slide, but this one earns its confidence in the next sentence ("almost none of them agreed to it") — that's the actual argument, not a warm-up. What I would *not* do is soften it into a qualified claim ("many teams…"); the flat statement is what makes the next line land as a reveal rather than a hedge. If you want a second opinion in the room: cut it only if you're trying to save length elsewhere, not because it's "too editorial" — it isn't, for this audience.

## Fact-check findings
- **[h3 · fact-check]** Claim: "HTTP/2, standardised in 2012, was supposed to make bundling obsolete overnight."
  Verdict: incorrect. Evidence: HTTP/2 was standardised by the IETF as RFC 7540, published in May 2015 (it derives from Google's SPDY, whose early drafts circulate from around 2009–2012 — that's likely the source of the mix-up, but the standard itself is 2015).
  Action: proposed correction "2012" → "2015", wrapped in `<mark data-redline="proposed" title="proposed correction: HTTP/2 was standardised as RFC 7540 in May 2015, not 2012">2015</mark>` in the "Where the budget actually goes" section. Awaiting your call.

## Verification
- Diff: 3 hunks (2 text edits + 1 block relocation), all traced to h1, h3, and d1/d2; every unannotated paragraph, the lede, the table, the blockquote, the footer, and the `<style>` block are byte-identical to `snapshot.html`.
- Widths: rendered `result.html` in a headless browser at 860px (the captured width) and 390px and inspected both screenshots. At 390px the relocated "three numbers" section reflows cleanly — the table's three columns stay legible without horizontal scroll (the existing `table { width: 100% }` rule already handled narrow widths before the move), the `<ul>` bullets wrap normally, and no heading collides with the block above or below it. No overflow introduced by the move.
- Proposed corrections: 1 `<mark data-redline="proposed">` element present and readable (renders with default browser `<mark>` styling since the page defines no override — it is visually distinct from body text at both widths).
- Not checked: I did not verify HTTP/2's RFC number against a live web fetch (no network tool available in this pass); the 2015 date and RFC 7540 reference are from prior knowledge and are the kind of fact worth a second, independent check before publication given the footer already flags this exact date as suspect.

Round 1 is written to `inbox/s-20260729-372474/round-1/`. Open `result.html` in
redline for round 2 — the proposed correction in the HTTP/2 section is waiting on you.
