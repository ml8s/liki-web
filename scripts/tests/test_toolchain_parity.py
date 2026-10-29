"""Toolchain pins must be single-sourced: local `make check` and CI run the
same golangci-lint script. 一个 pin 两处版本 = 本地过、CI 挂（或反过来）。
对账三方：deps/golangci-lint.version、ci.yml、本地 lint 脚本。
"""
from __future__ import annotations

import re
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
PIN_FILE = ROOT / "deps" / "golangci-lint.version"
WORKFLOW = ROOT / ".github" / "workflows" / "ci.yml"
MAKEFILE = ROOT / "Makefile"


def _pin() -> str:
    match = re.search(r"^version:\s*(\S+)\s*$", PIN_FILE.read_text(encoding="utf-8"), re.M)
    assert match, "deps/golangci-lint.version must carry a `version:` line"
    return match.group(1)


class ToolchainParityTest(unittest.TestCase):
    def test_pin_format(self):
        self.assertRegex(_pin(), r"^v\d+\.\d+\.\d+$")

    def test_ci_lints_through_the_local_gate(self):
        text = WORKFLOW.read_text(encoding="utf-8")
        self.assertNotIn(
            "golangci/golangci-lint-action",
            text,
            "CI must lint through scripts/lint-go.sh, not a separate action",
        )
        self.assertIn("make check", text, "CI must run the local static gate")
        self.assertIn(
            "deps/golangci-lint.version",
            text,
            "CI must provision golangci-lint from the single pin file",
        )

    def test_lint_go_uses_the_pin_script(self):
        makefile = MAKEFILE.read_text(encoding="utf-8")
        match = re.search(r"^lint-go:.*\n((?:\t.*\n)+)", makefile, re.M)
        self.assertTrue(match, "lint-go target missing")
        self.assertIn("scripts/lint-go.sh", match.group(1))
        self.assertNotIn("golangci-lint run", match.group(1))

    def test_lint_go_script_reads_the_pin(self):
        script = (ROOT / "scripts" / "lint-go.sh").read_text(encoding="utf-8")
        self.assertIn("deps/golangci-lint.version", script)


if __name__ == "__main__":
    unittest.main()
