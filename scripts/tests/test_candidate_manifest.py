import json
import unittest
from pathlib import Path
import sys

SCRIPTS = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(SCRIPTS))

from build_candidate_manifest import generate_manifest  # noqa: E402


class CandidateManifestTest(unittest.TestCase):
    def test_generates_exact_web_manifest(self):
        actual = generate_manifest(
            "b" * 40, "candidate-ccccccc-20261013010203", "sha256:" + "a" * 64,
            "127.0.0.1:5000",
        )
        self.assertEqual(actual, {
            "registry": "127.0.0.1:5000",
            "repository": "ml8s/liki-web",
            "source_commit": "b" * 40,
            "candidate_tag": "candidate-ccccccc-20261013010203",
            "schema_version": "liki.candidate-manifest.v1",
            "images": {"web": "sha256:" + "a" * 64},
        })

    def test_rejects_invalid_tag(self):
        with self.assertRaisesRegex(ValueError, "candidate_tag"):
            generate_manifest("b" * 40, "latest", "sha256:" + "a" * 64, "127.0.0.1:5000")

    def test_accepts_release_tag(self):
        actual = generate_manifest(
            "b" * 40, "v2026.10.16.0", "sha256:" + "a" * 64, "ghcr.io",
        )
        self.assertEqual(actual["candidate_tag"], "v2026.10.16.0")
        self.assertEqual(actual["registry"], "ghcr.io")


if __name__ == "__main__":
    unittest.main()
