# FAA Aircraft Registry CLI

**Every FAA aircraft lookup the registry website offers, plus a daily-synced offline copy of the entire US registry that unlocks fleet reports, hex decoding, ownership history, and expiration alerts no other tool has.**

The FAA registry is browser-only and un-scriptable; existing wrappers parse stale CSVs or do one-off hex math. This CLI does both live and local: every inquiry page as a typed-JSON command, and the full 315K-aircraft registry (plus deregistered and reserved data) in SQLite for fleet report, hex resolve, aircraft history, expiring, and instant offline lookups.

Learn more at [FAA Aircraft Registry](https://registry.faa.gov/aircraftinquiry/).

Created by [@omarshahine](https://github.com/omarshahine) (Omar Shahine).

## Install

The recommended path installs both the `faa-registry-pp-cli` binary and the `pp-faa-registry` agent skill (Claude Code, Codex, Cursor, Gemini CLI, GitHub Copilot, and other agents supported by the upstream [`skills`](https://github.com/vercel-labs/skills) CLI) in one shot:

```bash
npx -y @mvanhorn/printing-press-library install faa-registry
```

For CLI only (no skill):

```bash
npx -y @mvanhorn/printing-press-library install faa-registry --cli-only
```

For skill only — installs the skill into the same agents as the default command above, but skips the CLI binary (use this to update or reinstall just the skill):

```bash
npx -y @mvanhorn/printing-press-library install faa-registry --skill-only
```

To constrain the skill install to one or more specific agents (repeatable — agent names match the [`skills`](https://github.com/vercel-labs/skills) CLI):

```bash
npx -y @mvanhorn/printing-press-library install faa-registry --agent claude-code
npx -y @mvanhorn/printing-press-library install faa-registry --agent claude-code --agent codex
```

### Without Node (Go fallback)

If `npx` isn't available (no Node, offline), install the CLI directly via Go (requires Go 1.26.4 or newer):

```bash
go install github.com/mvanhorn/printing-press-library/library/travel/faa-registry/cmd/faa-registry-pp-cli@latest
```

This installs the CLI only — no skill.

### Pre-built binary

Download a pre-built binary for your platform from the [latest release](https://github.com/mvanhorn/printing-press-library/releases/tag/faa-registry-current). On macOS, clear the Gatekeeper quarantine: `xattr -d com.apple.quarantine <binary>`. On Unix, mark it executable: `chmod +x <binary>`.

<!-- pp-hermes-install-anchor -->
## Install for Hermes

Install the CLI binary first. The installer writes binaries to a per-user managed bin directory by default: `$HOME/.local/bin` on macOS/Linux and `%LOCALAPPDATA%\Programs\PrintingPress\bin` on Windows.

```bash
npx -y @mvanhorn/printing-press-library install faa-registry --cli-only
```

Then install the focused Hermes skill.

From the Hermes CLI:

```bash
hermes skills install mvanhorn/printing-press-library/cli-skills/pp-faa-registry --force
```

Inside a Hermes chat session:

```bash
/skills install mvanhorn/printing-press-library/cli-skills/pp-faa-registry --force
```

Restart the Hermes session or gateway if the newly installed skill is not visible immediately.

## Install for OpenClaw
Install both the CLI binary and the focused OpenClaw skill. The installer defaults binaries to a per-user bin directory (`$HOME/.local/bin` on macOS/Linux, `%LOCALAPPDATA%\Programs\PrintingPress\bin` on Windows):

```bash
npx -y @mvanhorn/printing-press-library install faa-registry --agent openclaw
```

Restart the OpenClaw session or gateway if the newly installed skill is not visible immediately.

## Use with Claude Desktop

This CLI ships an [MCPB](https://github.com/modelcontextprotocol/mcpb) bundle — Claude Desktop's standard format for one-click MCP extension installs (no JSON config required).

To install:

1. Download the `.mcpb` for your platform from the [latest release](https://github.com/mvanhorn/printing-press-library/releases/tag/faa-registry-current).
2. Double-click the `.mcpb` file. Claude Desktop opens and walks you through the install.

Requires Claude Desktop 1.0.0 or later. Pre-built bundles ship for macOS Apple Silicon (`darwin-arm64`) and Windows (`amd64`, `arm64`); for other platforms, use the manual config below.

<details>
<summary>Manual JSON config (advanced)</summary>

If you can't use the MCPB bundle (older Claude Desktop, unsupported platform), install the MCP binary and configure it manually.


```bash
go install github.com/mvanhorn/printing-press-library/library/travel/faa-registry/cmd/faa-registry-pp-mcp@latest
```

Add to your Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "faa-registry": {
      "command": "faa-registry-pp-mcp"
    }
  }
}
```

</details>

## Quick Start

```bash
# Flagship: full registration record for a tail number, live from registry.faa.gov
faa-registry-pp-cli aircraft lookup N101DQ

# Download the FAA's daily Releasable Aircraft Database into local SQLite (~73 MB, one-time then daily refresh)
faa-registry-pp-cli sync

# Aggregate an owner's entire fleet: models, engine classes, seats, years
faa-registry-pp-cli fleet report --owner "NETJETS SALES INC"

# Turn ADS-B Mode S hex codes into tail numbers and owners, offline (also reads stdin, one per line)
faa-registry-pp-cli hex resolve A008C5

# Ownership timeline including deregistration records the website hides
faa-registry-pp-cli aircraft history N101DQ

# Registrations lapsing in the next year — renewals cluster at month-ends, so use a wide window
faa-registry-pp-cli expiring --within 365 --state WA

```

## Unique Features

These capabilities aren't available in any other tool for this API.

### Local registry that compounds
- **`fleet report`** — One command turns an owner name into a full fleet profile: aircraft count, model mix, jet/turboprop/piston split, average seats and year built.

  _Reach for this when asked what aircraft an operator or person owns — it answers in aggregate instead of forcing a page-through of raw tail lists._

  ```bash
  faa-registry-pp-cli fleet report --owner "NETJETS SALES INC" --agent
  ```
- **`hex resolve`** — Resolve any number of ADS-B Mode S hex codes (args or stdin) to N-numbers, aircraft types, and owners — offline.

  _Use this to identify aircraft from ADS-B receiver logs or flight-tracker hex codes in bulk, instantly and without network access._

  ```bash
  faa-registry-pp-cli hex resolve A008C5 --agent
  ```
- **`models fleet`** — For any make/model, break down every registered example by registrant type (corporate, individual, LLC, co-owned) and state.

  _Market research on a model class: how many exist, who owns them, and where they are based._

  ```bash
  faa-registry-pp-cli models fleet --manufacturer CIRRUS --model SR22 --agent
  ```
- **`nnumber available`** — Check whether an N-number is assigned, reserved, or free — computed locally, with the reason.

  _Vanity tail-number shopping and registration planning without fighting the website's form validation._

  ```bash
  faa-registry-pp-cli nnumber available N500XA --agent
  ```

### Due diligence
- **`aircraft history`** — Chronological owner timeline for a tail number, stitching current registration with every deregistration record.

  _Pre-purchase and title research: see who held an aircraft, when registrations were cancelled, and export history in one answer._

  ```bash
  faa-registry-pp-cli aircraft history N101DQ --agent
  ```
- **`expiring`** — List registrations expiring within a window, filtered by owner or state, sorted soonest-first.

  _Catch a lapsing registration (a closing risk and airworthiness problem) before the FAA letter arrives._

  ```bash
  faa-registry-pp-cli expiring --within 365 --state WA --agent
  ```

## Recipes

### Identify a plane you flew on

```bash
faa-registry-pp-cli aircraft lookup N101DQ --agent --select status,description.serial_number,description.manufacturer,description.model,owner.name,other_owner_names
```

Full registration record narrowed to the fields that answer 'whose plane is this?' — including any co-owner names.

### Profile an operator's fleet

```bash
faa-registry-pp-cli fleet report --owner "NETJETS SALES INC" --agent
```

Counts, model mix, and age profile for every aircraft registered to the owner, computed from the local registry.

### Bulk-resolve ADS-B hex captures

```bash
faa-registry-pp-cli hex resolve A008C5 A11F35 --agent
```

Each Mode S hex becomes tail number + model + owner, joined offline against the FAA database. Pipe a file of codes to stdin for bulk runs.

### Pre-purchase due diligence

```bash
faa-registry-pp-cli aircraft history N123AB --agent
```

Chronological ownership timeline with deregistration and cancel dates the FAA website doesn't show.

### Find lapsing registrations

```bash
faa-registry-pp-cli expiring --within 365 --state WA --agent
```

Registrations expiring in the next 60 days, soonest first.

## Usage

Run `faa-registry-pp-cli --help` for the full command reference and flag list.

## Paths & environment variables

This CLI separates local files into four path kinds:

| Kind | Contents |
|------|----------|
| `config` | User-editable settings such as `config.toml` and saved profiles |
| `data` | Durable local data such as `data.db` |
| `state` | Runtime state such as persisted queries, jobs, and `teach.log` |
| `cache` | Regenerable HTTP/cache files |

Each kind resolves independently. The ladder is:

1. Per-kind env var: `FAA_REGISTRY_CONFIG_DIR`, `FAA_REGISTRY_DATA_DIR`, `FAA_REGISTRY_STATE_DIR`, or `FAA_REGISTRY_CACHE_DIR`
2. `--home <dir>` for this invocation
3. `FAA_REGISTRY_HOME` for a flat relocated root
4. XDG env vars: `XDG_CONFIG_HOME`, `XDG_DATA_HOME`, `XDG_STATE_HOME`, `XDG_CACHE_HOME`
5. Platform defaults matching existing installs

For containers and agent sandboxes, prefer a single relocated root:

```bash
export FAA_REGISTRY_HOME=/srv/faa-registry
faa-registry-pp-cli doctor
```

Under `FAA_REGISTRY_HOME=/srv/faa-registry`, the four dirs resolve to `/srv/faa-registry/config`, `/srv/faa-registry/data`, `/srv/faa-registry/state`, and `/srv/faa-registry/cache`.

MCP servers do not receive CLI flags from the host. Put relocation in the host `env` block:

```json
{
  "mcpServers": {
    "faa-registry": {
      "command": "faa-registry-pp-mcp",
      "env": {
        "FAA_REGISTRY_HOME": "/srv/faa-registry"
      }
    }
  }
}
```

Precedence matters in fleets: an ambient per-kind variable such as `FAA_REGISTRY_DATA_DIR` overrides an explicit `--home` for that kind. Use `FAA_REGISTRY_HOME` or the per-kind variables for durable fleet relocation; treat `--home` as the weaker per-invocation lever.

Relocation is one-way. Unsetting `FAA_REGISTRY_HOME` does not move files back to platform defaults, and `doctor` cannot find files left under a former root. Move the files manually before unsetting relocation variables.

Existing installs keep working because the platform-default rung matches the legacy layout. Run `faa-registry-pp-cli doctor --fail-on warn` to check path warnings in automation.

## Commands

### aircraft

Live FAA registry lookups for individual aircraft (registration detail pages).

- **`faa-registry-pp-cli aircraft by-serial`** - Find aircraft by manufacturer serial number.
- **`faa-registry-pp-cli aircraft lookup`** - Look up an aircraft's full registration record by N-number (tail number, with or without the leading N).

### dealers

Live dealer-certificate searches.

- **`faa-registry-pp-cli dealers`** - Search FAA dealer certificates by dealer name.

### documents

Live document-index searches (recorded documents for collateral like airframes and engines).

- **`faa-registry-pp-cli documents`** - Search the FAA document index by collateral identifier.

### engines

Live engine-reference searches.

- **`faa-registry-pp-cli engines`** - Search the engine reference table by engine manufacturer and model.

### models

Live registry searches by aircraft make/model and reference data.

- **`faa-registry-pp-cli models`** - Search the aircraft model reference by manufacturer and model name, including the number of aircraft assigned to each model code.

### owners

Live registry searches by registered owner name.

- **`faa-registry-pp-cli owners`** - List all aircraft registered to an owner name (paginated).

### regions

Live registry searches by geography.

- **`faa-registry-pp-cli regions by-country`** - List US-registered aircraft whose owners are located in a given country.
- **`faa-registry-pp-cli regions by-state`** - List aircraft registered in a state and county (paginated).


## Output Formats

```bash
# Human-readable table (default in terminal, JSON when piped)
faa-registry-pp-cli dealers --name "AVIATION SALES"

# JSON for scripting and agents
faa-registry-pp-cli dealers --name "AVIATION SALES" --json

# Filter to specific fields
faa-registry-pp-cli dealers --name "AVIATION SALES" --json --select id,name,status

# Dry run — show the request without sending
faa-registry-pp-cli dealers --name "AVIATION SALES" --dry-run

# Agent mode — JSON + compact + no prompts in one flag
faa-registry-pp-cli dealers --name "AVIATION SALES" --agent
```

## Agent Usage

This CLI is designed for AI agent consumption:

- **Non-interactive** - never prompts, every input is a flag
- **Pipeable** - `--json` output to stdout, errors to stderr
- **Filterable** - `--select id,name` returns only fields you need
- **Previewable** - `--dry-run` shows the request without sending
- **Read-only by default** - this CLI does not create, update, delete, publish, send, or mutate remote resources
- **Offline-friendly** - sync/search commands can use the local SQLite store when available
- **Agent-safe by default** - no colors or formatting unless `--human-friendly` is set

Exit codes: `0` success, `2` usage error, `3` not found, `5` API error, `7` rate limited, `10` config error.

## Health Check

```bash
faa-registry-pp-cli doctor
```

Verifies configuration and connectivity to the API.

## Configuration

Run `faa-registry-pp-cli doctor` to see the resolved config, data, state, and cache directories. The platform-default config path is `~/.config/faa-registry-pp-cli/config.toml`; `--home`, `FAA_REGISTRY_HOME`, and per-kind env vars can relocate it.

Static request headers can be configured under `headers`; per-command header overrides take precedence.

## Troubleshooting
**Not found errors (exit code 3)**
- Check the resource ID is correct
- Use `search <term>` (offline, after `sync`) or a live inquiry like `owners --name` / `models --manufacturer` to find records; `which <task>` suggests the right command

### API-specific
- **HTTP 403 Access Denied from registry.faa.gov** — The FAA blocks non-browser user agents; the CLI sends a Chrome User-Agent automatically — if you overrode headers, remove the override. If it persists, Akamai may be rate-limiting: wait a minute and retry.
- **fleet report / hex resolve / expiring return 'no local database'** — Run `faa-registry-pp-cli sync` first — offline commands need the daily FAA database snapshot.
- **aircraft lookup returns a record but offline commands disagree** — The bulk database refreshes nightly (~11:30pm Central) while the live site updates each federal working day at midnight; run `sync` to re-align.
- **Owner search returns too many or zero rows** — The FAA matches from the start of the name in ALL CAPS; try the exact registered prefix (e.g. 'NETJETS SALES') or use offline `search` for fuzzy FTS matching.

## Sources & Inspiration

This CLI was built by studying these projects and resources:

- [**icao-nnumber_converter**](https://github.com/guillaumemichel/icao-nnumber_converter) — Python (25 stars)
- [**FAA-registry-checker**](https://github.com/Jxck-S/FAA-registry-checker) — Python (19 stars)
- [**scrape-faa-releasable-aircraft**](https://github.com/simonw/scrape-faa-releasable-aircraft) — Python (10 stars)
- [**adsbtrack**](https://github.com/frankea/adsbtrack) — Python (9 stars)
- [**faa-aircraft-registry**](https://github.com/ClearAerospace/faa-aircraft-registry) — Python (5 stars)
- [**Aircraft-Registration-Lookup-API**](https://github.com/njfdev/Aircraft-Registration-Lookup-API) — TypeScript (4 stars)
- [**aircraft-registration-lookup-api**](https://github.com/SkyLink-API/aircraft-registration-lookup-api) — TypeScript (3 stars)
- [**faaDb**](https://github.com/ThreeSixes/faaDb) — Python (1 stars)

Generated by [CLI Printing Press](https://github.com/mvanhorn/cli-printing-press)
