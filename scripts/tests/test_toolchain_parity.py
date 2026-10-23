"""Toolchain pins must be single-sourced: local `make lint-go` and CI run the
same golangci-lint. 一个 pin 两处版本 = 本地过、CI 挂（或反过来）。
对账三方：deps/golangci-lint.version、ci.yml、本地 lint 脚本。
"""
from __future__ import annotations

import re
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
PIN_FILE = ROOT / "deps" / "golangci-lint.version"


def _pin() -> str:
    match = re.search(r"^version:\s*(\S+)\s*$", PIN_FILE.read_text(encoding="utf-8"), re.M)
    assert match, "deps/golangci-lint.version must carry a `version:` line"
    return match.group(1)


def _action_versions(workflow: Path) -> set[str]:
    # 正则解析而非 import yaml：CI 的 test-python 环境没有 PyYAML，
    # 测试自己引入本地/CI 依赖差异，正是本测试要消灭的那类漂移。
    text = workflow.read_text(encoding="utf-8")
    return set(
        re.findall(
            r"uses: golangci/golangci-lint-action@\S+[^\n]*\n\s+with:\n\s+version:\s*(\S+)",
            text,
        )
    )


class ToolchainParityTest(unittest.TestCase):
    def test_pin_format(self):
        self.assertRegex(_pin(), r"^v\d+\.\d+\.\d+$")

    def test_ci_matches_pin(self):
        versions = _action_versions(ROOT / ".github" / "workflows" / "ci.yml")
        self.assertTrue(versions, "CI must run golangci-lint-action")
        self.assertEqual(versions, {_pin()})

    def test_lint_go_uses_the_pin_script(self):
        makefile = (ROOT / "Makefile").read_text(encoding="utf-8")
        match = re.search(r"^lint-go:.*\n((?:\t.*\n)+)", makefile, re.M)
        self.assertTrue(match, "lint-go target missing")
        self.assertIn("scripts/lint-go.sh", match.group(1))
        self.assertNotIn("golangci-lint run", match.group(1))

    def test_lint_go_script_reads_the_pin(self):
        script = (ROOT / "scripts" / "lint-go.sh").read_text(encoding="utf-8")
        self.assertIn("deps/golangci-lint.version", script)


if __name__ == "__main__":
    unittest.main()
