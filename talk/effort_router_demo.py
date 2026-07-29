#!/usr/bin/env python3
"""effort_router_demo.py -- Act 0 + Act 2 companion for "Making the Most of Claude Opus 5".
Usage:
    python3 effort_router_demo.py effort_sweep [--model ID] [--efforts low,xhigh] [--prompt "..."]
    python3 effort_router_demo.py route ["a task description to classify"]

Act 0 (pre-run, just show output): run effort_sweep once tonight, keep the terminal
open, and project it verbatim -- "same prompt, same model, low vs xhigh, real
tokens." Act 2 (spoken over, not run): the LANES table below is the API-level
version of static + policy routing; `route` demonstrates a cheap Haiku classifier
choosing a lane for a sample task, mirroring the explorer-agent routing shown live
in Claude Code.
"""

from __future__ import annotations

import argparse
import json
import os
import sys
import time
from dataclasses import dataclass

try:
    import anthropic
except ImportError:
    print(
        "This script needs the 'anthropic' package. Install it with:\n"
        "    pip install anthropic",
        file=sys.stderr,
    )
    sys.exit(1)


# --------------------------------------------------------------------------
# LANES -- static routing policy (Act 2, layer 1-2: config + agent-definition
# routing). Shown/mentioned on stage, not executed. This is the API-parameter
# equivalent of CLAUDE_CODE_SUBAGENT_MODEL and .claude/agents/*.md frontmatter:
# a human (or Opus, at design time) decides which task classes get which
# (model, effort) pair; cheap classifiers or static dispatch just apply it.
#
#   task class                          -> (model, effort)
# --------------------------------------------------------------------------
LANES: dict[str, tuple[str, str]] = {
    # Fan-out zone: cheap, parallelizable, mechanical. Priced per attempt.
    "classify": ("claude-haiku-4-5", "low"),
    "extract": ("claude-haiku-4-5", "low"),
    "summarize": ("claude-haiku-4-5", "low"),
    # Negotiable middle: implementation work, not judgment calls.
    "implement": ("claude-sonnet-5", "medium"),
    "refactor": ("claude-sonnet-5", "medium"),
    # Judgment zone: one shot, wrong is expensive. Priced per mistake.
    "root-cause": ("claude-opus-5", "high"),
    "review": ("claude-opus-5", "high"),
    "architecture": ("claude-opus-5", "xhigh"),
}

DEFAULT_MODEL = "claude-opus-5"
DEFAULT_EFFORTS = ["low", "xhigh"]
DEFAULT_PROMPT = (
    "In three sentences, explain why cache invalidation is considered one of "
    "the two hard problems in computer science."
)


def _print_lanes() -> None:
    print()
    print("LANES -- static routing policy")
    print("-" * 60)
    print(f"{'task class':<14} {'model':<18} {'effort':<8}")
    print("-" * 60)
    for task_class, (model, effort) in LANES.items():
        print(f"{task_class:<14} {model:<18} {effort:<8}")
    print("-" * 60)
    print()


def _client() -> "anthropic.Anthropic":
    if not os.environ.get("ANTHROPIC_API_KEY"):
        print(
            "ANTHROPIC_API_KEY is not set.\n\n"
            "This script talks to the real Claude API -- no fabricated numbers,\n"
            "no mocked responses. Set your key and re-run:\n\n"
            "    export ANTHROPIC_API_KEY=sk-ant-...\n\n"
            "Get a key at https://console.anthropic.com/settings/keys",
            file=sys.stderr,
        )
        sys.exit(1)
    return anthropic.Anthropic()


@dataclass
class SweepResult:
    effort: str
    input_tokens: int
    output_tokens: int
    wall_time_s: float
    excerpt: str


def _excerpt(text: str, width: int = 88) -> str:
    one_line = " ".join(text.split())
    if len(one_line) <= width:
        return one_line
    return one_line[: width - 1] + "…"


def _run_one(client: "anthropic.Anthropic", model: str, effort: str, prompt: str) -> SweepResult:
    start = time.monotonic()
    response = client.messages.create(
        model=model,
        max_tokens=1024,
        output_config={"effort": effort},
        messages=[{"role": "user", "content": prompt}],
    )
    elapsed = time.monotonic() - start

    text = "".join(block.text for block in response.content if block.type == "text")
    return SweepResult(
        effort=effort,
        input_tokens=response.usage.input_tokens,
        output_tokens=response.usage.output_tokens,
        wall_time_s=elapsed,
        excerpt=_excerpt(text),
    )


