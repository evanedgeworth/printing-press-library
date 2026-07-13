from __future__ import annotations

import importlib.util
import unittest
from pathlib import Path


MODULE_PATH = Path(__file__).with_name("verify_release_scanners.py")
SPEC = importlib.util.spec_from_file_location("verify_release_scanners", MODULE_PATH)
assert SPEC and SPEC.loader
MODULE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(MODULE)


class ReleaseScannerContractTest(unittest.TestCase):
    def setUp(self) -> None:
        self.head = "a" * 40
        self.config = {
            "required_check_runs": ["Verify", "Govulncheck"],
            "required_commit_statuses": ["CodeRabbit"],
            "comment_scanners": [
                {
                    "name": "CodeRabbit",
                    "authors": ["coderabbitai[bot]"],
                    "success_patterns": ["walkthrough"],
                    "failure_patterns": ["rate limited", "review failed", "review limit reached", "review skipped", "draft detected", "all tool run failures"],
                }
            ],
        }
        self.snapshot = {
            "repository": "owner/repo",
            "pull_request": 4,
            "source_commit": "b" * 40,
            "generated_commit": "d" * 40,
            "release_version": "2026.7.3",
            "head_sha": self.head,
            "check_runs": [
                {"id": 1, "name": "Verify", "head_sha": self.head, "status": "completed", "conclusion": "success"},
                {"id": 2, "name": "Govulncheck", "head_sha": self.head, "status": "completed", "conclusion": "success"},
            ],
            "statuses": [{"id": 3, "context": "CodeRabbit", "state": "success"}],
            "comments": [
                {
                    "id": 4,
                    "user": {"login": "coderabbitai[bot]"},
                    "body": "## Walkthrough\nReview completed.",
                }
            ],
        }

    def verify(self):
        return MODULE.verify(self.config, self.snapshot, self.head)

    def test_accepts_only_complete_success_receipt(self) -> None:
        receipt = self.verify()
        self.assertTrue(receipt["compliant"])
        self.assertEqual(receipt["generated_commit"], "d" * 40)
        self.assertEqual(receipt["release_version"], "2026.7.3")

    def test_rejects_absent_check(self) -> None:
        self.snapshot["check_runs"] = self.snapshot["check_runs"][:1]
        self.assertFalse(self.verify()["compliant"])

    def test_rejects_absent_release_provenance(self) -> None:
        for field in ("source_commit", "generated_commit", "release_version"):
            with self.subTest(field=field):
                original = self.snapshot.pop(field)
                self.assertFalse(self.verify()["compliant"])
                self.snapshot[field] = original

    def test_rejects_cancelled_failed_and_incomplete_checks(self) -> None:
        for status, conclusion in (
            ("completed", "cancelled"),
            ("completed", "failure"),
            ("completed", "timed_out"),
            ("in_progress", None),
        ):
            with self.subTest(status=status, conclusion=conclusion):
                self.snapshot["check_runs"][0].update(status=status, conclusion=conclusion)
                self.assertFalse(self.verify()["compliant"])
        self.snapshot["check_runs"][0].update(status="completed", conclusion="success")

    def test_rejects_success_status_with_rate_limited_comment(self) -> None:
        self.snapshot["comments"][0]["body"] = "## Review limit reached\nRate limited. Walkthrough unavailable."
        receipt = self.verify()
        self.assertFalse(receipt["compliant"])
        self.assertTrue(receipt["comment_scanners"][0]["matched_failure_patterns"])

    def test_rejects_success_status_with_failed_comment(self) -> None:
        self.snapshot["comments"][0]["body"] = "## Review failed\n## Walkthrough"
        self.assertFalse(self.verify()["compliant"])

    def test_rejects_success_status_with_skipped_comment(self) -> None:
        self.snapshot["comments"][0]["body"] = "## Review skipped\nDraft detected."
        self.assertFalse(self.verify()["compliant"])

    def test_rejects_success_status_with_tool_failures(self) -> None:
        self.snapshot["comments"][0]["body"] = "## Walkthrough\n<!-- all tool run failures -->"
        self.assertFalse(self.verify()["compliant"])

    def test_rejects_wrong_head(self) -> None:
        self.assertFalse(MODULE.verify(self.config, self.snapshot, "c" * 40)["compliant"])


if __name__ == "__main__":
    unittest.main()
