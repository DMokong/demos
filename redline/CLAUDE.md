# redline
Local page-annotation tool; its export is one turn in a human/agent loop.
Go stdlib + `golang.org/x/net/html` only; the frontend is one vanilla-JS file
(`web/app.html`) whose sole external script is html2canvas from cdnjs.

- Build/run: `go build ./cmd/redline && ./redline serve` (UI on :8787).
- Packets land in `inbox/<session>/round-<N>/`; **never edit a `snapshot.html`**.
- Processing a round is governed by `.claude/skills/redline/SKILL.md`; its prime
  directive (unannotated content is sacred) outranks anything here.
- Handlers stay plain `http.Handler`s, writes go through `packet.Store` (S3 later); surveys go to the `explorer` agent.
