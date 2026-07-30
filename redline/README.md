# redline

**Annotate a page. Download the bundle. The agent takes its turn. Drag the
result back and go again.**

redline is a local web app for marking up a copy of any page — a URL, a local
file, or *the agent's own previous output* — the way you'd mark up a printout:
draw on it, highlight sentences, attach comments. Export downloads one
self-contained HTML file — the document plus an inert state block — and a Claude
Code skill picks it up as one round of an ongoing co-authoring loop.

It is not annotate-once-and-done. Round N's edited document is round N+1's
canvas.

```
   you annotate          →   bundle   →   agent applies, replies, verifies
        ↑                                              ↓
        └──────  drag the file back in  ←────  document + changes.md
```

## Build and run

```bash
go build ./cmd/redline
./redline serve                 # http://127.0.0.1:8787
./redline serve -port 8790      # flags: -port -addr
```

One binary, no install step: the frontend and the sample article are embedded
with `go:embed`. Dependencies: the Go standard library plus
`golang.org/x/net/html`, and the browser loads no external scripts at all. The
server never stores a round — annotations live in the browser's IndexedDB, keyed
by document, and the bundle is built client-side.

## Using it

1. **Open** something: the bundled sample draft, a URL, a local file, or a
   document the agent edited last round. Everything you ever annotated is listed
   in the gutter's document drawer and comes back when you reopen it.
2. **Highlight mode** (`1`): select text in the page. It highlights and opens a
   comment card in the margin. Give the comment an *intent*:

   | intent | what the agent does |
   |---|---|
   | `instruct` (default) | applies exactly that change to exactly that range |
   | `question` | **replies** in `changes.md` — never edits |
   | `fact-check` | verifies, then proposes a correction wrapped in `<mark data-redline="proposed">` |
   | `expand` | grows that section in your voice, additions listed |
   | `delete` | removes the range |

3. **Draw mode** (`2`): pen, box, arrow, note over the whole page. Box = scope,
   arrow = move (tail → head), circle = subject, note = words that override any
   guess. A note attaches to the last shape you drew.
4. **Export round** (`⌘E`) → your downloads folder gets
   `redline-<document>-r<N>.html`: the document exactly as annotated, carrying
   one inert block.

   ```html
   <script type="application/redline+json" id="redline-state">
   { "schema_version": "2.0", "document_key": "…", "round": 1,
     "viewport": {…}, "annotations": [ … ], "responses": { … } }
   </script>
   ```

   Highlight anchors are deliberately over-specified — CSS selector path,
   character offsets, the exact quoted text, and ~32 characters of
   prefix/suffix context — so they still resolve after the document has been
   edited. Draw marks are page-pixel and cannot survive a reflow, so they retire
   at export into the history drawer, carrying the text they covered.

5. Hand the file to Claude Code (below), then **drag what it wrote back onto
   redline**: replies land in their cards, applied edits break their anchors and
   fall into history, and the next round starts on the edited document.

Shortcuts: `1`/`2` modes · `p r a n` pen/box/arrow/note · `[` `]` stroke ·
`⌘Z` undo · `⌘E` export · `?` help. The width selector re-flows the page and
re-resolves every anchor, which is a decent live proof that the anchors hold.

## Demo prompts

Run these from `redline/`, where `./bundles` holds three ready-to-process
bundles: `redline-article-r1.html`, `redline-article-r2.html` (round 1's output,
annotated again), and `redline-landing-r1.html`.

**1 — the round (Opus, xhigh):**

> Process the redline bundle at ./bundles/redline-article-r1.html — apply the
> annotations, reply to questions, verify at both widths, and give me the change
> summary.

**2 — the survey (watch it fan out to the `explorer` agent):**

> Before changing anything, survey the document inside
> ./bundles/redline-article-r1.html — layout, styling system, and the author's
> tone — conclusions only.

**3 — the batch (a dynamic workflow):**

> Create a workflow to process every redline bundle in ./bundles: one agent per
> file following the redline skill end-to-end, then a final agent that
> cross-checks the results for consistency and compiles a single summary.

## What to look for when the agent comes back

- The `question` got a **reply**, not an edit.
- The `fact-check` correction is wrapped in `<mark data-redline="proposed">` —
  visible, and still yours to accept.
- Unannotated paragraphs are **byte-identical**. That is the prime directive:
  the author's words are the product.
- `changes.md` has all five sections: Applied · Assumptions · Replies ·
  Fact-check findings · Verification.

## Layout

```
cmd/redline/main.go          the CLI
internal/server/server.go    handlers (plain http.Handler): snapshot + assets
internal/snapshot/inline.go  fetch/read + inline CSS and images, strip scripts
web/app.html                 the whole frontend: viewer, draw, highlight, bundle
web/redline-core.js          pure logic: anchors, state, doc keys, bundle I/O
web/redline-store.js         IndexedDB — one record per document
web/test.html                the harness: open it in a browser, no runner needed
sample/article.html          the bundled demo target: a real-feeling draft
bundles/                     staged v2.0 bundles the demo prompts point at
inbox/                       audit trail: the packets those bundles came from
.claude/skills/redline/      SKILL.md — how a round gets processed
.claude/agents/explorer.md   model: haiku, read-only, conclusions not dumps
```

`inbox/` is history, not input. It holds the packets from the retired
directory-based workflow, kept because the talk's recorded material references
them; nothing in the code reads it, and a `snapshot.html` in there is still the
author's turn — never edit one.

Designed-in seam, not built here: the handlers are plain `http.Handler`s and the
server holds no state of its own — it freezes a page and serves the app, nothing
else — so mounting them in an existing Go service is a wiring change, not a
rewrite. The round lives entirely in the browser and in the bundle.

## Honest boundaries

- Inlining is best effort. Static and server-rendered pages come across well;
  SPAs degrade to whatever the server sent. Scripts are always stripped — a
  snapshot must not move under you while you annotate it.
- Anchoring works against the snapshot's DOM, which is the point: articles,
  landing pages, docs, slides-as-HTML.
- Annotations live in one browser profile's IndexedDB. Another browser, another
  machine, or a cleared profile starts empty — the bundle, not the store, is
  what travels.
