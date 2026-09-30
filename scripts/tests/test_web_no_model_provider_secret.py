import unittest
from pathlib import Path


PROJECT = Path(__file__).parents[2]
SCAN_ROOTS = (
    PROJECT / "cmd",
    PROJECT / "internal",
    PROJECT / "scripts",
    PROJECT / ".github",
    PROJECT / ".env.example",
)
FORBIDDEN = (
    "LIKI_LLM_API_KEY",
    "LIKI_LLM_MODEL",
    "LIKI_LLM_BASE_URL",
    "LIKI_LLM_THINKING",
    "LLMAPIKey",
    "LLMBaseURL",
    "LLMModel",
    "LLMThinking",
)


class WebModelSecretBoundaryTest(unittest.TestCase):
    def test_web_runtime_does_not_own_model_provider_credentials(self):
        hits = []
        for root in SCAN_ROOTS:
            if root.is_file():
                paths = [root]
            else:
                paths = root.rglob("*")
            for path in paths:
                if path == Path(__file__):
                    continue
                if not path.is_file() or path.suffix not in {
                    ".go", ".yml", ".yaml", ".sh", ".py", ".example"
                }:
                    continue
                text = path.read_text(encoding="utf-8", errors="ignore")
                for forbidden in FORBIDDEN:
                    if forbidden in text:
                        hits.append(f"{path.relative_to(PROJECT)}: {forbidden}")
        self.assertEqual(hits, [], "liki-web must not own model provider credentials")


if __name__ == "__main__":
    unittest.main()