def cmd_effort_sweep(args: argparse.Namespace) -> None:
    client = _client()
    efforts = [e.strip() for e in args.efforts.split(",") if e.strip()]

    print()
    print(f"effort_sweep -- model={args.model}")
    print(f'prompt: "{_excerpt(args.prompt, 100)}"')
    print()

    results: list[SweepResult] = []
    for effort in efforts:
        print(f"  running effort={effort} ...", file=sys.stderr)
        results.append(_run_one(client, args.model, effort, args.prompt))

    col_effort = max(6, max(len(r.effort) for r in results))
    header = (
        f"{'EFFORT':<{col_effort}}  {'IN':>6}  {'OUT':>6}  {'TIME':>7}  RESPONSE EXCERPT"
    )
    print(header)
    print("-" * len(header))
    for r in results:
        print(
            f"{r.effort:<{col_effort}}  {r.input_tokens:>6}  {r.output_tokens:>6}  "
            f"{r.wall_time_s:>6.1f}s  {r.excerpt}"
        )
    print()

    if len(results) == 2:
        lo, hi = results[0], results[1]
        if lo.output_tokens:
            ratio = hi.output_tokens / lo.output_tokens
            print(
                f"{hi.effort} used {ratio:.1f}x the output tokens of {lo.effort} "
                f"({hi.output_tokens} vs {lo.output_tokens})."
            )
    print("That's THE MAP as an API parameter.")
    print()


def cmd_route(args: argparse.Namespace) -> None:
    _print_lanes()

    if not args.task:
        print("(no task description given -- pass one to see the Haiku router in action)")
        print('  e.g. python3 effort_router_demo.py route "find why the checkout API times out"')
        return

    client = _client()
    task = args.task

    classify_prompt = (
        "Classify the following engineering task into exactly one of these "
        "classes: " + ", ".join(LANES.keys()) + ".\n"
        "Respond with only the class name, nothing else.\n\n"
        f"Task: {task}"
    )
    response = client.messages.create(
        model="claude-haiku-4-5",
        max_tokens=16,
        output_config={"effort": "low"},
        messages=[{"role": "user", "content": classify_prompt}],
    )
    label = "".join(b.text for b in response.content if b.type == "text").strip().lower()
    label = label.strip(".\"'")

    model, effort = LANES.get(label, (DEFAULT_MODEL, "high"))
    matched = "matched" if label in LANES else "unmatched -- defaulting to judgment zone"

    print(f'task: "{task}"')
    print(f"haiku classified as: {label!r} ({matched})")
    print(f"-> route to: model={model}, effort={effort}")
    print()
    print("Routing != orchestrating: Haiku picked the bucket; the policy in LANES")
    print("set the strategy. Cheap pattern-matching, not judgment.")
    print()


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="Act 0 / Act 2 demo: the effort dial and static routing, as real API calls."
    )
    sub = parser.add_subparsers(dest="command", required=True)

    sweep = sub.add_parser(
        "effort_sweep", help="run the same prompt at multiple effort levels and compare"
    )
    sweep.add_argument("--model", default=DEFAULT_MODEL, help=f"default: {DEFAULT_MODEL}")
    sweep.add_argument(
        "--efforts",
        default=",".join(DEFAULT_EFFORTS),
        help=f"comma-separated effort levels, default: {','.join(DEFAULT_EFFORTS)}",
    )
    sweep.add_argument("--prompt", default=DEFAULT_PROMPT, help="prompt to send at each effort level")
    sweep.set_defaults(func=cmd_effort_sweep)

    route = sub.add_parser(
        "route", help="print the LANES table and optionally route a sample task via Haiku"
    )
    route.add_argument("task", nargs="?", default=None, help="a task description to classify")
    route.set_defaults(func=cmd_route)

    return parser


def main(argv: list[str] | None = None) -> None:
    parser = build_parser()
    args = parser.parse_args(argv)
    args.func(args)


if __name__ == "__main__":
    main()
