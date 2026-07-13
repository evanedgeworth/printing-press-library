#!/usr/bin/env python3
"""Fail closed unless an exact PR head has a complete scanner receipt."""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path
from typing import Any


def _latest(items: list[dict[str, Any]], timestamp_keys: tuple[str, ...]) -> dict[str, Any]:
    return max(
        items,
        key=lambda item: tuple(str(item.get(key) or "") for key in timestamp_keys),
    )


def _is_commit_sha(value: Any) -> bool:
    candidate = str(value or "")
    return len(candidate) == 40 and all(character in "0123456789abcdef" for character in candidate.casefold())


def verify(config: dict[str, Any], snapshot: dict[str, Any], expected_head: str) -> dict[str, Any]:
    errors: list[str] = []
    evidence: dict[str, Any] = {
        "schema_version": 1,
        "repository": snapshot.get("repository"),
        "pull_request": snapshot.get("pull_request"),
        "source_commit": snapshot.get("source_commit"),
        "generated_commit": snapshot.get("generated_commit"),
        "release_version": snapshot.get("release_version"),
        "head_sha": snapshot.get("head_sha"),
        "checks": [],
        "statuses": [],
        "comment_scanners": [],
    }

    head_sha = str(snapshot.get("head_sha") or "")
    if not expected_head or head_sha != expected_head:
        errors.append(f"snapshot head {head_sha or '<absent>'} does not match expected head {expected_head or '<absent>'}")

    for field in ("source_commit", "generated_commit"):
        if not _is_commit_sha(snapshot.get(field)):
            errors.append(f"snapshot {field} is not a full commit SHA")
    if not str(snapshot.get("release_version") or "").strip():
        errors.append("snapshot release_version is absent")

    check_runs = snapshot.get("check_runs") or []
    for required_name in config.get("required_check_runs") or []:
        matches = [
            item
            for item in check_runs
            if item.get("name") == required_name and item.get("head_sha") == expected_head
        ]
        if not matches:
            errors.append(f"required check run is absent: {required_name}")
            continue
        item = _latest(matches, ("completed_at", "started_at", "id"))
        record = {
            "name": required_name,
            "status": item.get("status"),
            "conclusion": item.get("conclusion"),
            "url": item.get("html_url") or item.get("details_url"),
        }
        evidence["checks"].append(record)
        if item.get("status") != "completed" or item.get("conclusion") != "success":
            errors.append(
                f"required check run is not completed/success: {required_name} "
                f"status={item.get('status')!r} conclusion={item.get('conclusion')!r}"
            )

    statuses = snapshot.get("statuses") or []
    for required_context in config.get("required_commit_statuses") or []:
        matches = [item for item in statuses if item.get("context") == required_context]
        if not matches:
            errors.append(f"required commit status is absent: {required_context}")
            continue
        item = _latest(matches, ("updated_at", "created_at", "id"))
        record = {
            "context": required_context,
            "state": item.get("state"),
            "description": item.get("description"),
            "url": item.get("target_url"),
        }
        evidence["statuses"].append(record)
        if item.get("state") != "success":
            errors.append(f"required commit status is not success: {required_context} state={item.get('state')!r}")

    comments = snapshot.get("comments") or []
    for scanner in config.get("comment_scanners") or []:
        name = str(scanner.get("name") or "unnamed scanner")
        authors = {str(author).casefold() for author in scanner.get("authors") or []}
        matches = [
            item
            for item in comments
            if str((item.get("user") or {}).get("login") or "").casefold() in authors
        ]
        if not matches:
            errors.append(f"required scanner comment is absent: {name}")
            continue
        item = _latest(matches, ("updated_at", "created_at", "id"))
        body = str(item.get("body") or "")
        folded = body.casefold()
        failures = [pattern for pattern in scanner.get("failure_patterns") or [] if pattern.casefold() in folded]
        successes = [pattern for pattern in scanner.get("success_patterns") or [] if pattern.casefold() in folded]
        record = {
            "name": name,
            "author": (item.get("user") or {}).get("login"),
            "url": item.get("html_url"),
            "matched_success_patterns": successes,
            "matched_failure_patterns": failures,
        }
        evidence["comment_scanners"].append(record)
        if failures:
            errors.append(f"required scanner comment reports failure: {name} patterns={failures}")
        if scanner.get("success_patterns") and not successes:
            errors.append(f"required scanner comment lacks a success marker: {name}")

    evidence["compliant"] = not errors
    evidence["errors"] = errors
    return evidence


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--config", type=Path, required=True)
    parser.add_argument("--snapshot", type=Path, required=True)
    parser.add_argument("--expected-head", required=True)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()

    config = json.loads(args.config.read_text(encoding="utf-8"))
    snapshot = json.loads(args.snapshot.read_text(encoding="utf-8"))
    receipt = verify(config, snapshot, args.expected_head)
    args.output.write_text(json.dumps(receipt, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    if not receipt["compliant"]:
        for error in receipt["errors"]:
            print(f"release scanner contract: {error}", file=sys.stderr)
        return 1
    print(f"Release scanner contract passed for exact head {args.expected_head}.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
