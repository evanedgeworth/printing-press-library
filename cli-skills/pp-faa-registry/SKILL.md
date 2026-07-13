---
name: pp-faa-registry
description: "Every FAA aircraft lookup the registry website offers, plus a daily-synced offline copy of the entire US registry that unlocks fleet reports, hex decoding, ownership history, and expiration alerts no other tool has. Trigger phrases: `look up tail number`, `whose plane is N101DQ`, `who owns this aircraft`, `decode this mode s hex`, `check FAA registration`, `is this N-number available`, `NetJets fleet list`, `use faa-registry`, `run faa-registry`."
author: "Omar Shahine"
license: "Apache-2.0"
argument-hint: "<command> [args] | install cli|mcp"
allowed-tools: "Read Bash"
metadata:
  openclaw:
    requires:
      bins:
        - faa-registry-pp-cli
    install:
      - kind: go
        bins: [faa-registry-pp-cli]
        module: github.com/mvanhorn/printing-press-library/library/travel/faa-registry/cmd/faa-registry-pp-cli
---
<!-- GENERATED FILE — DO NOT EDIT.
     This file is a verbatim mirror of library/travel/faa-registry/SKILL.md,
     regenerated post-merge by tools/generate-skills/. Hand-edits here are
     silently overwritten on the next regen. Edit the library/ source instead.
     See the repository agent guide, section "Generated artifacts: registry.json, cli-skills/". -->

# FAA Aircraft Registry — Printing Press CLI

## Prerequisites: Install the CLI

This skill drives the `faa-registry-pp-cli` binary. **You must verify the CLI is installed before invoking any command from this skill.** If it is missing, install it first:

1. Install via the Printing Press installer. It defaults binaries to `$HOME/.local/bin` on macOS/Linux and `%LOCALAPPDATA%\Programs\PrintingPress\bin` on Windows:
   ```bash
   npx -y @mvanhorn/printing-press-library install faa-registry --cli-only
   ```
2. Verify: `faa-registry-pp-cli --version`
3. Ensure the reported install directory is on `$PATH` for the agent/runtime that will invoke this skill.

If the `npx` install fails (no Node, offline, etc.), fall back to a direct Go install (requires Go 1.26.4 or newer). This installs into `$GOPATH/bin` (default `$HOME/go/bin`), so add that directory to `$PATH` instead:

```bash
go install github.com/mvanhorn/printing-press-library/library/travel/faa-registry/cmd/faa-registry-pp-cli@latest
```

If `--version` reports "command not found" after install, the runtime cannot see the binary directory on `$PATH`. Do not proceed with skill commands until verification succeeds.

The FAA registry is browser-only and un-scriptable; existing wrappers parse stale CSVs or do one-off hex math. This CLI does both live and local: every inquiry page as a typed-JSON command, and the full 315K-aircraft registry (plus deregistered and reserved data) in SQLite for fleet report, hex resolve, aircraft history, expiring, and instant offline lookups.

## When to Use This CLI

Use this CLI whenever a task involves US aircraft identity: resolving a tail number or Mode S hex to an owner and aircraft type, checking registration status or expiration, researching who owns a fleet or a model class, or verifying N-number availability. It is authoritative (FAA source data), free, and works offline after one sync. It does not track live flights — pair it with a flight-tracking tool for positions.

## When Not to Use This CLI

Do not activate this CLI for requests that require creating, updating, deleting, publishing, commenting, upvoting, inviting, ordering, sending messages, booking, purchasing, or changing remote state. This printed CLI exposes read-only commands for inspection, export, sync, and analysis.

## Unique Capabilities

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

## Command Reference

**aircraft** — Live FAA registry lookups for individual aircraft (registration detail pages).

- `faa-registry-pp-cli aircraft by-serial` — Find aircraft by manufacturer serial number.
- `faa-registry-pp-cli aircraft lookup` — Look up an aircraft's full registration record by N-number (tail number, with or without the leading N).

