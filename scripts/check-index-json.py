#!/usr/bin/env python3
"""校验 V2 skills well-known index.json。

字段 ($schema = https://schemas.agentskills.io/discovery/0.2.0/schema.json):
  - name:      字母+数字+连字符, 1-64 字符
  - description: 非空, ≤1024 字符
  - type:      "skill-md" 或 "archive"
  - url:       非空字符串
  - digest:    sha256:<64 hex chars>

用法: python3 scripts/check-index-json.py [index.json ...]
"""

import json
import re
import sys

DISCOVERY_SCHEMA_V2 = "https://schemas.agentskills.io/discovery/0.2.0/schema.json"
DIGEST_RE = re.compile(r"^sha256:[a-f0-9]{64}$")


def check_skill_name(name: str) -> bool:
    if not isinstance(name, str) or len(name) < 1 or len(name) > 64:
        return False
    if not name.replace("-", "").isalnum():
        return False
    if name.startswith("-") or name.endswith("-") or "--" in name:
        return False
    return True


def check_v2(skills: list, path: str) -> int:
    errors = 0
    for s in skills:
        name = s.get("name", "")
        if not check_skill_name(name):
            print(f"FAIL {path}: name 无效 ({name!r})")
            errors += 1
            continue

        desc = s.get("description", "")
        if not isinstance(desc, str) or not desc or len(desc) > 1024:
            print(f"FAIL {path}: {name} description 无效")
            errors += 1
            continue

        typ = s.get("type")
        if typ not in ("skill-md", "archive"):
            print(f"FAIL {path}: {name} type 必须为 skill-md 或 archive (当前: {typ!r})")
            errors += 1
            continue

        url = s.get("url", "")
        if not isinstance(url, str) or not url:
            print(f"FAIL {path}: {name} 缺少 url")
            errors += 1
            continue

        digest = s.get("digest", "")
        if not isinstance(digest, str) or not DIGEST_RE.match(digest):
            print(f"FAIL {path}: {name} digest 格式错误 (当前: {digest!r}, 期望: sha256:<64 hex>)")
            errors += 1
            continue
    return errors


def check_index(path: str) -> int:
    with open(path) as f:
        data = json.load(f)

    skills = data.get("skills", [])
    if not isinstance(skills, list) or len(skills) == 0:
        print(f"FAIL {path}: skills 为空或非数组")
        return 1

    schema = data.get("$schema", "")
    if schema != DISCOVERY_SCHEMA_V2:
        print(f"FAIL {path}: 缺少或未知 $schema (期望: {DISCOVERY_SCHEMA_V2})")
        return 1

    errors = check_v2(skills, path)
    if errors == 0:
        print(f"OK   {path} V2 ({len(skills)} skills)")
    return errors


def main() -> int:
    paths = sys.argv[1:] if len(sys.argv) > 1 else [
        "web/.well-known/agent-skills/index.json",
    ]
    total = 0
    for p in paths:
        total += check_index(p)
    if total:
        print(f"\n{total} error(s)", file=sys.stderr)
    return min(total, 1)


if __name__ == "__main__":
    sys.exit(main())
