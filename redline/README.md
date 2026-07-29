# redline

**Annotate a page. Export the packet. The agent takes its turn. Open the result
and go again.**

redline is a local web app for marking up a copy of any page — a URL, a local
file, or *the agent's own previous output* — the way you'd mark up a printout:
draw on it, highlight sentences, attach comments. The export is a structured
packet, and a Claude Code skill picks it up as one round of an ongoing
co-authoring loop.

It is not annotate-once-and-done. Round N's `result.html` is round N+1's
canvas.

```
   you annotate          →   packet   →   agent applies, replies, verifies
        ↑                                              ↓
        └──────────  open result.html  ←────────  result.html + changes.md
```

## Build and run

```bash
go build ./cmd/redline
./redline serve                 # http://127.0.0.1:8787
./redline serve -port 8790 -inbox ./inbox
```

One binary, no install step: the frontend and the sample article are embedded
with `go:embed`. Dependencies: the Go standard library plus
`golang.org/x/net/html`. The browser loads exactly one external script,
html2canvas from cdnjs — and if that is unreachable the export still works,
minus `annotated.png` (the packet says so in `manifest.json`).

## Using it

1. **Open** something: the bundled sample draft, a URL, a local file, or a
   previous round's `result.html` from the inbox browser.
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
4. **Export round** (`⌘E`) → `inbox/<session>/round-<N>/`:

   ```
   manifest.json      session, round, parent round, source, viewport, counts
   snapshot.html      exactly what you annotated (may be last round's result)
   annotated.png      the flattened visual (best effort)
   annotations.json   draw vectors + highlight anchors
   ```

   Highlight anchors are deliberately over-specified — CSS selector path,
   character offsets, the exact quoted text, and ~32 characters of
   prefix/suffix context — so they still resolve after the document has been
   edited.

5. Hand it to Claude Code (below), then **open `result.html`** from the inbox
   browser: same session, next round.

Shortcuts: `1`/`2` modes · `p r a n` pen/box/arrow/note · `[` `]` stroke ·
`⌘Z` undo · `⌘E` export · `?` help. The width selector re-flows the page and
re-resolves every anchor, which is a decent live proof that the anchors hold.

## Demo prompts

**1 — the round (Opus, xhigh):**

> Process the newest round in ./inbox using the redline skill. Apply the
> annotations, reply to questions, verify at both widths, and give me the
> change summary.

**2 — the survey (watch it fan out to the `explorer` agent):**

> Before changing anything, survey snapshot.html's structure and voice —
> layout, styling system, and the author's tone — conclusions only.

**3 — the batch (a dynamic workflow):**

> Create a workflow to process every pending round in ./inbox: one agent per
> session following the redline skill end-to-end, then a final agent that
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
internal/server/server.go    handlers (plain http.Handler) + inbox browsing
internal/snapshot/inline.go  fetch/read + inline CSS and images, strip scripts
internal/packet/store.go     PacketStore interface + LocalDir (sessions/rounds)
web/app.html                 the whole frontend: viewer, draw, highlight, export
sample/article.html          the bundled demo target: a real-feeling draft
inbox/                       <session>/round-N/ packets and results
.claude/skills/redline/      SKILL.md — how a round gets processed
.claude/agents/explorer.md   model: haiku, read-only, conclusions not dumps
```

Designed-in seam, not built here: handlers are plain `http.Handler`s and every
write goes through the `packet.Store` interface, so mounting them in an
existing Go service with an S3-backed store is a wiring change, not a rewrite.

## Honest boundaries

- Inlining is best effort. Static and server-rendered pages come across well;
  SPAs degrade to whatever the server sent. Scripts are always stripped — a
  snapshot must not move under you while you annotate it.
- Anchoring works against the snapshot's DOM, which is the point: articles,
  landing pages, docs, slides-as-HTML.
- `annotated.png` needs html2canvas from the CDN. Offline, the packet is still
  complete and `manifest.json` records the omission.
