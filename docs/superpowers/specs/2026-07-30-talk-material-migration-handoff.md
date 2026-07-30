# Handoff — talk material affected by the redline redesign

**Date:** 2026-07-30
**For:** the session that owns the talk outline and deck
**Talk date:** 2026-07-31 (tomorrow)
**Status:** redline changes executing now; this document is the impact analysis, written against
the approved design rather than against finished code. A changelog of what actually landed is
appended as `## Changelog` when execution completes.

---

## 1. What changed in redline, in one paragraph

Annotations no longer live in per-round directories under `redline/inbox/`. They live in the
browser (IndexedDB), keyed by normalized document URL/path, and persist across rounds until their
anchor stops resolving. Claude Code no longer watches a directory: the human downloads a
**bundle** — the page itself with a `<script type="application/redline+json" id="redline-state">`
block carrying annotations — and the redline plugin reads it, edits a target, and writes replies
back into that same block. The Go server drops to a helper (fetch, inline, cache).

Full rationale: `docs/superpowers/specs/2026-07-30-cohesive-annotated-view-design.md`
Task-level detail: `docs/superpowers/plans/2026-07-30-cohesive-annotated-view-phase-1.md`

**Read the spec's §14 before changing anything.** It records that this was landed the day before
the talk against a documented recommendation to wait.

## 2. Every place the talk material is now wrong

Grepped across all 18 slides. The blast radius is one slide — the demo hub,
*"The dial, then a redline round"* — plus the runbook.

### 2.1 `talk/slides.html:962` — the Act 3 card

> "Three staged packets: **one agent per packet**, then a judge that cross-checks every result
> for consistency. Open the generated script."

Two independent problems:

- **"Three" was already wrong before this redesign.** A fourth session (`s-20260730-313937`) was
  exported at 2026-07-30T03:57Z. There are four pending rounds, five rounds total across four
  sessions.
- **"packet" is no longer the unit.** It is a bundle file. The fan-out is one agent per bundle.

### 2.2 `talk/slides.html:980` — speaker notes, Act 1

> "Export → lands in `./inbox`."

False. Export produces a browser download (`Blob` + object URL, no server round-trip). Nothing is
written to `inbox/`.

> *"Process the newest round in ./inbox using the redline skill. Apply the annotations, reply to
> questions, verify at both widths, and give me the change summary."*

This is the verbatim prompt printed on the slide. It no longer matches the skill's trigger. The
replacement form points at a bundle path. Take the exact wording from `redline/README.md` after
execution — do not retype it from here, or the slide and the README will drift again.

### 2.3 `talk/slides.html:981` — speaker notes, the closing beat

> "Then reopen `result.html` in redline: *'and now it's my turn again.'*"

The story survives; the gesture changes. There is no inbox browser to open a file from. You drag
the returned bundle onto redline and its `responses` merge into the annotations already on screen.

### 2.4 `talk/DEMO_RUNBOOK.md`

Being rewritten as part of execution (§2.4 session map, counts, and the bundle paths). **Verify it
rather than trusting it** — it was edited by an agent, not a human, and its §1 evidence claims were
originally established by a careful verification pass that has not been re-run against the new code.

### 2.5 The outline — not in this repo

`talk/` contains only the runbook, `slides.html`, `slides.pdf`, the cost/capability map, and
`effort_router_demo.py`. The runbook cites "the talk outline (§7)" and "the outline's own
fallback," so an outline exists somewhere outside this repository. It needs the same three
corrections as §2.1–§2.3. Locating it is the first job for whoever picks this up.

## 3. What got better — worth rewriting the narrative around, not just patching

These are genuine improvements to the demo's argument, not damage to work around.

**The Act 1 payoff moves onto the page.** Today, showing "the question got a reply, not an edit"
means opening `changes.md` and scrolling to *Replies*. Now the reply renders in the gutter,
attached to the annotation that asked it. You point at the document instead of at a markdown file.
That is a materially stronger version of *"it's a collaborator, not a compiler."*

**The loop becomes visible instead of narrated.** Round 1's answered annotation stays on screen
while round 2's are added. The cohesive view demonstrates the claim rather than requiring the
audience to hold two rounds in their head.

**History distinguishes "done" from "stale."** An annotation whose anchor broke *because the agent
did the work* is labelled **done**; one orphaned by unrelated drift is **stale**. That is a
concrete on-screen artifact of the agent having acted, available as a new pointing-place.

**The deterministic guard is unchanged and slightly stronger.** *"A pixel diff can tell you the
page changed; it can't read an arrow"* still holds, and draw marks now capture the text they
overlapped, so their meaning survives into history without an image.

## 4. Known gaps, carried forward honestly

- **`annotated.png` is dropped.** The new bundle does not call html2canvas, and neither the spec
  nor the plan addressed this — it is an oversight, not a decision. Consequences: `SKILL.md` §2's
  reading order tells the agent to look at that image to interpret drawn marks, and runbook §2.4
  calls Session A round-2's PNG *"the best single visual of the loop in the whole repo — consider
  putting it on a slide."* Existing PNGs survive under `inbox/`; new rounds produce none.
- **`slides.pdf` must be re-exported** after any `slides.html` edit. It is the wifi/projector
  fallback, and a stale fallback is worse than none because the mismatch surfaces only when relied on.
- **The demo will run on a code path nobody has rehearsed.** Runbook §1.5 budgets 25 minutes to run
  all three acts once. That has not happened against the new code.
