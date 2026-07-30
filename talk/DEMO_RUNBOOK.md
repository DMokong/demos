# DEMO RUNBOOK — "Making the Most of Claude Opus 5"

Live demo: **redline**, one tool, three acts. Everything below was independently
re-verified against the actual repo on 2026-07-29 (fresh build, fresh browser
run, every staged packet re-audited). Where something is still manual, it says
**TONIGHT** and gives the exact steps.

> **Amended 2026-07-30.** redline no longer has an inbox: the round is one
> self-contained HTML bundle, downloaded client-side and dragged back in.
> The three pending rounds were restaged as bundles under `redline/bundles/`
> and every demo prompt now names a bundle path. Two behaviours changed with
> it — **draw marks retire at export and never reach the agent**, and
> **annotations persist per document rather than per round**. §2.4 is the map;
> the Act 1 and Act 3 scripts carry the amended prompts.

Read §1 first, then rehearse with §2 open.

---

## 1. Pre-flight checklist

The eight items from the talk outline (§7), each marked as it actually stands.

### 1.1 Build redline, run the full loop once, stage the demo material — **DONE**

Evidence, all re-run by the acceptance pass:

- `GOTOOLCHAIN=local go vet ./...` → exit 0, no output.
- `go build ./cmd/redline` → exit 0, one 10.3 MB static-asset-embedded binary.
  The binary already sitting at `redline/redline` is **byte-identical**
  (md5 `d8a3263f…`) to a fresh build, so it is safe to run as-is.
- Non-stdlib dependency closure is exactly `golang.org/x/net/html` and
  `golang.org/x/net/html/atom`. No other module deps.
- Full loop in a real headless Chromium against a running server: sample article
  → real mouse-drag drew a box → real click + typed text attached a note
  (`attached_to` set, parent shape's `note` mirrored, undo cleared the mirror) →
  two highlights created through the real Selection API path (one `instruct`,
  one `question`) → clicked the actual **Export round** button → packet written,
  console errors: none from the app.
- **Four sessions, five rounds** sit in `redline/inbox` as the audit trail: three
  pending, two already processed. There is no `/inbox` route any more — the three
  pending rounds were restaged on 2026-07-30 as v2.0 bundles in `redline/bundles`,
  which is what the demo prompts point at (§2.4 has the full map).
- Every anchor in every staged packet was independently re-resolved against its
  own `snapshot.html` in a clean browser page: selector resolves, `container_tag`
  matches, `container_text_length` matches, `slice(start,end) === quoted_text`,
  the document offsets slice to the same text, and the 32-char prefix/suffix both
  match. **12 highlights, 12/12 exact.** Every quoted string occurs exactly once
  in its document, so the quoted-text fallback is unambiguous too.
- Re-verified after restaging: all three bundles parse through
  `RedlineCore.parseBundle()` with `schema_version` `"2.0"`, the expected
  `document_key` and round, and **8/8 anchors resolving on rung 1** against the
  bundle's own document. `web/test.html` reports **33 passed, 0 failed**.
- All five `annotated.png` files are valid PNGs at 1180 × (1344–1692), i.e.
  exactly page width 860 + gutter 320. None corrupt, none blank.
- Session A round-1's `result.html`/`changes.md` clear the brief's Part 3 bar —
  see §1.9 below for the specific evidence.

### 1.2 Test one real static page — **DONE (with a caveat that is yours to close)**

`POST /snapshot {"url":"https://pkg.go.dev/golang.org/x/net/html"}` succeeded:
3 stylesheets inlined, **0** leftover `<link rel=stylesheet>`, **55/55** images
converted to data URIs, **0** `<script>` tags surviving, 251 KB of HTML. Rendered
in a browser with *every* non-`data:` request blocked: 13,080 px tall, 42,492
characters of text, 55/55 images visible. Best-effort inlining is real, not
theater.

Caveat: this sandbox's egress proxy refuses most hosts (`example.com`, `go.dev`
and others return `Forbidden`), so only one live site could be exercised.
redline handled the refusals cleanly — HTTP 502 with a structured JSON error, no
crash, server healthy immediately after.

> **TONIGHT (5 min):** on the demo network, start redline and snapshot the *exact*
> page you plan to point at on stage. Paste its URL into the Open panel. If it
> mangles, fall back to the bundled sample — which is the primary demo target
> anyway. Do not discover this on stage.

### 1.3 sherlogs — **PARKED**

Not built, not staged, not in this repo. redline is the demo. Nothing to do.

### 1.4 Confirm Dynamic Workflows access — **TONIGHT (2 min, cannot be checked from here)**

This is a per-account entitlement (Max/Team/Enterprise, research preview) and is
invisible to this repo.

> **TONIGHT:** in Claude Code on the presenter laptop, run `/config` and confirm
> a **"Dynamic workflow size"** setting exists. If it does, set it to `medium`
> (<15 agents) and note where it is — you open `/config` on stage during Act 3
> anyway. If it does **not** exist, you have no access: switch to the Act 3
> fallback in §3.1 *before* the talk, not during it.

