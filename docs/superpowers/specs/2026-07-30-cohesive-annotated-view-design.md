# Cohesive annotated view — design

**Date:** 2026-07-30
**Status:** approved design, not yet planned
**Affects:** `redline/web/app.html`, `redline/internal/*`, `redline/.claude/skills/redline/SKILL.md`

---

## 1. Problem

redline today splits a document's review into per-round directories, and you view one
round at a time. Annotations belong to exactly one round and die there. To see what was
said about a paragraph two rounds ago you open a different directory and read a different
file. The tool's own loop — round N's `result.html` is round N+1's canvas — is real, but
the *reading* experience does not follow it.

Two consequences drove this redesign:

- **Reviewing is fragmented.** There is no view of "everything currently being said about
  this document." There is only a stack of rounds.
- **Integration is server-shaped.** Packets land in `inbox/` on the machine running the Go
  binary, which makes the backend the mechanism by which Claude Code participates. That
  couples annotation state to one filesystem and one user.

## 2. Goals

1. One cohesive view: the current document, with every still-valid annotation from every
   round shown in place, and the agent's replies attached to them.
2. Annotations that no longer resolve retire to a reviewable history that records **what
   they referenced** and **what was said**.
3. Annotation state lives client-side, so separate people annotating the same page never
   collide, and the tool is portable.
4. The backend becomes a helper (fetch, inline, cache), not the integration seam.
5. Claude Code integrates by reading a downloaded artifact, not by watching a directory.

## 3. Non-goals

- Real-time collaboration or shared cursors. Isolation is achieved by *not sharing state*,
  not by merging concurrent edits.
- Server-side accounts, auth, or persistence of annotations.
- Preserving draw marks as live annotations across document changes (see §6.3).
- A general-purpose diffing or version-control layer. Staleness is derived per-open, not
  tracked as history of the document.

## 4. Architecture

| Concern | Owner |
|---|---|
| Annotation state, replies, history | Browser (IndexedDB) |
| Anchor resolution and staleness derivation | Browser |
| Fetch page, inline CSS/images, strip scripts, cache | Go helper |
| Applying annotations, writing replies, publishing | redline plugin (Claude Code) |

**Removed:** `internal/packet` (`Store`, `LocalDir`, `Sessions`, `NextRound`, `Locate`),
the `inbox/` directory convention, and `POST /export/{id}` as an integration seam.

This knowingly retires two seams `redline/CLAUDE.md` documents as deliberately paid for:
the `packet.Store` interface (intended to allow an S3-backed store) and export-through-the-
server. Both existed to let the tool be mounted inside a larger Go service. Moving
integration to the client makes them dead weight. Accepted explicitly, not by accident.

**Retained:** `POST /snapshot`, `GET /snapshot/{id}`, `GET /healthz`, and the snapshot
pipeline in `internal/snapshot` — fetching a cross-origin page and inlining its assets is
the one thing a browser client cannot do for itself, and script-stripping remains a
correctness requirement (a snapshot must not move under the author while they annotate it).

**Staleness is never stored.** It is recomputed against the live document on every open,
because "stale" is a relationship between an anchor and current text, not a property of an
annotation.

## 5. Document identity and storage

Annotations are keyed by **normalized URL or file path**:

```
https://blog.dev/latency-budget     ← fragment, trailing slash, tracking params stripped
file:///Users/d/src/article.html
sample:article.html
```

A content hash is explicitly rejected as a key: it changes at exactly the moment the page
is edited, which would orphan every annotation precisely when continuity matters most.

Storage is **IndexedDB**. Annotations are small — anchors plus text, no snapshots retained
— but localStorage's synchronous API and ~5MB cap are the wrong shape. No eviction policy
until there is evidence one is needed.

**Known limitation:** a page that moves to a new URL strands its annotations. Mitigated in
practice because the embedded state block (§7) declares its own `document_key`, so opening
an agent-produced file re-attaches it to the right set regardless of path.

## 6. State model

State is derived from two independently known facts: whether the anchor resolves against
the current document, and whether the agent responded.

| anchor | agent response | state | lives |
|---|---|---|---|
| resolves | none | **open** | gutter |
| resolves | replied | **answered** (thread, reply visible) | gutter |
| resolves | applied | **applied, unverified** | gutter |
| broken | applied | **done** | history |
| broken | replied | **answered, text moved on** | history |
| broken | none | **stale** | history |
| any | user dismissed | **dismissed** | history |

