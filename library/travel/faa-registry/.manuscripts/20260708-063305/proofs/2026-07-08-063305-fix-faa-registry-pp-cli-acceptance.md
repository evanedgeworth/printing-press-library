# faa-registry — Phase 5 Acceptance Report

**Level:** Full Dogfood (binary-owned live matrix, `dogfood --live --level full`)
**Gate: PASS** — 86/86 executed tests passed, 45 skipped (no-positional / auth-gated shapes), 0 failed.
**Auth:** none (public site + public bulk database — freely testable, no credential).

## Test infrastructure note

The live-dogfood runner sandboxes `HOME`/`XDG_*` into a fresh temp dir per subprocess. For a sync-first offline CLI this means the runner's subprocesses cannot see the one-time-synced `~/.local/share/faa-registry-pp-cli/registry.db`, so all offline commands would otherwise fail with "database is empty" — a harness structural gap, not a CLI defect (flagged upstream via the output-review). Resolved cleanly by adding a **`FAA_REGISTRY_DB` env override** to the CLI (good practice regardless: custom data dirs, CI, shared caches) and exporting it before the run so the sandboxed subprocesses reach the real synced database. All offline commands then passed.

## Failures found and fixed inline (12 → 0 over 3 loops)

1. **10× "database is empty"** on offline commands — harness sandbox blind spot. Fix: `FAA_REGISTRY_DB` env override (CLI feature) + export it for the run. All 10 now pass against the real 921K-row database.
2. **`aircraft lookup` / `aircraft history` error_path** returned empty result + exit 0 on a malformed tail. Fix: `registrydb.ValidTail` format validation before the request → clean exit 1 with a specific message. (CLI fix.)
3. **`watch add/remove/check --dry-run --json`** emitted nothing (json_fidelity wants JSON). Fix: dry-run now prints a `{"dry_run":true,...}` stub. (CLI fix.)
4. **`watch check` happy_path** "missing runnable example" — the `Example:` used a `sync && watch check` compound the harness can't parse. Fix: single-invocation example. (CLI fix.)

## Also fixed from the agentic output-review (all shipped)

- **Compact-pruning field loss:** `hex resolve`, `hex to-tail/from-tail`, and `nnumber available` used `omitempty`, so `--compact`/`--agent`'s "field in ≥80% of rows" rule stripped owner/model from mixed batches. Removed `omitempty` on those payload structs — every row now carries every key. (This was the primary ADS-B-log use case.)
- **Silent prefix-match in `models fleet`:** `--model SR22` also matched SR22T (3,008 of 7,006). Added a `matched_models` per-variant breakdown so the aggregation is honest and auditable.
- **Raw YYYYMMDD dates:** `expiring`, `aircraft history`, `nnumber available` now render ISO 8601 (`2027-01-31`).
- **Empty `expiring` UX:** emits a stderr note with the soonest matching expiration date; documented examples widened to `--within 365` (FAA renewals cluster at month-ends, so short windows are legitimately empty).

## Behavioral spot-checks against the real synced DB (post-fix)

- `fleet report --owner "NETJETS SALES INC"` → 42 aircraft, all jet, 28× Cessna 560XL, avg year 2009.7.
- `hex resolve A008C5 A00001 BADHEX --agent` → registry hit keeps owner+model; computed fallback; invalid flagged — all keys present.
- `models fleet CIRRUS SR22` → 7,006 total, matched_models shows SR22 (3,998) + SR22T (3,008) split.
- `aircraft history N101DQ` → current NETJETS registration, ISO dates.
- `expiring --within 30 --state WA` → empty + "soonest matching expiration is 2027-01-31".
- `aircraft lookup __printing_press_invalid__` → exit 1, format error.
- `watch add/check/list --dry-run --json` → valid JSON.

## Doc-review (Phase 4.8/4.9) findings — all 7 warnings fixed

Stripped no-auth credential/secret boilerplate from SKILL "Paths and state"; added the sync prerequisite note atop Unique Capabilities; documented `--offline` + the 5-co-owner bulk cap in the aircraft recipe; replaced `example-value` placeholders with real dealer names; replaced README's nonexistent `list` command guidance with `search`/`which`; softened AGENTS mutation framing to match the read-only nature.
