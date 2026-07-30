# Cohesive Annotated View — Phase 1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

> **STATUS (2026-07-30): all 10 tasks executed and committed**, plus an unplanned Task 11
> (restage demo packets as bundles). An independent verification pass returned
> `all_clear: false` and found three defects; two were fatal to the bundle round trip and
> have since been fixed. Checkboxes below are left unticked deliberately — this header and
> the changelog in
> `docs/superpowers/specs/2026-07-30-talk-material-migration-handoff.md` are the record of
> what was actually done and verified, not the boxes.

**Goal:** Replace redline's per-round viewing with one view of the current document carrying every still-resolving annotation, with state persisted in the browser and Claude Code integrating via a downloaded bundle instead of `./inbox`.

**Architecture:** Pure logic (anchor resolution, state derivation, key normalization, bundle I/O) is extracted from `web/app.html` into `web/redline-core.js` so it can be tested without a framework. Persistence moves to IndexedDB behind `web/redline-store.js`. `web/app.html` keeps only UI and wiring. The Go server drops to a helper: it keeps `/snapshot` and gains a static-asset route, and loses `/inbox`, `/export`, and `internal/packet`.

**Tech Stack:** Vanilla JS (no framework, no build step, no bundler), IndexedDB, Go 1.24 stdlib + `golang.org/x/net/html`.

## Global Constraints

- **No new dependencies.** Go stdlib plus `golang.org/x/net/html` only. No npm, no bundler, no test framework, no task runner. (`redline/CLAUDE.md`)
- **Build with `GOTOOLCHAIN=local`** if the local Go is 1.24.x. `go.mod` pins `go 1.24.0` and `golang.org/x/net v0.48.0` deliberately.
- **Full check is** `go build ./... && go vet ./... && gofmt -l .` — all three must stay clean.
- **JS files are plain scripts, not ES modules.** They attach to a global (`window.RedlineCore`, `window.RedlineStore`) so `web/test.html` can load them over `file://` with no server and no CORS.
- **Scripts are always stripped from snapshots.** Non-negotiable: a snapshot must not move under the author while they annotate it.
- **Schema version for the embedded block is `"2.0"`.** The existing packet schema was `"1.0"`.
- **Do not land this before the talk.** See spec §14. All three demo prompts reference `./inbox`.

---

### Task 1: Extract pure logic into a testable core module

**Files:**
- Create: `redline/web/redline-core.js`
- Create: `redline/web/test.html`
- Modify: `redline/web/app.html:505-518` (`buildTextIndex`), `:636-665` (`nearestIndexOf`, `resolveAnchor`), `:606-635` (`rangeInElement`, `rangeFromGlobal`)
- Modify: `redline/embed.go:13`
- Modify: `redline/cmd/redline/main.go:119-126`
- Modify: `redline/internal/server/server.go:82-88`

**Interfaces:**
- Consumes: nothing (first task)
- Produces: `window.RedlineCore.resolveAnchor(doc, textIndex, anchor) → Range|null`, `window.RedlineCore.buildTextIndex(doc) → {text, nodes}`, `window.RedlineCore.nearestIndexOf(hay, needle, near) → number`. Route `GET /js/{name}` serving `web/{name}`.

This task is a pure refactor: identical behavior, now callable without globals. The existing `resolveAnchor` reads `S.doc` and `S.textIndex` directly, which is why it cannot be tested today.

- [ ] **Step 1: Write the failing test harness**

Create `redline/web/test.html`:

```html
<!doctype html>
<meta charset="utf-8">
<title>redline core tests</title>
<script src="./redline-core.js"></script>
<pre id="out"></pre>
<script>
let pass = 0, fail = 0;
function eq(actual, expected, name){
  const a = JSON.stringify(actual), e = JSON.stringify(expected);
  if (a === e){ pass++; log("PASS  " + name); }
  else { fail++; log("FAIL  " + name + "\n        expected " + e + "\n        actual   " + a); }
}
function log(s){ document.getElementById("out").textContent += s + "\n"; }
function docFrom(html){
  return new DOMParser().parseFromString(html, "text/html");
}

const ARTICLE = `<article id="post">
  <p id="lede">Alpha beta gamma.</p>
  <p>Delta epsilon zeta.</p>
</article>`;

// Rung 1: selector + offsets match exactly
(function(){
  const d = docFrom(ARTICLE);
  const idx = RedlineCore.buildTextIndex(d);
  const r = RedlineCore.resolveAnchor(d, idx, {
    selector: "#post > p:nth-of-type(2)",
    start_offset: 0, end_offset: 19,
    quoted_text: "Delta epsilon zeta.",
    prefix: "", suffix: "", document_start_offset: 0
  });
  eq(r && r.toString(), "Delta epsilon zeta.", "rung 1: selector+offsets");
})();

// Rung 2: selector is wrong, prefix+quote+suffix rescues it
(function(){
  const d = docFrom(ARTICLE);
  const idx = RedlineCore.buildTextIndex(d);
  const r = RedlineCore.resolveAnchor(d, idx, {
    selector: "#nonexistent",
    start_offset: 0, end_offset: 19,
    quoted_text: "Delta epsilon zeta.",
    prefix: "", suffix: "", document_start_offset: 0
  });
  eq(r && r.toString(), "Delta epsilon zeta.", "rung 2/3: falls through to text search");
})();

// Missing text resolves to null
(function(){
  const d = docFrom(ARTICLE);
  const idx = RedlineCore.buildTextIndex(d);
  const r = RedlineCore.resolveAnchor(d, idx, {
    selector: "#nope", start_offset: 0, end_offset: 5,
    quoted_text: "absent words", prefix: "", suffix: "", document_start_offset: 0
  });
  eq(r, null, "missing quote resolves to null");
})();

// nearestIndexOf picks the occurrence closest to the hint
eq(RedlineCore.nearestIndexOf("xx ab yy ab zz", "ab", 12), 9, "nearestIndexOf picks nearest");

log("\n" + pass + " passed, " + fail + " failed");
document.title = fail ? "FAIL" : "PASS";
</script>
```

- [ ] **Step 2: Run it to verify it fails**

Run: `open redline/web/test.html` (or load the `file://` path in a browser).
Expected: page errors — `RedlineCore is not defined`. No file exists yet.

- [ ] **Step 3: Create the core module**

Create `redline/web/redline-core.js`. Move these four functions out of `app.html` verbatim except that `S.doc` / `S.textIndex` become parameters:

