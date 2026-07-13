# faa-registry — Polish Pass

Scorecard 82 → 86 (Grade A). Verify 100%. Publish-validate FAIL → PASS. Tools-audit 1 → 0 pending.

Fixes applied by polish:
- Removed 2 dead functions (writeNoop/noopResult, responsePayloadParentAtPath) from helpers.go
- Rewrote thin `watch list` Short to a parameter-aware description
- Added mcp:read-only to `profile use`
- Spec-grounded MCP description overrides for all 9 typed tools (mcp-descriptions.json) → MCP Quality 8→10
- Assembled .manuscripts/<run>/{research,proofs}/ incl. passing phase5-acceptance.json
- gofmt normalized 7 files

Post-polish follow-up (closed by orchestrator, not deferred):
- Cache Freshness gap: wired a registrydb-native staleness hint (`emitRegistryStaleHint`) through all
  offline command choke points (fleet report, search, expiring, hex resolve, models fleet,
  aircraft history, nnumber available). Fires on stderr when the local registry is >7d old; suppressed
  under --quiet/--agent. Verified: backdated synced_at → hint fires with day count; --quiet silent.
  The scorecard dimension still reads 3/10 only because its live-check sandbox has no synced DB to
  age-check — environmental, same class as the documented live-check blind spot.

Skipped (false positives / structural): import.go rate-limit warn (typed 429 handling exists, single
daily GET), generic-Upsert warn (real sync is the custom registrydb importer), live-check failures
(empty-DB sandbox), MCP stdio-only (spec-edit + regen, out of scope).
