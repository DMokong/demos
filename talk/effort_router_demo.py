#!/usr/bin/env python3
"""effort_router_demo.py -- Act 0 + Act 2 companion for "Making the Most of Claude Opus 5".
Usage:
    python3 effort_router_demo.py effort_sweep [--model ID] [--efforts low,xhigh] [--prompt "..."] [--backend auto|api|bedrock]
    python3 effort_router_demo.py route ["a task description to classify"] [--backend auto|api|bedrock]

Needs credentials for ONE backend: the direct Claude API (ANTHROPIC_API_KEY) or
Amazon Bedrock (AWS credentials + AWS_REGION). --backend picks explicitly;
default "auto" tries the API key first, then Bedrock, then exits with setup
instructions for both. `route` with no task argument prints the LANES table
with zero API calls, regardless of backend/credentials.

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
# Model names here are the friendly, backend-neutral IDs -- the same ones used
# as CLI defaults below. They're translated to backend-specific wire IDs only
# at client-call time (see BEDROCK_MODEL_IDS / ClientContext.resolve_model).
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

# Amazon Bedrock model IDs -- the "anthropic.<first-party-id>" convention
# (bare on-demand model ID, `anthropic.` prefix, no date suffix). Source:
# Anthropic's Claude API skill / model-migration guide, Bedrock model-ID
# mapping table. VERIFY ON THE PRESENTER MACHINE: some accounts/regions only
# expose these models via a cross-region *inference profile* ID instead of
# the bare on-demand ID (e.g. "us.anthropic.claude-opus-5" rather than
# "anthropic.claude-opus-5") -- if a live call 404s with a model-not-found
# error, check the Bedrock console's "Cross-region inference" model IDs for
# your account/region and swap the value below.
BEDROCK_MODEL_IDS: dict[str, str] = {
    "claude-opus-5": "anthropic.claude-opus-5",
    "claude-haiku-4-5": "anthropic.claude-haiku-4-5",
    "claude-sonnet-5": "anthropic.claude-sonnet-5",
}

DEFAULT_MODEL = "claude-opus-5"
DEFAULT_EFFORTS = ["low", "xhigh"]
DEFAULT_PROMPT = (
    "In three sentences, explain why cache invalidation is considered one of "
    "the two hard problems in computer science."
)

_NO_CREDENTIALS_MESSAGE = """\
No Claude credentials found for either backend.

This script talks to the real Claude API -- no fabricated numbers, no mocked
responses. Set up ONE of the following and re-run:

  Direct API
    export ANTHROPIC_API_KEY=sk-ant-...
    Get a key at https://console.anthropic.com/settings/keys

  Amazon Bedrock (what Claude Code itself uses when CLAUDE_CODE_USE_BEDROCK=1)
    export AWS_REGION=us-east-1
    ...plus AWS credentials: AWS_ACCESS_KEY_ID / AWS_SECRET_ACCESS_KEY, an
    AWS_PROFILE, or an attached IAM role -- whichever your AWS setup uses.
    python3 effort_router_demo.py effort_sweep --backend bedrock

