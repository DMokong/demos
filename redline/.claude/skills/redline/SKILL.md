---
name: redline
description: Process a redline round — apply a human's page annotations (anchored highlight comments) to the annotated document and write the edited document plus changes.md. Use when pointed at a redline bundle — a downloaded redline-<slug>-r<N>.html, "process this redline bundle", or any .html file carrying a <script type="application/redline+json" id="redline-state"> block.
---

# redline — process one round of a co-authoring loop

You are the other half of a two-party loop. A human annotated a page; the bundle
they handed you is their turn. You produce the edited document + `changes.md`;
that is your turn. They open your document back in redline and annotate again.
Round N+1 is a different conversation than round N — behave like a collaborator
who will be read, not a formatter that will be diffed away.

---

## 0. PRIME DIRECTIVE — unannotated content is sacred

**Every byte the author did not annotate stays exactly as it is.**

- Do not rephrase, retitle, restructure, reorder, "tighten", or "improve"
  anything that has no annotation pointing at it. Not even obvious typos —
  unless an annotation asks for a proofread.
- Do not reformat the HTML. No re-indenting, no attribute reordering, no
  prettifier, no minifier, no tag normalisation, no "while I was in there".
  Copy the document you were given and edit in place with surgical string edits.
- Do not touch `<head>`, `<style>`, scripts, classes, or ids unless an
  annotation is about layout or styling.
- Minimal diff is the goal. If a note says "tighten this paragraph", one
  paragraph changes and nothing else does.
- A `question` never produces an edit. A `fact-check` never produces a *silent*
  edit.

The author's words are the product. You are editing someone's draft in their
house. Diffing the document you were given against the one you write should read
like a review, not a rewrite.

---

## 1. Trigger and scope

Triggers: a path to a redline bundle, "process this redline bundle", or a `.html`
file containing a `<script type="application/redline+json" id="redline-state">`
block.

**The bundle is the page.** One file carries both the document and the state:

    redline-<slug>-r<N>.html
      <html>…</html>                     the document, exactly as annotated
      <script type="application/redline+json" id="redline-state">
        { schema_version, document_key, round, annotations[], responses{} }

Read `schema_version` before anything else; this skill handles `"2.0"`. If the
file has no block, it is not a bundle — say so and stop rather than guessing.

One bundle = one unit of work. Do not edit other bundles, and do not edit the
tool's source.

---

## 1a. Which document do you edit

1. An explicit path the user names → that file.
2. The source you **already have in session context** — you published or
   generated the page during this conversation → that file.
3. Neither → the bundled document itself.

Rung 2 requires knowledge you already hold. Do **not** search the filesystem
for a plausible source, infer one from the page title, or match on content
similarity. Editing the wrong file is the worst failure this tool can
produce, and it is worse than falling through to rung 3. When in doubt, drop
to rung 3 and say so.

On rungs 1–2 the anchors were captured against *rendered* output, so CSS
selectors and textContent offsets may not apply. Resolution degrades to
quoted-text search. Report which rung you used and which resolution step
succeeded, so a weak match is visible rather than silent.

---

## 2. Reading order (do not skip, do not reorder)

1. **The `redline-state` block** — `schema_version` first, then `document_key`,
   `round`, and `viewport` (the width the author was looking at).
2. **`annotations`** — the instructions. Read all of them before editing
   anything; two annotations often constrain each other. `responses` from
   earlier rounds is your own prior turn — read it so you do not repeat or
   contradict yourself.
3. **The document** — the one you picked in §1a. Read the regions the
   annotations point at, plus enough around them to match voice and markup
   conventions. You do not need to read the whole file to change two paragraphs.

If `round > 1` and a previous `changes.md` sits beside the target, skim it: it
tells you what you already told this author, what they accepted, and what they
are pushing back on.

---

## 3. The state block — the schema you will actually get

Top level (`schema_version: "2.0"`):

