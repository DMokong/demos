# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A local page-annotation tool whose export is **one turn in a human/agent loop**.
A human marks up a page; the bundle they download is their turn. An agent writes
the edited document + `changes.md`; that is its turn. The agent's document is the
next round's canvas. The tool exists to produce that bundle — the loop is the
product.

`.claude/skills/redline/SKILL.md` governs how a round gets processed, and its
prime directive — **unannotated content is sacred** — outranks anything in this
file. Do not paraphrase that skill here; read it.

## Commands

```bash
go build ./cmd/redline && ./redline serve   # UI on http://127.0.0.1:8787
./redline serve -port 8790 -addr 0.0.0.0    # flags: -port -addr
go build ./... && go vet ./... && gofmt -l .  # the full check; all three are clean today
```

`serve` is the only subcommand. **There are no Go tests** — no `*_test.go`, no
Makefile, no test runner. The JS has one: open `web/test.html` in a browser (it
needs `file://` and nothing else) and read the last line, `<N> passed, <M>
failed`. Don't introduce a test framework, a bundler, or a task runner to run it.

## Architecture

Dependencies run one way, `cmd` → `server` → `snapshot`, and the server holds no
durable state at all: it freezes a page and serves the app. Everything about a
round — annotations, responses, history, round number — lives in the browser.

| Unit | Responsibility |
|---|---|
| `cmd/redline/main.go` | Flags, listener, graceful shutdown, asset resolution |
| `internal/server/server.go` | Plain `http.Handler`s: snapshot create/get, static assets |
| `internal/snapshot/inline.go` | Fetch/read a page, inline CSS + images as data URIs, **strip all scripts** |
| `web/redline-core.js` | Pure logic: anchor resolution, state derivation, doc keys, bundle parse/serialize |
| `web/redline-store.js` | IndexedDB, one record per document key |
| `web/app.html` | UI and wiring only — no logic worth testing should land here |
| `web/test.html` | The harness for the two JS modules; plain scripts, no modules |
| `embed.go` | Lives at the module root because `go:embed` requires it |

Routes are Go 1.22 pattern-mux: `GET /{$}`, `GET /view/{id}`, `GET /js/{name}`,
`POST /snapshot`, `GET /snapshot/{id}`, `GET /healthz`. There is no export route
and no inbox: the round is downloaded client-side as one HTML file.

**The seam is the missing storage.** Handlers stay plain `http.Handler`s with no
store behind them, so mounting this in an existing Go service is a wiring change
rather than a rewrite. Adding a write path to the server — an `os.WriteFile`, a
database, a round directory — puts state back on the wrong side of the loop.

**JS files are plain scripts, not ES modules.** They attach to `window.RedlineCore`
and `window.RedlineStore` so `web/test.html` loads them over `file://` with no
server and no CORS.

**Scripts are always stripped from snapshots.** Not a sanitisation detail — a
snapshot must not move under the author while they annotate it.

## Cross-file contracts

These are the invariants you cannot see from any single file:

- **The embedded state block is the cross-file contract.** Schema `"2.0"`, written
  by `web/redline-core.js` (`serializeBundle`/`parseBundle`), produced by
  `web/app.html`'s Export, and consumed by `SKILL.md` §3. Changing its shape means
  changing all three together. The skill reads `schema_version` before anything
  else.
- **The anchor format is a two-sided contract.** `web/app.html` emits
  deliberately over-specified anchors — CSS selector path, `textContent`
  offsets, exact quoted text, ~32 chars of prefix/suffix, plus document-level
  offsets — `redline-core.js` resolves them, and `SKILL.md` §4 consumes them via
  the same four-step ladder so anchors survive an edited document. Never simplify
  one side alone.
- **State is derived, never stored.** `RedlineCore.deriveState` recomputes
  stale/answered/applied/needs-reanchor from (resolution × response) on every
  mount. Persisting a status flag would let the badge and the document disagree.
- **Ambiguity is a state, not a guess.** A quote that matches several places
  resolves to `needs-reanchor` in the app and to "stop and report it" in
  `SKILL.md` §4.4. Keep those identical: a wrong attachment is the worst failure
  this tool can produce.
- **Document identity is the join key.** `RedlineCore.normalizeKey` derives a
  `document_key` from the source; IndexedDB records, the bundle's
  `document_key`, and response merges are all keyed by it.
- **An embedded state block outranks the path.** `afterMount` reads
  `#redline-state` out of the mounted document and prefers its `document_key`
  over `deriveDocKey()`. This is what lets a returned bundle — which lives at a
  different path than the document it came from — re-attach to its annotations
  instead of stranding them. The snapshot pipeline preserves that one script
  (`isRedlineState` in `internal/snapshot/inline.go`) for exactly this reason;
  it is inert data, so keeping it does not weaken the strip-all-scripts
  invariant. `TestRedlineStateBlockSurvivesInlining` pins both halves.
- **KNOWN HOLE: identity forks by how you opened the document.**
  `deriveDocKey` branches on `src.kind` *before* normalizing, so the bundled
  sample opened via the button is `sample:sample/article.html` while the same
  file opened by path is `file:///…/sample/article.html` — two keys, two
  annotation sets. Fixing it means changing the canonical key scheme, which
  would orphan the staged bundles under `bundles/`, so it is deliberately
  deferred rather than patched in a hurry. Do not document this as solved.

## Gotchas

- **The binary is cwd-sensitive.** `assetFS()` and `readSample()` prefer on-disk
  `web/*` and `sample/article.html` when they exist, falling back to the embedded
  copies — so the frontend can be edited without rebuilding, but running from
  another directory silently switches to the embedded versions.
- **`embed.go` lists each web asset by name.** A new `web/*.js` file is invisible
  to a built binary until it is added to the `go:embed` directive *and* loaded by
  `app.html`.
- **`sample/landing.html` is orphaned.** Only `sample/article.html` is in the
  `go:embed` set and served as the bundled demo target; nothing references
  `landing.html`. Either embed it or drop it — don't assume it is reachable.
- **Annotations live in one browser profile.** IndexedDB is the source of truth;
  a different browser or a cleared profile starts empty. The bundle, not the
  store, is what travels.
- **Draw marks are round-scoped.** They are page-pixel coordinates against one
  layout, so they cannot survive a reflow: Export retires them into history
  carrying the text they covered, and they never appear in the bundle.
- `inbox/` holds packets from the retired directory-based workflow and is kept
  for the talk's recorded material. Nothing in the code reads it, and a
  `snapshot.html` in there is still the author's turn — never edit one.
- The compiled `/redline` binary is gitignored; `.mcp.json` is too, because it
  holds literal credentials.

## Conventions

Go stdlib plus `golang.org/x/net/html` only — adding a dependency needs a
reason, and the frontend has none at all: no npm, no bundler, no framework, no
external script. Package comments carry the "why" and are worth reading before
editing. Structural surveys ("how is this laid out", "what is the author's
voice") go to the read-only `explorer` agent, which returns conclusions rather
than file dumps. The demo prompts live in `README.md`.
