---
name: printing-press-contribution
description: Use in printing-press-library when asked to claim an issue, add or reprint a CLI, or publish generated CLI output. Do not use for read-only catalog questions or ordinary fixes to an existing published CLI.
---

# Printing Press Contribution

## Claim an issue before you start working on it

Multiple agents and humans work this repo simultaneously. Before touching code for an issue, **claim it visibly** and **check whether someone else already has** — otherwise two agents land overlapping PRs and one of you wasted a session.

**Before starting, check both signals for an existing claim:**

```bash
# 1. Is the issue assigned to someone?
gh issue view <num> --json assignees,state,title

# 2. Has anyone commented claiming it (look for phrases like "I'll take this",
#    "claiming this", "working on this", or another agent's claim comment)?
gh issue view <num> --comments
```

If `assignees` is non-empty OR a recent comment claims the issue, **don't start work**. Pick a different issue. If the existing claim looks stale (no follow-up activity for more than a few days and no linked PR), leave a polite comment asking the original claimer if they're still working on it before you proceed — don't just take it.

**To claim an issue yourself, do both:**

```bash
# 1. Try to self-assign. This may fail silently or with a permissions error
#    on forks or for accounts without write access — that's expected.
gh issue edit <num> --add-assignee @me 2>/dev/null || true

# 2. Always leave a claim comment. This is the durable signal that works
#    regardless of repo permissions and is visible to humans skimming the
#    issue thread.
gh issue comment <num> --body "Claiming this — starting work now. Will open a PR within <reasonable timeframe>."
```

