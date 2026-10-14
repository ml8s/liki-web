#!/usr/bin/env python3
"""Generate the release record for a pushed liki-web candidate image."""
import argparse
import json
import re
from pathlib import Path

TAG_RE = re.compile(r"^candidate-[0-9a-f]{7}-\d{14}$")
DIGEST_RE = re.compile(r"^sha256:[0-9a-f]{64}$")
COMMIT_RE = re.compile(r"^[0-9a-f]{40}$")


def generate_manifest(commit: str, candidate_tag: str, image_digest: str, registry: str) -> dict:
    if not COMMIT_RE.fullmatch(commit):
        raise ValueError("source_commit must be a full 40-character Git SHA")
    if not TAG_RE.fullmatch(candidate_tag):
        raise ValueError("candidate_tag must be candidate-<7-char-sha>-<YYYYmmddHHMMSS>")
    if not DIGEST_RE.fullmatch(image_digest):
        raise ValueError("image digest must be a sha256 digest")
    if not registry or registry.startswith(("http://", "https://")):
        raise ValueError("registry must be a registry host without a scheme")
    return {
        "registry": registry,
        "repository": "ml8s/liki-web",
        "source_commit": commit,
        "candidate_tag": candidate_tag,
        "schema_version": "liki.candidate-manifest.v1",
        "images": {"web": image_digest},
    }


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--output", required=True, type=Path)
    parser.add_argument("--source-commit", required=True)
    parser.add_argument("--candidate-tag", required=True)
    parser.add_argument("--registry", required=True)
    parser.add_argument("--metadata-file", required=True, type=Path)
    args = parser.parse_args()

    metadata = json.loads(args.metadata_file.read_text(encoding="utf-8"))
    image_digest = metadata.get("containerimage.digest") if isinstance(metadata, dict) else None
    if not isinstance(image_digest, str):
        raise ValueError("metadata-file is missing containerimage.digest")

    manifest = generate_manifest(args.source_commit, args.candidate_tag, image_digest, args.registry)
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(manifest, indent=2) + "\n", encoding="utf-8")


if __name__ == "__main__":
    try:
        main()
    except (OSError, ValueError, json.JSONDecodeError) as exc:
        raise SystemExit(f"candidate manifest: {exc}") from exc
