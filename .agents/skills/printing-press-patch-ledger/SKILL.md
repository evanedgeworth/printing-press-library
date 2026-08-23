---
name: printing-press-patch-ledger
description: Use in printing-press-library when code inside an existing published CLI is hand-modified and the customization must survive regeneration. Do not use for README-only or SKILL-only edits.
---

# Printing Press Patch Ledger

## `.printing-press-patches/` records library-side customizations

If you modify a published CLI under `library/<cat>/<slug>/` beyond what the generator produced, **catalog the change as one file per patch under `.printing-press-patches/`** at the CLI's root (parallel to `.printing-press.json`). SKILL.md / README.md edits are owned by `tools/sweep-canonical/` or direct edit and don't need a patch entry; this convention is for code-level customizations.

```
library/<cat>/<slug>/.printing-press-patches/
  <id>.json     one self-contained patch per file
  _meta.json    only when the CLI carries CLI-global lists (see below)
  .gitkeep      present even at zero patches (git won't track an empty dir)
```

Each `<id>.json` is a single self-contained patch object:

```json
{
  "schema_version": 2,
  "id": "short-identifier",
  "applied_at": "YYYY-MM-DD",
  "base_run_id": "<copy from .printing-press.json>",
  "base_printing_press_version": "<copy from .printing-press.json>",
  "summary": "The durable behavior a regen must preserve — the lesson, not the diff (one sentence).",
  "reason": "Why this API/runtime needs it and the failure mode it prevents (one or two sentences).",
  "files": ["internal/cli/foo.go"],
  "validated_outcome": "Optional: non-obvious test result that confirms the fix.",
  "upstream_issue": "Optional: https://github.com/mvanhorn/cli-printing-press/issues/<n>"
}
```

The filename is `slugify(id).json`. **One PR = one new file**, so two concurrent PRs on the same CLI write different files and never conflict on patch metadata — the whole point of the directory layout ([mvanhorn/cli-printing-press#2496](https://github.com/mvanhorn/cli-printing-press/issues/2496)). CLI-global lists that don't belong to any single patch (`upstream_tracking`, `deferred_to_upstream`, …) live in `_meta.json`; that is the one remaining shared file and it changes rarely.

**Why this is not the legacy single-array file.** This used to be a single `.printing-press-patches.json` with a `patches: [...]` array that every PR appended to — a guaranteed merge conflict between any two same-CLI PRs. That shape is now **converted automatically**: the post-merge `normalize-patches.yml` workflow (source: `.github/scripts/normalize-patches/normalize.py`) explodes any legacy single-array file that lands on `main` into this directory. Older Printing Press versions still emit the single file, and CI tolerates it on PRs — you don't have to convert it yourself, the normalizer does it after merge. New work should write the directory form directly.

Each `<id>.json` is an **index entry**, not a second copy of the diff — and the bar is *altitude*, not just brevity. A fresh print overwrites this whole tree; these entries are what survive to steer the next regen away from re-making the mistake, so write each one as a **reprint-guard**: capture the durable behavioral contract or API/runtime reality the customization encodes, not the line-level changes (git already has those). Litmus test — the entry should still read true after a full regen.

- **Flag moving targets instead of enshrining them.** *"The client version is whatever the live desktop currently sends; a regen must re-discover it"* — not *"set User-Agent to 7.299.0"*.
- **Let the `id` read as the lesson.** `d6-read-only-applies-to-all-desktop-token-sources`, not `add-tokensource-enum`.
- **Keep `summary`/`reason` short.** A table of field renames or SQL transformations means you've dropped to changelog altitude; that belongs in the commit message, not here.

**Delete stale workaround entries.** A `reason` field that describes a verifier or pipeline bug (e.g. *"the package verifier currently treats X as Y; this entry exists to silence the false positive"*) is a placeholder, not a real customization. When the underlying bug is fixed, delete the file — leaving it behind makes future contributors think there's a hand-edit to preserve when there isn't.

A worked example lives at `library/payments/kalshi/.printing-press-patches/`.