```js
/* redline core — pure logic, no DOM globals, no app state.
   Loaded as a plain script so web/test.html can run it over file://. */
(function(global){
"use strict";

function buildTextIndex(doc){
  const nodes = [], walker = doc.createTreeWalker(doc.body, NodeFilter.SHOW_TEXT);
  let text = "", n;
  while ((n = walker.nextNode())){
    nodes.push({ node: n, start: text.length });
    text += n.nodeValue;
  }
  return { text, nodes };
}

function nearestIndexOf(hay, needle, near){
  let best = -1, bestD = Infinity, i = hay.indexOf(needle);
  while (i >= 0){
    const d = Math.abs(i - near);
    if (d < bestD){ bestD = d; best = i; }
    i = hay.indexOf(needle, i + 1);
  }
  return best;
}

function rangeInElement(doc, el, start, end){
  const walker = doc.createTreeWalker(el, NodeFilter.SHOW_TEXT);
  let pos = 0, n, range = doc.createRange(), started = false;
  while ((n = walker.nextNode())){
    const len = n.nodeValue.length;
    if (!started && pos + len >= start){ range.setStart(n, start - pos); started = true; }
    if (started && pos + len >= end){ range.setEnd(n, end - pos); return range; }
    pos += len;
  }
  return null;
}

function rangeFromGlobal(doc, idx, gs, ge){
  const find = p => {
    for (const e of idx.nodes){
      if (p >= e.start && p <= e.start + e.node.nodeValue.length) return { node: e.node, off: p - e.start };
    }
    return null;
  };
  const a = find(gs), b = find(ge);
  if (!a || !b) return null;
  const range = doc.createRange();
  range.setStart(a.node, a.off);
  range.setEnd(b.node, b.off);
  return range;
}

/* Resolution order mirrors SKILL.md exactly: selector+offsets, then
   prefix+quote+suffix, then the quote nearest its recorded position. */
function resolveAnchor(doc, idx, a){
  if (!doc) return null;
  let el = null;
  try { el = doc.querySelector(a.selector); } catch(_){}
  if (el && el.textContent.slice(a.start_offset, a.end_offset) === a.quoted_text){
    return rangeInElement(doc, el, a.start_offset, a.end_offset);
  }
  const T = idx ? idx.text : "";
  let i = -1;
  if (a.prefix || a.suffix){
    const j = T.indexOf((a.prefix||"") + a.quoted_text + (a.suffix||""));
    if (j >= 0) i = j + (a.prefix||"").length;
  }
  if (i < 0) i = nearestIndexOf(T, a.quoted_text, a.document_start_offset || 0);
  if (i < 0) return null;
  return rangeFromGlobal(doc, idx, i, i + a.quoted_text.length);
}

global.RedlineCore = { buildTextIndex, nearestIndexOf, rangeInElement, rangeFromGlobal, resolveAnchor };
})(window);
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: reload `redline/web/test.html`.
Expected: `4 passed, 0 failed`, tab title `PASS`.

- [ ] **Step 5: Point app.html at the core module**

In `redline/web/app.html`, delete the now-duplicated `buildTextIndex`, `nearestIndexOf`, `rangeInElement`, `rangeFromGlobal`, and `resolveAnchor` bodies. Add before the main script block:

```html
<script src="/js/redline-core.js"></script>
```

Replace the five call sites with thin shims so no other code changes:

```js
function buildTextIndex(){ if (!S.textIndex && S.doc) S.textIndex = RedlineCore.buildTextIndex(S.doc); }
function resolveAnchor(a){ buildTextIndex(); return RedlineCore.resolveAnchor(S.doc, S.textIndex, a); }
function rangeInElement(el, s, e){ return RedlineCore.rangeInElement(S.doc, el, s, e); }
function rangeFromGlobal(gs, ge){ buildTextIndex(); return RedlineCore.rangeFromGlobal(S.doc, S.textIndex, gs, ge); }
```

- [ ] **Step 6: Serve and embed the new asset**

`redline/embed.go:13` — replace the directive:

```go
//go:embed web/app.html web/redline-core.js sample/article.html
var Assets embed.FS
```

`redline/internal/server/server.go` — add to the route table at line 82:

```go
mux.HandleFunc("GET /js/{name}", s.handleAsset)
```

And add the handler beside `handleApp`:

```go
// handleAsset serves a static frontend file from the asset FS. Only files
// directly under web/ are reachable, and the name is a single path segment,
// so it cannot traverse.
func (s *Server) handleAsset(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" || strings.ContainsAny(name, `/\.`) != strings.HasSuffix(name, ".js") {
		http.NotFound(w, r)
		return
	}
	b, err := fs.ReadFile(s.assets, "web/"+name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Write(b)
}
```

`redline/cmd/redline/main.go:121` — `assetFS()` already prefers the on-disk `web/` directory when `web/app.html` exists, so it needs no change; the new file is picked up from disk automatically during development.

- [ ] **Step 7: Verify the app still works end to end**

Run: `cd redline && GOTOOLCHAIN=local go build ./cmd/redline && ./redline serve`
Then: open `http://127.0.0.1:8787`, load the sample article, drag-select a paragraph, confirm a highlight card appears.
Expected: identical behavior to before. The network tab shows `GET /js/redline-core.js` returning 200.

- [ ] **Step 8: Run the full check**

Run: `cd redline && GOTOOLCHAIN=local go build ./... && GOTOOLCHAIN=local go vet ./... && gofmt -l .`
Expected: all clean, `gofmt -l` prints nothing.

- [ ] **Step 9: Commit**

```bash
git add redline/web/redline-core.js redline/web/test.html redline/web/app.html \
        redline/embed.go redline/internal/server/server.go
git commit -m "refactor: extract anchor resolution into testable core module"
```

---

### Task 2: Make resolution report ambiguity instead of guessing

**Files:**
- Modify: `redline/web/redline-core.js`
- Modify: `redline/web/test.html`

**Interfaces:**
- Consumes: `RedlineCore.resolveAnchor` from Task 1
- Produces: `RedlineCore.resolve(doc, textIndex, anchor) → {status, range, rung}` where `status` is `"resolved" | "ambiguous" | "missing"` and `rung` is `1 | 2 | 3 | 0`

Spec §11: when the quoted text occurs several times with no prefix/suffix match, the annotation must be marked **needs re-anchoring** rather than attached to a plausible guess. Today `nearestIndexOf` silently picks the closest. `resolveAnchor` is kept as a thin wrapper so Task 1's call sites keep working.

- [ ] **Step 1: Write the failing tests**

Append to `redline/web/test.html` before the summary `log(...)` line:

```js
const AMBIG = `<article id="post">
  <p>The same words here.</p>
  <p>Filler in between.</p>
  <p>The same words here.</p>
</article>`;

// Ambiguous: quote occurs twice, no prefix/suffix to disambiguate
(function(){
  const d = docFrom(AMBIG);
  const idx = RedlineCore.buildTextIndex(d);
  const r = RedlineCore.resolve(d, idx, {
    selector: "#gone", start_offset: 0, end_offset: 20,
    quoted_text: "The same words here.", prefix: "", suffix: "", document_start_offset: 0
  });
  eq(r.status, "ambiguous", "ambiguous quote is refused, not guessed");
  eq(r.range, null, "ambiguous resolution yields no range");
})();

// Ambiguous text becomes unambiguous when prefix/suffix pin it
(function(){
  const d = docFrom(AMBIG);
  const idx = RedlineCore.buildTextIndex(d);
  const r = RedlineCore.resolve(d, idx, {
    selector: "#gone", start_offset: 0, end_offset: 20,
    quoted_text: "The same words here.",
    prefix: "Filler in between.\n  ", suffix: "", document_start_offset: 0
  });
  eq(r.status, "resolved", "prefix disambiguates a repeated quote");
  eq(r.rung, 2, "and reports it resolved on rung 2");
})();

// Exact selector hit reports rung 1
(function(){
  const d = docFrom(ARTICLE);
  const idx = RedlineCore.buildTextIndex(d);
  const r = RedlineCore.resolve(d, idx, {
    selector: "#post > p:nth-of-type(2)", start_offset: 0, end_offset: 19,
    quoted_text: "Delta epsilon zeta.", prefix: "", suffix: "", document_start_offset: 0
  });
  eq(r.status, "resolved", "exact selector resolves");
  eq(r.rung, 1, "and reports rung 1");
})();

// Absent text is missing, not ambiguous
(function(){
  const d = docFrom(ARTICLE);
  const idx = RedlineCore.buildTextIndex(d);
  const r = RedlineCore.resolve(d, idx, {
    selector: "#gone", start_offset: 0, end_offset: 5,
    quoted_text: "absent words", prefix: "", suffix: "", document_start_offset: 0
  });
  eq(r.status, "missing", "absent quote is missing");
})();
```

- [ ] **Step 2: Run to verify they fail**

Run: reload `redline/web/test.html`.
Expected: FAIL — `RedlineCore.resolve is not a function`.

- [ ] **Step 3: Implement `resolve`**

In `redline/web/redline-core.js`, add `countOccurrences` and `resolve`, then redefine `resolveAnchor` as a wrapper:

```js
function countOccurrences(hay, needle){
  if (!needle) return 0;
  let c = 0, i = hay.indexOf(needle);
  while (i >= 0){ c++; i = hay.indexOf(needle, i + 1); }
  return c;
}

/* Same ladder as resolveAnchor, but reports which rung landed and refuses
   to guess when the quote is ambiguous (SKILL.md §4 step 4, client-side). */
function resolve(doc, idx, a){
  const miss = { status: "missing", range: null, rung: 0 };
  if (!doc) return miss;
  let el = null;
  try { el = doc.querySelector(a.selector); } catch(_){}
  if (el && el.textContent.slice(a.start_offset, a.end_offset) === a.quoted_text){
    const r = rangeInElement(doc, el, a.start_offset, a.end_offset);
    if (r) return { status: "resolved", range: r, rung: 1 };
  }
  const T = idx ? idx.text : "";
  if (a.prefix || a.suffix){
    const j = T.indexOf((a.prefix||"") + a.quoted_text + (a.suffix||""));
    if (j >= 0){
      const i = j + (a.prefix||"").length;
      const r = rangeFromGlobal(doc, idx, i, i + a.quoted_text.length);
      if (r) return { status: "resolved", range: r, rung: 2 };
    }
  }
  const n = countOccurrences(T, a.quoted_text);
  if (n === 0) return miss;
  if (n > 1) return { status: "ambiguous", range: null, rung: 0 };
  const i = T.indexOf(a.quoted_text);
  const r = rangeFromGlobal(doc, idx, i, i + a.quoted_text.length);
  return r ? { status: "resolved", range: r, rung: 3 } : miss;
}

function resolveAnchor(doc, idx, a){ return resolve(doc, idx, a).range; }
```

Add `resolve` and `countOccurrences` to the exported object.

- [ ] **Step 4: Run tests to verify they pass**

Run: reload `redline/web/test.html`.
Expected: `11 passed, 0 failed`.

- [ ] **Step 5: Commit**

```bash
git add redline/web/redline-core.js redline/web/test.html
git commit -m "feat: refuse ambiguous anchors instead of guessing"
```

---

### Task 3: Document key normalization and state derivation

**Files:**
- Modify: `redline/web/redline-core.js`
- Modify: `redline/web/test.html`

**Interfaces:**
- Consumes: nothing from prior tasks
- Produces: `RedlineCore.normalizeKey(raw) → string`, `RedlineCore.deriveState({resolution, response, dismissed}) → string` returning one of `"dismissed" | "needs-reanchor" | "open" | "answered" | "applied-unverified" | "done" | "answered-moved" | "stale"`

Implements spec §5 (identity) and §6 (the seven-row state table plus the ambiguity state from §11).

- [ ] **Step 1: Write the failing tests**

Append to `redline/web/test.html`:

```js
// --- normalizeKey ---
eq(RedlineCore.normalizeKey("https://blog.dev/post/#section"), "https://blog.dev/post", "strips fragment");
eq(RedlineCore.normalizeKey("https://blog.dev/post/"), "https://blog.dev/post", "strips trailing slash");
eq(RedlineCore.normalizeKey("https://blog.dev/p?utm_source=x&id=7"), "https://blog.dev/p?id=7", "strips tracking params, keeps others");
eq(RedlineCore.normalizeKey("file:///Users/d/a.html"), "file:///Users/d/a.html", "file urls pass through");
eq(RedlineCore.normalizeKey("sample:article.html"), "sample:article.html", "sample scheme passes through");

// --- deriveState: all seven rows of spec §6 plus ambiguity ---
const st = (resolution, response, dismissed) =>
  RedlineCore.deriveState({ resolution, response, dismissed });

eq(st({status:"resolved"}, null, false),                    "open",               "resolved + none");
eq(st({status:"resolved"}, {kind:"replied"}, false),        "answered",           "resolved + replied");
eq(st({status:"resolved"}, {kind:"applied"}, false),        "applied-unverified", "resolved + applied");
eq(st({status:"missing"},  {kind:"applied"}, false),        "done",               "missing + applied");
eq(st({status:"missing"},  {kind:"replied"}, false),        "answered-moved",     "missing + replied");
eq(st({status:"missing"},  null, false),                    "stale",              "missing + none");
eq(st({status:"resolved"}, null, true),                     "dismissed",          "dismissal wins over resolved");
eq(st({status:"missing"},  {kind:"applied"}, true),         "dismissed",          "dismissal wins over done");
eq(st({status:"ambiguous"}, null, false),                   "needs-reanchor",     "ambiguous");
eq(st({status:"ambiguous"}, {kind:"replied"}, false),       "needs-reanchor",     "ambiguous outranks response");
```

