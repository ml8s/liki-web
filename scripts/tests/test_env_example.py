import re
import unittest
from pathlib import Path


PROJECT = Path(__file__).parents[2]
GO_ROOTS = (PROJECT / "cmd", PROJECT / "internal")
ENV_EXAMPLE = PROJECT / ".env.example"


class EnvExampleTest(unittest.TestCase):
    def test_every_runtime_env_var_has_a_template_entry(self):
        source_vars = set()
        env_call = re.compile(
            r"\b(?:os\.Getenv|envOr(?:Bool|Int)?)\(\s*\"([A-Z][A-Z0-9_]+)\""
        )
        for root in GO_ROOTS:
            for path in root.rglob("*.go"):
                if path.name.endswith("_test.go"):
                    continue
                source_vars.update(env_call.findall(path.read_text(encoding="utf-8")))

        template_vars = set(
            re.findall(r"^([A-Z][A-Z0-9_]+)=", ENV_EXAMPLE.read_text(encoding="utf-8"), re.MULTILINE)
        )
        self.assertFalse(
            source_vars - template_vars,
            "runtime environment variables missing from .env.example",
        )

    def test_security_sensitive_template_values_stay_empty(self):
        template = ENV_EXAMPLE.read_text(encoding="utf-8")
        for name in (
            "JWT_SECRET", "REFERRAL_PAYOUT_SECRET", "RESEND_API_KEY",
            "DODO_API_KEY", "DODO_WEBHOOK_KEY", "XUNHU_APPSECRET",
            "ALIPAY_PRIVATE_KEY", "LIKI_AGENTS_INTERNAL_TOKEN",
            "FEEDBACK_API_KEY", "METRICS_TOKEN",
        ):
            match = re.search(rf"^{name}=(.*)$", template, re.MULTILINE)
            self.assertIsNotNone(match, f".env.example missing {name}")
            self.assertEqual(match.group(1).strip(), "", f"{name} must not contain a real value")


if __name__ == "__main__":
    unittest.main()