**dealers** — Live dealer-certificate searches.

- `faa-registry-pp-cli dealers` — Search FAA dealer certificates by dealer name.

**documents** — Live document-index searches (recorded documents for collateral like airframes and engines).

- `faa-registry-pp-cli documents` — Search the FAA document index by collateral identifier.

**engines** — Live engine-reference searches.

- `faa-registry-pp-cli engines` — Search the engine reference table by engine manufacturer and model.

**models** — Live registry searches by aircraft make/model and reference data.

- `faa-registry-pp-cli models` — Search the aircraft model reference by manufacturer and model name

**owners** — Live registry searches by registered owner name.

- `faa-registry-pp-cli owners` — List all aircraft registered to an owner name (paginated).

**regions** — Live registry searches by geography.

- `faa-registry-pp-cli regions by-country` — List US-registered aircraft whose owners are located in a given country.
- `faa-registry-pp-cli regions by-state` — List aircraft registered in a state and county (paginated).


### Finding the right command

When you know what you want to do but not which command does it, ask the CLI directly:

```bash
faa-registry-pp-cli which "<capability in your own words>"
```

`which` resolves a natural-language capability query to the best matching command from this CLI's curated feature index. Exit code `0` means at least one match; exit code `2` means no confident match — fall back to `--help` or use a narrower query.

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

## Auth Setup

No authentication required.

Run `faa-registry-pp-cli doctor` to verify setup.

## Agent Mode

Add `--agent` to any command. Expands to: `--json --compact --no-input --no-color --yes`.

- **Pipeable** — JSON on stdout, errors on stderr
- **Filterable** — `--select` keeps a subset of fields. Dotted paths descend into nested structures; arrays traverse element-wise. Critical for keeping context small on verbose APIs:

  ```bash
  faa-registry-pp-cli dealers --name "AVIATION SALES" --agent --select id,name,status
  ```
- **Previewable** — `--dry-run` shows the request without sending
- **Offline-friendly** — sync/search commands can use the local SQLite store when available
- **Non-interactive** — never prompts, every input is a flag
- **Read-only** — do not use this CLI for create, update, delete, publish, comment, upvote, invite, order, send, or other mutating requests

### Response envelope

Commands that read from the local store or the API wrap output in a provenance envelope:

```json
{
  "meta": {"source": "live" | "local", "synced_at": "...", "reason": "..."},
  "results": <data>
}
```

Parse `.results` for data and `.meta.source` to know whether it's live or local. A human-readable `N results (live)` summary is printed to stderr only when stdout is a terminal AND no machine-format flag (`--json`, `--csv`, `--compact`, `--quiet`, `--plain`, `--select`) is set — piped/agent consumers and explicit-format runs get pure JSON on stdout.

## Paths and state

Agents should treat the CLI's path resolver as part of the runtime contract:

- Use `--home <dir>` for one invocation, or set `FAA_REGISTRY_HOME=<dir>` to relocate all four path kinds under one root.
- Use per-kind env vars only when a specific kind must diverge: `FAA_REGISTRY_CONFIG_DIR`, `FAA_REGISTRY_DATA_DIR`, `FAA_REGISTRY_STATE_DIR`, `FAA_REGISTRY_CACHE_DIR`.
- Resolution order is per-kind env var, `--home`, `FAA_REGISTRY_HOME`, XDG (`XDG_CONFIG_HOME`, `XDG_DATA_HOME`, `XDG_STATE_HOME`, `XDG_CACHE_HOME`), then platform defaults.
- `config` contains settings like `config.toml` and profiles. `data` contains the local registry database (`registry.db`, built by `sync`) and the framework `data.db`. `state` contains persisted queries and `teach.log`. `cache` contains regenerable HTTP/cache files. Set `FAA_REGISTRY_DB` to point the registry database elsewhere.
- Run `faa-registry-pp-cli doctor --fail-on warn` to surface path and credential-location warnings. `agent-context` exposes a schema v4 `paths` block for agents that need the resolved dirs.
- For MCP, pass relocation through the MCP host config. The MCP binary does not inherit CLI flags:

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

