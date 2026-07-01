# Contributing to Abacus

Thanks for your interest in improving Abacus. This guide covers local setup and
the conventions CI enforces.

## Prerequisites

- **Go 1.26+**
- **Node 20+** (frontend)
- Optional: Docker (full-stack run), `appimagetool` (AppImage build)

## Project layout

See [`CLAUDE.md`](CLAUDE.md) for the directory map and architecture, and
[`docs/architecture.md`](docs/architecture.md) for the layer diagram. Key rule:
**layer discipline** — each layer imports only from the layer below, and core
never references a specific wallet importer.

## Build & test

The Go binary embeds `web/dist` via `go:embed`, so build the frontend first.

```bash
make frontend          # build web/dist (required before `go build`)
go build ./...
go vet ./...
go test ./...          # add -race for the sync/concurrency code

cd web
npm ci
npm run lint           # oxlint
npm test               # vitest
npm run build
```

> **Toolchain note:** `go.mod` declares `go 1.26.1`. `golangci-lint`/`gosec`
> built against an older Go will refuse to analyze the module, so they are not
> wired into CI yet — `go vet` is the lint gate for now. Use a matching Go
> toolchain locally.

CI (`.github/workflows/ci.yml`) runs the Go build/vet/test (+race), the frontend
lint/test/build, and a Docker build. **All must pass before merge.** The Go job
also enforces a **test-coverage floor** (currently 25%); a change that drops
total coverage below it fails CI, so add tests alongside new code.

## Conventions

- **Money is integer satoshis or cents — never floats.** See `CLAUDE.md` Key
  Invariants. Ledger financial fields are immutable; corrections go through
  `JournalEntry`.
- **No private data** — never store keys, seeds, or signing material; never parse
  `.mv.db`.
- **Branches:** never commit directly to `main`. Work on a feature branch, open
  a PR, wait for CI to go green, then squash-merge. Full workflow (including
  why tag pushes and remote branch deletion need a local machine) is in
  [`CLAUDE.md`](CLAUDE.md) → Git Workflow.
- **Commits:** conventional-commit style prefixes are used (`feat:`, `fix:`,
  `test:`, `docs:`, `chore:`, `ci:`). Keep commits focused.
- **Tests:** add/extend tests with behaviour changes. Pure functions
  (accounting, parsers) should have unit tests; API handlers get handler tests.

## Adding common things

- **A wallet importer:** implement `importer.WalletImporter`, register it in
  `cmd/abacus/main.go`, reuse `internal/importer/common`. See `CLAUDE.md`.
- **A cost-basis method:** add a pure `Run<Method>` in `internal/accounting/`,
  a constant in `internal/domain/accounting.go`, and wire it into the service +
  API validation.

## Pull requests

Fill in the PR template (summary, changes, test plan). Ensure CI is green. For
security issues, **do not** open a public PR/issue — see [`SECURITY.md`](SECURITY.md).
