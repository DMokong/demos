# Making the Most of Claude Opus 5
*30 min talk (incl. live demo) + 15 min Q&A — audience: software engineers, plus platform engineers and designers*

---

## 1. Hook (2 min)
- "Five days ago, Anthropic shipped a model that doubles its predecessor's score on their hardest coding benchmark — at the same price."
- Framing: this release is about the **economics of daily use** — near-frontier intelligence you can afford to run all day.
- Cold-open story: the FreeCAD task — given a drawing of a machine part with no way to *view* the image, Opus 5 wrote its own computer-vision pipeline to extract the geometry from raw pixels and rebuilt the part. No competing model solved it in five attempts.

## 2. The Mid-2026 Lineup in 60 Seconds (2 min)
- Tier structure: Mythos class (Fable 5) sits above Opus.
- **Opus 5** (Jul 24) — the workhorse: near-Fable intelligence at half Fable's price ($5/$25 per MTok, unchanged from 4.8). Now the default Opus model in Claude Code.
- **Sonnet 5** (Jun 30) — most agentic Sonnet; performs close to Opus 4.8; $2/$10 intro until Aug 31, then $3/$15.
- **Haiku 4.5** — legacy but load-bearing: parallel execution, subagents, latency-sensitive paths.
- One-liner: *last month's flagship capability is now the mid-tier price.*

## 3. Opus 5 vs Opus 4.8 — What Actually Changed (3 min)

> ⚠️ **VERIFICATION STATUS (checked 2026-07-31 by the Act 3 workflow — read before asserting any number below).**
> Six launch claims were cross-checked against independent sources. **Only two survived: the $5/$25 price and the July 24 ship date.** Details in §3a. Say "Anthropic-reported" out loud for the rest — someone in this room has read the same coverage.

- Benchmarks: new SOTA on Frontier-Bench & GDPval-AA; >2× Opus 4.8's Frontier-Bench score at lower cost per task; within 0.5% of Fable 5's CursorBench peak at half the cost.
  - ⚠️ **Frontier-Bench: self-reported, not reproducible.** Every instance of 43.3% traces to one internal Anthropic run. The benchmark's own maintainers (Harbor/Terminal-Bench, independent of Anthropic) have **not published the dataset or code**, and their leaderboard marks the entry self-reported. Epoch AI — genuinely independent — scores Opus 5 at **159 vs Fable 5's 161**, and rates them **tied** on the software-engineering sub-index. "Surpasses all other models" is a collage of vendor self-reports, not one head-to-head test.
  - ⚠️ **CursorBench: one measurement, many channels.** Cursor's numbers (70.0% @ $8.23/task vs Fable 5 70.5% @ $17.32) check out arithmetically, but Cursor alone builds, runs, grades and publishes it, with no public test set. BenchLM excludes it from scoring for exactly that reason. Also worth knowing: Cursor retracted a Grok 4.5 score after training contamination.
- **Effort is now a 5-level dial**: low / medium / high / xhigh / max, set via `output_config.effort`. API default is `high`.
- **Thinking on by default**; can only be disabled at effort ≤ high — and thinking-on at low effort usually beats thinking-off at similar cost.
- **1M token context** — default *and* max; behavior stays consistent through the window.
- Fast mode: ~2.5× speed at 2× price (interactive paths only).
  - ⚠️ **The price is verified; the speed is not.** $10/$50 is confirmed live in production (OpenRouter). The 2.5× figure has **never been measured by anyone but Anthropic** — Artificial Analysis, the one org that benchmarks throughput independently, clocked base Opus 5 at 53.8 tok/s and has **no Fast-mode entry at all**. Not contradicted; unmeasured. Phrase it as "Anthropic says ~2.5×."
- Behavior: verifies its own work unprompted; iterates until it succeeds; pushes back on flawed designs; review accuracy holds even at low effort.
- Story options (pick one): the package-manager bug — Opus 5 found the root cause and fixed an edge case the community's own patch had missed, while a competing model fixed the surface symptom and reported it resolved · the trading-firm feed — built a market data feed for a new exchange in one session, and with no live feed to validate against, built its own test harness to check its parsing.
- Honest counterweight (engineers respect it): CodeRabbit's early review — stronger builder than reviewer, "less anxious," much better design judgment than 4.8 — but reads ~50% more and writes ~65% more per review call than baseline. Verbosity costs money → segue to the migration checklist.

