# demos

Assets for the talk **"Making the Most of Claude Opus 5"** (30 min + live demo).

Two things live here: the demo tool, and the talk's supporting material.

```
redline/                     the live-demo tool — a human/agent co-authoring loop
  cmd/redline/main.go        the CLI (serve)
  internal/server/           HTTP handlers (plain http.Handler on a stdlib mux)
  internal/snapshot/         fetch/read a page, inline CSS+images, strip scripts
  internal/packet/           PacketStore interface + LocalDir (sessions/rounds)
  web/app.html               the entire frontend: viewer, draw mode, highlight mode, export
  sample/article.html        bundled demo target — a draft tech-blog article
  sample/landing.html        second demo target — a product landing page
  inbox/                     staged packets: <session-id>/round-<N>/
  .claude/skills/redline/    SKILL.md — how the agent processes one round
  .claude/agents/explorer.md model: haiku, Read/Grep/Glob, conclusions only
  README.md                  redline's own docs + the three demo prompts

talk/
  opus-5-cost-capability-map.html   "THE MAP" — interactive cost/capability chart (§4)
  effort_router_demo.py             Act 0 effort sweep + Act 2 LANES routing table
  DEMO_RUNBOOK.md                   >>> read this before going on stage <<<
```

## Build and run redline (three commands)

```bash
cd redline
go build ./cmd/redline
./redline serve                 # http://127.0.0.1:8787
```

One self-contained binary — the frontend and `sample/article.html` are compiled
in with `go:embed`. Dependencies: the Go standard library plus
`golang.org/x/net/html`. The browser loads exactly one external script
(html2canvas from cdnjs); without it, export still works minus `annotated.png`.

Run it from `redline/` so that `./inbox` and `sample/landing.html` resolve, and
so Claude Code picks up `.claude/skills/redline/SKILL.md`.

## The loop, in one line

Annotate a page in the browser → **Export** writes a packet to
`inbox/<session>/round-<N>/` → Claude Code processes it with the redline skill
and writes `result.html` + `changes.md` into the same directory → open
`result.html` back in redline for round N+1.

Details, intents, and the three demo prompts: [`redline/README.md`](redline/README.md).
Stage directions, fallbacks, and known sharp edges: [`talk/DEMO_RUNBOOK.md`](talk/DEMO_RUNBOOK.md).
