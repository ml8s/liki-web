import importlib.util
import json
import unittest
from pathlib import Path


MODULE_PATH = Path(__file__).parents[1] / "feedback-read.py"
SPEC = importlib.util.spec_from_file_location("feedback_read", MODULE_PATH)
feedback_read = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
SPEC.loader.exec_module(feedback_read)


class FeedbackReadTest(unittest.TestCase):
    def test_decode_record_reads_problem_summary(self):
        payload = {
            "schema_version": "feedback-v1",
            "meta": {"source": "skill-agent", "skill_version": "2026.09.25", "engine_version": "2026.09.25"},
            "agent": {"name": "codex", "version": "1"},
            "llm": {"provider": "unknown", "model": "unknown"},
            "problem": {
                "type": "clarity",
                "severity": "medium",
                "summary": " resolved_period   解释入口不直观 ",
                "expected": " clearer entry ",
                "observed": " unclear ",
            },
        }
        record = {
            "id": "fb-1",
            "status": "pending",
            "created_at": "2026-09-25T00:00:00Z",
            "issue_url": "",
            "payload": json.dumps(payload, ensure_ascii=False),
        }

        row = feedback_read.decode_record(record)

        self.assertEqual(row["summary"], "resolved_period 解释入口不直观")
        self.assertEqual(row["source"], "skill-agent")
        self.assertEqual(row["type"], "clarity")
        self.assertEqual(row["severity"], "medium")

    def test_render_table_shows_summary_and_versions(self):
        row = {
            "id": "fb-1", "status": "pending", "type": "clarity", "severity": "medium",
            "summary": "summary", "tool": "-", "expected": "-", "observed": "-",
            "source": "skill-agent", "skill_version": "2026.09.25", "engine_version": "2026.09.25",
            "agent": "codex 1", "llm": "unknown / unknown", "issue_url": "-",
            "schema_version": "feedback-v1", "created_at": "2026-09-25T00:00:00Z",
        }
        rendered = feedback_read.render_table([row], details=False)
        self.assertIn("summary", rendered)
        self.assertIn("2026.09.25", rendered)


if __name__ == "__main__":
    unittest.main()