## 3a. The verification pass — what actually survived (2 min, optional but strong)
*Output of the Act 3 workflow, run 2026-07-31. Full article: `talk/thread.html`. Use this either as a short beat here or as Q&A ammunition.*

- **6 claims cross-checked. 2 survived: the price and the ship date.**
- **The through-line, which is the interesting part:** the claims that survived are not the *true* ones — they are the **administrative** ones. A price is a number Anthropic decides; a ship date is an event Anthropic schedules. Neither can be falsified by experiment. What looks like corroboration is **propagation**: OpenRouter and Requesty publish $5/$25 because they *resell* at $5/$25. Nobody ran a test.
- **What sorted the claims was infrastructure, not honesty.** A claim survived if an apparatus existed for an outsider to run the test. Frontier-Bench: maintainers haven't published dataset or code, so it is unreproducible *even in principle*. Fast mode speed: no independent throughput measurement exists. CursorBench: one company builds, runs and grades it.
- **ARC-AGI-3 is the exception, and it proves the rule.** It is the only launch claim whose benchmark has an independent administrator running its own evals on a published harness — and it is the only one that produced a live adversarial dispute.
- 🎤 **The line:** *contradiction is a marker of instrumentation. A vendor number that draws fire is better-evidenced than one that draws applause — because drawing fire means someone was equipped to check it.* Read silence as a warning, not a clearance.
- 🎤 **Meta-beat if you want it:** this section was produced by the Act 3 workflow, and it fact-checked *this outline*. Three claims on my own slides did not survive. That is the demo arguing with the talk.

> 🚨 **Q&A LANDMINE — know this before you walk on.** On **2026-07-30** (yesterday) OpenAI published *"How two settings tripled our ARC-AGI-3 scores"*: GPT-5.6 Sol at **38.3%** on the public set vs Opus 5's **30.2%**, using two general Responses API settings. ARC Prize's co-founder confirmed the settings are legitimate but off-leaderboard for parity. It is OpenAI self-reported and unreplicated on a private set — say so — but if you assert unqualified Opus dominance and someone in the room read that post, you are on the back foot. Get there first.

## 4. THE MAP — Cost vs Output Across Models × Effort (3 min)
*(→ interactive chart: `opus-5-cost-capability-map.html`)*
- One visual that carries the whole argument: each model is a **curve** through cost/output space, not a point — effort moves you along the curve, model choice jumps you between curves.
- Key readings:
  - Opus 5 low/medium land near where Opus 4.8 high used to sit — for less.
  - Sonnet 5 at xhigh approaches Opus-4.8-class cost — at that point, jump tiers instead.
  - Opus 5 max is *not* always better than xhigh (Frontier-Bench peaks at xhigh).
  - **Top of the market is an overlap, not a hierarchy**: independent scores now put Opus 5 (max) at 61 on the AA Intelligence Index — highest of 170 tracked models, one point above Fable — plus >100 Elo ahead on GDPval-AA v2 and +9.6 on Frontier-Bench, at half the price. Fable keeps SWE-bench Pro (+0.8), factual knowledge, and creative polish. Chart shows the curves interleaving at the top.
  - Haiku sits far left: the fan-out zone.
