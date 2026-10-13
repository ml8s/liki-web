from __future__ import annotations

import json
from pathlib import Path

import pytest


ROOT = Path(__file__).resolve().parents[2]
MAKEFILE = (ROOT / "Makefile").read_text(encoding="utf-8")
APP_PACKAGE = json.loads((ROOT / "web/app/package.json").read_text(encoding="utf-8"))
VITE_CONFIG = (ROOT / "web/app/vite.config.ts").read_text(encoding="utf-8")


def test_go_coverage_target_matches_ci_policy() -> None:
    assert "test-go-coverage:" in MAKEFILE
    assert "go test -race -count=1" in MAKEFILE
    assert "-coverprofile=coverage.out" in MAKEFILE
    assert "-coverpkg=./internal/..." in MAKEFILE
    assert "go tool cover -func=coverage.out" in MAKEFILE
    assert 'Go coverage $$COV% < 65%' in MAKEFILE


def test_frontend_coverage_target_is_standardized() -> None:
    assert "test-js-coverage:" in MAKEFILE
    assert APP_PACKAGE["scripts"]["test:coverage"] == "vitest run --coverage"


def test_vitest_uses_v8_provider_and_reports() -> None:
    assert '"@vitest/coverage-v8"' in (ROOT / "web/app/package.json").read_text(encoding="utf-8")
    assert "provider: 'v8'" in VITE_CONFIG
    assert "reportsDirectory" in VITE_CONFIG
    assert "text-summary" in VITE_CONFIG
    assert "html" in VITE_CONFIG
    assert "statements: 29" in VITE_CONFIG
    assert "branches: 25" in VITE_CONFIG
    assert "functions: 18" in VITE_CONFIG
    assert "lines: 30" in VITE_CONFIG


def test_frontend_domain_sources_are_included() -> None:
    for path in ("src/api/**", "src/stores/**", "src/components/**", "src/utils/**"):
        assert path in VITE_CONFIG