The comment is mandatory, the self-assign is best-effort. Self-assign needs write access on the repo (which agents running from forks, lower-trust accounts, or unauthenticated CI contexts often don't have), and even when it succeeds it isn't surfaced in `gh issue list` output the same way comments are. The comment is the convention; the assignee is a courtesy when permissions allow.

**If you abandon a claim** (the task turned out larger than expected, you got blocked, etc.), leave a follow-up comment saying so and `gh issue edit <num> --remove-assignee @me` if you self-assigned successfully. Don't silently disappear — the next agent looking at the issue uses your claim comment to decide whether the issue is still in flight.

## Adding a new CLI or reprinting an existing one — use the Printing Press, not a hand-built PR

**New CLI additions and reprints do not start in this repo.** They are produced by `cli-printing-press` and shipped here through the publish phase of `/printing-press` (which invokes the `/printing-press-publish` skill at the end of a run). Hand-constructed PRs that try to assemble the canonical CLI shape from scratch in this repo systematically miss things: wrong directory layout (`<slug>-pp-cli/` instead of `<slug>/`), missing `.manuscripts/<run-id>/{research,proofs}/`, missing `dogfood-results.json`, hand-authored `cli-skills/pp-<slug>/SKILL.md` (which is a generated mirror), wrong PR title scope (`feat(library)` instead of `feat(<slug>)`), wrong branch name (`add-<slug>` instead of `feat/<slug>`), and PR descriptions that are free-form prose instead of the validation-table-bearing template the publish skill produces. We get those PRs every week and they are not mergeable as-is.

This rule applies to **any agent working in this repo** that has been asked to "add the X CLI", "publish the X CLI", "reprint X under the new template", or anything that produces a fresh `library/<category>/<slug>/` tree. Stop before you start writing `main.go`.

### When this rule applies

Treat the change as a **new CLI / reprint** (subject to this rule) if any of these are true:

- A brand-new directory `library/<category>/<slug>/` is being created.
- `library/<category>/<slug>/.printing-press.json` is being added, or the existing one is being replaced wholesale (not a small `printing_press_version` bump or single-field tweak).
- You are about to write more than two of the canonical foundational files for a slug that did not exist on `main`: `cmd/<slug>-pp-cli/main.go`, `internal/cli/root.go`, `internal/client/client.go`, `SKILL.md`, `README.md`, `AGENTS.md`, `Makefile`, `LICENSE`, `NOTICE`, `.goreleaser.yaml`, `.golangci.yml`, `go.mod`, `go.sum`.

The following are **not** new CLI / reprint changes and are fine to land via a normal hand-authored PR from this repo:

- Bug fixes or behavior tweaks in an already-published CLI's `internal/cli/*.go`, `internal/client/*.go`, etc. Record any code-level customization in `.printing-press-patches/` per the convention below.
- README/SKILL.md polish on an already-published CLI (edit `library/<cat>/<slug>/SKILL.md`, never the `cli-skills/` mirror).
- AST-level patches via `printing-press patch` from the generator repo.
- `tools/sweep-canonical/` runs that retrofit canonical shape across all entries.
- CI changes under `.github/workflows/`, repo-root docs, or installer changes under `npm/`.
- Manual edits to the generator-output files (`registry.json`, `cli-skills/pp-*/SKILL.md`) — don't; those are bot-regenerated post-merge.
- Manual release-version bumps or changelog release entries — don't; `CHANGELOG.md`, `.printing-press-release.json`, and runtime `version` stamping are owned by the post-merge release-ledger workflow described below.

### What to do instead, when the change *is* a new CLI / reprint

1. **Stop editing files in this repo** and tell the user: "This is a new CLI / reprint. The canonical path is `/printing-press <api>` (or `/printing-press-reprint <api>` for a regen) in a workspace that has the `cli-printing-press` generator available. That flow runs research → generate → dogfood → verify → archive → publish end-to-end, and Phase 6 opens a properly-shaped PR back into this repo by invoking `/printing-press-publish` for you. You almost never invoke the publish skill directly."
2. If the CLI is already generated and archived under `~/printing-press/library/<slug>/` and only the publish step remains, run `/printing-press-publish <slug>` directly — that skill handles category resolution, validation, branch naming, registry update, manuscript inclusion, and PR description shape end-to-end. Don't replicate any of those steps by hand.
3. If the user does not have the generator and is committed to a hand-built submission anyway, you must reproduce the canonical shape exactly. The publish-skill PR template is the contract; deviations get bounced. See the next section.

### Canonical PR shape (the publish-skill contract)

If you are constructing a new-CLI PR by any path, it must match this shape:

- **Branch:** `feat/<slug>` exactly. Not `add-<slug>`, not `feat/<slug>-pp-cli`, not `feature/<slug>`.
- **Title:** `feat(<slug>): add <slug>` exactly. No trailing dash-suffix description (`— Korean startup database`), no `feat(library)` scope, no Co-Authored-By or "Claude Code" trailers.
- **Directory:** `library/<category>/<slug>/`. Slug only — never `library/<category>/<slug>-pp-cli/`. The `-pp-cli` infix lives in binary names, not directory names.
- **Files present at minimum:** `.printing-press.json` (full manifest with `api_name`, `cli_name`, `spec_format`, `spec_checksum`, `spec_source`, `printing_press_version`, plus MCP fields if applicable), `cmd/<slug>-pp-cli/main.go`, `internal/cli/root.go` and the per-resource files, `internal/client/client.go`, `SKILL.md`, `README.md`, `AGENTS.md`, `LICENSE`, `NOTICE`, `Makefile`, `.goreleaser.yaml`, `.golangci.yml`, `go.mod`, `go.sum`, `dogfood-results.json`, and a populated `.manuscripts/<run-id>/research/` plus `.manuscripts/<run-id>/proofs/`. A reprint that drops the manuscripts is not a publishable reprint.
- **Files NOT in the PR:** `cli-skills/pp-<slug>/SKILL.md` and `registry.json` (both regenerated post-merge by `generate-skills.yml` and `generate-registry.yml` — `verify-library-conventions.yml` hard-fails if either is present in the PR diff); committed binaries; `.env`, `session-state.json`, or other files with real credentials.
- **PR body:** matches the publish-skill template — `## <slug>` heading, description paragraph, `**API:** … | **Category:** … | **Press version:** …` line, `**Spec:** …` line, then `### CLI Shape` (with the verbatim `--help` output in a fenced code block), `### What This CLI Does`, `### Manuscripts` (links to the in-PR `.manuscripts/<run-id>/research/` and `.manuscripts/<run-id>/proofs/` paths), `### Validation Results` (a table with PASS/FAIL for Manifest, Phase 5, `go mod tidy`, `go vet`, `go build`, `--help`, `--version`, `verify-skill`, `govulncheck`, Manuscripts), and an optional `### Gaps` section. **No** `## Summary` / `## Why This Matters` / `## What It Does` / `## Endpoints` / `## Test plan` / generic-template prose; that shape signals a hand-built submission and a likely missing-files PR.

### Signs you've drifted off the canonical path

If you're constructing a new-CLI PR and you notice any of these, stop and route through `/printing-press publish` instead:

- You're writing the PR description from scratch in prose rather than filling in the publish-skill template.
- You're using `## Summary`, `## Why This Matters`, `## Endpoints`, `## Test plan`, or appending `🤖 Generated with Claude Code` — those are universal-PR-template tells, not publish-skill output.
- Your validation evidence is a `## Testing` code block with `go build ./...` and a smoke command, instead of the structured `### Validation Results` table.
- You can't link to manuscripts because there are no `.manuscripts/<run-id>/research/` or `.manuscripts/<run-id>/proofs/` files in the diff.
- You're hand-editing `cli-skills/pp-<slug>/SKILL.md` or `registry.json`. Both are bot-regenerated; hand-edits get overwritten and create merge conflicts.
- The PR diff has fewer than ~30 files for a new CLI — the canonical layout always lands dozens of files (`internal/cli/*.go` per endpoint plus `internal/cliutil/`, `internal/client/`, optional `internal/mcp/`, `cmd/<slug>-pp-cli/`, plus the docs/build files). A new-CLI PR with only a single `main.go` is missing nearly all of it.

When any of these fire, **the right move is not to fix the PR by adding more sections**. The right move is to drop the hand-built branch, re-enter `/printing-press <api>` (or `/printing-press-publish <slug>` if the CLI is already generated and only the publish step remains), and let the skill open the PR.

### CI gates for new-CLI / reprint shape

The `verify-library-conventions.yml` workflow runs `verify_publish_package.py` on every PR that touches `library/**`. For PRs that **add** a `library/<cat>/<slug>/.printing-press.json` (the classifier for new-CLI submissions), it hard-fails on missing publish artifacts (the patches index — either `.printing-press-patches/` or the legacy `.printing-press-patches.json` — `AGENTS.md`, `README.md`, `SKILL.md`, `go.mod`, `.goreleaser.yaml`, `LICENSE`, `NOTICE`, `cmd/<cli_name>/main.go`), missing `.manuscripts/<run-id>/{research,proofs}/`, missing `run_id` / `printer` / `printing_press_version` in the manifest, missing `novel_features`, and missing MCP artifacts (`manifest.json`, `tools-manifest.json`) when MCP is advertised. It also emits advisory notices when the PR body lacks `### Publication Path` or `### Novel Commands`. Don't try to placate the verifier file by file — if it fires, you've drifted off the publish flow; re-run the publish skill.

## Automated code review with Greptile

Every PR against this repo gets an automated review from **Greptile** alongside the verify-* workflows. Greptile posts a top-level summary comment with a **confidence score on a 0-5 scale** (5 = "Production ready", 4 = "Minor polish needed", 3 = "Implementation issues", 2 = "Significant bugs", 0-1 = "Critical problems"), plus inline comments tagged with **P0 / P1 / P2** severity (P0 = must fix before merge, P1 = should fix, P2 = consider fixing) and categorized as Logic / Syntax / Style. Status is shown via 👀 (analyzing) → 👍 (done) or 😕 (failed); Greptile does NOT use GitHub's approve / request-changes flow.

**The bar is resolving every Greptile finding before merge** — the 0-5 score is a confidence signal, not a guarantee, so don't treat the number itself as the gate. 4/5 and 5/5 are both acceptable end states; the score will land in that range naturally once threads are addressed. A 5/5 with open P1s is still not ready; a 4/5 with everything resolved is ready. Treat every P0 and P1 as blocking; P2s require either a fix or a concrete reply explaining why we're deferring.

Greptile feedback is not limited to GitHub review threads. It also edits top-level PR summary comments, and those summaries can contain actionable issue blocks, including `Comments Outside Diff`, even when the thread list has zero unresolved comments. Before saying a PR is ready, read the latest `greptile-apps` top-level summary yourself, then run the repo-owned review-state helper:

```bash
python3 .github/scripts/pr-review-state/greptile_feedback.py <PR_NUMBER>
```

`PR_NUMBER` is the GitHub pull request number, for example `1093` — not a branch name, URL, issue number, or commit SHA. The helper defaults to `mvanhorn/printing-press-library` and exits non-zero until all of these are true: Greptile Review passes, Greptile policy gate passes, there are no unresolved non-outdated review threads, the latest `greptile-apps` top-level comment reviewed the current PR head SHA, and that latest comment has no actionable markers such as `Issue 1 of`, `Fix the following`, `Comments Outside Diff`, `remaining open item`, or `Safe to merge after fixing/reviewing`. The helper is a guardrail, not a substitute for reading the summary; phrases like `one small fix` or `gap remains` are still blocking.

If you (an agent) opened the PR, you own driving it to ready-to-merge:

1. **Watch for the review.** Greptile posts within a few minutes of PR open or push. Read findings with `gh pr view <PR> --comments`; check the summary comment for the score and the inline threads for P0/P1/P2 tags.
2. **Address every finding in code or in a reply.** Push fixes when a finding is valid. When you genuinely believe a finding is wrong or out of scope, reply with a concrete reason (not "won't fix" — explain *why* the code is right as written, or *why* the deferral is justified) and ask the thread to be resolved.
3. **Re-trigger after pushes.** Greptile re-reviews automatically on push. A stuck review can be re-run via the "Re-trigger Greptile" button in the summary comment footer.
4. **If the `Greptile policy gate` job fails with `Timed out waiting for Greptile Review to complete`,** Greptile silently skipped the PR — usually because the diff is over its plan's size cap (large new-CLI prints are the common trigger). The gate workflow auto-posts `@greptileai review` after ~3 minutes of no check appearing, which forces a manual review with no rate limit; if that nudge also didn't recover, post `@greptileai review` yourself as a single PR comment and wait for the bot to start. Don't tag `@greptileai` preemptively on PRs you just opened — only after a documented timeout — or you'll double up on reviews.
5. **Don't merge with unresolved Greptile threads.** If a thread won't resolve because a finding looks like a genuine false positive, escalate to a human reviewer on the thread before merging. A high score is not a substitute for closing the thread.
6. **Greptile is configured by `greptile.json` at the repo root.** That config encodes repo-specific rules — manuscript-content judgment for new CLIs, reprint classification, PR title / branch / body shape. Don't disable rules to silence a finding; the rule exists because something burned us. If you believe a rule is mis-firing across the board, file a separate PR amending `greptile.json` with reasoning.

The same expectation applies to non-CLI PRs (CI fixes, bug fixes, doc edits, sweep-canonical runs): resolve every comment before merge. The strictness is uniform; the rule-set Greptile applies varies with what you touched.