```jsonc
{
  "schema_version": "2.0",
  "document_key": "file:///Users/d/src/article.html",  // identity, not a path
  "round": 2,
  "viewport": { "width": 860, "gutter_width": 320 },
  "annotations": [ /* highlight objects, in DOCUMENT order */ ],
  "responses": {  /* yours, from earlier rounds — see §9 */ }
}
```

`annotations` carries only the annotations still awaiting you: answered,
dismissed and retired ones stay in the browser and never reach the bundle. It is
sorted by position in the document. Work through it in that order: an edit near
the top can shift nothing below it if you keep diffs minimal, but reading in
order is how you notice two comments that constrain each other.

Draw marks are page-pixel coordinates against one rendered layout, so they are
round-scoped: redline retires them in the browser at round end and they do not
appear in the block. §6 tells you how to read one if it ever reaches you by
another route.

**Highlight object** — a comment anchored to an exact text range:

```jsonc
{
  "id": "h1",
  "number": 1,                      // the badge number the author saw in the gutter
  "intent": "instruct",             // instruct | question | fact-check | expand | delete
  "comment": "tighten this — two sentences, not four",
  "anchor": {
    "selector": "#post > p:nth-of-type(2)",  // CSS path to the CONTAINING element
    "container_tag": "p",
    "container_text_length": 412,            // that element's textContent length
    "start_offset": 0,                       // char offsets into el.textContent
    "end_offset": 137,
    "quoted_text": "Ask a team where their page weight lives…",
    "prefix": "…32 chars of context before",
    "suffix": "…32 chars of context after",
    "document_start_offset": 1804,           // same offsets into body.textContent
    "document_end_offset": 1941
  },
  "rects": [ {"x": 40, "y": 612, "w": 780, "h": 19} ],  // page px at viewport.width
  "created_at": "…"
}
```

**Draw object** — a whiteboard mark in page pixel coordinates:

```jsonc
{
  "id": "d1",
  "type": "pen",                    // pen | rect | arrow | note
  "color": "#ef4444",
  "stroke_width": 4,
  "points": [ {"x": 120, "y": 300}, … ],   // pen: the path
                                            // rect/arrow: exactly two points
                                            // note: one point (its top-left)
  "bounds": {"x":120,"y":300,"w":340,"h":80},
  "note": "move this above the table",      // text, or "" 
  "attached_to": "d2",                      // set on notes: the shape they explain
  "created_at": "…"
}
```

Coordinates are CSS pixels in the document rendered at `viewport.width`, origin
at the document's top-left. `x >= viewport.width` means the mark is in the
comment gutter, not on the page.

---

## 4. Locating an anchor (the algorithm, in order)

Parse the document you are editing and, for each annotation:

1. **Selector + offsets.** `el = querySelector(anchor.selector)`. If
   `el.textContent.slice(start_offset, end_offset) === anchor.quoted_text`,
   you have it. This is the normal case and it is exact.
2. **Context search.** Otherwise search the document's `body.textContent` for
   `prefix + quoted_text + suffix`. A unique hit is the range.
3. **Quote search.** Otherwise search for `quoted_text` alone and take the
   occurrence nearest `document_start_offset`. Only accept it if it is
   unambiguous.
4. **Stop.** If none of the above lands, or the quote appears several times with
   no way to choose: **do not guess and do not edit.** Record it under
   *Assumptions* in changes.md as an unresolved anchor, quoting the text and the
   comment verbatim so the author can re-mark it next round. A missed
   annotation is a small failure; editing the wrong paragraph is a large one.

Notes:
- Offsets are into `textContent`, so they ignore tags but include whitespace
  exactly as it appears in the file. Do not trim before slicing.
- The range may cross inline tags (`<strong>`, `<a>`). Edit the HTML so the
  rendered text changes; preserve inline markup unless the annotation is about it.
- Never rewrite the surrounding block to make an edit easier.

Sanity-check yourself before editing: if a highlight's `quoted_text` does not
appear in your intended edit's neighbourhood, you are in the wrong place.

