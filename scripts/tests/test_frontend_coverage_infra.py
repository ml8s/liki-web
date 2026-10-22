from __future__ import annotations

import json
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[2]
MAKEFILE = (ROOT / "Makefile").read_text(encoding="utf-8")
APP_PACKAGE = json.loads((ROOT / "web/app/package.json").read_text(encoding="utf-8"))
VITE_CONFIG = (ROOT / "web/app/vite.config.ts").read_text(encoding="utf-8")


class FrontendCoverageInfraTest(unittest.TestCase):
    def test_go_coverage_target_matches_ci_policy(self) -> None:
        self.assertIn("test-go-coverage:", MAKEFILE)
        # race 自 acccc92 起拆为独立并行目标（语义保留）。
        self.assertIn("test-go-race:", MAKEFILE)
        self.assertIn("go test -race -count=1 ./...", MAKEFILE)
        self.assertIn("-coverprofile=coverage.out", MAKEFILE)
        self.assertIn("-coverpkg=./internal/...", MAKEFILE)
        self.assertIn("go tool cover -func=coverage.out", MAKEFILE)
        # 覆盖率地板 40（acccc92：65→40，门禁只防归零）：判定值与提示必须一致。
        self.assertIn('-v m="40"', MAKEFILE)
        self.assertIn("Go coverage $$COV% < 40%", MAKEFILE)

    def test_frontend_coverage_target_is_standardized(self) -> None:
        self.assertIn("test-js-coverage:", MAKEFILE)
        self.assertEqual(APP_PACKAGE["scripts"]["test:coverage"], "vitest run --coverage")

    def test_vitest_uses_v8_provider_and_reports(self) -> None:
        package_lock = (ROOT / "web/app/package.json").read_text(encoding="utf-8")
        self.assertIn('"@vitest/coverage-v8"', package_lock)
        self.assertIn("provider: 'v8'", VITE_CONFIG)
        self.assertIn("reportsDirectory", VITE_CONFIG)
        self.assertIn("text-summary", VITE_CONFIG)
        self.assertIn("html", VITE_CONFIG)
        self.assertIn("statements: 29", VITE_CONFIG)
        self.assertIn("branches: 25", VITE_CONFIG)
        self.assertIn("functions: 18", VITE_CONFIG)
        self.assertIn("lines: 30", VITE_CONFIG)

    def test_frontend_domain_sources_are_included(self) -> None:
        for path in ("src/api/**", "src/stores/**", "src/components/**", "src/utils/**"):
            self.assertIn(path, VITE_CONFIG)


if __name__ == "__main__":
    unittest.main()
