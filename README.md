# Abacus

**Bitcoin Accounting Engine**

```
◉
│││
│││
│││
```

*Wallets manage bitcoin. Abacus manages the books.*

---

Abacus is an open-source, self-hosted Bitcoin accounting engine.
Import your wallet data from Sparrow or Nunchuk and get an immutable financial ledger with FIFO and Average Cost accounting.

## What Abacus does

- Imports wallet data from **Sparrow** and **Nunchuk** (descriptor, BSMS, BIP329 labels)
- Builds an **immutable ledger** from your transaction history
- Runs **FIFO and Average Cost** cost basis calculations
- Tracks **UTXO age and cost basis** per coin
- Provides a **REST API** for all accounting data
- Generates **reports** (balance sheet, P&L, CSV/PDF/Excel)

## What Abacus does NOT do

- Store private keys or seed phrases
- Connect to your wallet or sign transactions
- Replace Sparrow, Nunchuk, or any other wallet

## Quick Start

```bash
git clone https://github.com/storagebirddrop/abacus
cd abacus
cp .env.example .env
docker compose up --build
```

Open http://localhost:8080

## Import your wallet

1. Export your wallet data from Sparrow or Nunchuk
2. Open Abacus → Wallets → Import
3. Upload the file — Abacus detects the format automatically

**Supported formats:**
- Sparrow JSON wallet export
- Sparrow transaction CSV
- Nunchuk JSON export
- BSMS files (BIP-129, multisig descriptor)
- BIP329 label files (.jsonl)

## Architecture

```
Importer (Sparrow / Nunchuk / BSMS / BIP329)
    ↓
Normalization (wallet-agnostic)
    ↓
Ledger Engine (immutable)
    ↓
Accounting Engine (FIFO / Average Cost)
    ↓
REST API
    ↓
Web UI
```

See [docs/architecture.md](docs/architecture.md) for the full design.

## Development

```bash
# Backend
go build ./...
go test ./...

# Frontend
cd web && npm install && npm run dev

# Full stack
docker compose up --build
```

API spec: [docs/api/swagger.yaml](docs/api/swagger.yaml)

## Roadmap

| Phase | Status |
|---|---|
| 0 — Architecture & Foundation | ✅ |
| 1 — Sparrow + Nunchuk importer | 🔜 |
| 2 — Ledger Engine | 🔜 |
| 3 — Accounting Engine (FIFO / Avg Cost) | 🔜 |
| 4 — REST API + Dashboard | 🔜 |
| 5 — Reports (PDF / Excel / CSV) | 🔜 |
| 6 — Plugin system + additional wallets | 🔜 |

## License

MIT