- [ ] **Step 2: Run to verify they fail**

Run: reload `redline/web/test.html`.
Expected: FAIL — `RedlineCore.normalizeKey is not a function`.

- [ ] **Step 3: Implement both functions**

Add to `redline/web/redline-core.js`:

```js
const TRACKING = /^(utm_[a-z_]+|fbclid|gclid|mc_eid|ref)$/i;

/* Stable identity for an annotated document. Deliberately NOT a content
   hash: a hash changes exactly when the page is edited, which is the
   moment continuity matters most. */
function normalizeKey(raw){
  const s = String(raw || "").trim();
  if (!/^https?:/i.test(s)) return s;              // file:, sample:, anything else verbatim
  let u;
  try { u = new URL(s); } catch(_){ return s; }
  u.hash = "";
  for (const k of Array.from(u.searchParams.keys())){
    if (TRACKING.test(k)) u.searchParams.delete(k);
  }
  let out = u.origin + u.pathname.replace(/\/+$/, "");
  const q = u.searchParams.toString();
  return q ? out + "?" + q : out;
}

/* Spec §6. Two independent facts — does the anchor resolve, and did the
   agent respond — plus the user's dismissal, which outranks both. */
function deriveState({ resolution, response, dismissed }){
  if (dismissed) return "dismissed";
  const status = resolution ? resolution.status : "missing";
  if (status === "ambiguous") return "needs-reanchor";
  const kind = response ? response.kind : null;
  if (status === "resolved"){
    if (kind === "replied") return "answered";
    if (kind === "applied") return "applied-unverified";
    return "open";
  }
  if (kind === "applied") return "done";
  if (kind === "replied") return "answered-moved";
  return "stale";
}

const LIVE_STATES = ["open", "answered", "applied-unverified", "needs-reanchor"];
function isLive(state){ return LIVE_STATES.indexOf(state) >= 0; }
```

Add `normalizeKey`, `deriveState`, `isLive`, and `LIVE_STATES` to the exported object.

- [ ] **Step 4: Run tests to verify they pass**

Run: reload `redline/web/test.html`.
Expected: `26 passed, 0 failed`.

- [ ] **Step 5: Commit**

```bash
git add redline/web/redline-core.js redline/web/test.html
git commit -m "feat: add document key normalization and annotation state derivation"
```

---

### Task 4: Bundle parse and serialize

**Files:**
- Modify: `redline/web/redline-core.js`
- Modify: `redline/web/test.html`

**Interfaces:**
- Consumes: nothing from prior tasks
- Produces: `RedlineCore.parseBundle(htmlString) → {schema_version, document_key, round, annotations, responses} | null`, `RedlineCore.serializeBundle(htmlString, state) → string`

Implements spec §7. The block is `<script type="application/redline+json" id="redline-state">`. `serializeBundle` must replace an existing block rather than append a second one, so a bundle can round-trip repeatedly.

- [ ] **Step 1: Write the failing tests**

Append to `redline/web/test.html`:

```js
// --- bundle round-trip ---
(function(){
  const html = "<!doctype html><html><body><p>Hi</p></body></html>";
  const state = { schema_version:"2.0", document_key:"sample:article.html",
                  round:2, annotations:[{id:"h1"}], responses:{} };
  const out = RedlineCore.serializeBundle(html, state);
  const back = RedlineCore.parseBundle(out);
  eq(back.document_key, "sample:article.html", "round-trip preserves key");
  eq(back.round, 2, "round-trip preserves round");
  eq(back.annotations.length, 1, "round-trip preserves annotations");
})();

// Re-serializing replaces the block, never appends a second one
(function(){
  const html = "<!doctype html><html><body><p>Hi</p></body></html>";
  const once = RedlineCore.serializeBundle(html, { schema_version:"2.0", document_key:"k", round:1, annotations:[], responses:{} });
  const twice = RedlineCore.serializeBundle(once, { schema_version:"2.0", document_key:"k", round:2, annotations:[], responses:{} });
  const count = (twice.match(/id="redline-state"/g) || []).length;
  eq(count, 1, "serialize replaces the existing block");
  eq(RedlineCore.parseBundle(twice).round, 2, "and the surviving block is the new one");
})();

// A document with no block parses as null, not a throw
eq(RedlineCore.parseBundle("<!doctype html><html><body></body></html>"), null, "no block parses as null");

// Malformed JSON inside the block parses as null, not a throw
eq(RedlineCore.parseBundle('<script type="application/redline+json" id="redline-state">{oops</scr'+'ipt>'), null, "malformed block parses as null");
```

- [ ] **Step 2: Run to verify they fail**

Run: reload `redline/web/test.html`.
Expected: FAIL — `RedlineCore.serializeBundle is not a function`.

- [ ] **Step 3: Implement parse and serialize**

Add to `redline/web/redline-core.js`:

```js
const BLOCK_RE = /<script\s+type="application\/redline\+json"\s+id="redline-state">([\s\S]*?)<\/script>/i;

function parseBundle(html){
  const m = BLOCK_RE.exec(String(html || ""));
  if (!m) return null;
  try { return JSON.parse(m[1]); } catch(_){ return null; }
}

/* Inert in a normal browser; meaningful only to redline. Replaces any
   existing block so a bundle can round-trip repeatedly. */
function serializeBundle(html, state){
  const json = JSON.stringify(state, null, 2).replace(/<\/script/gi, "<\\/script");
  const block = '<script type="application/redline+json" id="redline-state">\n' + json + '\n</script>';
  const s = String(html || "");
  if (BLOCK_RE.test(s)) return s.replace(BLOCK_RE, block);
  if (/<\/body>/i.test(s)) return s.replace(/<\/body>/i, block + "\n</body>");
  return s + "\n" + block;
}
```

Add `parseBundle` and `serializeBundle` to the exported object.

