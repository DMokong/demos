# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

Assets for a 30-minute talk, **"Making the Most of Claude Opus 5"**, plus its live demo. The
deliverable is a rehearsed stage performance, not a shipped product — so *content* here (staged
annotation packets, session IDs, slide text) is as load-bearing as code. Changing it can break a
narrative that was verified beat by beat on 2026-07-29.

Two areas: `redline/` is the demo tool (Go), `talk/` is the supporting material.

## Where authority lives

This file covers only the repo-wide picture. Three documents outrank it, and none of their
content is repeated here:

| Document | Governs |
|---|---|
| `redline/CLAUDE.md` | everything under `redline/` — architecture, the two designed-in seams, cross-file contracts, gotchas |
| `redline/.claude/skills/redline/SKILL.md` | how one round gets processed; its prime directive (**unannotated content is sacred**) outranks everything |
| `talk/DEMO_RUNBOOK.md` | stage operation — act script, fallbacks, and what is verified vs still manual |

`redline/README.md` holds the three verbatim demo prompts. The `redline` skill is
directory-scoped: it applies to files under `redline/` only.

## Commands

```bash
# redline (run from redline/ — the binary is cwd-sensitive; see redline/CLAUDE.md)
cd redline && go build ./cmd/redline && ./redline serve    # UI on http://127.0.0.1:8787
go build ./... && go vet ./... && gofmt -l .               # the full check; all three clean today

# talk/ — Act 0 and Act 2 material (all subcommands need: pip install anthropic)
python3 talk/effort_router_demo.py route                  # LANES table; no credentials, no API call
python3 talk/effort_router_demo.py effort_sweep --backend bedrock   # needs AWS creds + AWS_REGION
python3 talk/effort_router_demo.py effort_sweep           # --backend auto: ANTHROPIC_API_KEY, else Bedrock
```

**`anthropic` is not installed in this checkout, so every subcommand currently fails.** The import
sits at module top level behind a `try/except ImportError` that exits 1, so it gates `route` too —
`route` is free of *credentials*, not of dependencies. `boto3` (1.43.28) is present, so the Bedrock
path needs only `pip install anthropic`. Fix this before rehearsing; the runbook describes `route` as
costing nothing to run, which is true only once the package is there.

Build with `GOTOOLCHAIN=local` if your Go is 1.24.x. `go.mod` pins Go 1.24.0 and
`golang.org/x/net v0.48.0` deliberately — x/net v0.57.0 needs Go ≥ 1.25 and would trigger a
toolchain download on the demo laptop.

**There are no tests** anywhere — no `*_test.go`, no Makefile, no test runner. `redline/CLAUDE.md`
covers what to do if you add the first one.

`effort_sweep` defaults to `claude-opus-5` at efforts `low,xhigh`. It never fabricates numbers: with
no usable backend it prints setup instructions for both paths and exits nonzero — so a silent
wrong-looking table is not a failure mode, but an empty one is.

## The demo loop

redline's export *is* one turn in a human/agent loop. A human annotates a page in the browser →
Export writes a packet to `redline/inbox/<session>/round-<N>/` → an agent applies it via the redline
skill, writing `result.html` + `changes.md` into that same directory → round N's `result.html` is
round N+1's canvas. A round is "pending" when it has `snapshot.html` but no `result.html`.

The talk's three acts map onto this: Act 1 runs one round live, Act 2 shows routing (the read-only
`explorer` agent, `model: haiku`), Act 3 fans out over every pending round via a generated workflow.
`talk/slides.html` is the primary screen (20 slides, THE MAP embedded in slide 7);
`talk/slides.pdf` is projector/wifi insurance and must be re-exported whenever the HTML changes.

## Demo-critical invariants

**Never edit a staged packet.** `snapshot.html`, `manifest.json`, `annotations.json`, and
`annotated.png` are the author's turn and immutable. Every anchor in every staged packet was
independently re-resolved against its own snapshot (11 highlights, 11/11 exact); the runbook's §1.9
quality claims are byte-level and rest on those files being untouched.

**Two facts have drifted from `DEMO_RUNBOOK.md` §2.4 — verify before trusting it:**

1. A **fourth** session `s-20260730-313937/round-1` is staged (pending, created
   `2026-07-30T03:57:05Z`) and is **untracked in git**. The runbook says "three sessions, four
   rounds, three pending"; it is now four sessions, five rounds, four pending. Because demo prompt 1
   selects the pending round with the latest `manifest.created_at`, it will now pick *this* session,
   not Session C as the runbook states. Either delete it or update the runbook before rehearsing.
2. **Two** packets carry non-portable absolute paths from the original build sandbox
   (`/home/user/demos/...`), not one: Session B `s-20260729-2a0b58/round-1` (`source.ref` →
   `sample/landing.html`) and Session A `s-20260729-372474/round-2` (`source.ref` → round-1's
   `result.html`). The runbook flags only Session B. Neither path exists in this checkout, so
   `manifest.source.ref` needs editing for either to resolve here.

Relatedly, all shell paths in `DEMO_RUNBOOK.md` are `/home/user/demos/...`. This checkout is
elsewhere — translate them rather than pasting.

## Claude Code wiring in this repo

`.mcp.json` (gitignored) runs an r2mcp persistent-memory server with **`R2MCP_SCOPE=claudeclaw`** —
this project deliberately shares the claudeclaw memory pool rather than an isolated scope, so writes
from here land in production memory on purpose. It holds literal credentials rather than `${VAR}`
expansions, which is why it must never be committed.

Consequently `.claude/settings.local.json` denies `mcp__memory__meditate` and `mcp__memory__lint`:
both are confined to the *current* scope, and `meditate` archives by default, so from this repo they
would mutate claudeclaw's real data. Run them from the claudeclaw project instead. Note also that
`stats` has no scope predicate on any of its queries — it reports whole-table totals for every scope,
so never use it to judge scope isolation.
