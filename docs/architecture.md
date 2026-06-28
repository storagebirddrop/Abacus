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
┌──────────────────────▼──────────────────────┐
│             Accounting Engine               │  Pure functions
│          FIFO / Average Cost                │  Deterministic
└──────────────────────┬──────────────────────┘
                       │
┌──────────────────────▼──────────────────────┐
│               Ledger Engine                 │  Immutable entries
│        Blockchain + Metadata                │  Audit trail
└──────────────────────┬──────────────────────┘
                       │
┌──────────────────────▼──────────────────────┐
│             Normalization Layer             │  Wallet-agnostic
│        domain.Transaction / UTXO           │  Standard structs
└──────────────────────┬──────────────────────┘
                       │
┌──────────────────────▼──────────────────────┐
│              Importer Layer                 │  Plugin-based
│      Sparrow │ Nunchuk │ (future)           │  WalletImporter interface
└──────────────────────┬──────────────────────┘
                       │
┌──────────────────────▼──────────────────────┐
│                  SQLite                     │  golang-migrate
│              (single file)                  │  Immutable ledger
└─────────────────────────────────────────────┘
```

## Key Principles

**Each layer only depends on the layer directly below it.**

### Importer Layer
- Every wallet importer implements `WalletImporter`
- Core never contains wallet-specific code
- Auto-detection: upload any file, Abacus picks the right importer
- Shared parsers in `internal/importer/common/` (BSMS, BIP329)

### Normalization Layer
- Converts wallet-specific data to `domain.*` structs
- Result is identical regardless of import source

### Ledger Engine
- Stores immutable `LedgerEntry` records
- Never edits blockchain data
- All corrections via `JournalEntry` (audit log)
- Blockchain data + metadata = financial ledger

### Accounting Engine
- Pure functions — no side effects
- Input: ledger entries + price snapshots
- Output: `CostBasisRecord` per UTXO
- Supported: FIFO, Average Cost
- Future: LIFO, HIFO, Specific Identification

### API Layer
- Every UI feature is also available via API
- Frontend is a client — API is the product
- OpenAPI 3.1 spec in `docs/api/swagger.yaml`

## Data Integrity

- All satoshi values stored as `INTEGER` (never FLOAT)
- All timestamps stored as Unix epoch `INTEGER`
- `ledger_entries` is append-only
- Wallet private data never stored (no keys, no seeds)

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