Or pass --backend {api,bedrock} to skip auto-detection."""


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


@dataclass
class ClientContext:
    """The result of the single client factory shared by both subcommands.

    Wraps whichever SDK client backend construction produced and knows how to
    translate a friendly model ID (as used in LANES / CLI defaults) into the
    ID that backend's wire protocol expects.
    """

    backend: str  # "api" or "bedrock"
    client: object  # anthropic.Anthropic | anthropic.AnthropicBedrock
    region: str | None = None  # set for backend == "bedrock"

    def resolve_model(self, friendly_model: str) -> str:
        if self.backend == "bedrock":
            return BEDROCK_MODEL_IDS.get(friendly_model, friendly_model)
        return friendly_model

    def header(self, friendly_model: str) -> str:
        """One-line, projector-visible summary of which backend + model ID
        is about to be called -- printed before any request goes out."""
        resolved = self.resolve_model(friendly_model)
        if self.backend == "bedrock":
            region = f" region={self.region}" if self.region else ""
            return f"backend: bedrock{region} -- model {friendly_model} -> {resolved}"
        return f"backend: api -- model {friendly_model}"


def _bedrock_credentials_discoverable() -> bool:
    """Auto-detect signal: does this environment look like it's set up for
    Bedrock? Mirrors how Claude Code itself decides (CLAUDE_CODE_USE_BEDROCK=1)
    plus the standard AWS credential/region env vars."""
    return bool(
        os.environ.get("AWS_ACCESS_KEY_ID")
        or os.environ.get("AWS_PROFILE")
        or os.environ.get("AWS_REGION")
        or os.environ.get("CLAUDE_CODE_USE_BEDROCK") == "1"
    )


def _check_bedrock_dependencies() -> None:
    """boto3/botocore are only needed for AWS SigV4 request signing. If the
    presenter is using AWS_BEARER_TOKEN_BEDROCK (a Bedrock API key) instead of
    IAM credentials, neither is required, so skip the check in that case."""
    if os.environ.get("AWS_BEARER_TOKEN_BEDROCK"):
        return
    try:
        import boto3  # noqa: F401
        import botocore  # noqa: F401
    except ImportError:
        print(
            "Bedrock backend needs the 'boto3' package (AWS SigV4 request "
            "signing) unless you're using AWS_BEARER_TOKEN_BEDROCK. Install "
            "it with:\n\n    pip install boto3\n",
            file=sys.stderr,
        )
        sys.exit(1)


def _resolve_backend(requested: str) -> str:
    """Turn --backend {auto,api,bedrock} into a concrete "api" or "bedrock"
    choice. Only "auto" does detection; "api"/"bedrock" pass through as-is
    (their own credential checks happen in _build_client)."""
    if requested != "auto":
        return requested

    if os.environ.get("ANTHROPIC_API_KEY"):
        return "api"
    if _bedrock_credentials_discoverable():
        return "bedrock"

    print(_NO_CREDENTIALS_MESSAGE, file=sys.stderr)
    sys.exit(1)


def _build_client(requested_backend: str) -> ClientContext:
    """The single client factory used by both effort_sweep and route.

    Never fabricates output: if the resolved backend has no usable
    credentials, this exits nonzero with setup instructions instead of
    returning a client that would fail (or worse, silently produce nothing).
    """
    backend = _resolve_backend(requested_backend)

    if backend == "api":
        if not os.environ.get("ANTHROPIC_API_KEY"):
            print(
                "--backend api requires ANTHROPIC_API_KEY.\n\n"
                "    export ANTHROPIC_API_KEY=sk-ant-...\n\n"
                "Get a key at https://console.anthropic.com/settings/keys\n\n"
                "(Drop --backend to auto-detect, or use --backend bedrock if "
                "you have AWS credentials instead.)",
                file=sys.stderr,
            )
            sys.exit(1)
        return ClientContext(backend="api", client=anthropic.Anthropic())

    if backend == "bedrock":
        _check_bedrock_dependencies()
        # anthropic.AnthropicBedrock() resolves credentials via the standard
        # boto3 chain (env vars, shared credentials/config file, AWS_PROFILE,
        # or an attached IAM/instance role) and resolves the region from
        # AWS_REGION, then a boto3 session's configured region, falling back
        # to us-east-1 with a warning if neither is set. No explicit args
        # needed -- construction never fails for missing credentials; a bad
        # or absent credential surfaces as an auth error on the first request.
        client = anthropic.AnthropicBedrock()
        return ClientContext(backend="bedrock", client=client, region=client.aws_region)

    print(f"Unknown --backend {backend!r} (expected auto, api, or bedrock).", file=sys.stderr)
    sys.exit(1)


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


def _run_one(ctx: ClientContext, model: str, effort: str, prompt: str) -> SweepResult:
    resolved_model = ctx.resolve_model(model)
    start = time.monotonic()
    response = ctx.client.messages.create(
        model=resolved_model,
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
    ctx = _build_client(args.backend)
    efforts = [e.strip() for e in args.efforts.split(",") if e.strip()]

    print()
    print(ctx.header(args.model))
    print(f"effort_sweep -- model={args.model}")
    print(f'prompt: "{_excerpt(args.prompt, 100)}"')
    print()

    results: list[SweepResult] = []
    for effort in efforts:
        print(f"  running effort={effort} ...", file=sys.stderr)
        results.append(_run_one(ctx, args.model, effort, args.prompt))

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

    ctx = _build_client(args.backend)
    task = args.task

    classify_prompt = (
        "Classify the following engineering task into exactly one of these "
        "classes: " + ", ".join(LANES.keys()) + ".\n"
        "Respond with only the class name, nothing else.\n\n"
        f"Task: {task}"
    )
    classifier_model = "claude-haiku-4-5"
    print(ctx.header(classifier_model))
    response = ctx.client.messages.create(
        model=ctx.resolve_model(classifier_model),
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
        description=(
            "Act 0 / Act 2 demo: the effort dial and static routing, as real API "
            "calls -- over the direct Claude API or Amazon Bedrock."
        )
    )
    sub = parser.add_subparsers(dest="command", required=True)

    # Shared --backend flag for both subcommands, so client construction is
    # identical (single factory: _build_client) regardless of which one runs.
    backend_parent = argparse.ArgumentParser(add_help=False)
    backend_parent.add_argument(
        "--backend",
        choices=["auto", "api", "bedrock"],
        default="auto",
        help=(
            "which Claude backend to call: 'api' for the direct Claude API "
            "(ANTHROPIC_API_KEY), 'bedrock' for Amazon Bedrock (AWS "
            "credentials + AWS_REGION), or 'auto' (default) to pick the API "
            "key if set, else Bedrock if AWS credentials are discoverable, "
            "else exit with setup instructions for both"
        ),
    )

    sweep = sub.add_parser(
        "effort_sweep",
        parents=[backend_parent],
        help="run the same prompt at multiple effort levels and compare",
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
        "route",
        parents=[backend_parent],
        help="print the LANES table and optionally route a sample task via Haiku",
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
