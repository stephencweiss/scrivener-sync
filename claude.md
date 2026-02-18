# Claude Instructions

## Project Overview
- This repository contains `scriv-sync`, a Go CLI for bi-directional sync between Markdown folders and Scrivener projects.
- Entry point: `cmd/scriv-sync/main.go`.
- Core logic is in:
  - `internal/sync/` for sync planning, state, and reconciliation.
  - `internal/scrivener/` for Scrivener project read/write.
  - `internal/rtf/` for Markdown <-> RTF conversion.
  - `internal/config/` for user config handling.

## Local Development
- Build binary: `make build`
- Run tests: `make test` (or `go test ./...`)
- Format: `make fmt`
- Lint (if installed): `make lint`
- Cross-build: `make build-all`

## Code Expectations
- Keep changes focused and minimal.
- Preserve existing CLI behavior and flags unless explicitly changing UX.
- Prefer table-driven tests for new behavior in Go packages.
- Add or update tests when modifying sync logic, Scrivener parsing/writing, or RTF conversion.
- Avoid broad refactors unless requested.

## Validation Before Finalizing
- Run `go test ./...` after meaningful code changes.
- If touching formatting-sensitive code, run `go fmt ./...`.
- If linting is available, run `golangci-lint run`.

## Safety Notes
- Scrivener project structure and `.scrivx` XML must be preserved; avoid destructive writes.
- Sync state and conflict/deletion handling are critical paths. Treat behavior changes as high-risk and test accordingly.
- Respect user filesystem paths and do not hardcode machine-specific paths in code.

## Useful Reference Files
- `README.md` for CLI usage and config format.
- `TEST_PLAN.md` for existing test scope and expectations.
- `testdata/` for fixtures used by unit/integration tests.
