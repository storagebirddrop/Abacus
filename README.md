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
Import your wallet data and get an immutable financial ledger with multi-method cost basis accounting, blockchain sync, and jurisdiction-specific tax reports.

## What Abacus does

- Imports wallet data from **Sparrow**, **Nunchuk**, **Coldcard**, **Specter Desktop**, **Electrum**, and any wallet that exports a descriptor or BIP329 labels
- Builds an **immutable ledger** from your transaction history
- Runs **FIFO, Average Cost, LIFO, HIFO, Specific ID, and UK Section 104** cost basis calculations
- Tracks **UTXO age and cost basis** per coin
- Syncs transaction history live via **Esplora, Electrum, or Bitcoin Core**
- Generates **tax reports** for the Netherlands (Box 3), Germany (§23 EStG), United Kingdom (HMRC CGT / Section 104), and United States (IRS Form 8949)
- Generates **generic reports** (balance sheet, P&L, CSV/PDF/Excel)
- Provides a **REST API** for all accounting data
- Serves a **React dashboard** at `localhost:8080`

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

1. Export your wallet data from your wallet app
2. Open Abacus → Wallets → Import
3. Upload the file — Abacus detects the format automatically

**Supported formats:**
- Sparrow JSON wallet export, transaction CSV
- Nunchuk JSON export
- Coldcard `coldcard-export.json`
- Specter Desktop JSON descriptor export
- Electrum JSON wallet export (unencrypted)
- BSMS files (BIP-129, multisig descriptor)
- BIP329 label files (`.jsonl`)
- Any JSON with a `descriptor` or `desc` field (Jade, Passport, SeedSigner, etc.)

## Architecture

```
Blockchain (Esplora / Electrum / Bitcoin Core)
    ↓
Sync Layer (address derivation → tx fetch → persist)
    ↓
Importer (Sparrow / Nunchuk / Coldcard / Specter / Electrum / BIP329 / BSMS)
    ↓
Normalization (wallet-agnostic)
    ↓
Ledger Engine (immutable)
    ↓
Accounting Engine (FIFO / AvgCost / LIFO / HIFO / SpecificID / Section 104)
    ↓
Report Engine (CSV / PDF / Excel / Tax Reports)
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

| Item | Status |
|---|---|
| Phase 0 — Architecture & Foundation | ✅ |
| Phase 1 — Sparrow + Nunchuk importer | ✅ |
| Phase 2 — Ledger Engine | ✅ |
| Phase 3 — Accounting Engine (FIFO / Avg Cost) | ✅ |
| Phase 4 — Dashboard (React / Vite) | ✅ |
| Phase 5 — Reports (PDF / Excel / CSV) | ✅ |
| Phase 6 — Extended wallet importers (Coldcard, Specter, Electrum, generic) | ✅ |
| Phase 7 — Blockchain sync (Esplora / Electrum / Bitcoin Core) | ✅ |
| Backlog 1 — Ledger & UTXO endpoints | ✅ |
| Backlog 5 — Jurisdiction tax reports (NL / DE / UK / US) + LIFO / HIFO / Section 104 | ✅ |
| Backlog 6 — BIP329 label export (`POST /labels`) | 🔜 |
| Backlog 7 — Linux AppImage packaging | 🔜 |

## License

MIT