- Punchline: **tune effort before you switch models; switch models when the curves cross.**
- 🎤 *Speaker note — the shaded zones (likely question: "what's the shading?"):*
  - **Fan-out zone** (left, teal): cheap end — Haiku, DeepSeek-class, Sonnet at low effort. For mechanical, parallelizable work: reading a hundred files, grep, summarize, extract. Defining property: you can run many instances at once and afford to waste some output — an irrelevant file read costs a fraction of a cent. Volume is the strategy. (= the `model: haiku` explorer agent, the mechanical nodes of a workflow.)
  - **Judgment zone** (right, coral): expensive end — Opus 5 high/xhigh, Fable. For work where a single decision carries the consequences: root cause, architecture, interpreting a hundred subagent reports, final review. Defining property: you run it once, and being wrong is expensive — a misdiagnosed root cause costs days, so 15–20× per task is good economics. (= the orchestrator's seat.)
  - The unshaded middle (Sonnet 5, Opus 5 low/med) is deliberate: negotiable territory where curve-crossing and effort sweeps decide.
  - One-liner: **the fan-out zone is priced per attempt; the judgment zone is priced per mistake.** The router's real job is deciding which zone a task's *failure cost* lives in — not just its difficulty.
- **"Don't take my word for it" beat (cross-lab prior art):**
  - Artificial Analysis's Intelligence Index vs Cost per Task chart is this map at industry scale — Pareto frontier framing ("everything off the frontier is paying more for less"), now literally on SF billboards. They plot effort levels as curves too: for any GPT-5.6 Terra effort level, a Luna/Sol level beats it — same "curves cross" logic.
  - Every lab converged on the dial in 2026: OpenAI max reasoning + "ultra" parallel mode, Gemini thinking levels + Deep Think, Grok effort param. "Which model?" → "which model, at what effort?"
  - Academic backing: Pareto-frontier price-performance is improving ~5–10x per year — this map is a snapshot of a curve sliding down-and-right fast (good closing stat).
  - Live moment: hit **"show other labs"** on the chart to reveal ghost curves for GPT-5.6, Gemini 3, Grok, DeepSeek.
  - Caveat if asked: the dials aren't apples-to-apples across labs — cross-lab comparisons normalize by token spend, not parameter names; several headline numbers are vendor-reported.

## 5. Migration Checklist — What to DELETE From Your Prompts (3 min)
- ❌ "Double-check your answer" / "verify with a subagent" → over-verification, wasted tokens, no quality gain.
- ❌ Legacy harness verification steps → same problem.
- ❌ "Only report high-severity issues" in review prompts → it obeys literally and under-reports; ask for everything, filter in a second pass.
- ❌ Effort defaults carried over from 4.8 → recalibrated levels; re-run a sweep on your own evals.
- ✅ ADD: explicit conciseness instructions, scope constraints, delegation caps, narration cadence.
- Theme: **the model got more agentic, so your prompts should get less paranoid.**
- *The operational sequel now has its own slide (**slide 11, "The trial period"**) — the Delete Protocol trial-period checklist, sourced from Boris Cherny's YC Startup School 2026 talk: archive then delete · a bare week, don't guess · add a line back only after the same stumble, repeatedly · charge rent per line · keep evals, expect them to saturate. Its speaker notes carry the two experiments (`claude --system-prompt`, `CLAUDE_CODE_SIMPLE=1`); the five self-maintenance routines now live on slide 15 ("Steal these"), which bridges into §6.*

## 6. Introduction to Dynamic Workflows (5 min)
*(How Opus 5's delegation instincts become infrastructure.)*

### What it is
- A Claude Code feature (research preview since May): you describe a complex task → Claude **writes a JavaScript orchestration script** → the workflow runtime executes it in the background, fanning work out across subagents → intermediate results stay in script variables → Claude receives only the coordinated final result.
- Contrast with normal subagent work: usually Claude orchestrates turn by turn and every result flows back into the conversation. With a workflow, the coordination moves out of the conversation and into a script — your context stays clean and the fan-out runs in parallel.
- Available in CLI, Desktop, and the VS Code extension for Max/Team/Enterprise plans (Enterprise admin-controlled).

### What it's for (and not for)
- For: wide, parallelizable work — codebase-wide bug hunts, migrations touching hundreds of files, security audits, performance profiling across services.
- Not for: a faster way to write a single function. Interactive, ambiguous work stays in the normal conversation loop.
- Safety rule: workflows that modify code are only as safe as your test gates — the workflow can run tests as part of the process, *but only if the tests exist*. Best argument yet for writing tests first.

### Guardrails & mechanics worth showing
- Default size guideline is "medium" (aim for <15 agents); change via "Dynamic workflow size" in `/config`.
- Subagents can nest to depth 3 by default (`CLAUDE_CODE_MAX_SUBAGENT_SPAWN_DEPTH=1` to disable).
- Combine with `CLAUDE_CODE_SUBAGENT_MODEL=claude-sonnet-5` so the fan-out bills mid-tier while Opus 5 does the decomposition — the map's two zones, wired together.
- Effort composes: low effort for mechanical workflow nodes, xhigh for the judge/verify steps.
- 🎤 **Gotcha worth showing, learned the hard way while building this talk's demo:** `CLAUDE_CODE_SUBAGENT_MODEL` **does not reach workflow agents.** They inherit the main-loop model unless the *script* says otherwise. Agent definitions (`model: haiku`) only apply if the script passes that agent type. **For a workflow, routing lives in the generated script — nowhere else.** I built the Act 3 demo, forgot to ask for routing, and ran six mechanical web fetches on Opus. That is exactly the failure Act 2 warns about, committed while building the demo for Act 2.
  - The fix is a **prompt** clause, not a setting: *"route each phase to the cheapest model that can do it."* Which is the better demo anyway — you state the principle, Opus makes the assignments, and you read them off the generated script.

### Two stories that sell it
- The Bun creator ported 535,496 lines across 1,448 files from Zig to Rust in 11 days (May 3–14, 2026) — up to 64 parallel Claude instances, with steering and adversarial review, and the existing test suite expected to pass at the end. A task previously scoped in quarters.
- Community demo: someone sent Opus 5 to build an F1 showroom in Blender — it drafted the car, then unexpectedly fired off workflows with **27 agents, one per part**, each building a high-detail piece. Nobody asked it to; the decomposition instinct is in the model.
- *The template (task · verifier · exit) and the five self-maintenance routines now have their own slide — **slide 15, "Steal these"** — the take-home that closes this section, sourced from the same Boris Cherny YC Startup School 2026 talk.*
- 🎤 *Pocket note (tabled topic, keep for Q&A if someone asks "isn't this just LangGraph?"):* there's a live "loops vs graphs" debate on X this month — one-liner answer: a loop is one node, a graph is several; Dynamic Workflows' twist is that the model writes the graph per task instead of a developer maintaining it. Multi-agent caution stats if pressed: ~15× token cost vs single chat (Anthropic eng), 14 catalogued failure modes (UC Berkeley).

## 7. LIVE DEMO (8 min) — "One tool, three acts"
*Two interchangeable scenario kits, same three-act structure — pick after tonight's rehearsal:*
- ***redline*** *(→ `demo-brief-redline.md`, primary): a co-authoring loop. A local tool (Go single binary, JS whiteboard) for annotating a copy of any page — including the agent's own previous output: draw on it, highlight text, attach comments with intents (instruct/question/fact-check/expand/delete). Export = a packet; Claude Code + a purpose-built skill processes it as one round — surgical edits, replies to questions, marked fact-check corrections — and the result loads back in for the next round. Works for articles, slides, pages, UIs. Hits all three audiences. Hermetic (localhost).*
- ***sherlogs*** *(→ `demo-brief-observability.md`, alternate): incident archaeology on the real Grafana/Loki/Mimir stack — SLO burn, quiet root cause, red-herring deploy annotation, alert-rule audit with a backtest gate. Flashier for platform engineers; bigger failure surface (network, API keys, ingestion windows).*
- *Decision rule: rehearse both tonight if time allows; redline is the default. (Earlier flaky-test kit retired; brief kept in `demo-repo-brief.md` if ever wanted.)*

### Act 0 — The dial, in 60 seconds (pre-run, just show output)
- Show the terminal output of `effort_sweep` from `effort_router_demo.py`: the *same prompt* on Opus 5 at `low` vs `xhigh`, with input/output token counts side by side.
- One sentence: "That's THE MAP as an API parameter. Everything else today is about deciding who gets which setting."

### Act 1 — Judgment + collaboration (Opus 5, xhigh) — *the scribble becomes a turn in the work*
> ⚠️ **redline was rebuilt on 2026-07-30. `./inbox` no longer exists.** Rounds are now **bundles**: Export writes a downloaded `.html` file carrying an inert `redline-state` block. Annotations persist per document in the browser, and the agent's reply now renders **in the gutter next to the annotation**, not only in `changes.md`. Full changelog: `docs/superpowers/specs/2026-07-30-talk-material-migration-handoff.md`.

- Live on the projector: open the sample draft article in redline, then **invite a volunteer** — highlight a paragraph + "tighten this" (instruct), highlight the dubious claim + fact-check, highlight a sentence + type a *question*, draw one arrow moving a section up. **Export → a file lands in your downloads folder.**
- In Claude Code: *"Process the redline bundle at `<path to the export>` — apply the annotations, reply to questions, verify at both widths, and give me the change summary."* Kick it off, talk over it (Act 2), return for the payoff.
- What to point at when you return: the **question got a reply, not an edit** — "it's a collaborator, not a compiler"; the fact-check correction is visually marked as *proposed*, awaiting the author; unannotated paragraphs are untouched — **the prime directive: the author's words are sacred**; the two-width verification note.
- **The payoff is stronger than it was.** Drag the returned file back onto redline: the reply appears **in the annotation's own card**, under a green ANSWERED chip. You point at the page, not at a markdown file. The `instruct` annotation's anchor breaks because the text changed, so it retires to history as **done** — visible proof the agent acted. *"And now it's my turn again."* The loop is the product.

### Act 2 — Smart routing (while Act 1 runs)
- Show the three routing layers, cheapest decision-maker first:
  1. **Static routing (config):** `CLAUDE_CODE_SUBAGENT_MODEL=claude-sonnet-5` — every delegation bills mid-tier by default. One env var, permanent saving.
  2. **Policy routing (agent definitions):** `.claude/agents/explorer.md` → `model: haiku`, tools Read/Grep/Glob only, "return conclusions, not file dumps." The policy is written once by a human/Opus; Haiku just executes it.
  3. **Judgment routing (the model itself):** in a second session:
     > *"Before changing anything, survey snapshot.html's styling system — layout approach, spacing scale, color tokens — conclusions only."*
     Watch Opus 5 fan the survey out to the explorer instead of reading everything itself — and note your conversation stays clean.
- 🎤 One-liner for the slide: **routing gets smarter as you go down the list, and more expensive — so push every decision as far up as it can live.** (The API version of layer 1–2 is the LANES table in `effort_router_demo.py` — mention, don't run.)
- 🎤 *Speaker note — "why would Haiku pick the model? Isn't that brains work?"* (near-certain Q&A):
  - **Routing ≠ orchestrating.** Routing is triage into buckets — cheap pattern-matching, Haiku's lane. Orchestration (decomposing, sequencing, judging results) stays with Opus.
  - The judgment lives in the **policy** (LANES table / agent frontmatter), written by you or Opus at design time — Haiku applies it, it doesn't set strategy.
  - **Economics:** a router runs on every request; it must cost less than the cheapest lane it routes to, or you pay Opus tokens just to learn a task was cheap.
  - **Safety:** bias to escalate on uncertainty; misrouting up costs cents, misrouting down costs quality.
  - In Claude Code, Opus *does* pick models/effort per subagent — it's already in the loop, so routing comes free. The standalone Haiku router is **the front door, not the brain.**

### Act 3 — Dynamic Workflow (the research fan-out) — **REWRITTEN 2026-07-31, rehearsed**
*Replaces the old redline-batch version. Why: the bundle batch demonstrated the mechanism over synthetic work you staged yourself. This does real research the room can judge, re-runs without spoiling, and — critically — has **zero dependency on redline**, which changed the night before the talk.*

- **The prompt** (this exact wording was run and works):
  > *"Create a workflow to research the Opus 5 release. Fan out one agent per source across Hacker News, the Anthropic newsroom, two other frontier labs' blogs, and two benchmark trackers. Then cross-check every factual claim against at least two independent sources, discarding anything only one source carries. Then a final agent finds the through-line those sources don't state outright and writes it as a standalone HTML article to `talk/thread.html`. **Route each phase to the cheapest model that can do it.**"*
- **Measured run (2026-07-31):** 13 agents · 0 errors · **11m34s** · 6 sources, all reachable · 102 raw claims · 6 cross-checked · **2 corroborated, 4 dropped**.
- ⏱ **STAGING: 11.5 min does not fit inside an 8-minute demo slot.** Kick it off at the *start* of Act 1, let it run underneath Acts 1 and 2, and return to it in Act 3 — exactly the pattern already used for Act 1. Or show tonight's pre-run. **Do not stand and watch it.**
- 🎤 *Deterministic guard, unchanged and still true:* **a pixel diff can tell you the page changed; it can't read an arrow.** Judgment per item — if a tool could decide it, you'd use the tool.
- While it spins up: open `/config` → "Dynamic workflow size", mention nesting depth 3.
- **The payoff moment — now visible in the code, not asserted over a slide.** Open the generated JS script: `model: 'haiku'` sits on the six parallel fetchers, `model: 'sonnet'` on the cross-checkers, one `model: 'opus'` on the synthesis. *"In Act 2 the routing was our policy. Here, Opus 5 wrote the routing itself — per task. Fan-out zone does the sweep; judgment zone finds the thread. That's the whole map, in one script."*
- **The second payoff — the result argues with the talk.** Open `talk/thread.html`: only the price and the ship date survived, and three claims from *these slides* did not. See §3a.
- **Close the loop into Act 1:** open `talk/thread.html` in redline. The workflow just wrote a draft; now annotate it. **Meta-punchline: the demo was the feedback loop your team wishes it had** — and the thing being reviewed was written by the previous act.
- ⚠️ **Network-dependent, unlike redline.** Wifi dies → Act 3 dies. Fallback: tonight's pre-run output (`talk/thread.html` + the counts above), narrated. Keep the old redline-bundle batch documented in `DEMO_RUNBOOK.md` as the offline alternate.
- ⚠️ **Honest limitation, worth owning if asked:** 102 raw claims deduped to 100 — the grouping was naive string matching, so "top 6 by corroboration count" was really "6 claims that happened to share a prefix." A better run would cluster claims semantically before ranking. The verdicts on the 6 checked are sound; the *selection* of those 6 was not principled.

### Pre-flight checklist — **REVISED 2026-07-31, morning of**
- [ ] **Clear stale redline state before rehearsing.** Your browser holds annotations under `sample:sample/article.html` from yesterday's testing, and imported responses can be *refused* by the id-collision guard. In the console: `await RedlineStore.saveDoc("sample:sample/article.html", {round:0, annotations:[], responses:{}, dismissed:{}, draw_history:[]})`
- [ ] **Hard-reload redline.** The Go server sends no cache headers; `app.html` caches aggressively and you can end up demoing yesterday's frontend.
- [ ] **Run Act 1 end to end once** — annotate with a *real* question (placeholder comments produce an empty round: the skill correctly refuses to invent replies, and you see nothing).
- [ ] Test one real static page so "point it at any web page" isn't theater; if it mangles, demo the bundled page only.
- [ ] ~~sherlogs~~ — parked, not built, not in the repo. Nothing to do.
- [ ] Confirm Dynamic Workflows access (`/config` → "Dynamic workflow size"). No access → Act 3 falls back to the redline-bundle batch documented in `DEMO_RUNBOOK.md`.
- [ ] **Act 3 timing decision:** it runs ~11.5 min. Either kick it off at the start of Act 1 and return in Act 3, or show tonight's pre-run. Decide before you walk on.
- [ ] Keep `talk/thread.html` from tonight's run as the wifi fallback — Act 3 is the only network-dependent act.
- [ ] Second terminal with `effort_sweep` output ready for Act 0 — needs `pip install anthropic` first; the import is top-level and gates `route` too.
- [ ] Timer check: if running long, Act 3's payoff (the generated script + the verification result) is the keep; Act 0 is the cut.
- [ ] Fold in Dustin's benchmark report when ready → recalibrate chart positions from it.

## 8. Quick Hits for Platform Engineers & Designers (2 min)
*Slide dropped 2026-07-31 in favor of the §3a verdict table (slide 18); quick hits now live in the close slide's notes as an optional verbal beat.*
- Platform: mid-conversation tool changes without cache invalidation (beta); automatic fallbacks so safety-flagged requests route instead of block (beta); org-wide default models for Team/Enterprise; no data-retention requirement for general access.
- Designers: much stronger visual output (animations, 3D, interactive artifacts); frontend self-checking — it opened its pages at desktop *and* phone widths, caught a product below the mobile fold, fixed it before handing back; vision works best with tools to crop/verify iteratively.

## 9. Close (2 min)
1. Make Opus 5 the daily driver — tune cost with **effort**, not by downgrading models.
2. Update prompts by *removing* things — verification scaffolding now costs money and adds nothing.
3. Route deliberately: Opus for judgment, Sonnet for implementation, Haiku for fan-out — and reach for a Dynamic Workflow when the work is wide, parallel, and test-gated.
- Final slide: links — Opus 5 announcement, Opus 5 prompting guide, effort docs, Sonnet 5 announcement, Dynamic Workflows docs, Artificial Analysis Intelligence-vs-Cost chart.

---

## Still open
- ~~Which repo for the Part B workflow demo?~~ **Resolved** — Act 3 is now the research fan-out, no repo needed and no test-gate risk.
- ~~Do you have Dynamic Workflows access?~~ **Confirmed working** — two runs completed 2026-07-30/31.
- Want a Go version of the API demo as backup? (Python assumed for broadest reach.)
- Chart positions are labeled "illustrative" — per-effort scores aren't published. **Now doubly important given §3a:** the Frontier-Bench and CursorBench positions rest on numbers that did not survive verification. Keep saying "illustrative" out loud.
- ~~`slides.html` needs updating~~ **Done 2026-07-31** — demo-hub Act 3 card and speaker notes now describe the research fan-out, the bundle flow, the 11.5-min timing, and the verification payoff. **`slides.pdf` still needs re-exporting** — it is the wifi fallback, and a stale fallback is worse than none.
- **Decide the §3a beat**: run it as a 2-minute section, or hold it entirely for Q&A. It is the most interesting thing in the talk and also the most likely to derail the timer.