- [ ] **Step 4: Run tests to verify they pass**

Run: reload `redline/web/test.html`.
Expected: `33 passed, 0 failed`.

- [ ] **Step 5: Commit**

```bash
git add redline/web/redline-core.js redline/web/test.html
git commit -m "feat: add embedded state block parse and serialize"
```

---

### Task 5: IndexedDB persistence layer

**Files:**
- Create: `redline/web/redline-store.js`
- Modify: `redline/embed.go:13`
- Modify: `redline/web/app.html` (add the `<script src>`)

**Interfaces:**
- Consumes: nothing from prior tasks
- Produces: `window.RedlineStore` with `open() → Promise<void>`, `loadDoc(key) → Promise<{key, round, annotations, responses, dismissed, draw_history}>`, `saveDoc(key, doc) → Promise<void>`, `listDocs() → Promise<Array<{key, round, updated_at}>>`, `mergeResponses(key, responses) → Promise<void>`

One object store, `documents`, keyed by the normalized document key. Annotations are small — anchors and text only, no snapshots retained (spec §5, §7).

- [ ] **Step 1: Write the store**

Create `redline/web/redline-store.js`:

```js
/* redline store — IndexedDB persistence, one record per document key.
   Deliberately not a generic ORM: one store, five operations. */
(function(global){
"use strict";

const DB_NAME = "redline", DB_VERSION = 1, STORE = "documents";
let dbp = null;

function open(){
  if (dbp) return dbp;
  dbp = new Promise((res, rej) => {
    const req = indexedDB.open(DB_NAME, DB_VERSION);
    req.onupgradeneeded = () => {
      const db = req.result;
      if (!db.objectStoreNames.contains(STORE)) db.createObjectStore(STORE, { keyPath: "key" });
    };
    req.onsuccess = () => res(req.result);
    req.onerror = () => rej(req.error);
  });
  return dbp;
}

function tx(mode, fn){
  return open().then(db => new Promise((res, rej) => {
    const t = db.transaction(STORE, mode);
    const out = fn(t.objectStore(STORE));
    t.oncomplete = () => res(out && out.result !== undefined ? out.result : undefined);
    t.onerror = () => rej(t.error);
  }));
}

function blank(key){
  return { key, round: 0, annotations: [], responses: {}, dismissed: {}, draw_history: [], updated_at: null };
}

function loadDoc(key){
  return tx("readonly", s => s.get(key)).then(r => r || blank(key));
}

function saveDoc(key, doc){
  const rec = Object.assign(blank(key), doc, { key, updated_at: new Date().toISOString() });
  return tx("readwrite", s => s.put(rec)).then(() => undefined);
}

function listDocs(){
  return tx("readonly", s => s.getAll()).then(rows =>
    (rows || []).map(r => ({ key: r.key, round: r.round, updated_at: r.updated_at }))
                .sort((a, b) => String(b.updated_at).localeCompare(String(a.updated_at))));
}

/* Merge by annotation id. A stale bundle must never clobber newer local
   state, so this only adds or overwrites the ids it carries. */
function mergeResponses(key, responses){
  return loadDoc(key).then(doc => {
    doc.responses = Object.assign({}, doc.responses, responses || {});
    return saveDoc(key, doc);
  });
}

global.RedlineStore = { open, loadDoc, saveDoc, listDocs, mergeResponses };
})(window);
```

- [ ] **Step 2: Embed and load it**

`redline/embed.go:13`:

```go
//go:embed web/app.html web/redline-core.js web/redline-store.js sample/article.html
var Assets embed.FS
```

In `redline/web/app.html`, beside the core script tag:

```html
<script src="/js/redline-store.js"></script>
```

- [ ] **Step 3: Verify it round-trips in the browser**

Run: `cd redline && GOTOOLCHAIN=local go build ./cmd/redline && ./redline serve`, open `http://127.0.0.1:8787`, then in the devtools console:

```js
await RedlineStore.saveDoc("sample:article.html", { round: 1, annotations: [{id:"h1"}] });
await RedlineStore.loadDoc("sample:article.html");
await RedlineStore.listDocs();
```

Expected: the loaded record has `round: 1` and one annotation; `listDocs()` returns one entry. `loadDoc("never-seen")` returns a blank record rather than throwing.

- [ ] **Step 4: Run the full check**

Run: `cd redline && GOTOOLCHAIN=local go build ./... && GOTOOLCHAIN=local go vet ./... && gofmt -l .`
Expected: all clean.

- [ ] **Step 5: Commit**

```bash
git add redline/web/redline-store.js redline/embed.go redline/web/app.html
git commit -m "feat: add IndexedDB persistence for annotations per document"
```

---

### Task 6: Persist and restore annotations on mount

**Files:**
- Modify: `redline/web/app.html:335-356` (`S`), `:439-481` (`mountDoc`, `afterMount`), `:694-716` (`addHighlight`), `:723-729` (`removeHighlight`)

**Interfaces:**
- Consumes: `RedlineStore.loadDoc/saveDoc`, `RedlineCore.normalizeKey/resolve/deriveState`
- Produces: `S.docKey` (normalized key for the mounted document), `S.responses`, `S.dismissed`, `S.states` (map of annotation id → state string), and `persist()` / `recomputeStates()` available to later tasks

This is where the cohesive view becomes real: opening a document restores every annotation ever made against that key and re-derives each one's state against the current text.

- [ ] **Step 1: Extend the state object**

In `redline/web/app.html`, add to `S` (line 335):

```js
  docKey: null,             // normalized identity for the mounted document
  round: 0,                 // incremented by downloadBundle(); lineage only
  responses: {},            // annotation id → {kind, text}
  dismissed: {},            // annotation id → true
  states: {},               // annotation id → derived state string
  drawHistory: [],          // retired draw marks
```

- [ ] **Step 2: Derive the key and restore on mount**

In `afterMount()` (line 452), after the document is mounted and `S.doc` is set, add:

```js
  S.docKey = deriveDocKey();
  const rec = await RedlineStore.loadDoc(S.docKey);
  S.round        = rec.round || 0;
  S.highlights   = rec.annotations || [];
  S.responses    = rec.responses   || {};
  S.dismissed    = rec.dismissed   || {};
  S.drawHistory  = rec.draw_history || [];
  S.hseq = S.highlights.reduce((m, h) => Math.max(m, h._seq || 0), 0);
  recomputeStates();
  renumber();
  syncCards();
```

- [ ] **Step 3: Add state recomputation, key derivation, and persistence**

