# Changelog

All notable changes to this project are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project aims to follow [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added
- **Exchange account imports** — five new exchange importers: Bitvavo, Bitonic,
  Kraken (ledger CSV with refid-pairing), Coinbase, and Strike (Lightning +
  trades). Upload a CSV export from any of these exchanges to import buy/sell/
  deposit/withdrawal/lightning_receive/lightning_send trades.
- `ExchangeTrade` domain entity and `exchange_trades` table (migration 006);
  `ledger_entries.transaction_id` made nullable with a new `exchange_trade_id`
  FK so exchange trades appear in the transaction history.
- `RunExchangeFIFO` accounting engine: FIFO lot-matching with partial-lot
  splitting for exchange trade histories; fiat cost/proceeds recorded at
  trade time (no external price lookup needed).
- Frontend **Add Wallet** dialog now has a wallet-type toggle — choose
  "On-chain wallet" (existing flow) or "Exchange account" (selects exchange
  from a dropdown, hides descriptor field).
- Shared CSV helpers (`ParseBTCSats`, `ParseFiatCents`) in
  `internal/importer/common/exchange_csv.go`.

### Changed
- Documentation: version-history accuracy pass (CHANGELOG entries for
  0.1.1/0.1.2, dead `VERSION` env var removed, stale version references in
  README/backlog generalized) and a backlog accuracy pass checking off items
  that had already landed in code.
- `SECURITY.md`/`.env.example`/`README.md`: recommend `openssl rand -hex 32`
  for `API_TOKEN`.

## [0.1.2] - 2026-07-01

### Fixed
- Docs accuracy pass: removed a phantom "Bitcoin Core" sync-backend claim
  that was never implemented; documented the actual supported backends.

### Added
- Drag-and-drop import zone in the Import tab — drop a wallet export file
  directly onto the drop zone (or click to browse); upload starts
  immediately on file selection.

## [0.1.1] - 2026-07-01

### Fixed
- Removed an unused `Button` import and vestigial form-submit calls from
  `ImportTab`.

### Changed
- Dependency bumps: GitHub Actions, npm minor/patch group, `jsdom`.

## [0.1.0] - 2026-07-01

First tagged release. All phases (0–7) and two rounds of independent audit
remediation are included.

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
- Web UI attaches the `API_TOKEN` bearer token when saved on the Settings page,
  so enabling auth no longer breaks the bundled UI.
- Server-side transaction search, status filter, and sort on the API and the
  Transactions tab (works across the whole history, not just the current page).
- Frontend: dark mode + accessibility pass, responsive sidebar, toast +
  confirm dialogs, a React error boundary, and surfaced initial-load errors.
- Release artifacts are signed with cosign (keyless) and published with a
  `sha256sums.txt`; verification steps are in the release notes.
- `NOTICE` file attributing bundled third-party components.
- CI: Go coverage floor gate and `npm audit` on the frontend dependencies.

### Changed
- Germany §23 EStG: year-dependent Freigrenze (€1,000 from 2024) and loss
  offsetting within the year.
- Version is now baked into the binary at build time via
  `-ldflags "-X main.version=<tag>"` rather than read from a runtime env var;
  untagged/dev builds report `dev`.

### Fixed
- Average-cost precision loss (multiply-before-divide).
- LIFO: only fall back to the disposal-time price when no lot was matched, so a
  matched zero-cost lot is no longer overridden by the disposed UTXO's own price.
- Germany §23 EStG Freigrenze is strictly-less-than the threshold ("weniger
  als"), so a gain of exactly €600/€1,000 is taxable rather than exempt.
- Electrum sync BTC→sats conversion now exact (no float truncation).
- Blockchain sync: race-free Esplora rate limiter; cancellable, bounded sync
  goroutine with graceful shutdown; sync endpoints return 404 (not 400/empty)
  for a non-existent wallet.
- Rate limiter ignores spoofable `X-Forwarded-For` unless `TRUST_PROXY` is set;
  the auth health/version exemption is an exact path match.
- Import uploads bounded (413 over the 32 MiB cap) to prevent memory exhaustion.

[Unreleased]: https://github.com/storagebirddrop/Abacus/compare/v0.1.2...HEAD
[0.1.2]: https://github.com/storagebirddrop/Abacus/releases/tag/v0.1.2
[0.1.1]: https://github.com/storagebirddrop/Abacus/releases/tag/v0.1.1
[0.1.0]: https://github.com/storagebirddrop/Abacus/releases/tag/v0.1.0
