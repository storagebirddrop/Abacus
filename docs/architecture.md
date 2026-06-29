# Abacus Architecture

## Layers

```
┌─────────────────────────────────────────────┐
│                  Web UI                     │  React + Vite
│              (React + Vite)                 │  No business logic
└──────────────────────┬──────────────────────┘
                       │ HTTP / REST
┌──────────────────────▼──────────────────────┐
│                  REST API                   │  Go / chi router
│           /api/v1/...                       │  OpenAPI 3.1 spec
└──────────────────────┬──────────────────────┘
                       │
         ┌─────────────┼──────────────┐
         │             │              │
┌────────▼──────┐ ┌────▼─────┐ ┌─────▼──────────┐
│  Accounting   │ │ Reports  │ │   Portfolio    │
│    Engine     │ │  Engine  │ │    Summary     │
│ FIFO/AvgCost  │ │ CSV/PDF/ │ │ cross-wallet   │
│ LIFO/HIFO     │ │ XLSX/Tax │ │ aggregation    │
│ SpecificID    │ │ NL/DE/   │ └────────────────┘
│ Section 104   │ │ UK/US    │
└────────┬──────┘ └──────────┘
         │
┌────────▼──────────────────────────────────────┐
│               Ledger Engine                   │  Immutable entries
│        Blockchain + Metadata                  │  Audit trail
└──────────────────────┬────────────────────────┘
                       │
┌──────────────────────▼──────────────────────┐
│             Normalization Layer             │  Wallet-agnostic
│        domain.Transaction / UTXO           │  Standard structs
└──────────────────────┬──────────────────────┘
                       │
         ┌─────────────┴──────────────┐
         │                            │
┌────────▼──────┐          ┌──────────▼──────────┐
│   Importer    │          │     Sync Layer      │
│    Layer      │          │  address derivation │
│  Sparrow/     │          │  + blockchain fetch │
│  Nunchuk/     │          │  Esplora/Electrum/  │
│  Coldcard/    │          │  Bitcoin Core       │
│  Specter/     │          └─────────────────────┘
│  Electrum/    │
│  BIP329/BSMS  │
└────────┬──────┘
         │
┌────────▼──────────────────────────────────────┐
│                   SQLite                      │  golang-migrate
│              (single file)                    │  Immutable ledger
└───────────────────────────────────────────────┘
```

## Key Principles

**Each layer only depends on the layer directly below it.**

### Importer Layer
- Every wallet importer implements `WalletImporter`
- Core never contains wallet-specific code
- Auto-detection: upload any file, Abacus picks the right importer
- Shared parsers in `internal/importer/common/` (BSMS, BIP329)
- Supported: Sparrow, Nunchuk, Coldcard, Specter Desktop, Electrum, generic descriptor

### Sync Layer
- Derives addresses from output descriptor (wpkh, sh(wpkh), pkh)
- Fetches transaction history per address from a blockchain backend
- Gap limit 20 — stops scanning when 20 consecutive addresses have no history
- Backends: Esplora REST (default: mempool.space), Electrum TCP, Bitcoin Core RPC
- Writes through the same ledger pipeline as file importers

### Normalization Layer
- Converts wallet-specific data to `domain.*` structs
- Result is identical regardless of import source

### Ledger Engine
- Stores immutable `LedgerEntry` records
- Never edits blockchain data
- All corrections via `JournalEntry` (audit log)
- Blockchain data + metadata = financial ledger

### Accounting Engine
- Pure functions — no side effects, no DB access
- Input: UTXOs + spend times + price lookup
- Output: `CostBasisRecord` per UTXO
- Methods: FIFO, Average Cost, LIFO, HIFO, Specific Identification, UK Section 104
- UK Section 104 implements TCGA 1992 s.104/105/106A: same-day rule → 30-day rule → pool

### Report Engine
- Generic reports: transactions list, P&L, balance sheet (CSV / PDF / XLSX)
- Tax reports: NL Box 3, DE §23 EStG, UK HMRC CGT SA108, US IRS Form 8949 (CSV / PDF)
- UK tax report runs Section 104 in-memory; does not overwrite stored cost basis records

### API Layer
- Every UI feature is also available via API
- Frontend is a client — API is the product
- OpenAPI 3.1 spec in `docs/api/swagger.yaml`

## Data Integrity

- All satoshi values stored as `INTEGER` (never FLOAT)
- All timestamps stored as Unix epoch `INTEGER`
- `ledger_entries` is append-only
- Wallet private data never stored (no keys, no seeds, no encrypted wallet files)

## Plugin Architecture

```go
type WalletImporter interface {
    Name() string
    SupportedFormats() []string
    Detect(filename string, r io.ReadSeeker) bool
    Import(ctx context.Context, walletID string, r io.Reader) (*ImportResult, error)
}
```

To add a new wallet: implement the interface, call `importer.Register()` in `main.go`.