Add beside `resolveAnchor`. The snapshot's `source` object carries `{kind, ref}` where
`kind` is one of `sample | file | url | result` — that pair is what identity is built from:

```js
/* Identity comes from the snapshot's source, not the browser location:
   the same document opened from a file path and from its URL must not
   split into two annotation sets by accident. */
function deriveDocKey(){
  const src = S.snap && S.snap.source;
  if (!src || !src.ref) return "unknown:" + ((S.snap && S.snap.id) || "none");
  if (src.kind === "url")  return RedlineCore.normalizeKey(src.ref);
  if (src.kind === "file") return RedlineCore.normalizeKey(
    /^file:/i.test(src.ref) ? src.ref : "file://" + src.ref);
  // sample and result are repo-relative; keep the scheme so they never
  // collide with a real path
  return RedlineCore.normalizeKey(src.kind + ":" + src.ref);
}
```

Then:

```js
/* Staleness is never stored — it is a relationship between an anchor and
   the current text, so it is recomputed on every open. */
function recomputeStates(){
  buildTextIndex();
  S.states = {};
  for (const h of S.highlights){
    const resolution = RedlineCore.resolve(S.doc, S.textIndex, h.anchor);
    S.states[h.id] = RedlineCore.deriveState({
      resolution,
      response: S.responses[h.id] || null,
      dismissed: !!S.dismissed[h.id],
    });
    h._rects = resolution.range ? rectsForRange(resolution.range) : [];
  }
}

function persist(){
  if (!S.docKey) return Promise.resolve();
  return RedlineStore.saveDoc(S.docKey, {
    round: (S.round || 0),
    annotations: S.highlights,
    responses: S.responses,
    dismissed: S.dismissed,
    draw_history: S.drawHistory,
  });
}
```

- [ ] **Step 4: Persist on every mutation**

At the end of `addHighlight` (line 716) and `removeHighlight` (line 729), and wherever a comment or intent is edited, add:

```js
  recomputeStates();
  persist();
```

- [ ] **Step 5: Verify persistence across a reload**

Run: `./redline serve`, open the sample article, add two highlights with comments, then hard-reload the page and reopen the same sample.
Expected: both highlights reappear with their comments, positioned correctly. Devtools → Application → IndexedDB → `redline` → `documents` shows one record.

- [ ] **Step 6: Verify staleness is derived, not stored**

Run: in the console, mutate the mounted document's text under a highlight, then `recomputeStates(); S.states`.
Expected: that annotation's state flips to `"stale"` without any stored flag changing.

- [ ] **Step 7: Commit**

```bash
git add redline/web/app.html
git commit -m "feat: restore annotations per document and derive state on mount"
```

---

### Task 7: Gutter states, threads, dismiss, and the history drawer

**Files:**
- Modify: `redline/web/app.html:730-779` (`syncCards`, `paintCard`), `:807-835` (`layoutCards`)

**Interfaces:**
- Consumes: `S.states`, `S.responses`, `S.dismissed`, `persist()`, `recomputeStates()` from Task 6; `RedlineCore.isLive`
- Produces: `dismiss(id)`, `renderHistory()`

Implements spec §9. The gutter shows only live annotations; everything else moves into a collapsed drawer.

- [ ] **Step 1: Filter the gutter to live annotations**

In `syncCards()` (line 730), replace the iteration source:

```js
  const live = S.highlights.filter(h => RedlineCore.isLive(S.states[h.id] || "open"));
```

Use `live` wherever the function currently iterates `S.highlights`.

- [ ] **Step 2: Paint state chips and agent replies**

In `paintCard()` (line 765), after the existing quote element, add:

```js
  const state = S.states[h.id] || "open";
  const chip = card.querySelector(".chip") || card.appendChild(document.createElement("div"));
  chip.className = "chip chip-" + state;
  chip.textContent = ({
    "open": "open", "answered": "answered",
    "applied-unverified": "applied · verify", "needs-reanchor": "needs re-anchoring",
  })[state] || state;

  const resp = S.responses[h.id];
  if (resp){
    const reply = card.querySelector(".reply") || card.appendChild(document.createElement("div"));
    reply.className = "reply";
    reply.textContent = resp.text || resp.summary || "";
  }

  const dis = card.querySelector(".dismiss") || card.appendChild(document.createElement("button"));
  dis.className = "dismiss";
  dis.textContent = "dismiss";
  dis.onclick = () => dismiss(h.id);
```

- [ ] **Step 3: Implement dismiss**

```js
function dismiss(id){
  S.dismissed[id] = true;
  recomputeStates();
  syncCards();
  renderHistory();
  persist();
}
```

- [ ] **Step 4: Render the history drawer**

```js
/* Spec §6.2: every retired entry records what it referenced and what was
   said — for draw marks too, which carry referenced_text instead of a quote. */
function renderHistory(){
  const box = $("#history");
  const retired = S.highlights.filter(h => !RedlineCore.isLive(S.states[h.id] || "open"));
  const rows = [];
  for (const h of retired){
    const resp = S.responses[h.id];
    rows.push(
      '<li class="hist ' + esc(S.states[h.id]) + '">' +
      '<div class="hs">' + esc(S.states[h.id]) + '</div>' +
      '<div class="hr">referenced: “' + esc(truncate(h.anchor.quoted_text, 120)) + '”</div>' +
      '<div class="hy">you said: ' + esc(h.comment || "(no comment)") + '</div>' +
      (resp ? '<div class="ha">agent said: ' + esc(resp.text || resp.summary || "") + '</div>' : "") +
      '</li>'
    );
  }
  for (const d of S.drawHistory){
    rows.push(
      '<li class="hist draw">' +
      '<div class="hs">' + esc(d.type) + ' · retired at round end</div>' +
      '<div class="hr">referenced: “' + esc(truncate(d.referenced_text || "(no text under mark)", 120)) + '”</div>' +
      '<div class="hy">you said: ' + esc(d.note || "(no note)") + '</div>' +
      '</li>'
    );
  }
  $("#history-count").textContent = rows.length;
  box.innerHTML = rows.join("");
}
```

- [ ] **Step 5: Add the drawer markup and styles**

In the gutter container in `redline/web/app.html`:

```html
<div id="stale-summary"></div>
<details id="history-drawer">
  <summary>HISTORY (<span id="history-count">0</span>)</summary>
  <ul id="history"></ul>
</details>
```

- [ ] **Step 6: Add the bulk-staleness summary**

Spec §11 — returning to a much-changed page must not dump a wall of orphans. At the top of `syncCards()`:

