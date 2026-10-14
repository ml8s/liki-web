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


def test_candidate_smokes_locally_before_remote_publication():
    script = Path(__file__).resolve().parents[1].joinpath("build-candidate.sh").read_text(encoding="utf-8")
    assert script.index("docker buildx build") < script.index("scripts/smoke-image.sh")
    assert "docker push" not in script
    assert "--push" not in script
    assert script.index("ensure_candidate_registry") < script.index("scripts/smoke-image.sh")
    assert "candidate-publish" not in script


class CandidateManifestTest(unittest.TestCase):
    def test_generates_exact_web_manifest(self):
        with tempfile.TemporaryDirectory() as tmp:
            metadata = Path(tmp) / "buildx.json"
            output = Path(tmp) / "candidate-digests.json"
            metadata.write_text(json.dumps({"containerimage.digest": "sha256:" + "a" * 64}))
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


if __name__ == "__main__":
    unittest.main()


class CandidateManifestValidatorTest(unittest.TestCase):
    def valid_manifest(self) -> dict:
        return {
            "registry": "127.0.0.1:5000",
            "repository": "ml8s/liki-web",
            "source_commit": "b" * 40,
            "candidate_tag": "candidate-ccccccc-20261013010203",
            "schema_version": "liki.candidate-manifest.v1",
            "images": {"web": "sha256:" + "a" * 64},
        }

    def test_accepts_exact_web_contract(self):
        from validate_candidate_manifest import validate_manifest

        self.assertEqual(
            validate_manifest(self.valid_manifest(), "b" * 40),
            {"web": "sha256:" + "a" * 64},
        )

    def test_rejects_contract_drift(self):
        from validate_candidate_manifest import validate_manifest

        with self.assertRaisesRegex(ValueError, "contract"):
            validate_manifest({**self.valid_manifest(), "schema_version": "legacy"}, "b" * 40)

    def test_cli_writes_action_outputs(self):
        import os
        import subprocess
        import tempfile

        output = Path(tempfile.gettempdir()) / "liki-web-validator-outputs"
        try:
            subprocess.run(
                [
                    "python3",
                    str(SCRIPTS / "validate_candidate_manifest.py"),
                    "--source-commit",
                    "b" * 40,
                    "--output",
                    str(output),
                ],
                env={**os.environ, "DIGESTS_JSON": json.dumps(self.valid_manifest())},
                check=True,
            )
            self.assertEqual(
                output.read_text(encoding="utf-8"),
                f"web_digest=sha256:{'a' * 64}\n",
            )
        finally:
            output.unlink(missing_ok=True)


def test_publish_candidate_requires_semantic_evidence_contract():
    script = Path(__file__).resolve().parents[1].joinpath("publish-candidate.sh").read_text(encoding="utf-8")
    assert 'schema_version") != "liki.assembly-evidence.v1"' in script


def test_promote_workflow_uses_tested_validator():
    workflow = Path(__file__).resolve().parents[2].joinpath(".github/workflows/promote-release.yml").read_text(encoding="utf-8")
    assert "scripts/validate_candidate_manifest.py" in workflow
    assert '--source-commit "$actual_commit"' in workflow