### 1.5 Pre-warm: run all three acts once, screenshot every stage — **TONIGHT (25 min)**

Nothing in this repo can substitute for this: Acts 1–3 need a live Claude Code
session on your account. Exact steps in §2.

**Before you touch anything else, per the outline's revised pre-flight (§7):**

- **Clear stale redline state.** Your browser likely holds annotations under
  `sample:sample/article.html` from earlier testing, and imported responses can
  be *refused* by the id-collision guard. In the browser console:
  `await RedlineStore.saveDoc("sample:sample/article.html", {round:0, annotations:[], responses:{}, dismissed:{}, draw_history:[]})`
- **Hard-reload redline.** The Go server sends no cache headers; `app.html`
  caches aggressively and you can end up rehearsing (or demoing) yesterday's
  frontend.
- **Annotate with a real question, not a placeholder**, when you run Act 1 below.
  A placeholder comment produces an empty round — the skill correctly refuses to
  invent replies, and you see nothing come back.

As you go, capture:

1. redline with the sample article open, before annotating.
2. Mid-annotation: highlight + comment card visible, arrow drawn.
3. The "Round exported" modal showing the written path.
4. Claude Code mid-run on demo prompt 1.
5. `changes.md` open, scrolled to **Replies** (the money shot).
6. `result.html` reopened in redline — round 2 badge visible.
7. Act 2: Claude Code delegating the survey to the `explorer` agent.
8. Act 3: the generated JS orchestration script, and the final summary.

Put them in a folder you can open in one keystroke. If wifi dies, you narrate
over screenshots and the story still lands — **redline itself needs no network**
except for `annotated.png` (see §3.2).

### 1.6 Second terminal with `effort_sweep` output ready for Act 0 — **TONIGHT (3 min, needs an API key OR AWS Bedrock)**

