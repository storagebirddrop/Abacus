# Abacus — Post-Phase-7 Backlog

Parked ideas to work through after Phase 7 (blockchain sync) is complete.
Listed roughly in implementation order, not priority — reprioritise as needed.

---

## 1. Ledger & UTXO endpoints (currently 501)

Implement the two routes that exist in the router but return 501:

- `GET /api/v1/wallets/{id}/ledger` — paginated ledger entries
- `GET /api/v1/wallets/{id}/ledger/{entryID}` — single entry + journal audit trail
- `GET /api/v1/wallets/{id}/utxos` — current UTXO set

---

## 2. Transaction metadata editing

Implement the PATCH endpoint:

- `PATCH /api/v1/wallets/{id}/transactions/{txid}` — update category, note, counterparty
- Each change appends a `JournalEntry` (immutable audit trail)
- Wire into the dashboard: inline edit on the transaction row

---

## 3. Additional cost basis methods

Extend the accounting engine with two more methods alongside FIFO and AvgCost:

- **LIFO** (Last-In-First-Out) — common in some jurisdictions
- **HIFO** (Highest-In-First-Out) — minimises short-term gain
- **Specific Identification** — user manually matches lots to disposals

All as pure functions following the existing `fifo.go` / `avgcost.go` pattern.

---

## 4. Multi-wallet portfolio view

Aggregate across all wallets in a single dashboard view:

- Combined balance (total BTC held, total cost basis)
- Combined P&L summary
- Cross-wallet accounting run (single currency, single method)
- New API route: `GET /api/v1/portfolio/summary`

---

## 5. Tax report templates per jurisdiction

Pre-built report layouts that match local tax authority requirements:

- **NL** — Belastingdienst box 3 (wealth tax on 1 Jan valuation)
- **DE** — Freigrenzen, Haltefrist (1-year exemption)
- **UK** — HMRC Section 104 pooling rules
- **US** — Form 8949 / Schedule D layout

Each as a new report format alongside the existing CSV/PDF/XLSX generators.

---

## 6. Label & POST label endpoint (currently 501)

- `POST /api/v1/wallets/{id}/labels` — create or update a BIP329 label
- Wire into the dashboard for address/tx annotation
- Export labels back as BIP329 `.jsonl`

---

## 7. Linux AppImage packaging

Distribute Abacus as a single-file portable Linux executable:

- `packaging/appimage/Abacus.desktop` — XDG desktop entry
- `packaging/appimage/icon.png` — 256×256 app icon
- `Makefile` target `appimage`:
  1. Build frontend (`npm run build`)
  2. Build Go binary inside an Ubuntu 20.04 Docker container (old glibc = broad distro compatibility; CGO required for go-sqlite3)
  3. Assemble AppDir: `Abacus.AppDir/usr/bin/abacus`, `Abacus.AppDir/usr/share/abacus/web/dist/`, `.desktop`, icon, `AppRun` symlink
  4. Run `appimagetool Abacus.AppDir Abacus-x86_64.AppImage`
- Output: `dist/Abacus-x86_64.AppImage` — users `chmod +x` and run; no install needed
- The binary serves the bundled `web/dist/` via `FRONTEND_DIR` (already supported by config)

---