### 6.1 Why two axes rather than a lifecycle

Applying an `instruct` breaks that annotation's own anchor. Without the agent's response,
a successfully applied edit is indistinguishable from an annotation orphaned by unrelated
drift. The response record is what separates **done** from **stale**.

### 6.2 History record

Each history entry satisfies the two stated requirements literally — what it referenced,
and what was said:

```
▌ stale · round 1 · 2026-07-30
  referenced:  "This is a draft of an argument I keep making in review meetings…"
               (#post > p:nth-of-type(2))
  you said:    "Why do we have to state this? Does it make the argument better?"
  agent said:  "Cut sentence one, keep the thesis." — replied, no edit
```

### 6.3 Draw marks

Draw marks are page-pixel coordinates against one rendered layout; there is no anchor
ladder for them and they cannot survive a reflow. They are therefore **always round-scoped
and retire at round end**, never appearing as live annotations.

They are *not* dropped. At draw time the DOM is available, so redline captures the text of
the elements the mark's bounds overlap and stores it as `referenced_text`. History then
reads:

```
▭ rect · round 1 · retired at round end
  referenced:  "A performance budget is not a diet. It is a currency…"
  you said:    "where did this quote come from ?"
```

This is better than a cropped thumbnail: no image storage, and no dependency on
html2canvas, which the demo runbook already flags as the tool's one network-fragile
component.

## 7. The bundle

The transport artifact **is the page**, not a sidecar JSON. Downloading produces
`redline-<key>-r<N>.html`: the document with a single embedded block.

```html
<script type="application/redline+json" id="redline-state">
{
  "schema_version": "2.0",
  "document_key": "file:///Users/d/src/article.html",
  "round": 2,
  "annotations": [ /* anchors + comments, browser → plugin */ ],
  "responses": {                              /* plugin → browser */
    "h2": { "kind": "replied", "text": "cut sentence one, keep the thesis" },
    "h5": { "kind": "applied", "summary": "tightened to two sentences" }
  }
}
</script>
```

One artifact type, one schema, symmetric in both directions. Inert in any browser; the file
opens and renders normally. The block declares `document_key`, so an agent-produced file
under a different path still re-attaches to the correct annotation set.

The download is produced **entirely client-side** (a `Blob` and an object URL) with no
server round-trip — `POST /export` is gone, and the helper is not involved. The document
HTML is serialized from the live view at download time; no snapshot is retained in
IndexedDB between sessions.

IndexedDB remains the source of truth. The block is a transport copy and is merged back
**by annotation id**, so opening a stale bundle cannot clobber newer local state.

## 8. Plugin contract

### 8.1 Target selection ladder

1. An explicit path the user names → that file.
2. The source the agent **already has in session context** — it published or generated the
   page during this conversation → that file.
3. Neither → the bundled document itself.

Rung 2 requires knowledge the agent already holds. It must **not** search the filesystem
for a plausible source, infer one from the page title, or match on content similarity.
Editing the wrong file is the worst failure this tool can produce, and it is worse than
falling through to rung 3. When in doubt, drop to rung 3 and say so.

On rungs 1 and 2 the anchors were captured against *rendered* output, so CSS selectors and
`textContent` offsets may not apply to the source (templating, Markdown, a build step).
Resolution degrades to quoted-text search. **The plugin must report which rung it used and
which resolution step succeeded**, so a weak match is visible rather than silent.

### 8.2 `SKILL.md` changes

- **§1 trigger** becomes "the bundle you were pointed at" instead of "the newest round in
  `./inbox`". This deletes the multi-pending-round tie-break rules entirely — sessions no
  longer share a directory, so there is nothing to disambiguate.
- **Add** the target ladder (§8.1) and the requirement to emit the state block.
- **Add** the reporting requirement for which resolution rung was used.
- **Unchanged:** §0 prime directive (unannotated content is sacred), §4 anchor ladder, §5
  intent semantics, §6 draw semantics, §7 ambiguity handling.

### 8.3 `changes.md`

Survives as the human-readable artifact. Rule: **the embedded block is canonical, and
`changes.md` may not contain a claim absent from the block.** Without that rule the prose
and the UI drift and begin disagreeing.

## 9. View

