# CLAUDE.md

This file provides guidance for AI coding agents and contributors working in this repository.

## Project Summary

`scriv-sync` is a Go CLI for bi-directional synchronization between local Markdown files and Scrivener projects (`.scriv`).

Primary capabilities:
- Initialize project aliases and folder mappings
- Detect changes on both sides
- Sync in either direction (`pull`, `push`) or both (`sync`)
- Report pending changes (`status`)
- Track sync state for conflict/orphan detection

## Tech Stack

- Language: Go
- CLI framework: `cobra`
- Config format: YAML (`gopkg.in/yaml.v3`)
- State format: JSON

## Repository Layout

- `cmd/scriv-sync/main.go`: CLI entrypoint and command wiring
- `internal/config/`: global config loading/saving and project options
- `internal/sync/`: sync planning/execution, deletion/orphan handling, state tracking
- `internal/scrivener/`: Scrivener project reader/writer (`.scrivx` + `content.rtf`)
- `internal/rtf/`: RTF <-> Markdown conversion
- `testdata/`: fixture Scrivener projects and RTF samples

## Common Commands

- Build: `make build`
- Test: `make test`
- Format: `make fmt`
- Lint (if installed): `make lint`
- Cross-compile: `make build-all`

Run CLI locally:
- `./scriv-sync init --local <md_dir> --scriv <project.scriv> --alias <name>`
- `./scriv-sync sync <alias>`
- `./scriv-sync pull <alias>`
- `./scriv-sync push <alias>`
- `./scriv-sync status <alias>`

## Runtime Paths

Global data is stored under `~/.scriv-sync/`:
- Config: `~/.scriv-sync/config.yaml`
- Per-project state: `~/.scriv-sync/state/<alias>.json`

## Development Guidelines

- Keep changes scoped and package-local when possible.
- Prefer explicit error wrapping (`fmt.Errorf("...: %w", err)`).
- Preserve existing behavior for conflict/deletion defaults unless intentionally changing policy.
- Avoid changing Scrivener XML structure except where required by feature/fix.
- Add/adjust tests when modifying:
  - sync planning or conflict logic (`internal/sync/*_test.go`)
  - RTF conversion (`internal/rtf/rtf_test.go`)
  - Scrivener read/write behavior (`internal/scrivener/*_test.go`)

## Validation Checklist

Before finalizing changes:
1. Run `make fmt`.
2. Run `make test`.
3. If command behavior changed, run a quick manual smoke test with `status` or `sync --dry-run`.

## Notes for Agents

- Treat this as a stateful sync tool: regressions often appear in edge cases (orphan detection, conflict resolution, deleted files).
- Prefer deterministic tests over ad hoc fixtures.
- If adding CLI flags/commands, update `README.md` command docs accordingly.
