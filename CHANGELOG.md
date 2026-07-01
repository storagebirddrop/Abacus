# Changelog

All notable changes to this project are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project aims to follow [Semantic Versioning](https://semver.org/).

## [Unreleased]

_No unreleased changes at this time._

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

[Unreleased]: https://github.com/storagebirddrop/Abacus/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/storagebirddrop/Abacus/releases/tag/v0.1.0
