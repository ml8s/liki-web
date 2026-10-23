import re
import unittest
from pathlib import Path


PROJECT = Path(__file__).parents[2]
WORKFLOW = PROJECT / ".github" / "workflows" / "ci.yml"


class CIWorkflowTest(unittest.TestCase):
    def test_docker_e2e_uses_absolute_runtime_webhook_name(self):
        docker_test = (PROJECT / "scripts" / "docker-test.sh").read_text(encoding="utf-8")
        self.assertIn(
            "LIKI_PAYMENT_WEBHOOK_URL=http://localhost:8080/api/payments/webhook",
            docker_test,
        )
        self.assertNotRegex(docker_test, r"^PAYMENTS_WEBHOOK_URL=", re.MULTILINE)

    def test_docker_e2e_declares_complete_business_contract(self):
        docker_test = (PROJECT / "scripts" / "docker-test.sh").read_text(encoding="utf-8")
        required = {
            "LIKI_CHAT_ONE_TIME_PRICE_CNY_MINOR_UNITS": "7900",
            "LIKI_CHAT_ONE_TIME_PRICE_USD_MINOR_UNITS": "2990",
            "LIKI_CHAT_SUBSCRIPTION_PRICE_USD_MINOR_UNITS": "1990",
            "LIKI_CHAT_ONE_TIME_ACCESS_DAYS": "30",
            "LIKI_CHAT_KOL_REVENUE_SHARE_BASIS_POINTS": "4000",
            "LIKI_CHAT_SCOUT_REVENUE_SHARE_BASIS_POINTS": "1000",
            "LIKI_REFERRAL_PAYOUT_ENCRYPTION_KEY": "e2e-referral-payout-encryption-key-32-bytes",
        }
        for name, value in required.items():
            self.assertIn(f"{name}={value}", docker_test)

    def test_image_smoke_declares_complete_business_contract(self):
        smoke_image = (PROJECT / "scripts" / "smoke-image.sh").read_text(encoding="utf-8")
        required = {
            "LIKI_PAYMENT_WEBHOOK_URL": "http://127.0.0.1:$PORT/api/payments/webhook",
            "LIKI_PUBLIC_BASE_URL": "http://127.0.0.1:$PORT",
            "LIKI_REFERRAL_PAYOUT_ENCRYPTION_KEY": "web-image-smoke-referral-payout-encryption-key-32-bytes",
            "LIKI_CHAT_ONE_TIME_PRICE_CNY_MINOR_UNITS": "7900",
            "LIKI_CHAT_ONE_TIME_PRICE_USD_MINOR_UNITS": "2990",
            "LIKI_CHAT_SUBSCRIPTION_PRICE_USD_MINOR_UNITS": "1990",
            "LIKI_CHAT_ONE_TIME_ACCESS_DAYS": "30",
            "LIKI_CHAT_KOL_REVENUE_SHARE_BASIS_POINTS": "4000",
            "LIKI_CHAT_SCOUT_REVENUE_SHARE_BASIS_POINTS": "1000",
        }
        for name, value in required.items():
            self.assertIn(f"{name}={value}", smoke_image)
        self.assertNotRegex(smoke_image, r"^\s*-e PAYMENTS_WEBHOOK_URL=", re.MULTILINE)

    def test_local_runtimes_use_listen_address_not_addr(self):
        for contract_file in ("dev/docker-compose.yml", "scripts/dev-start.sh", "scripts/smoke-image.sh"):
            self.assertNotRegex(
                (PROJECT / contract_file).read_text(encoding="utf-8"),
                r"LIKI_HTTP_LISTEN_ADDR(?!RESS)\s*=",
                f"{contract_file} must use LIKI_HTTP_LISTEN_ADDRESS",
            )

    def test_docker_entrypoint_does_not_add_removed_addr_flag(self):
        dockerfile = (PROJECT / "Dockerfile").read_text(encoding="utf-8")
        self.assertIn('ENTRYPOINT ["/usr/local/bin/liki-web"]', dockerfile)
        self.assertNotIn("--addr", dockerfile)


    def test_ci_runs_standardized_coverage_targets_and_uploads_frontend_report(self):
        text = WORKFLOW.read_text(encoding="utf-8")
        self.assertIn("make test-go-coverage", text)
        self.assertIn("make test-js-coverage", text)
        self.assertIn("name: frontend-coverage", text)
        self.assertIn("web/app/coverage", text)


if __name__ == "__main__":
    unittest.main()
