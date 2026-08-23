#!/usr/bin/env python3
"""Keep agent discovery deterministic and below host input limits."""

from pathlib import Path


ROOT = Path(__file__).resolve().parents[2]
AGENTS = ROOT / "AGENTS.md"
PORTABLE_SKILLS = ROOT / ".agents" / "skills"
CLAUDE_SKILLS = ROOT / ".claude" / "skills"


def main() -> None:
    size = AGENTS.stat().st_size
    if size >= 32_768:
        raise SystemExit(f"AGENTS.md is {size} bytes; keep it below 32768 bytes")
    if CLAUDE_SKILLS.resolve() != PORTABLE_SKILLS.resolve():
        raise SystemExit(".claude/skills must resolve to .agents/skills")

    contribution = (PORTABLE_SKILLS / "printing-press-contribution" / "SKILL.md").read_text()
    patch_ledger = (PORTABLE_SKILLS / "printing-press-patch-ledger" / "SKILL.md").read_text()
    required = {
        "printing-press-contribution": (
            contribution,
            "add or reprint a CLI",
            "read-only catalog questions",
            "Automated code review with Greptile",
        ),
        "printing-press-patch-ledger": (
            patch_ledger,
            "hand-modified",
            "README-only or SKILL-only edits",
            ".printing-press-patches/",
        ),
    }
    for name, (body, positive, negative, procedure) in required.items():
        for phrase in (positive, negative, procedure):
            if phrase not in body:
                raise SystemExit(f"{name} is missing required trigger or procedure: {phrase}")

    print(f"agent instructions verified: AGENTS.md={size} bytes")


if __name__ == "__main__":
    main()
