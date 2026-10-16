#!/usr/bin/env python3
"""Validate and unpack the liki-web release candidate manifest."""
from __future__ import annotations

import argparse
import json
import os
import re
import sys
from pathlib import Path
from typing import Any

TAG_RE = re.compile(r"^(?:candidate-[0-9a-f]{7}-\d{14}|v[0-9]{4}\.\d{2}\.\d{2}\.\d+)$")
DIGEST_RE = re.compile(r"^sha256:[0-9a-f]{64}$")
COMMIT_RE = re.compile(r"^[0-9a-f]{40}$")


def validate_manifest(document: Any, source_commit: str | None = None) -> dict[str, str]:
    """Return the exact web digest when document matches the release contract."""
    if not isinstance(document, dict):
        raise ValueError("manifest must be a JSON object")
    required = {
        "registry", "repository", "source_commit", "candidate_tag",
        "schema_version", "images",
    }
    if set(document) != required:
        missing = sorted(required - set(document))
        unexpected = sorted(set(document) - required)
        raise ValueError(f"invalid fields: missing={missing}, unexpected={unexpected}")
    if document["repository"] != "ml8s/liki-web":
        raise ValueError("repository must be ml8s/liki-web")
    if not isinstance(document["source_commit"], str) or not COMMIT_RE.fullmatch(document["source_commit"]):
        raise ValueError("source_commit must be a lowercase 40-character Git SHA")
    if source_commit is not None and document["source_commit"] != source_commit:
        raise ValueError("source_commit does not match the release tag")
    if document["schema_version"] != "liki.candidate-manifest.v1":
        raise ValueError("schema_version contract must be liki.candidate-manifest.v1")
    if not isinstance(document["registry"], str) or not document["registry"]:
        raise ValueError("registry must be non-empty")
    if not isinstance(document["candidate_tag"], str) or not TAG_RE.fullmatch(document["candidate_tag"]):
        raise ValueError("candidate_tag is invalid")
    images = document["images"]
    if not isinstance(images, dict) or set(images) != {"web"}:
        raise ValueError("images must contain exactly web")
    if not isinstance(images["web"], str) or not DIGEST_RE.fullmatch(images["web"]):
        raise ValueError("web digest must be sha256 plus 64 lowercase hex characters")
    return {"web": images["web"]}


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--source-commit")
    parser.add_argument("--output", type=Path, help="append Actions outputs here")
    args = parser.parse_args()
    try:
        document = json.loads(os.environ.get("DIGESTS_JSON", ""))
        images = validate_manifest(document, args.source_commit)
    except (json.JSONDecodeError, TypeError, ValueError) as error:
        raise SystemExit(f"invalid candidate manifest: {error}") from error
    lines = [f"{name}_digest={digest}\n" for name, digest in sorted(images.items())]
    if args.output:
        args.output.parent.mkdir(parents=True, exist_ok=True)
        with args.output.open("a", encoding="utf-8") as handle:
            handle.writelines(lines)
    else:
        sys.stdout.writelines(lines)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