---

## 5. Intent semantics

| intent | what you do | what you must NOT do |
|---|---|---|
| `instruct` | Apply exactly that change to exactly that range/scope. | Extend the edit past the anchor. |
| `expand` | Grow that section in the author's established voice and structure. Mark every addition in changes.md so they know what is new. | Rewrite what was already there. Add sections they did not ask for. |
| `delete` | Remove the range. Fix only references that are *immediately* broken by the removal (a dangling "as shown above"), and list each such fix. | Tidy up the paragraph "while you're there". |
| `question` | **Do not edit anything.** Answer it in *Replies*, quoting the anchor text. Be direct and useful; a real answer, with a recommendation if you have one. | Change the text. "Improve" it because the question implied doubt. |
| `fact-check` | Verify the claim (web research if tools allow, otherwise reason from what you know and say so). Report in *Fact-check findings* with what you checked against. If the claim is wrong, apply the correction **wrapped** so it is visually distinct: `<mark data-redline="proposed" title="proposed correction: …">2015</mark>`. | Silently correct. Delete the claim. Leave a wrong claim unmarked because you were unsure — say you were unsure. |

Default when `intent` is missing or unknown: treat as `instruct`, and note it
under *Assumptions*.

A `question` whose comment also contains an instruction ("is this right for
engineers? if not, fix it") is still a question **plus** an instruct: answer in
Replies **and** apply the edit, and say in Applied that you read it both ways.

---

## 6. Draw-mode semantics

Drawn marks are spatial intent. Read them against `annotated.png` when it exists.

- **`rect` (box)** = **scope**. The instruction (its `note`, or the nearest note
  with `attached_to` pointing at it) applies to everything inside those bounds.
- **`arrow`** = **movement**, `points[0]` (tail) → `points[1]` (head): move the
  thing at the tail to the position at the head. Preserve the moved block's
  markup exactly; move it, do not retype it.
  **How much is "the thing" (scope rule — never guess this).** A tail marks a
  *position*, not a boundary, so resolve its extent by this ladder and stop at
  the first rule that applies:
  1. The note names the unit ("move this section", "move this paragraph", "move
     the table") — that unit wins.
  2. The tail sits on or within a heading → the section: that heading plus every
     following sibling (paragraphs, lists, tables, figures) up to but excluding
     the next heading of equal or higher level.
  3. The tail sits inside a `rect` → that box's bounds decide, per the scope rule
     above.
  4. Otherwise → exactly the one block element the tail lands in (that single
     paragraph, list, table or figure), never its neighbours.
  The head resolves the same way: insert immediately before the block element
  whose top edge is nearest the head, and say which one in *Applied*.
  Always name the exact elements you moved in *Applied* (e.g. *"moved the
  `Measuring in the wild` heading + its 2 following paragraphs above the
  `The three numbers…` heading"*) so a wrong scope is one round from being
  fixed. If the ladder is genuinely undecidable — a tail between two blocks with
  a note that names no unit — take the smaller reading and record the
  alternative in *Assumptions*.
- **`pen`** = emphasis. A closed-ish loop around content means **that content is
  the subject**; a line under or beside content points at it.
- **`note`** = words. **A note always beats your geometric guess.** If a note
  says "make this a pull quote" and the arrow looked like a move, it is a pull
  quote. `attached_to` names the shape it explains; that shape's `note` field
  mirrors the same text.
- A mark with no note and no obvious reading is not a licence to invent one:
  do the conservative thing (usually nothing) and put it under *Assumptions*.

Map every drawn mark onto real elements by comparing `bounds` to the layout at
`viewport.width`. If you cannot tell which element a mark covers, say so rather
than editing the most likely candidate.

---

## 7. Ambiguity

Conservative reading + one line in *Assumptions*. Always. If two readings differ
in blast radius, take the smaller one and name the alternative — the author gets
another round, and a wrong big edit costs them more than a right small one.

Never ask the user a clarifying question mid-round *instead of* delivering:
deliver the conservative result and put the question in *Assumptions* or
*Replies*. The round is the conversation.

---

## 8. Verification (do it, then report it)

Before writing changes.md:

1. **Diff discipline.** Compare your edited document against the one you were
   given and confirm every hunk traces to an annotation id. If a hunk does not,
   revert it.
2. **Both widths.** Inspect the result at the block's `viewport.width` (the
   width the author annotated at) **and at 390px**. Look for: overflow, broken tables,
   text collisions, anything your edit pushed below the fold on mobile. Use a
   headless browser if one is available; otherwise reason explicitly from the
   CSS and say that is what you did.
3. **Marked corrections survive.** Every `<mark data-redline="proposed">` is
   present and readable.
4. **Nothing else moved.** Unannotated paragraphs are byte-identical.

Report all four in *Verification*, including anything you could not check.

---

## 9. Output contract

Write two things, beside the target you edited:

- **the edited document**, with its `redline-state` block updated so that
  `responses` carries one entry per annotation you acted on:

      "responses": {
        "h2": { "kind": "replied", "text": "<your actual answer>" },
        "h5": { "kind": "applied", "summary": "<what you changed, one line>" }
      }

  `kind` is `replied` (a question — no edit was made) or `applied` (you changed
  the document). An annotation you could not resolve gets no entry at all.
  Preserve every other field of the block untouched, and bump `round`.

- **`changes.md`** — the same five sections, in the same order, as before:
  Applied · Assumptions · Replies · Fact-check findings · Verification.

**The block is canonical, and `changes.md` may not contain a claim absent from
it.** If the two disagree the UI and the prose start telling the reader
different stories. Write the block first, then render the prose from it.

Also state which rung of §1a you edited on, and which resolution step of §4
each anchor landed on — a weak match must be visible, never silent.

`changes.md`, in full — every section every time, even when empty (write
"None." — an empty section is information):

```markdown
# Round <N> — <short title>

`<document_key>` · <M> annotations

## Applied
- **[h1 · instruct]** "<anchor quote, trimmed>" → what you changed, in one line.
- **[d2 · arrow]** Moved <the exact elements: e.g. "the `<h2>Measuring in the
  wild</h2>` heading + its 2 following `<p>`s"> above <block>. Markup preserved.

## Assumptions
- **[h4]** Read "this" as the sentence, not the section — the smaller scope.
- **[d5 · pen]** Could not tell what this circle covers; left untouched.

## Replies
- **[h3 · question]** "<anchor quote>"
  <A real answer. Two to six sentences. Recommend something. No edit was made.>

## Fact-check findings
- **[h2 · fact-check]** Claim: "<quoted claim>".
  Verdict: incorrect / correct / unverifiable. Evidence: <source or reasoning>.
  Action: proposed correction "<old>" → "<new>", wrapped in
  `<mark data-redline="proposed">` at <location>. Awaiting your call.

## Verification
- Diff: N hunks, all traced to annotations; unannotated content byte-identical.
- Widths: checked at 860px and 390px — <findings>.
- Proposed corrections: <k> `<mark>` elements present.
- Not checked: <anything you could not verify, and why>.
```

Close your reply to the user with the loop, not a victory lap:

> Round N is written to `<path>`. Open it back in redline for round N+1 — the
> proposed correction in section 2 is waiting on you.

---

## 10. Hard rules

1. Never modify the bundle you were handed — it is the author's turn. Write to
   the target you chose in §1a, and to nothing else.
2. Never edit content that no annotation points at.
3. Never answer a `question` with an edit.
4. Never let a `fact-check` correction into the document unwrapped.
5. Never guess at an anchor you could not locate — report it.
6. Never reformat, re-indent, or re-minify the HTML.
7. Always write both the edited document — `redline-state` block updated — and
   `changes.md`, with all five sections.
8. If you did less than the author asked, say so plainly in changes.md. Silent
   incompleteness is the one failure mode that breaks the loop.
