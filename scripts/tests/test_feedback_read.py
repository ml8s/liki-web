import importlib.util
import json
import unittest
from pathlib import Path
from unittest import mock


MODULE_PATH = Path(__file__).parents[1] / "feedback-read.py"
SPEC = importlib.util.spec_from_file_location("feedback_read", MODULE_PATH)
feedback_read = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
SPEC.loader.exec_module(feedback_read)


class FeedbackReadTest(unittest.TestCase):
    def test_fetch_records_single_page(self):
        page = [{"id": "fb-1"}, {"id": "fb-2"}]
        with mock.patch.object(feedback_read, "_http_get_json", return_value=page) as get:
            records = feedback_read.fetch_records("https://x/api/feedback", "k", None, 50)
        self.assertEqual([r["id"] for r in records], ["fb-1", "fb-2"])
        self.assertIn("limit=50", get.call_args[0][0])
        self.assertIn("offset=0", get.call_args[0][0])

    def test_fetch_records_all_pages_traverses_offset(self):
        page1 = [{"id": f"fb-{i}"} for i in range(50)]
        page2 = [{"id": "fb-50"}, {"id": "fb-51"}]
        with mock.patch.object(
            feedback_read, "_http_get_json", side_effect=[page1, page2]
        ) as get:
            records = feedback_read.fetch_records(
                "https://x/api/feedback", "k", "done", 50, fetch_all=True
            )
        self.assertEqual(len(records), 52)
        second_call = get.call_args_list[1][0][0]
        self.assertIn("offset=50", second_call)
        self.assertIn("status=done", second_call)

    def test_fetch_records_rejects_non_array(self):
        with mock.patch.object(feedback_read, "_http_get_json", return_value="oops"):
            with self.assertRaises(RuntimeError):
                feedback_read.fetch_records("https://x/api/feedback", "k", None, 50)

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