```js
  const staleCount = S.highlights.filter(h => (S.states[h.id] === "stale")).length;
  $("#stale-summary").textContent = staleCount
    ? staleCount + " annotation" + (staleCount === 1 ? "" : "s") + " no longer resolve"
    : "";
```

- [ ] **Step 7: Build the document switcher**

Spec §9 — this replaces the inbox browser, which Task 10 removes. It is what makes
"come back to a URL and your work is there" true, so it must land before the inbox
browser goes away.

```html
<details id="doc-switcher">
  <summary>DOCUMENTS (<span id="doc-count">0</span>)</summary>
  <ul id="doc-list"></ul>
</details>
```

```js
/* Recent documents by key, with the count of annotations still awaiting a
   response. Replaces the inbox browser: there is no server listing any more. */
async function renderDocSwitcher(){
  const docs = await RedlineStore.listDocs();
  const rows = await Promise.all(docs.map(async d => {
    const rec = await RedlineStore.loadDoc(d.key);
    const open = (rec.annotations || []).filter(h =>
      !rec.dismissed[h.id] && !rec.responses[h.id]).length;
    const current = d.key === S.docKey ? " current" : "";
    return '<li class="doc' + current + '" data-key="' + esc(d.key) + '">' +
           '<span class="dk">' + esc(d.key) + '</span>' +
           '<span class="dc">' + open + ' open</span>' +
           '<span class="dr">round ' + (d.round || 0) + '</span></li>';
  }));
  $("#doc-count").textContent = docs.length;
  $("#doc-list").innerHTML = rows.join("");
}
```

Call `renderDocSwitcher()` at the end of `afterMount()` and after `persist()` resolves.
Clicking a row reopens that document through the existing snapshot-loading path; a row
whose key is not currently loadable (a URL you are no longer on) stays listed but is
marked unavailable rather than removed — its annotations are still real.

- [ ] **Step 8: Verify the states render**

Run: `./redline serve`, add three highlights. In the console:

```js
S.responses["h1"] = {kind:"replied", text:"cut sentence one"};
S.responses["h2"] = {kind:"applied", summary:"tightened"};
recomputeStates(); syncCards(); renderHistory();
```

Expected: `h1` shows an `answered` chip with the reply beneath it; `h2` shows `applied · verify`; `h3` shows `open`. Clicking dismiss on `h3` moves it into the drawer, and the count increments. The document switcher lists one document with `2 open`.

- [ ] **Step 9: Commit**

```bash
git add redline/web/app.html
git commit -m "feat: gutter states, inline replies, history drawer, and document switcher"
```

---

### Task 8: Download and import the bundle, replacing POST /export

**Files:**
- Modify: `redline/web/app.html:1185-1240` (`buildAnnotations`, the export handler)
- Modify: `redline/internal/server/server.go:87` (remove the export route)

**Interfaces:**
- Consumes: `RedlineCore.serializeBundle/parseBundle`, `RedlineStore.mergeResponses`, `persist()`, `recomputeStates()`
- Produces: `downloadBundle()`, `importBundle(file)`

Spec §7: the download is produced entirely client-side (`Blob` + object URL) with no server round-trip.

- [ ] **Step 1: Implement the download**

```js
/* Download is the round boundary: it increments the round and retires
   every draw mark (spec §6.3, §9). */
async function downloadBundle(){
  S.round = (S.round || 0) + 1;
  retireDrawMarks();
  const html = "<!doctype html>\n" + S.doc.documentElement.outerHTML;
  const state = {
    schema_version: "2.0",
    document_key: S.docKey,
    round: S.round,
    viewport: { width: S.pageW, gutter_width: S.gutterW },
    annotations: S.highlights.filter(h => RedlineCore.isLive(S.states[h.id] || "open")),
    responses: S.responses,
  };
  const blob = new Blob([RedlineCore.serializeBundle(html, state)], { type: "text/html" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = "redline-" + slug(S.docKey) + "-r" + S.round + ".html";
  a.click();
  URL.revokeObjectURL(url);
  await persist();
  toast("round " + S.round + " downloaded");
}

function slug(key){ return String(key).replace(/^[a-z]+:\/*/i, "").replace(/[^a-z0-9]+/gi, "-").slice(0, 40); }
```

- [ ] **Step 2: Implement the import**

```js
/* Merge is by annotation id, so opening a stale bundle can never clobber
   newer local state (spec §7). */
async function importBundle(file){
  const text = await file.text();
  const parsed = RedlineCore.parseBundle(text);
  if (!parsed){ toast("no redline block in that file", true); return; }
  const key = parsed.document_key || S.docKey;
  await RedlineStore.mergeResponses(key, parsed.responses || {});
  if (key === S.docKey){
    S.responses = Object.assign({}, S.responses, parsed.responses || {});
    recomputeStates(); syncCards(); renderHistory();
  }
  toast("merged " + Object.keys(parsed.responses || {}).length + " response(s)");
}
```

- [ ] **Step 3: Wire the file input and drop target**

```html
<input type="file" id="import" accept=".html" hidden>
```

```js
$("#import").onchange = e => { if (e.target.files[0]) importBundle(e.target.files[0]); };
document.addEventListener("dragover", e => e.preventDefault());
document.addEventListener("drop", e => {
  e.preventDefault();
  if (e.dataTransfer.files[0]) importBundle(e.dataTransfer.files[0]);
});
```

- [ ] **Step 4: Remove the export route**

In `redline/internal/server/server.go`, delete line 87 (`mux.HandleFunc("POST /export/{id}", s.handleExport)`) and the `handleExport` method.

- [ ] **Step 5: Verify the round-trip**

Run: `./redline serve`, annotate the sample, click Export. Open the downloaded file in a text editor and confirm the `redline-state` block is present with your annotations. Hand-edit it to add `"responses": {"h1": {"kind":"replied","text":"test reply"}}`, save, and drag it back onto redline.
Expected: `h1` flips to `answered` with "test reply" shown in its card.

- [ ] **Step 6: Run the full check**

Run: `cd redline && GOTOOLCHAIN=local go build ./... && GOTOOLCHAIN=local go vet ./... && gofmt -l .`
Expected: all clean.

- [ ] **Step 7: Commit**

```bash
git add redline/web/app.html redline/internal/server/server.go
git commit -m "feat: client-side bundle download and import, drop POST /export"
```

---

### Task 9: Draw marks capture referenced text and retire at round end

**Files:**
- Modify: `redline/web/app.html:877-892` (`commitShape`), `:922-945` (`addNote`)

