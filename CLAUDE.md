# Abacus — Codebase Guide

## Project

Bitcoin Accounting Engine. Self-hosted, wallet-agnostic, immutable ledger.
Motto: *Wallets manage bitcoin. Abacus manages the books.*

## Tech Stack

- **Backend**: Go 1.23, chi router, SQLite
- **Frontend**: React + Vite + TypeScript
- **Database**: SQLite via `golang-migrate`
- **Deploy**: Docker Compose

## Directory Map

```
cmd/abacus/main.go          entrypoint — registers importers, starts HTTP
internal/domain/            core types (no external deps)
internal/importer/          plugin importers
  importer.go               WalletImporter interface + registry
  common/                   shared parsers: bsms.go, bip329.go
  sparrow/                  Sparrow wallet importer
  nunchuk/                  Nunchuk wallet importer
internal/normalizer/        wallet-agnostic normalization
internal/ledger/            ledger engine
internal/accounting/        FIFO / Average Cost calculations
internal/repository/        SQLite data access
internal/api/               HTTP handlers and router
internal/config/            config from env vars
migrations/                 numbered SQL migration files
docs/architecture.md        layer diagram and principles
docs/domain-model.md        all entities described
docs/api/swagger.yaml       OpenAPI 3.1 spec
web/                        React + Vite frontend
```

## Key Invariants

1. **Satoshis only** — all monetary values are `int64` satoshis, never floats
2. **LedgerEntry is immutable** — never UPDATE ledger_entries; use JournalEntry for corrections
3. **No private data** — never store keys, seeds, or signing material
4. **Layer discipline** — each layer only imports from the layer below
5. **Plugin importers** — core never references Sparrow/Nunchuk directly

## Adding a New Wallet Importer

1. Create `internal/importer/<wallet>/<wallet>.go`
2. Implement `importer.WalletImporter` interface
3. Call `importer.Register(<wallet>.New())` in `cmd/abacus/main.go`
4. Reuse `internal/importer/common/bsms.go` and `bip329.go` where applicable

## Commands

```bash
go build ./...              build
go vet ./...                lint
go test ./...               test
docker compose up --build   full stack
```

## Database

SQLite at `DB_PATH` (default `./abacus.db`).
Migrations in `migrations/` — files are numbered `001_`, `002_`, etc.
Run migrations via `golang-migrate` on startup.

## API

All routes registered in `internal/api/router.go`.
Spec: `docs/api/swagger.yaml`.
Unimplemented routes return `501 Not Implemented`.
