# FAA Aircraft Registry — Discovery Report

Run: 20260708-063305 · Method: direct HTTP contract probing (probe-reachability = standard_http; no browser needed)

## Reachability

- `registry.faa.gov` sits behind Akamai. Non-browser User-Agents (`curl/*`, `Go-http-client/*`) get **403 Access Denied** (edgesuite error page). Any Chrome-like UA gets **200**.
- Runtime: **standard HTTP + required User-Agent header**. No cookies, no CSRF, no auth. `probe-reachability` confidence 0.95 for both stdlib and surf-chrome transports.
- The ASP.NET anti-forgery token rendered in forms is **not enforced**: bare POST and bare GET both return full results.

## Live inquiry contract (all replayable via GET)

Base: `https://registry.faa.gov/aircraftinquiry`

| Inquiry | Endpoint (GET works) | Params |
|---|---|---|
| N-number lookup | `/Search/NNumberResult` | `NNumberTxt` (with or without leading N) |
| Serial number | `/Search/SerialNumberResult` | `Serialtxt`, `sort_option` (1=N-number, 2=serial, 3=mfr, 4=model, 5=name) |
| Owner name | `/Search/NameResult` | `nametxt`, `sort_option`, `Page` |
| Make/Model | `/Search/MakeModelResult` | `Mfrtxt`, `Modeltxt`, `Page` |
| Engine reference | `/Search/EngineReferenceResult` | `MfrNametxt`, `Modeltxt` |
| Dealer | `/Search/DealerResult` | `Dealertxt` |
| Document index | `/Search/DocumentIndexResult` | `Colltxt` |
| State/County | `/Search/StateCountyResult` | `StateName`, `CountyName` |
| Country | `/Search/CountryResult` | `Countrytxt` |
| N-number availability | `/Search/NNumberAvailabilityResult` | `Starttxt`, `Endingtxt`, `Characterstxt` — 302s on naive GET/POST; may need token or specific param shape. Availability is fully computable offline from MASTER+RESERVED, so live endpoint is optional. |

Pagination: `&Page=N` query param; result pages embed `Page=` links (name search "DELTA AIR LINES" → 32 pages).

## Response shape (HTML, highly semantic)

- Detail pages (N-number result): sections as `<table>` with `<caption class="devkit-table-title">` (Aircraft Description, Registered Owner, Other Owner Names, Temporary Certificates, Fuel Modifications) and every `<td>` tagged `data-label="Field Name"`. ~40 distinct labels observed (Serial Number, Manufacturer Name, Model, Mode S Code (Base 16 / Hex), Expiration Date, Status, owner address fields, engine fields, TCDS, etc.).
- List pages (name/make-model/state results): `<th>` headers + rows with `data-label` cells; each row links `/Search/NNumberResult?NNumberTxt=...`.
- Extraction plan: generated endpoints use `response_format: html` (page mode); flagship commands hand-parse the data-label tables into typed JSON (parser in `internal/faaparse`), which is robust given the semantic markup.

## Offline database (the data layer)

`https://registry.faa.gov/database/ReleasableAircraft.zip` — 72.8 MB zip, **updated daily** (Last-Modified moves each night; files timestamped previous evening). HEAD is blocked (503) but GET works (IIS origin, supports Range).

Contents (2026-07-08 snapshot):

| File | Rows | Key fields |
|---|---|---|
| MASTER.txt | 314,529 | N-NUMBER, SERIAL, MFR MDL CODE, ENG MFR MDL, YEAR MFR, TYPE REGISTRANT, NAME, address, LAST ACTION DATE, CERT ISSUE DATE, CERTIFICATION, TYPE AIRCRAFT/ENGINE, STATUS CODE, MODE S CODE + HEX, FRACT OWNER, AIR WORTH DATE, OTHER NAMES(1-5), **EXPIRATION DATE**, UNIQUE ID, KIT MFR/MODEL |
| DEREG.txt | 382,709 | deregistered aircraft: CANCEL-DATE, EXP-COUNTRY, mail+physical addresses, MODE-S-CODE HEX, ... |
| RESERVED.txt | 125,995 | reserved N-numbers: REGISTRANT, RSV DATE, EXP DATE, PURGE DATE, N-NUM-CHG |
| ACFTREF.txt | 93,871 | model reference: MFR, MODEL, TYPE-ACFT, TYPE-ENG, AC-CAT, NO-ENG, **NO-SEATS, AC-WEIGHT, SPEED**, TC-DATA-SHEET/HOLDER |
| ENGINE.txt | 4,747 | engine reference: MFR, MODEL, TYPE, **HORSEPOWER, THRUST** |
| DEALER.txt | — | dealer certificates |
| DOCINDEX.txt | — | document index |
| ardata.pdf | — | official field documentation |

CSV-ish fixed-pad format with trailing commas; first line headers; values right-padded with spaces (need TrimSpace on import).

## Verified sample invocations

- `GET /Search/NNumberResult?NNumberTxt=N172SP` → 200, Cessna 172S SKYHAWK detail page (same bytes as token-bearing POST).
- `GET /Search/NameResult?nametxt=DELTA+AIR+LINES&sort_option=1` → 200, paginated fleet list, 32 pages.
- `GET /Search/MakeModelResult?Mfrtxt=CIRRUS&Modeltxt=SR22` → 200, model reference table with "Number of Aircraft Assigned".
- `GET /Search/SerialNumberResult?Serialtxt=17280005&sort_option=1` → 200.
