# Changelog

All notable changes to this project are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project aims to follow [Semantic Versioning](https://semver.org/).

## [Unreleased]

Everything built so far lives here until the first tagged release (`v1.0.0`).

### Added
- Wallet importers: Sparrow, Nunchuk, Coldcard, Specter, Electrum, and a generic
  descriptor fallback (BSMS / BIP329 support).
- Immutable ledger engine with UTXO tracking and a journal audit trail.
- Cost-basis engines: FIFO, Average Cost, LIFO, HIFO, Specific ID, UK Section 104.
- Jurisdiction tax reports: NL (Box 3), DE (§23 EStG), UK (HMRC CGT), US (Form 8949).
- Generic reports (transactions, P&L, balance sheet) as CSV / PDF / XLSX.
- Blockchain sync (opt-in): Esplora, Electrum, Bitcoin Core backends; runtime
  configuration via the Settings page.
- Cross-wallet portfolio summary.
- React dashboard; self-contained binary + Linux AppImage packaging.
- API security: optional bearer-token auth (`API_TOKEN`) and per-IP rate limiting.

### Changed
- Germany §23 EStG: year-dependent Freigrenze (€1,000 from 2024) and loss
  offsetting within the year.

### Fixed
- Average-cost precision loss (multiply-before-divide).
- Electrum sync BTC→sats conversion now exact (no float truncation).
- Blockchain sync: race-free Esplora rate limiter; cancellable, bounded sync
  goroutine with graceful shutdown.
- Import uploads bounded (413 over the 32 MiB cap) to prevent memory exhaustion.

[Unreleased]: https://github.com/storagebirddrop/Abacus/commits/main