`talk/effort_router_demo.py` is verified working: `python3 -m py_compile` passes,
`--help` and both subcommand helps render, and the `route` subcommand prints the
LANES table with no API call and no credentials. The script now supports
`--backend {auto,api,bedrock}` (default `auto`): direct API if
`ANTHROPIC_API_KEY` is set, else Amazon Bedrock if AWS credentials are
discoverable, else it prints setup instructions for **both** options and exits
nonzero — never fabricated numbers. `anthropic` 0.120.2 + `boto3` are importable
here (`boto3` is a presenter-machine prerequisite for the Bedrock path only —
`pip install boto3` if it's missing). **Gate for Act 0: `pip install anthropic`
first** — the import is top-level, so it also gates the `route` subcommand, which
needs no API key or credentials at all.

What has **never run**: a live sweep. This sandbox has no API key and no working
Bedrock credentials (the ambient `AWS_ACCESS_KEY_ID` here is a sandbox
placeholder — a real Bedrock call against it 403s with "security token
included in the request is invalid"), so no real token counts or timings exist
yet. **You have Bedrock, not an Anthropic API key — use the Bedrock path
tonight.**

> **TONIGHT (Bedrock — this is your path, no ANTHROPIC_API_KEY needed):**
> ```bash
> cd /home/user/demos/talk
> AWS_REGION=us-east-1 python3 effort_router_demo.py effort_sweep --backend bedrock
> ```
> (Swap `us-east-1` for whichever region has your Opus 5 / Haiku 4.5 Bedrock
> access enabled; AWS credentials resolve the normal way — env vars, an
> `AWS_PROFILE`, or an instance role.) The banner line printed before the
> table confirms `backend: bedrock` and the resolved `anthropic.claude-*`
> model ID — that's your check that it's really hitting Bedrock, not silently
> no-op'ing.
>
> **If you get an Anthropic API key instead:**
> ```bash
> export ANTHROPIC_API_KEY=sk-ant-...
> cd /home/user/demos/talk
> python3 effort_router_demo.py effort_sweep          # opus-5 at low vs xhigh, --backend auto picks the API key
> ```
> Leave that terminal on screen 2, scrolled to the table. Sanity-check that the
> output-token ratio is big enough to be the punchline; if `low` and `xhigh` land
> too close on your prompt, re-run with `--prompt` set to something with more
> reasoning headroom (a design question, not a lookup).

### 1.7 Timer check — **noted**

Outline's rule stands: if you are running long, **Act 3's payoff (the generated
script) is the keep; Act 0 is the cut.** Act 0 is 60 seconds of terminal output
you can also just describe.

### 1.8 Fold in Dustin's benchmark report / recalibrate chart positions — **NOT DONE (no report in repo)**

`talk/opus-5-cost-capability-map.html` is present, self-contained (zero external
scripts, zero network requests), renders clean at 1920×1080 with no console
errors, and the **"show other labs"** button works — it reveals ghost curves for
GPT-5.6 (Luna/Sol), Gemini 3 Pro, Grok 4.5 and DeepSeek v4, and flips its own
label to "hide other labs". Positions are still the "illustrative" ones.

> **TONIGHT (optional, only if the report exists):** update the data arrays in the
> `<script>` block. If not, keep saying "illustrative" out loud — the outline
> already commits to that.
>
> **Projector note:** the page is ~1269 px tall at 1920×1080 and ~1065 px at
> 1280×800, so the x-axis caption sits just below the fold on a 16:9 projector.
> Hit **⌘−** once before presenting so the whole chart fits.

### 1.9 The Part-3 quality bar (brief §Part 3, item 1) — **DONE, verified independently**

Session A round-1 already contains an agent-written `result.html` + `changes.md`.
Re-checked from scratch, not taken on trust:

| bar | result |
|---|---|
| the `question` got a **reply**, not an edit | The `question` anchors the lede paragraph. `<p id="lede">` is **byte-identical** between `snapshot.html` and `result.html`; the answer is a 700-character entry under **Replies**. |
| the fact-check correction is `<mark>`-wrapped | Exactly **one** `<mark data-redline="proposed" title="…RFC 7540 in May 2015, not 2012">2015</mark>`, visible and non-overflowing at both 860 px and 390 px. |
| unannotated paragraphs are byte-identical | Block-level multiset diff: **17 blocks in, 17 out, exactly 2 changed** — the `instruct` paragraph and the `fact-check` paragraph. `<head>`, `<style>`, `<footer>`, the blockquote and the table are byte-identical strings. The section the arrow moved is byte-identical, only repositioned. |
| two-width verification note | Present, and independently confirmed: no horizontal overflow at 860 px or 390 px (`scrollWidth === clientWidth` at both). |
| all five `changes.md` sections in order | Applied · Assumptions · Replies · Fact-check findings · Verification. Yes. |

The loop is real too: `round-2/snapshot.html` is **byte-identical** to
`round-1/result.html`, `parent_round: 1`, `parent_ref: "round-1/result.html"`,
`source.kind: "result"`, same `session_id`.

---

## 2. Act-by-act script

### 2.0 Setup, before the room fills

```bash
cd /home/user/demos/redline
./redline serve                 # http://127.0.0.1:8787
```

(`go build ./cmd/redline` first only if you touched Go source; the checked-in
binary is already current.)

**Run from `redline/`.** Three things depend on it: `./bundles` resolves to the
staged bundles the demo prompts name, `sample/landing.html` resolves as a file
path (it is *not* compiled into the binary — only `sample/article.html` is), and
Claude Code picks up `.claude/skills/redline/SKILL.md` and
`.claude/agents/explorer.md`.

**Use one browser profile for the whole talk.** Annotations live in that
profile's IndexedDB, keyed by document — a different browser or a cleared
profile starts empty, and the document drawer you show in Act 1's closing move
goes with it.

Open a **second terminal**, also in `/home/user/demos/redline`, running
`claude`. That is where all three demo prompts go. A **third** terminal holds the
Act 0 `effort_sweep` output.

**Materials:** `talk/slides.html` is the primary screen — it's the deck you
click through on stage. THE MAP is embedded in slide 7, so there's no
tab-switch to a separate map page. `talk/slides.pdf` is wifi/projector
insurance — keep it open in a PDF viewer on the presenter laptop as a fallback
if `slides.html` won't render.

> **TONIGHT:** click through all 20 slides once at the projector's real
> resolution — confirm nothing clips and slide 7's map renders before the room
> fills.

### Act 0 — The dial, 60 seconds

Switch to terminal 3. Show the pre-run table: same prompt, Opus 5 at `low` vs
`xhigh`, input/output tokens and wall time side by side.

> "That's THE MAP as an API parameter. Everything else today is about deciding
> who gets which setting."

Command (already run tonight, do not run live):
```bash
python3 /home/user/demos/talk/effort_router_demo.py effort_sweep
```

### Act 1 — Judgment + collaboration (the scribble becomes a turn)

**On the projector, in redline:**

1. Open the bundled sample: **Open** panel → *"Use the sample draft article"*.
2. Invite the volunteer. Coach them through four marks (this is the exact shape
   the skill handles best):
   - **Highlight mode** (`1`) — drag across a paragraph, intent `instruct`,
     type *"tighten this"*.
   - Highlight the dubious sentence (**"HTTP/2, standardised in 2012"**), intent
     **`fact-check`**, type *"double-check this date"*.
   - Highlight one sentence, intent **`question`**, type a real question — e.g.
     *"is this the right framing for engineers?"*. **This is the one that
     matters.** Make sure they type a question, not an instruction.
   - **Draw mode** (`2`) → `a` for arrow → drag from *inside* a section heading
     up to where it should go. Then `n` for note, click, and type
     *"move this whole section above X"*. Naming the unit ("this whole section")
     makes the scope unambiguous.

     > **Know before you do this:** draw marks **retire at export** and do not
     > travel in the bundle — they are page-pixel coordinates that cannot
     > survive a reflow. The mark drops into the history drawer carrying the
     > text it covered, and the agent never sees it. Draw it for the story if
     > you like ("this is how designers actually mark up"), but **do not
     > promise a moved section in the payoff** — the three highlight intents
     > are what Act 1 pays off on. §2.4 has the full note.

3. **Export round** (`⌘E`) → **your downloads folder**, not a round directory:
   `redline-<document>-r<N>.html`. Read the filename out loud. That one file is
   the document plus an inert state block; there is no packet directory any more.

**In Claude Code (terminal 2), demo prompt 1 — verbatim from `redline/README.md`,
with the path swapped for the file you just exported:**

> Process the redline bundle at ~/Downloads/redline-article-r1.html — apply the
> annotations, reply to questions, verify at both widths, and give me the change
> summary.

If the live export goes sideways, the staged bundle takes the same prompt
unchanged: `./bundles/redline-article-r1.html`.

Kick it off. **Talk over it — that's Act 2.** Come back for the payoff.

**What to point at when you return** (open the `changes.md` it wrote beside the
bundle):

- **Replies** — the question got an *answer*, not an edit. The anchored sentence
  is untouched. *"It's a collaborator, not a compiler."*
- **Fact-check findings** — HTTP/2 is RFC 7540, **May 2015**, not 2012. The
  correction is applied wrapped in `<mark data-redline="proposed">`: visible,
  and still the author's call. Never a silent change.
- **Applied** vs everything else — `diff snapshot.html result.html` reads like a
  review, not a rewrite. *"The prime directive: the author's words are sacred."*
- **Verification** — it checked its own work at the captured width and 390 px
  without being asked twice.

**The closing move:** back in redline, **drag the file the agent wrote onto the
page**. Replies land in their cards, applied edits break their anchors and fall
into history, and the round badge reads **2**. Same document, next round — and
the gutter's document drawer shows it as one document with a history, not two
unrelated rounds.

> "And now it's my turn again." The loop is the product.

If you want that beat pre-baked instead of live, Session A already demonstrates
it end to end (§2.4).

### Act 2 — Smart routing (delivered while Act 1 runs)

Three layers, cheapest decision-maker first:

1. **Static routing (config):** `CLAUDE_CODE_SUBAGENT_MODEL=claude-sonnet-5` —
   one env var, every delegation bills mid-tier, permanently.
2. **Policy routing (agent definitions):** show
   `redline/.claude/agents/explorer.md` — `model: haiku`, `tools: Read, Grep,
   Glob`, *"5–12 bullets, conclusions only, never paste file contents."* The
   judgment is in the policy, written once; Haiku just executes it.
3. **Judgment routing (the model itself):** in a **second Claude Code session**,
   demo prompt 2 — verbatim from `redline/README.md`:

   > Before changing anything, survey snapshot.html's structure and voice —
   > layout, styling system, and the author's tone — conclusions only.

   Watch Opus 5 hand the survey to the `explorer` agent instead of reading
   everything itself, and note that your conversation stays clean.

Slide one-liner: **routing gets smarter as you go down the list, and more
expensive — so push every decision as far up as it can live.**

Mention (do not run) the **LANES** table in `talk/effort_router_demo.py` — the
API-level version of layers 1–2. If you want it on screen it costs nothing and
needs no key:

```bash
python3 /home/user/demos/talk/effort_router_demo.py route
```

Q&A ammo ("why would Haiku pick the model?"): routing ≠ orchestrating; the
judgment lives in the policy; a router runs on every request so it must cost less
than the cheapest lane it routes to; bias to escalate — misrouting up costs
cents, misrouting down costs quality.

### Act 3 — Dynamic Workflow (the research fan-out) — **REWRITTEN 2026-07-31**

Replaces the old redline-bundle-batch version. Real research the room can judge,
re-runs without spoiling, and — critically — has **zero dependency on redline**,
which changed the night before the talk. The bundle batch is now the fallback,
below.

**⏱ Timing — decide before you walk on.** The run takes ~11.5 minutes, which does
not fit inside Act 3's slot. **Kick it off at the very start of Act 1** — before
you invite the volunteer up — so it runs underneath Acts 1 and 2, and return to it
here for the payoff. Or skip the live run and show tonight's pre-run output
instead. **Do not stand and watch it run.**

**The prompt** (verbatim from the outline §7 — this exact wording was run and
works):

> Create a workflow to research the Opus 5 release. Fan out one agent per source
> across Hacker News, the Anthropic newsroom, two other frontier labs' blogs, and
> two benchmark trackers. Then cross-check every factual claim against at least
> two independent sources, discarding anything only one source carries. Then a
> final agent finds the through-line those sources don't state outright and
> writes it as a standalone HTML article to `talk/thread.html`. **Route each
> phase to the cheapest model that can do it.**

**Measured run (2026-07-31):** 13 agents · 0 errors · **11m34s** · 6 sources, all
reachable · 102 raw claims · 6 cross-checked · **2 corroborated, 4 dropped**.

Deterministic guard, say it on stage — unchanged, still true: **a pixel diff can
tell you the page changed; it can't read an arrow.** Judgment per item — if a tool
could decide it, you'd use the tool.

While it spins up (or while narrating the pre-run): open `/config` →
**"Dynamic workflow size"**, mention subagent nesting depth 3.

**Payoff 1 — the routing, now visible in the code, not asserted over a slide.**
Open the generated JS orchestration script: `model: 'haiku'` sits on the six
parallel fetchers, `model: 'sonnet'` on the cross-checkers, one `model: 'opus'` on
the synthesis.

> "In Act 2 the routing was our policy. Here, Opus 5 wrote the routing itself —
> per task. Fan-out zone does the sweep; judgment zone finds the thread. That's
> the whole map, in one script."

**Payoff 2 — the result argues with the talk.** Open `talk/thread.html`: only the
price and the ship date survived independent cross-checking, and three claims
from these slides did not (outline §3a).

**Close the loop into Act 1:** open `talk/thread.html` in redline. The workflow
just wrote a draft; now annotate it. **Meta-punchline: the demo was the feedback
loop your team wishes it had — and the thing being reviewed was written by the
previous act.**

⚠️ **Network-dependent, unlike redline.** Wifi dies → Act 3 dies. Fallback:
tonight's pre-run `talk/thread.html` plus the counts above, narrated.

#### Fallback — offline or no Dynamic Workflows access: the redline bundle batch

Use this if wifi is down (this path touches only local files, no external
fetches) or if `/config` showed no "Dynamic workflow size" option in pre-flight
(§1.4) — in that second case there is no workflow runtime at all, so also see
§3.1's further, workflow-free fallback.

**Three bundles are already staged in `redline/bundles`.** Fire demo prompt 3 —
verbatim from `redline/README.md`:

> Create a workflow to process every redline bundle in ./bundles: one agent per
> file following the redline skill end-to-end, then a final agent that
> cross-checks the results for consistency and compiles a single summary.

Deterministic guard for this path too: **a pixel diff can tell you the page
changed; it can't read an arrow.** Interpreting redlines is design judgment per
annotation — if a tool could decide it, you'd use the tool.

While it spins up: open `/config` → **"Dynamic workflow size"**, mention subagent
nesting depth 3.

**The payoff:** open the generated JS orchestration script. One agent per bundle,
a cross-checking judge at the end, results aggregating in variables.

> "In Act 2 the routing was our policy. Here, Opus 5 wrote the routing itself —
> per task. Fan-out zone does the bundles; judgment zone does the consistency
> check. That's the whole map, in one script."

Then close the loop: Act 1's change summary is done — show the designer their
scribble, applied and verified. **Meta-punchline: the demo was the feedback loop
your team wishes it had.**

§2.4 below has the full bundle map (which session, which document, which
annotations) for this fallback path.

### 2.4 What is staged where — bundle map

**Four sessions, five rounds, three of them pending — and each pending round is
now a bundle in `redline/bundles/`.** That is what the demo prompts point at.
`redline/inbox/` still holds all five original packets, but it is the audit
trail now, not the input: nothing in the code reads it and there is no `/inbox`
route to list it.

| bundle (run from `redline/`) | document | round | annotations | from |
|---|---|---|---|---|
| `./bundles/redline-article-r1.html` | the sample article | 1 | 3 highlights | Session C round-1 |
| `./bundles/redline-article-r2.html` | round-1's *own output* | 2 | 2 highlights | Session A round-2 |
| `./bundles/redline-landing-r1.html` | the landing page | 1 | 3 highlights | Session B round-1 |

Each is one self-contained HTML file: the `snapshot.html` from its packet
**byte-for-byte**, plus one inert block —

```html
<script type="application/redline+json" id="redline-state">
{ "schema_version": "2.0", "document_key": "…", "round": N,
  "viewport": {…}, "annotations": [ … ], "responses": {} }
</script>
```

All three parse through `RedlineCore.parseBundle()` and **all 8 anchors
re-resolve on rung 1** — exact selector + offsets, quoted text matching
character for character — against the bundle's own document. Verified
2026-07-30, after restaging.

**Two things changed under the old packet story. Say them if asked, and know
them before Act 1:**

- **Draw marks retire at export and never travel in the bundle.** They are
  page-pixel coordinates against one layout, so they cannot survive a reflow.
  Export moves each one into the local history drawer carrying the *text it
  covered* (captured at draw time — no screenshot involved), and the bundle
  carries highlights only. **Consequence for Act 1: the arrow the volunteer
  draws will not reach Claude Code.** The three highlight intents carry the act;
  Act 1's step 2 has the amended coaching.
- **Annotations persist per document, not per round.** IndexedDB keyed by
  `document_key` is the source of truth, so reopening a document brings back
  every annotation ever made against it, with each one's state re-derived
  against the current text. There is no round directory and no server-side
  store; the bundle, not the store, is what travels.

Because the three staged bundles were captured in the original sandbox, their
`document_key`s carry that machine's paths — `sample:sample/article.html`,
`result:/home/user/demos/…/round-1/result.html`, and
`file:///home/user/demos/redline/sample/landing.html`. Only the first will join
up with anything you annotate live on the presenter laptop. Harmless for the
demo; worth knowing if you open one and wonder why the gutter is otherwise
empty.

#### Session A — `s-20260729-372474` — the article, mid-refinement (**the loop story**)

**round-1 — PROCESSED** (has `result.html` + `changes.md`, kept in `inbox/`)

| # | kind | anchor | what it says | agent did |
|---|---|---|---|---|
| 1 | `question` | lede, *"Every product team I have worked with has a performance budget."* | *is this too editorial an opener for an engineering audience?* | **Replied only.** Paragraph byte-identical. |
| 2 | `instruct` | *"This is a draft of an argument…"* | *tighten this — two sentences, not four* | Rewrote that paragraph, nothing else. |
| 3 | `fact-check` | *"HTTP/2, standardised in 2012"* | *double-check this date* | Verdict **incorrect** (RFC 7540, May 2015); correction applied inside `<mark data-redline="proposed">`. |
| d1+d2 | arrow + note | tail in the "three numbers" section, head above "Where the budget actually goes" | *move this whole section — heading, list, and table — above 'Where the budget actually goes'* | Moved heading + `<p>` + `<ul>` + `<p>` + `<table>` as a unit, markup verbatim. |

**round-2 → `./bundles/redline-article-r2.html`**, and it is round-1's *output*:
the document in this bundle is byte-identical to round-1's `result.html`. Two
highlights: `instruct` (*bump the byline `v0.3` to `v0.4` — round 1 landed*) and
`expand` (*grow the thin "Measuring in the wild" paragraph with a concrete
field-vs-lab example*). The block says `"round": 2`, so the loop reads correctly
on screen with no directory to explain. The packet's `annotated.png` visibly
carries the yellow proposed-correction mark forward — **that image is the best
single visual of the loop in the whole repo.** Consider putting it on a slide.

#### Session B — `s-20260729-2a0b58` → `./bundles/redline-landing-r1.html`

The landing page (`sample/landing.html`). Different document type on purpose —
it proves "works for articles, slides, pages, UIs."

| # | kind | target | comment |
|---|---|---|---|
| 1 | `instruct` | hero subhead | *cut this to one punchy sentence — lead with "no dashboard archaeology"* |
| 2 | `delete` | the public-beta banner | *we're GA now — delete this whole banner* |
| 3 | `expand` | thin "deploys" lead paragraph | *2-3 more sentences, walk through an actual incident scenario* |

The packet also holds a rect + note over the pricing card (*add an annual toggle
here — "2 months free" framing*). **That mark is not in the bundle** — draw marks
retire at export. It survives in `inbox/` if you want to tell the story, but do
not promise the agent will act on it.

#### Session C — `s-20260729-dff0fd` → `./bundles/redline-article-r1.html`

The article again, fresh session. Deliberately non-overlapping targets so Act 3's
cross-check judge sees varied work. Highlights only, no draw marks — which is why
this is the safest single bundle to demo:

| # | kind | target | comment |
|---|---|---|---|
| 1 | `instruct` | the longest of the three metric bullets | *trim this — match the other two* |
| 2 | `question` | *"field numbers are true"* | *too cute, or does it land? genuinely torn* |
| 3 | `instruct` | the blockquote | *make this the pull quote — bump the font size* |

#### Session D — `s-20260730-313937` — PROCESSED, and the best Q&A ammo in the repo

No bundle: this round already has its `result.html` + `changes.md` in `inbox/`.
Keep it closed unless someone challenges the "a pixel diff can't read an arrow"
line — then open its `changes.md`.

Eight marks: **one** highlight (labelled `instruct`, but the comment is two
questions) and **seven** draw marks — five pen strokes, one rect, one note. The
agent changed **nothing**, and said why: `result.html` is byte-identical to
`snapshot.html`. Two pen strokes form an X over the table's `Owner` column with
no note attached, and it refused to guess between *delete this column*, *these
owners are wrong*, and *who owns INP?* — then wrote down what each reading would
have cost. It also downgraded the mislabelled `instruct` to a question rather
than rewriting the thesis paragraph on a guess.

> "It did nothing, and that is the demo. Six of the eight marks carried no
> words. A tool that acted anyway would have deleted a column."

Note the mark that shaped the current design: the rect's note reads *"where did
this quote come from?"* — that text is now captured **at draw time** and retired
into the history drawer with the mark, which is why draw marks no longer need to
travel in the bundle at all.

**Choosing a bundle for demo prompt 1:** there is no tie-break rule any more —
the skill takes the path you name, so name it. Default to
`./bundles/redline-article-r1.html` (Session C: three clean highlights, one of
them a `question`, no draw marks to explain).

---

## 3. Fallbacks

### 3.1 No Dynamic Workflows access → Act 3 degrades, it does not disappear

The outline's own fallback: **sequential skill runs plus the explorer fan-out you
already showed in Act 2.** Concretely, in Claude Code:

> Process each bundle in ./bundles one at a time with the redline skill —
> redline-landing-r1.html first, then redline-article-r1.html, then
> redline-article-r2.html. For each, use the explorer agent to survey the
> document before editing. When all three are done, cross-check the three edited
> documents for consistency — shared nav/footer must still match — and compile
> one summary.

You lose the generated JS script (the payoff moment) but keep every substantive
beat: fan-out to Haiku, one agent per bundle, a judgment-zone consistency pass at
the end. Say the quiet part out loud — *"without the workflow runtime this is the
same graph, run by hand and paid for in conversation context"* — which is
precisely the argument for the feature.

Also worth knowing: on the free/Pro tier `/config` will simply not show
"Dynamic workflow size". Check tonight (§1.4), decide before you walk on.

### 3.2 CDN or wifi loss → almost everything still works

redline is hermetic (localhost) apart from one thing.

| still works offline | needs the network |
|---|---|
| serving the UI, the sample article, `sample/landing.html` | snapshotting a **URL** (local files and the bundled sample are fine) |
| draw mode, highlight mode, comments, intents, undo | `annotated.png` (needs html2canvas from cdnjs) |
| **Export** — one self-contained bundle, built client-side | live `fact-check` research (falls back to model knowledge, and says so) |
| the document drawer, IndexedDB persistence, drag-back import, the round-N+1 loop | Claude Code itself |
| `opus-5-cost-capability-map.html` (zero external requests) | `effort_router_demo.py` (which is why Act 0 is pre-run) |

The offline export path was tested for real here, with cdnjs genuinely blocked:
the toolbar chip reads **"png offline"**, Export still succeeds, and
`manifest.json` gains two `notes` entries explaining that `annotated.png` was
omitted and to work from the anchors. The skill has a documented fallback for
exactly that case. **Nothing about the demo's argument depends on the PNG** — it
is a nicety, and the packet is self-sufficient without it.

If wifi dies entirely, Claude Code cannot run: switch to the §1.5 screenshots and
narrate. Every claim you make on stage is visible in `changes.md`, which you can
open from disk.

### 3.3 If a live annotation goes sideways

- Volunteer highlights the wrong thing → the **×** on the comment card deletes it;
  `⌘Z` undoes the last draw mark.
- Selection won't take → make sure you're in **Highlight mode** (`1`); in Draw
  mode the overlay swallows mouse events by design.
- Note box appears but text doesn't land → click *into* the note input before
  typing (it autofocuses, but a stray click elsewhere steals it), then **Enter**.
- Export button appears to hang → it is waiting up to 9 s for html2canvas. It
  will fall through to the offline path on its own.
- Volunteer draws a beautiful arrow and you need it to matter → it won't reach
  the agent (draw marks retire at export). Recover by having them highlight the
  same range with an `instruct` comment saying the same thing.
- Worst case, skip the live annotation: run demo prompt 1 against
  `./bundles/redline-article-r1.html` and narrate Session A round-1's
  already-written `changes.md` from `inbox/`.

---

## 4. Known limitations and sharp edges

Honest list from the verification pass. Nothing here blocks the demo; several are
worth knowing before someone in the room asks.

**Unresolved / accepted**

1. **`snapshot.html` is a normalized re-serialization, not a byte copy.** Every
   snapshot round-trips through `x/net/html` Parse+Render even when nothing needs
   inlining, so straight quotes in body text become `&#34;`, `<!doctype html>`
   becomes `<!DOCTYPE html>`, and `<head>` reflows. It renders identically and no
   annotation is affected, but `diff sample/article.html snapshot.html` shows
   noise. Both the brief and `SKILL.md` describe `snapshot.html` as "the exact
   content annotated", which is very slightly overstated. *Not fixed — cosmetic,
   and the fix touches the snapshot pipeline on demo day.*
2. **No SRI hash on the html2canvas `<script>` tag** — a deliberate trade: a hash
   mismatch would fail silently mid-demo. It also means the page trusts whatever
   cdnjs serves. Fine for a localhost demo laptop; call it out if a security-minded
   attendee asks.
3. **`annotated.png` can get large and slow.** 411 kB at 1180×1343 for the sample.
   A 13,500 px page like pkg.go.dev will run html2canvas for several seconds with
   no hard runtime cap (only a 9 s cap on *loading* the library). Exporting a very
   tall page is the slowest moment in the tool — don't do it on stage.
4. **The staged `annotated.png` files were not produced by html2canvas.** cdnjs is
   unreachable from the build sandbox, so they are real headless-Chromium
   screenshots of the same annotated stage — same DOM, same marks, captured a
   different way. Every affected `manifest.json` records this in `notes`, so it is
   disclosed rather than hidden. **The html2canvas path has therefore never been
   exercised over a real network.** *Load the redline UI once on the demo wifi and
   confirm the toolbar chip reads "png ready" rather than "png offline."*
5. **Exporting twice bumps the round twice and downloads two files.** Legitimate
   for "I forgot an annotation", but a double-click on Export leaves you with
   `…-r1.html` and `…-r2.html` in Downloads and a round counter one ahead of the
   story. Worse, the **first** export already retired the draw marks, so the
   second bundle is missing nothing but the round number is wrong. Click once,
   and hand the agent the *highest*-numbered file.
6. **The round counter is per document and it remembers.** It lives in IndexedDB
   under the document's key, so re-annotating the sample article during rehearsal
   leaves the counter incremented for the real run. Nothing breaks; the number on
   screen just won't match the story. If it matters, rehearse in a private window
   or clear the `redline` IndexedDB database beforehand.
7. **`sample/landing.html` is not embedded in the binary** — only
   `sample/article.html` is. Opening the landing page requires the path
   `sample/landing.html` relative to the process's working directory, i.e. run
   `./redline serve` from `redline/`. The staged landing bundle carries the
   original sandbox's absolute path as its `document_key`
   (`file:///home/user/demos/redline/sample/landing.html`), so it will not join up
   with a locally-opened copy of the same file. The bundle still processes fine —
   the document travels inside it.
8. **The skill's two-width verification assumes a browser.** If the presenter's
   Claude Code session has no browser tool, `SKILL.md` instructs it to reason from
   the CSS and say so. That reads fine, but it is a weaker claim than "I looked."
9. **The anchor guarantee rests on the iframe DOM never being mutated.** Highlights
   are canvas-drawn, never injected. Any future change that writes into the
   snapshot DOM would silently invalidate every offset in the bundle's
   `redline-state` block.
10. **`SKILL.md` gives no guidance for an `annotated.png` that exists but is
    useless** (blank, wrong size, partial capture) — only for one that is *absent*.
    An agent could over-trust a bad image. Low risk with the staged packets, which
    are all good.
11. **Toolbar wraps to two rows below ~1900 px viewport width.** Readable, but check
    it at the projector's actual resolution before the room fills.
12. **Session A round-1's `changes.md` has one cosmetic slip:** it says the arrow
    moved "the heading + its 3 following blocks" and then correctly lists four
    (`<p>`, `<ul>`, `<p>`, `<table>`). The *edit* is correct — verified block by
    block — only the count in the prose is off by one. Don't read that sentence
    aloud verbatim.
13. **The `fact-check` demo without web access changes what the room sees.** The
    agent will correctly flag that it verified from model knowledge rather than a
    live source. That is intended behaviour and arguably a better beat — but it is
    a different beat.

**Environment facts, not defects**

- `example.com`, `go.dev` and most hosts are refused by the build sandbox's egress
  proxy. redline surfaces those as clean HTTP 502s with a JSON error body and stays
  healthy. On a normal network this does not apply.
- `go.mod` pins Go 1.24.0 and `golang.org/x/net v0.48.0` on purpose: x/net v0.57.0
  requires Go ≥ 1.25 and would force a toolchain download on the demo laptop. Build
  with `GOTOOLCHAIN=local` if your Go is 1.24.x.
- The compiled binary at `redline/redline` is gitignored, and current.

---

## 5. One-screen cheat sheet

```
cd /home/user/demos/redline && ./redline serve          # terminal 1 → :8787
cd /home/user/demos/redline && claude                   # terminal 2 → prompts
python3 ../talk/effort_router_demo.py effort_sweep      # terminal 3 → Act 0 (pre-run)

redline shortcuts:  1 highlight · 2 draw · p r a n tools · [ ] stroke
                    ⌘Z undo · ⌘E export · ? help

Act 1 →  "Process the redline bundle at <path> — apply the annotations, reply to
          questions, verify at both widths, and give me the change summary."
          <path> = the file ⌘E just put in ~/Downloads, or, if that went sideways,
          ./bundles/redline-article-r1.html
Act 2 →  "Before changing anything, survey the document inside
          ./bundles/redline-article-r1.html — layout, styling system, and the
          author's tone — conclusions only."
Act 3 →  "Create a workflow to research the Opus 5 release. Fan out one agent per
          source across Hacker News, the Anthropic newsroom, two other frontier
          labs' blogs, and two benchmark trackers. Then cross-check every factual
          claim against at least two independent sources, discarding anything
          only one source carries. Then a final agent finds the through-line
          those sources don't state outright and writes it as a standalone HTML
          article to talk/thread.html. Route each phase to the cheapest model
          that can do it."
          ~11.5 min — kick off at the START of Act 1, don't stand and watch it.
          Offline or no Dynamic Workflows access → bundle-batch fallback, see Act 3.

Staged:  ./bundles/redline-article-r1.html  r1 · 3 highlights (article, session C)
         ./bundles/redline-article-r2.html  r2 · 2 highlights (r1's own output)
         ./bundles/redline-landing-r1.html  r1 · 3 highlights (landing page)
         ./inbox/  audit trail only — 4 sessions, 5 rounds, no /inbox route

Remember: draw marks retire at export and never reach the agent.
          Annotations persist per document_key in IndexedDB, not per round.
```
