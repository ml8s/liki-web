import json
import tempfile
import unittest
from pathlib import Path
import sys

SCRIPTS = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(SCRIPTS))

from build_candidate_manifest import generate_manifest  # noqa: E402


def test_candidate_build_fails_fast_before_expensive_gate():
    script = Path(__file__).resolve().parents[1].joinpath("build-candidate.sh").read_text(encoding="utf-8")
    assert script.index("RELEASE_VERSION=") < script.index("make gate")
    assert script.index("cmd/liki-web/VERSION") < script.index("make gate")


class CandidateManifestTest(unittest.TestCase):
    def test_generates_exact_web_manifest(self):
        with tempfile.TemporaryDirectory() as tmp:
            metadata = Path(tmp) / "buildx.json"
            output = Path(tmp) / "candidate-digests.json"
            metadata.write_text(json.dumps({"containerimage.digest": "sha256:" + "a" * 64}))
            actual = generate_manifest("b" * 40, "candidate-ccccccc-20261013010203", "sha256:" + "a" * 64)
            self.assertEqual(actual, {
                "repository": "ml8s/liki-web",
                "source_commit": "b" * 40,
                "candidate_tag": "candidate-ccccccc-20261013010203",
                "images": {"web": "sha256:" + "a" * 64},
            })

    def test_rejects_invalid_tag(self):
        with self.assertRaisesRegex(ValueError, "candidate_tag"):
            generate_manifest("b" * 40, "latest", "sha256:" + "a" * 64)


if __name__ == "__main__":
    unittest.main()