- **Original packets are preserved.** `redline/inbox/` is left in place as an audit trail even
  though nothing reads it any more.

## 5. Suggested order for the picking-up session

1. Find the outline; apply §2.1–§2.3 to it.
2. Fix `slides.html:962` (count + noun) and `:980-981` (the three speaker-note lines), taking the
   prompt wording verbatim from `redline/README.md`.
3. Re-export `slides.pdf`.
4. Decide the `annotated.png` question: restore image capture, or amend `SKILL.md` §2 to stop
   promising an image that will not exist.
5. Run all three acts end to end against the new code before the talk.

---

## Changelog

Executed 2026-07-30 as a 12-agent workflow (all 12 completed, 0 errors), followed by a fix pass.
Net: **22 files changed, +2883 / −1159** across `redline/`.

### Commits, oldest first

| SHA | What |
|---|---|
| `43ba3d0` | Extract anchor resolution into `web/redline-core.js`; add `web/test.html` harness; add `GET /js/{name}` asset route |
| `0a1bd20` | `resolve()` refuses ambiguous anchors instead of guessing; reports which ladder rung landed |
| `8710494` | `normalizeKey()` + `deriveState()` — the seven-row state table plus `needs-reanchor` |
| `b523c7c` | `parseBundle()` / `serializeBundle()` for the embedded state block |
| `69f68dd` | `web/redline-store.js` — IndexedDB persistence, one record per document key |
| `5c64cb7` | Restore annotations on mount; derive state against current text |
| `c00b63e` | Gutter state chips, inline agent replies, dismiss, history drawer, document switcher |
| `4f4d170` | Client-side bundle download and import; `POST /export` removed |
| `5a29d90` | Draw marks capture `referenced_text`; retire at round end |
| `9befafd` | Retarget `SKILL.md` to bundles; delete `internal/packet`; remove `GET /inbox` |
| `b9e6225` | Restage demo packets as v2.0 bundles; update `README.md` prompts and `DEMO_RUNBOOK.md` |
| `cf5b4e8` | Fix two defects fatal to the bundle round trip, plus two flagged during execution |

### Structural changes

**Added:** `web/redline-core.js`, `web/redline-store.js`, `web/test.html`,
`internal/snapshot/inline_test.go` (the repo's first Go test), `bundles/` (three v2.0 bundles).
**Deleted:** `internal/packet/store.go`, `POST /export/{id}`, `GET /inbox`, the `-inbox` flag.
**Preserved:** `redline/inbox/` remains on disk as an audit trail; nothing reads it.

### Defects found by the independent verification pass

The verifier returned **`all_clear: false`**. This is recorded rather than smoothed over,
because two findings were holes in the spec and plan themselves, not agent error.

1. **FIXED — the state block was destroyed by its own snapshot pipeline.** Spec §4 retained
   "scripts are always stripped from snapshots" while §7 put the state in a `<script>` block.
   Those two requirements cancel: reopening a bundle silently lost every annotation. Fixed in
   `cf5b4e8` by preserving that single block — a script with an unknown type is inert data and
   is never executed, so keeping it cannot let a snapshot move under the author, which is the
   whole reason scripts are stripped. `TestRedlineStateBlockSurvivesInlining` and
   `TestExecutableScriptsAreStillStripped` pin both halves.
2. **FIXED — `deriveDocKey` ignored the block's `document_key`.** Spec §7 promised an
   agent-produced file "under a different path still re-attaches to the correct annotation set."
   The plan never wired it, so the promise was false. `afterMount` now prefers the embedded key
   and merges the bundle's annotations and responses **by id**, so a stale bundle cannot clobber
   newer local state.
3. **FIXED — `serializeBundle` corrupted blocks containing `$` patterns.** Flagged by the Task 4
   agent, unreachable by its own tests but live once user comments enter the block: `$&`, `$'`,
   `` $` `` and `$1` in a replacement *string* are interpreted as patterns. Now a replacer function.
4. **FIXED — internal bookkeeping leaked into the bundle.** `_seq`, `_rects`, `_cardTop`,
   `_cardH` were exported alongside the documented fields of a contract `SKILL.md` consumes.
5. **OPEN — identity forks by how a document was opened.** The bundled sample via the button is
   `sample:sample/article.html`; the same file via the Open box is `file:///…/sample/article.html`
   — two keys, two annotation sets. Deferred deliberately: fixing it changes the canonical key
   scheme and would orphan the three staged bundles. `redline/CLAUDE.md` now documents it as a
   known hole rather than claiming the guarantee it previously claimed.

### Verified after the fix pass

- JS harness **35 passed, 0 failed** in real Chromium
- `go build ./... && go vet ./... && gofmt -l .` clean; `go test ./internal/snapshot/` passes
- A staged bundle reopens with `docKey` taken from the embedded block, 3 annotations restored,
  states re-derived, dismissed entry correctly in history
- Snapshotting a bundle retains exactly one script — the state block — with executables stripped

### Deviations worth knowing

- **Three bundles, not four.** Only three of the five staged rounds were pending; two already
  had `result.html`.
- **The previously-untracked fourth session was committed.** The runbook now cites it as Session D,
  so leaving it untracked would have made the runbook reference a file missing from a fresh clone.
- **The Playwright MCP blocks `file://`**, so every agent ran the harness over a throwaway
  `python3 -m http.server`. No dependency was added; the plan's `file://` instruction is wrong.
- **The "do not land before the talk" constraint was overridden** by explicit instruction. That is
  why Task 11 exists.
