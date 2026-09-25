#!/usr/bin/env python3
"""Read and decode Liki feedback-v1 records.

The API intentionally stores the self-describing feedback payload verbatim. This
command renders the stable fields used for triage without requiring reviewers to
inspect raw JSON.
"""

from __future__ import annotations

import argparse
import json
import os
import sys
import urllib.error
import urllib.parse
import urllib.request
from typing import Any


DEFAULT_URL = "https://liki.hk/api/feedback"


def parse_args(argv: list[str] | None = None) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--url", default=os.environ.get("LIKI_FEEDBACK_URL", DEFAULT_URL))
    parser.add_argument("--status", choices=("pending", "processing", "done", "rejected"))
    parser.add_argument("--limit", type=int, default=50)
    parser.add_argument("--format", choices=("table", "json"), default="table")
    parser.add_argument(
        "--details",
        action="store_true",
        help="include expected/observed and issue URL in table output",
    )
    return parser.parse_args(argv)


def fetch_records(url: str, api_key: str, status: str | None, limit: int) -> list[dict[str, Any]]:
    query: list[str] = []
    if status:
        query.append(f"status={urllib.parse.quote(status)}")
    if limit > 0:
        query.append(f"limit={limit}")
    endpoint = url + ("?" + "&".join(query) if query else "")
    request = urllib.request.Request(
        endpoint,
        headers={"Authorization": f"Bearer {api_key}", "Accept": "application/json"},
        method="GET",
    )
    try:
        with urllib.request.urlopen(request, timeout=10) as response:
            body = json.load(response)
    except urllib.error.HTTPError as exc:
        raise RuntimeError(f"feedback API returned HTTP {exc.code}") from exc
    except (urllib.error.URLError, json.JSONDecodeError) as exc:
        raise RuntimeError(f"feedback API unavailable: {exc}") from exc

    records = body.get("data", []) if isinstance(body, dict) else body
    if not isinstance(records, list):
        raise RuntimeError("feedback API response must be an array")
    if any(not isinstance(record, dict) for record in records):
        raise RuntimeError("feedback API returned a non-object record")
    return records


def decode_record(record: dict[str, Any]) -> dict[str, Any]:
    try:
        payload = json.loads(record.get("payload") or "{}")
    except (TypeError, json.JSONDecodeError):
        payload = {}
    if not isinstance(payload, dict):
        payload = {}

    meta = text_object(payload.get("meta"))
    problem = text_object(payload.get("problem"))
    agent = text_object(payload.get("agent"))
    llm = text_object(payload.get("llm"))
    return {
        "id": str(record.get("id") or "-"),
        "status": str(record.get("status") or "-"),
        "created_at": str(record.get("created_at") or "-"),
        "type": str(problem.get("type") or "-"),
        "severity": str(problem.get("severity") or "-"),
        "summary": one_line(str(problem.get("summary") or "No problem summary")),
        "tool": str(problem.get("tool") or "-"),
        "expected": one_line(str(problem.get("expected") or "-")),
        "observed": one_line(str(problem.get("observed") or "-")),
        "source": str(meta.get("source") or "-"),
        "skill_version": str(meta.get("skill_version") or "-"),
        "engine_version": str(meta.get("engine_version") or "-"),
        "agent": f"{agent.get('name') or '-'} {agent.get('version') or '-'}".strip(),
        "llm": f"{llm.get('provider') or '-'} / {llm.get('model') or '-'}",
        "issue_url": str(record.get("issue_url") or "-"),
        "schema_version": str(payload.get("schema_version") or "-"),
    }


def text_object(value: Any) -> dict[str, Any]:
    return value if isinstance(value, dict) else {}


def one_line(value: str, limit: int = 120) -> str:
    compact = " ".join(value.split())
    if len(compact) <= limit:
        return compact
    return compact[: limit - 1] + "…"


def render_table(rows: list[dict[str, Any]], details: bool) -> str:
    headers = ["id", "status", "type", "severity", "summary", "skill", "engine"]
    if details:
        headers.extend(["tool", "expected", "observed", "issue"])
    values: list[list[str]] = []
    for row in rows:
        line = [
            row["id"],
            row["status"],
            row["type"],
            row["severity"],
            row["summary"],
            row["skill_version"],
            row["engine_version"],
        ]
        if details:
            line.extend([row["tool"], row["expected"], row["observed"], row["issue_url"]])
        values.append(line)

    widths = [len(header) for header in headers]
    for line in values:
        for index, value in enumerate(line):
            widths[index] = min(max(widths[index], len(value)), 80)

    def format_line(line: list[str]) -> str:
        return "  ".join(value[: widths[index]] for index, value in enumerate(line))

    output = [format_line(headers), "  ".join("-" * width for width in widths)]
    output.extend(format_line(line) for line in values)
    return "\n".join(output)


def main(argv: list[str] | None = None) -> int:
    args = parse_args(argv)
    api_key = os.environ.get("FEEDBACK_API_KEY", "").strip()
    if not api_key:
        print("FEEDBACK_API_KEY is required", file=sys.stderr)
        return 2
    if args.limit < 1:
        print("--limit must be at least 1", file=sys.stderr)
        return 2

    try:
        records = fetch_records(args.url, api_key, args.status, args.limit)
    except RuntimeError as exc:
        print(exc, file=sys.stderr)
        return 1

    rows = [decode_record(record) for record in records]
    if args.format == "json":
        print(json.dumps(rows, ensure_ascii=False, indent=2))
    else:
        print(render_table(rows, args.details))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