```
┌─────────────────────────────────┬──────────────────────┐
│  the current document           │  ▌2 open             │
│                                 │   "check this date"  │
│  …HTTP/2, standardised in 2012  │                      │
│                                 │  ▌1 answered      ⌄  │
│  …whoever merged last.          │   you: why state…    │
│                                 │   agent: cut sen-    │
│                                 │   tence one…         │
│                                 │        [dismiss]     │
│                                 ├──────────────────────┤
│                                 │  HISTORY (4)      ▸  │
└─────────────────────────────────┴──────────────────────┘
```

- **Gutter** — live annotations only, document order, state as a chip. Answered entries
  collapse to one line and expand on click.
- **History drawer** — collapsed by default, grouped by retirement reason.
- **Document switcher** — replaces the inbox browser: recent documents by key with
  open-annotation counts.

Rounds are not a navigation concept. They survive only as a counter in the state block for
lineage. **Download is the round boundary**: it increments the round and retires draw marks.

## 10. Publish

Publish is the inverse of the round loop: it strips redline metadata out of the document
and deposits the accumulated conversation as durable provenance.

1. **Browser** emits `redline-publish-<key>.html` — the document plus a block containing
   the complete history, live and retired.
2. **Plugin** writes the history into the workspace as a human-readable record (default
   `docs/redline/<slug>.md`), and strips the `redline+json` block from the document,
   leaving clean publishable output.
3. **Browser** clears local state for that key. Clearing is gated on **the publish bundle
   download having completed** — that is the only precondition the browser can actually
   observe, since it has no way to know whether the plugin later wrote the workspace
   record. So the guarantee is precise but narrow: annotations are never dropped without a
   file containing the full history having left the browser first. The document remains in
   the switcher marked *published*.

This also resolves the one standing objection to the embedded block: metadata does not ship,
because publish removes it.

## 11. Edge cases

- **Ambiguous anchor.** If the quoted text occurs several times with no clear winner, the
  annotation is marked **needs re-anchoring** rather than attached to a plausible guess.
  This is SKILL.md §4's "do not guess" rule enforced client-side.
- **Bulk staleness.** Returning to a much-changed page shows one summary line
  ("7 annotations no longer resolve") linking into history, rather than a wall of orphans.
- **Unknown `document_key` on import** is adopted as a new document rather than erroring.
  This is the portability path: change machines, or open someone else's bundle. Merge is by
  annotation id, and ids are session-unique, so two people's sets combine without collision.
- **Two people, same URL.** Separate browsers, separate IndexedDB, no shared state, no
  collision — isolation by construction rather than by locking.

## 12. Testing

There are no tests in the repo today, and `redline/CLAUDE.md` forbids introducing a test
framework or task runner to add the first one. The anchor resolver and state derivation are
pure functions and deserve the first coverage.

**Approach:** a self-contained `web/test.html` with hand-rolled assertions, printing
pass/fail in the browser. Zero new dependencies, no runner, constraint honored.

Cases to pin:

- each rung of the resolution ladder (selector+offset, prefix+quote+suffix, nearest quote)
- the ambiguous-quote refusal
- all seven rows of the §6 state table
- `document_key` override beating URL-derived identity
- merge-by-id not clobbering newer local state

## 13. Suggested phase split

This is large for a single implementation plan. §10 (publish) is cleanly separable and
depends on everything before it, so a two-phase split is natural:

- **Phase 1** — §4–§9: client state, document identity, state model, bundle round-trip,
  the cohesive view, and the `SKILL.md` rewrite. Delivers the whole of the original
  complaint on its own.
- **Phase 2** — §10: publish and workspace history archival. Without it, IndexedDB grows
  without bound and metadata ships in published documents; neither blocks Phase 1 from
  being useful.

Each phase gets its own plan.

## 14. Sequencing risk — read before planning

**This design breaks the live demo, and the demo is imminent.**

`talk/DEMO_RUNBOOK.md` is dated 2026-07-29 and its pre-flight items are marked **TONIGHT**.
All three demo prompts in `redline/README.md` reference `./inbox` explicitly, Act 3 fans out
over "every pending round in `./inbox`", and four packets are staged there. Removing
`inbox/` and retargeting the skill's trigger invalidates all of it: the staged packets, the
act script, and the §3.1 fallback.

**Recommendation: do not land any of this until after the talk.** The work is well-bounded
and will keep. If it must proceed sooner, the demo path has to be migrated in the same
change — restaging every packet as a bundle and rewriting all three prompts and the runbook
— which roughly doubles the scope and puts a rehearsed performance on a fresh code path.
