# faa-registry — Shipcheck Report

## Final verdict: PASS (7/7 legs) — recommendation: ship

| Leg | Result |
|---|---|
| verify | PASS |
| validate-narrative | PASS (strict + full examples: 12 commands resolved) |
| dogfood | PASS |
| workflow-verify | PASS |
| apify-audit | PASS |
| verify-skill | PASS |
| scorecard | PASS — **80/100, Grade A** (MCP-remote/auth/live-api dims omitted from denominator) |

## Fix loops (2 used)

**Loop 1:** validate-narrative FAIL — two examples used shell pipelines (`echo 'A008C5' | ...`, `cat hexes.txt | ...`) the validator can't resolve. Rewrote to argument form (`hex resolve A008C5`) in research.json + README + SKILL; stdin behavior still documented in the command's own help.

**Loop 2 (advisory findings from dogfood):**
- Added `// pp:data-source local` annotations to all 6 novel command files.
- Added typed HTTP 429 handling to the sync downloader.
- Synced root.go Short/Long to the narrative headline (was the truncated spec cli_description).

## Behavioral verification against real targets (in addition to the mechanical legs)

- `aircraft lookup N101DQ --json` → full parsed record: NETJETS SALES INC, serial 560-6513, Mode S A008C5, expiration 06/30/2033, 12 other-owner names incl. the user's, temporary certificate T263103. Matches the live FAA page field-for-field.
- `sync` → exact row counts: master 314,529 / dereg 382,709 / reserved 125,995 / acftref 93,871 / engine 4,747 (verified against the extracted files with `wc -l`).
- `fleet report --owner "NETJETS SALES INC"` → 42 aircraft, 28× Cessna 560XL, all-jet, avg year 2009.7.
- `hex resolve` (stdin batch) → registry hit (A008C5→N101DQ w/ owner), computed fallback (A00001→N1), invalid detection (ADF7C8 outside block).
- `hex to-tail A008C5` = N101DQ — algorithm independently matches the FAA's own MODE S CODE HEX column.
- `expiring --within 30 --state WA` → empty, **verified correct** via direct SQL (earliest future WA expiration is 2027-01-31); `--within 365` returns the 2027-01-31 batch with days_left 206.
- `models fleet CIRRUS SR22` → 7,006 registered, split across 8 registrant types, years 2001-2026.
- `nnumber available` → assigned (N101DQ w/ owner) / free / invalid-format rejection (N99999X, N101DQS, NI23).
- `watch add/check/list` → baseline then no-change on stable data.
- `owners --all-pages` → 988 rows across 20 pages, rows == total.
- `aircraft by-serial --serial 560-6513` → finds N101DQ (after fixing the server's mandatory sort_option).
- `regions by-state WA --county KING` → 32 rows == total 32 (after excluding the FAA's summary footer table).

## Bugs found and fixed during build/test (fix-before-ship, all fixed)
1. encoding/csv swallowed 67K ACFTREF rows (bare quotes in model names like `"DR 107"` treated as quoted fields) → manual line splitter; counts now exact.
2. `SerialNumberResult`/`NameResult` 302 without `sort_option` → default `--sort 1`.
3. County listing off-by-one → FAA appends an aggregate "SubTotal" table; parser now excludes summary sections.
4. `OTHER NAMES(1)` header normalization mismatch → co-owner columns imported as NULL; fixed key map (fractional-owner search depends on it).
5. `nnumber available` accepted invalid formats → added N-number format validation with reasons.
6. An early test tail with an illegal character (O is not used in N-numbers) → N500XA everywhere; caught by the new validator.

## Known non-defects
- Sample Output Probe shows 5/6 offline commands failing with "local registry database is empty" in the probe's isolated sandbox — expected: offline commands require a one-time 73 MB sync; the error message is the designed remediation guidance. All 6 verified working against the real synced database (above).
- Offline records carry at most 5 co-owner names (bulk MASTER file limit); live `aircraft lookup` shows the complete list. Documented in help + SKILL.
- Scorecard "insight 4/10" gap noted; novel-feature set was user-approved scope.