Fleet precedence: an inherited per-kind env var overrides an explicit `--home` for that kind. Use `FAA_REGISTRY_HOME` or per-kind vars as durable fleet levers, and use `--home` only for a single invocation. Relocation is not reversible by unsetting env vars; move files manually before clearing `FAA_REGISTRY_HOME`, or `doctor` will not find credentials left under the former root.

## Agent Feedback

When you (or the agent) notice something off about this CLI, record it:

```
faa-registry-pp-cli feedback "the --since flag is inclusive but docs say exclusive"
faa-registry-pp-cli feedback --stdin < notes.txt
faa-registry-pp-cli feedback list --json --limit 10
```

Entries are stored locally as `feedback.jsonl` under the resolved data dir. They are never POSTed unless `FAA_REGISTRY_FEEDBACK_ENDPOINT` is set AND either `--send` is passed or `FAA_REGISTRY_FEEDBACK_AUTO_SEND=true`. Default behavior is local-only.

Write what *surprised* you, not a bug report. Short, specific, one line: that is the part that compounds.

## Output Delivery

Every command accepts `--deliver <sink>`. The output goes to the named sink in addition to (or instead of) stdout, so agents can route command results without hand-piping. Three sinks are supported:

| Sink | Effect |
|------|--------|
| `stdout` | Default; write to stdout only |
| `file:<path>` | Atomically write output to `<path>` (tmp + rename) |
| `webhook:<url>` | POST the output body to the URL (`application/json` or `application/x-ndjson` when `--compact`) |

Unknown schemes are refused with a structured error naming the supported set. Webhook failures return non-zero and log the URL + HTTP status on stderr.

## Named Profiles

A profile is a saved set of flag values, reused across invocations. Use it when a scheduled agent calls the same command every run with the same configuration - HeyGen's "Beacon" pattern.

```
faa-registry-pp-cli profile save briefing --json
faa-registry-pp-cli --profile briefing dealers --name "AVIATION SALES"
faa-registry-pp-cli profile list --json
faa-registry-pp-cli profile show briefing
faa-registry-pp-cli profile delete briefing --yes
```

Explicit flags always win over profile values; profile values win over defaults. `agent-context` lists all available profiles under `available_profiles` so introspecting agents discover them at runtime.

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 2 | Usage error (wrong arguments) |
| 3 | Resource not found |
| 5 | API error (upstream issue) |
| 7 | Rate limited (wait and retry) |
| 10 | Config error |

## Argument Parsing

Parse `$ARGUMENTS`:

1. **Empty, `help`, or `--help`** → show `faa-registry-pp-cli --help` output
2. **Starts with `install`** → ends with `mcp` → MCP installation; otherwise → see Prerequisites above
3. **Anything else** → Direct Use (execute as CLI command with `--agent`)

## MCP Server Installation

1. Install the MCP server:
   ```bash
   go install github.com/mvanhorn/printing-press-library/library/travel/faa-registry/cmd/faa-registry-pp-mcp@latest
   ```
2. Register with Claude Code:
   ```bash
   claude mcp add faa-registry-pp-mcp -- faa-registry-pp-mcp
   ```
3. Verify: `claude mcp list`

## Direct Use

1. Check if installed: `which faa-registry-pp-cli`
   If not found, offer to install (see Prerequisites at the top of this skill).
2. Match the user query to the best command from the Unique Capabilities and Command Reference above.
3. Execute with the `--agent` flag:
   ```bash
   faa-registry-pp-cli <command> [subcommand] [args] --agent
   ```
4. If ambiguous, drill into subcommand help: `faa-registry-pp-cli <command> --help`.
