# faa-registry — Phase 3 Build Log

## What was built

**Priority 0 — data layer (`internal/registrydb/`, new package):**
- SQLite schema: `faa_master` (314K active registrations), `faa_dereg` (383K), `faa_reserved` (126K), `faa_acftref` (94K models w/ seats/weight/speed), `faa_engine` (4.7K w/ HP/thrust), `faa_meta`, `faa_watches` + `faa_watch_state`, and `faa_master_fts` (FTS5 over names/co-owners/model/mfr). Separate `registry.db` file so the big import never bloats the framework cache DB.
- Header-driven CSV importer: maps normalized header names → columns, so the FAA's known layout drift (e.g. 2024 DOCINDEX column addition) degrades gracefully instead of breaking. BOM-aware, TrimSpace on every field, single-tx batch insert.
- `Sync`: GET with browser UA (Akamai rejects non-browser UAs), honors If-Modified-Since, caches the zip next to the DB.
- Code decoders from ardata.pdf: registrant type, aircraft type, engine type, region, ~40 status codes, airworthiness classes.
- N-number ⇄ ICAO24 algorithm (US block A00001–ADF7C7), both directions, validated against the FAA's own MODE S CODE HEX data (N101DQ↔A008C5) plus reference pairs and a full-block roundtrip sweep.
- Typed queries: LookupTail/LookupHex (3-table join), Fleet (aggregation), History (MASTER+DEREG timeline), Expiring, ModelFleet, Available (MASTER ∪ RESERVED with reason), Search (FTS), watch CRUD + snapshot diff.

**Priority 1 — absorbed (live inquiries):**
- `internal/faaparse/` (new package): parses aircraftinquiry HTML via the semantic `data-label`/`caption` markup. Handles the FAA's nested-captions-as-section-dividers quirk, label-cells vs value-cells, error banners, "Showing X - Y of Z (Page N of M)" pagination. Auto-detects detail vs list pages. Table-driven tests against live-captured fixtures (N101DQ detail incl. fractional owner data, Delta 1558-aircraft name search, CIRRUS SR22 make/model, serial result).
- All 9 generated live commands re-pointed from generic page extraction to `parseFAAHTMLResponse` (typed JSON out): aircraft lookup / by-serial, owners, models, engines, dealers, documents, regions by-state / by-country.
- `aircraft lookup` extended: positional tail arg, `--hex` (DB-then-algorithm resolution), `--offline` (local-DB answer).
- `owners --all-pages` + `--max-pages` (default cap 40): fetches and merges every server-side page.
- `sync` (new), `search` (new, FTS), `watch add/remove/list/check` (new; snapshot-diff based change detection — beats FAA-registry-checker's exact-caps single-owner design).
- `hex to-tail` / `hex from-tail` (pure algorithm, no DB needed).

**Priority 2 — transcendence (all 6 approved novel features implemented, zero stubs):**
- `fleet report --owner` — MASTER×ACFTREF aggregation: count, model mix, engine classes, states, avg seats/year, `--aircraft` for per-tail rows.
- `hex resolve` — batch stdin/args; registry-joined; algorithm fallback tagged `source: computed`.
- `aircraft history` — DEREG+MASTER chronological timeline.
- `expiring --within N [--owner] [--state]` — soonest-first with days_left.
- `models fleet --manufacturer [--model]` — registrant-type + state breakdown, year range.
- `nnumber available` — batch, with reason (assigned/reserved+purge date/free).

## Verification during build
- `go build ./...`, `go vet ./...`, `go test ./...` all green (faaparse: 5 tests, registrydb: 8 tests incl. import fixture zip, hex sweep; generated novel-wiring smoke tests pass).
- Live: `aircraft lookup N101DQ --json` returns the full parsed record (matches the user's pasted page exactly); `hex to-tail A008C5` = N101DQ.

## Intentionally deferred / notes
- Live `NNumberAvailabilityResult` endpoint dropped from spec: rejects scripted requests with format-error redirects even with session+CSRF (probed extensively). The offline computed `nnumber available` replaces it and explains the reason — strictly better.
- `regions`/`models`/`engines`/`dealers`/`documents` keep single-page fetch + `--page`; only `owners` (the fleet-list case) got `--all-pages`.
- DEALER.txt and DOCINDEX.txt not imported offline (live commands cover them; low query value vs 18MB+ import cost). Can be added to tableSpecs in one edit.

## Generator limitations found (for retro)
- `response_format: html` page-mode extraction is title/description-only — any data-bearing HTML page needs a hand parser. A `html_extract.mode: labeled-tables` (data-label/caption semantics) would have covered this API generator-side.
- Novel-feature scaffolds for commands that nest under generated resource parents emit a warning ("maps to generated command path; skipping novel stub") but the scaffold IS generated and wired — the warning is misleading.
