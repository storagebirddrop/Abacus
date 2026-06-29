# Abacus — Codebase Guide

## Project

Bitcoin Accounting Engine. Self-hosted, wallet-agnostic, immutable ledger.
Motto: *Wallets manage bitcoin. Abacus manages the books.*

## Tech Stack

- **Backend**: Go 1.26, chi router, SQLite
- **Frontend**: React + Vite + TypeScript
- **Database**: SQLite via `golang-migrate`
- **Deploy**: Docker Compose

## Directory Map

```
cmd/abacus/main.go          entrypoint — registers importers, starts HTTP
internal/domain/            core types (no external deps)
internal/importer/          plugin importers
  importer.go               WalletImporter interface + registry
  service.go                import orchestration + ledger wiring
  common/                   shared parsers: bsms.go, bip329.go
  sparrow/                  Sparrow wallet importer
  nunchuk/                  Nunchuk wallet importer
  coldcard/                 Coldcard hardware wallet (descriptor export)
  specter/                  Specter Desktop (descriptor export)
  electrum/                 Electrum wallet (transaction history)
  descriptor/               Generic descriptor fallback (Jade, Passport, SeedSigner, etc.)
internal/normalizer/        wallet-agnostic normalization
internal/ledger/            ledger engine (Build: tx → entries + UTXOs)
internal/accounting/        cost basis calculations — all pure functions
  accounting.go             Service, PriceLookup, AccountingSummary
  fifo.go                   RunFIFO
  avgcost.go                RunAvgCost
  lifo.go                   RunLIFO
  hifo.go                   RunHIFO
  specificid.go             RunSpecificID
  section104.go             RunSection104 (UK HMRC — same-day, 30-day, pool)
  accounting_test.go        unit tests (no DB)
internal/repository/        SQLite data access
  wallet_repo.go
  transaction_repo.go
  ledger_repo.go
  utxo_repo.go
  cost_basis_repo.go
  price_snapshot_repo.go
  import_job_repo.go
  journal_repo.go
  label_repo.go
  sync_repo.go
internal/api/               HTTP handlers and router
  router.go                 all routes registered here
  wallets.go                wallet + import + transaction + label handlers
  accounting.go             accounting + price handlers
  ledger.go                 ledger entry + UTXO handlers
  reports.go                CSV/PDF/XLSX report handlers + tax report handler
  sync.go                   blockchain sync handlers
  portfolio.go              cross-wallet portfolio summary handler
internal/reports/           report generators
  data.go                   shared data structs (TransactionRow, PnLRow, BalanceRow)
  csv.go                    CSV generators for all three generic reports
  pdf.go                    PDF generators (maroto v2)
  excel.go                  Excel generators (excelize)
  tax_nl.go                 Netherlands Box 3 (Wet IB 2001 Art. 5.2)
  tax_de.go                 Germany §23 EStG
  tax_uk.go                 UK HMRC CGT Section 104 (TCGA 1992)
  tax_us.go                 US IRS Form 8949
internal/sync/              blockchain sync layer
  backend.go                BlockchainBackend interface
  derive.go                 DeriveAddresses from output descriptor
  service.go                SyncService orchestration
  esplora/                  Esplora REST backend (Mempool.space)
  electrum/                 Electrum JSON-RPC TCP backend
  bitcoincore/              Bitcoin Core JSON-RPC backend
internal/config/            config from env vars
migrations/                 numbered SQL migration files
docs/architecture.md        layer diagram and principles
docs/domain-model.md        all entities described
docs/api/swagger.yaml       OpenAPI 3.1 spec
web/                        React + Vite frontend
```

## Key Invariants

1. **Satoshis only** — all monetary values are `int64` satoshis or cents, never floats
2. **LedgerEntry financial fields are immutable** — never UPDATE `sats`, `fiat_amount`, `fiat_currency`, or `transaction_id` on `ledger_entries`. Metadata fields (`category`, `note`, `counterparty_id`) may be updated via `UpdateMetadata`. Use `JournalEntry` to record any correction.
3. **No private data** — never store keys, seeds, or signing material; never import `.mv.db` (encrypted)
4. **Layer discipline** — each layer only imports from the layer below
5. **Plugin importers** — core never references Sparrow/Nunchuk directly

## Adding a New Wallet Importer

1. Create `internal/importer/<wallet>/<wallet>.go`
2. Implement `importer.WalletImporter` interface
3. Call `importer.Register(<wallet>.New())` in `cmd/abacus/main.go`
4. Reuse `internal/importer/common/bsms.go` and `bip329.go` where applicable

## Adding a New Cost Basis Method

1. Create `internal/accounting/<method>.go` with a pure `Run<Method>(...)` function
2. Add a constant in `internal/domain/accounting.go`
3. Add a `case` in `accounting.Service.Run()` and in `api.AccountingHandler.RunAccounting()`

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

### Wallet & Import
- `GET/POST /api/v1/wallets` — list, create
- `GET/DELETE /api/v1/wallets/{id}` — get, delete
- `POST /api/v1/wallets/{id}/import` — upload Sparrow/Nunchuk/Coldcard/Specter/Electrum/BIP329/BSMS
- `GET /api/v1/wallets/{id}/import-jobs` — list import history
- `GET /api/v1/import-jobs/{id}` — job status

### Transactions & Labels
- `GET /api/v1/wallets/{id}/transactions` — paginated tx list
- `GET /api/v1/wallets/{id}/transactions/{txid}` — single tx
- `PATCH /api/v1/wallets/{id}/transactions/{txid}` — update metadata (category, note)
- `GET /api/v1/wallets/{id}/labels` — BIP329 labels
- `POST /api/v1/wallets/{id}/labels` — add label (501 — not yet implemented)

### Ledger & UTXOs
- `GET /api/v1/wallets/{id}/ledger` — paginated ledger entries
- `GET /api/v1/wallets/{id}/ledger/{entryID}` — single entry + journal audit trail
- `GET /api/v1/wallets/{id}/utxos` — UTXO set (`?unspent=true` to filter)

### Accounting
- `POST /api/v1/wallets/{id}/accounting/run` — run cost basis (`method`: fifo|avgcost|lifo|hifo|specificid|section104)
- `GET /api/v1/wallets/{id}/accounting/summary` — portfolio summary
- `GET /api/v1/wallets/{id}/accounting/cost-basis` — per-UTXO records

### Prices
- `GET /api/v1/prices` — price snapshots (currency + date range)
- `POST /api/v1/prices` — manual price entry

### Reports
- `GET /api/v1/wallets/{id}/reports/transactions` — transaction report (`format`: csv|pdf|xlsx)
- `GET /api/v1/wallets/{id}/reports/pnl` — P&L report (`format`: csv|pdf|xlsx)
- `GET /api/v1/wallets/{id}/reports/balance-sheet` — balance sheet (`format`: csv|pdf|xlsx)
- `GET /api/v1/wallets/{id}/reports/tax` — jurisdiction tax report (`jurisdiction`: nl|de|uk|us; `year`; `format`: csv|pdf)

### Sync
- `POST /api/v1/wallets/{id}/sync` — start blockchain sync job
- `GET /api/v1/wallets/{id}/sync-jobs` — list sync jobs
- `GET /api/v1/sync-jobs/{jobID}` — job status

### Portfolio
- `GET /api/v1/portfolio/summary` — cross-wallet portfolio summary

### System
- `GET /api/v1/health`
- `GET /api/v1/version`