**Interfaces:**
- Consumes: `S.drawHistory`, `persist()`
- Produces: `textUnderBounds(bounds) → string`, `retireDrawMarks()`

Spec §6.3: draw marks can never survive a reflow, so they are always round-scoped. But their meaning is preserved by capturing the text under them **at draw time**, which needs no image and no html2canvas.

- [ ] **Step 1: Capture text under the mark**

```js
/* At draw time the DOM is available, so a mark's meaning can be preserved
   as text rather than pixels — no html2canvas, no image storage. */
function textUnderBounds(b){
  if (!S.doc || !b) return "";
  const out = [];
  const walker = S.doc.createTreeWalker(S.doc.body, NodeFilter.SHOW_ELEMENT);
  let el;
  while ((el = walker.nextNode())){
    if (!el.getBoundingClientRect) continue;
    const r = el.getBoundingClientRect();
    const win = $("#page").contentWindow;
    const top = r.top + (win ? win.scrollY : 0), left = r.left + (win ? win.scrollX : 0);
    const overlaps = left < b.x + b.w && left + r.width > b.x && top < b.y + b.h && top + r.height > b.y;
    if (overlaps && el.children.length === 0 && el.textContent.trim()) out.push(el.textContent.trim());
  }
  return truncate(out.join(" ").replace(/\s+/g, " "), 240);
}
```

- [ ] **Step 2: Store it on every committed shape**

In `commitShape()` (line 877), before the shape is pushed to `S.draw`:

```js
  d.referenced_text = textUnderBounds(d.bounds);
```

- [ ] **Step 3: Retire draw marks at round end**

```js
function retireDrawMarks(){
  for (const d of S.draw){
    S.drawHistory.push({
      id: d.id, type: d.type, note: d.note || "",
      referenced_text: d.referenced_text || "",
      round: S.round, retired_at: new Date().toISOString(),
    });
  }
  S.draw = [];
  S.dseq = 0;
  render();
  renderHistory();
}
```

- [ ] **Step 4: Verify capture and retirement**

Run: `./redline serve`, open the sample, draw a rect over the blockquote, attach the note "where did this quote come from?", then click Export.
Expected: the canvas clears; the history drawer gains one entry reading `referenced: "A performance budget is not a diet. It is a currency…"` with `you said: where did this quote come from ?`. Reload the page and confirm the entry survives.

- [ ] **Step 5: Commit**

```bash
git add redline/web/app.html
git commit -m "feat: capture text under draw marks and retire them at round end"
```

---

### Task 10: Retarget the skill and delete the packet layer

**Files:**
- Modify: `redline/.claude/skills/redline/SKILL.md` (§1, §9, and add the target ladder)
- Modify: `redline/README.md` (the three demo prompts)
- Delete: `redline/internal/packet/`
- Modify: `redline/internal/server/server.go:86` (remove the inbox route), `:60-80` (drop the store dependency)
- Modify: `redline/cmd/redline/main.go:69-74` (drop store construction and the `-inbox` flag)
- Modify: `redline/CLAUDE.md` (architecture table, seams, gotchas)

**Interfaces:**
- Consumes: the bundle format from Task 4 and Task 8
- Produces: a skill whose trigger is a bundle path, not a directory

- [ ] **Step 1: Rewrite SKILL.md §1**

Replace the whole "1. Trigger and scope" section with this text. Note what is *deleted*: the
`inbox/` directory layout and the entire multi-pending-round tie-break rule — sessions no
longer share a directory, so there is nothing left to disambiguate.

```markdown
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
```

- [ ] **Step 2: Add the target ladder to SKILL.md**

Add a new section after §1, copied from spec §8.1 exactly:

```markdown
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
```

- [ ] **Step 3: Rewrite the output contract**

Replace the whole "9. Output contract" section with this text:

```markdown
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

Close your reply to the user with the loop, not a victory lap:

> Round N is written to `<path>`. Open it back in redline for round N+1 — the
> proposed correction in section 2 is waiting on you.
```

§0 (prime directive), §4 (anchor ladder), §5 (intent semantics), §6 (draw semantics) and §7 (ambiguity) are unchanged. §8 (verification) is unchanged except that "both widths" now means the width recorded in the block's `viewport`, not a manifest.

- [ ] **Step 4: Update the three demo prompts**

In `redline/README.md`, replace `./inbox` references. Prompt 1 becomes:

> Process the redline bundle at ./redline-article-r1.html — apply the annotations, reply to questions, verify at both widths, and give me the change summary.

Prompt 3 (the fan-out) becomes one agent per bundle file rather than per session directory.

- [ ] **Step 5: Delete the packet layer**

```bash
git rm -r redline/internal/packet
```

Remove the `-inbox` flag and store construction from `cmd/redline/main.go`, the `store` field and `handleInbox` from `internal/server/server.go`, and the `GET /inbox` route.

- [ ] **Step 6: Update redline/CLAUDE.md**

The architecture table, the "two designed-in seams" paragraph, and the packet-schema cross-file contract all describe code that no longer exists. Replace them with the new layout: core/store/app split, the embedded block as the cross-file contract, and the bundle as the integration surface.

- [ ] **Step 7: Run the full check**

Run: `cd redline && GOTOOLCHAIN=local go build ./... && GOTOOLCHAIN=local go vet ./... && gofmt -l .`
Expected: all clean, no unused imports left behind by the deletions.

- [ ] **Step 8: Verify the whole loop end to end**

Run: `./redline serve`. Annotate the sample article with one `instruct` and one `question`, Export. In a second terminal, run Claude Code and point it at the downloaded bundle with prompt 1. Confirm it writes an edited document with a `responses` block. Drag that file back onto redline.
Expected: the `instruct` annotation's anchor breaks and it shows as **done** in history; the `question` stays in place as **answered** with the reply visible in its card.

- [ ] **Step 9: Commit**

```bash
git add -A redline/
git commit -m "feat: retarget skill to bundles and remove the packet layer"
```

---

## Verification of the whole phase

- [ ] `redline/web/test.html` reports `33 passed, 0 failed`
- [ ] `GOTOOLCHAIN=local go build ./... && go vet ./... && gofmt -l .` all clean
- [ ] Annotations survive a reload, keyed by document
- [ ] An edited paragraph flips its annotation to `stale` with no stored flag
- [ ] A repeated quote yields `needs-reanchor`, never a silent wrong attachment
- [ ] Bundle round-trips: download → hand-edited responses → drag back → replies render
- [ ] Draw marks retire at Export carrying their referenced text
- [ ] `grep -r "inbox" redline/ --include=*.go` returns nothing
