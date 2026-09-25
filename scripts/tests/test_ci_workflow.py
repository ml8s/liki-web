import re
import unittest
from pathlib import Path


PROJECT = Path(__file__).parents[2]
WORKFLOW = PROJECT / ".github" / "workflows" / "ci.yml"


class CIWorkflowTest(unittest.TestCase):
    def test_e2e_uses_required_runtime_environment(self):
        text = WORKFLOW.read_text(encoding="utf-8")
        match = re.search(
            r"- name: Start servers\n(?P<indent>[ \t]+)env:\n(?P<body>(?:(?P=indent)[ \t]+.*\n)+)",
            text,
        )
        self.assertIsNotNone(match, "CI e2e must declare its server runtime environment")
        env = match.group("body")

        required = {
            "JWT_SECRET",
            "REFERRAL_PAYOUT_SECRET",
            "LISTEN_ADDR",
            "RETURN_URL",
            "LIKI_LLM_API_KEY",
            "LIKI_LLM_MODEL",
            "XUNHU_APPID",
            "XUNHU_APPSECRET",
        }
        for name in required:
            self.assertIsNotNone(
                re.search(rf"^[ \t]*{name}:\s*\S+", env, re.MULTILINE),
                f"missing CI e2e env {name}",
            )

        self.assertIn("LISTEN_ADDR: :8081", env)
        self.assertNotIn("-addr :8081", text, "liki-web listens through LISTEN_ADDR, not a CLI flag")

    def test_docker_entrypoint_does_not_add_removed_addr_flag(self):
        dockerfile = (PROJECT / "Dockerfile").read_text(encoding="utf-8")
        self.assertIn('ENTRYPOINT ["/usr/local/bin/liki-web"]', dockerfile)
        self.assertNotIn("--addr", dockerfile)


if __name__ == "__main__":
    unittest.main()
