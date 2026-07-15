# Abacus — Backlog

## Status

**Phase 0–7 and the original post-Phase-7 backlog (items 1–7 below) are complete and merged.**
A first independent product audit drove a round of remediation, and the
**Second Batch** that followed is now entirely landed. A **second independent audit**
drove the **Third Batch**, which is also now largely complete. A **Fourth Batch**
(post-launch code review + dark-first redesign, 2026-07-13) has since landed too — see below.
The remaining open items are listed below (last verified against the code on 2026-07-13).

---

# Open Items

## Medium / nice-to-have — ✅ all five closed (2026-07-13)
- [x] **`AbortController` on data fetches** — `apiFetch` already forwarded
  `RequestInit` (including `signal`), so `listWallets`/`getWallet`/
  `listTransactions` now accept an optional `AbortSignal`. Wired into the two
  effects most exposed to rapid navigation: `WalletPage`'s wallet load
  (keyed on route `id`) and `TransactionsTab`'s list load (keyed on
  walletID/page/filters) — each aborts its in-flight request on
  dep-change/unmount and ignores `AbortError` in the catch handler.
- [x] **Uncleared `setTimeout`s** — `Toast.tsx` now tracks pending dismiss
  timers in a ref and clears them on provider unmount; `SettingsPage.tsx`'s
  two "Saved" timers are ref-tracked and cleared on unmount / re-trigger.
- [x] **Timezone not indicated on dates** — added a shared `formatDate()` in
  `lib/utils.ts` (`Intl.DateTimeFormat` with `timeZoneName: 'short'`) and
  applied it to the two highest-traffic date displays:
  `TransactionsTab`'s date column and `PricesPage`'s snapshot date column.
- [x] **No client-side descriptor validation** — `AddWalletDialog` now runs a
  loose sanity regex (`DESCRIPTOR_RE`) against a manually-entered descriptor
  before submit — catches obvious typos early; the server remains the source
  of truth for real validation.
- [x] **Test coverage gaps (partial)** — added `internal/sync/service_test.go`
  covering `StartSync` (wallet-not-found, no-descriptor, backend-factory
  error) and `runSync` (backend error fails the job, `networkToParams`
  covers all networks); added wallet-not-found negative tests for
  `wallet_repo.go` and `ledger_repo.go`. Most other repos and ~10 frontend
  page components remain untested — this was a partial pass, not a full
  sweep; left as a smaller follow-up if further coverage is wanted.

## Carried over
- [ ] **Tax constants by-year audit** — NL Box 3 methodology, UK annual exempt
  amounts, German loss carry-forward (Verlustvortrag). Needs legal care.
- [ ] **Performance** — UTXO endpoint pagination, frontend code-splitting
  (~471 KB bundle as of the redesign, up from 419 KB), memoisation; review
  indexes for large wallets.
- [ ] **Opportunities** — journal diff/audit viewer over the immutable ledger.
  (Portfolio dashboard with a value/BTC history chart shipped in the Fourth
  Batch — see below.)
- [ ] **Cross-platform release** — Docker image publish, Windows/macOS, arm64.
- [x] **Accessibility audit** — aria attributes and keyboard navigation were
  never actually verified (the "Dark mode + accessibility pass" line in the
  Second Batch below turned out to only cover responsive layout; accessibility
  specifically was never separately audited). Done: drag-and-drop import zones
  (`ImportTab`, `AddWalletDialog`) are now keyboard-operable (`role="button"`,
  `tabIndex`, Enter/Space); the mobile nav drawer (`Layout.tsx`) is a real
  modal dialog (focus moves in on open, `inert` on `<main>` while open,
  Escape closes, focus returns to the toggle); the wallet detail tab strip
  (`WalletPage.tsx`) uses proper ARIA tab semantics; sortable table headers
  carry `aria-sort`; segmented/toggle button groups (`PortfolioChart`,
  `AddWalletDialog` wallet-type toggle) use `aria-pressed` + `role="group"`;
  the transaction category cell has accessible labels. Added `vitest-axe` for
  automated regression coverage (`Layout.test.tsx`, extended
  `WalletPage.test.tsx`, new `ImportTab.a11y.test.tsx` /
  `AddWalletDialog.a11y.test.tsx`) so this class of regression is now caught
  by `npm test`, not just manual review.
- [ ] **Bitcoin Core sync backend** — `blockchain_backend: bitcoincore` is listed
  in the architecture but was never implemented. Add `internal/sync/bitcoincore/`
  with a JSON-RPC client (`getaddresstxids` / `scantxoutset`) and wire it into
  `main.go` and `settings.go`.

## Already done — checked off in an accuracy pass (2026-07-02)
Verified against the code; these had landed but were left unchecked:
- [x] **Delete-button `aria-label`** — `aria-label={\`Delete wallet ${w.name}\`}`
  in `WalletsPage.tsx`.
- [x] **`CODEOWNERS` granularity** — per-module owners added for
  `accounting`/`reports`/`ledger`/`sync`/`importer`/`middleware.go`/`migrations`.
- [x] **SECURITY.md vuln-response window** — "we aim to acknowledge reports
  within a few days" is present.
- [x] **`PricesPage` stale data on currency switch** — `load()` clears
  `prices`/`error` before refetching (PR #64).
- [x] **`listWallets()` null handling** — already defensive (`data ?? []`); the
  API also never returns `null`. Non-issue.
- [x] **Unused type imports** — would fail `npm run lint` (oxlint) in CI; none
  present.

Note: an earlier version of this file also duplicated the AppImage-release
item here (marked open) while the real entry — see "Release & Housekeeping"
below — was already done. Removed the stale duplicate; see that section for
the actual release history.

---

# Third Batch (second independent audit) — ✅ all complete

## Correctness (money path)
- [x] **LIFO fallback bug** — `internal/accounting/lifo.go`: fallback only when no
  lot was matched; phantom-gain on unmatched disposal fixed. Unit test added.

## High value, self-contained
- [x] **Frontend swallows initial-load errors** — error state added to
  `AccountingTab`, `ImportTab`, `SyncPanel`, `WalletPage`.
- [x] **Rate limiter ignores `X-Forwarded-For`** — opt-in `TRUST_PROXY` mode added
  to `internal/api/middleware.go`; documented in `SECURITY.md`.
- [x] **Release signing + checksums** — cosign keyless signing + `sha256sums.txt`
  added to `.github/workflows/release.yml`.
- [x] **Third-party `NOTICE` file** — generated and committed.
- [x] **CI: coverage gate + `npm audit`** — Go coverage floor and frontend audit
  step added to `.github/workflows/ci.yml`.

## Medium
- [x] **Sync handler status codes** — `StartSync` and `ListSyncJobs` now return
  404 for a non-existent wallet.
- [x] **React error boundary** — router wrapped with boundary + recovery UI.
- [x] **Auth-bypass hardening** — `middleware.go` health/version exemption uses
  exact `==` path match instead of `strings.HasSuffix`.

## Discarded — false positives / already done
- **Section 104 30-day rule off-by-one** — *false positive.* Verified correct.
- **`API_TOKEN` not wired to the UI** — done (PR #51).
- **US holding-period precision** — non-issue; standard whole-day count.
- **`golangci-lint` / `gosec` missing** — deferred; `go 1.26.1` toolchain blocks
  these tools. `go vet` is the lint gate for now.
- **`usePoll` stale closure** — retracted by the auditor; the hook is correct.

---

# Fourth Batch (post-launch code review + dark-first redesign, 2026-07-13) — ✅ complete

## Post-launch code review — four real, silent (no-error) bugs found and fixed
- [x] **Auto-release tag-push race** — two merges landing close together could
  both read the same "latest tag" and the second tag push would silently
  fail, dropping a release. Fixed with a `concurrency` group in
  `auto-release.yml` (PR #92).
- [x] **Multisig `OP_M`/`OP_N` miscompilation above 16-of-16** —
  `internal/sync/derive.go`'s opcode encoding only covers 1–16; above that it
  silently landed on unrelated opcodes and produced a wrong address with no
  error. Now rejected with a clear error instead (PR #93).
- [x] **Exchange-import dedup + Kraken currency bugs** — Coinbase/Strike CSV
  re-imports could insert full duplicate trades (no synthetic dedup key);
  Kraken hardcoded EUR and silently zeroed non-EUR trade amounts/fees. Added
  `common.SyntheticExternalID`; Kraken now recognizes USD/GBP/CAD/JPY/CHF/AUD
  and captures the fiat-side fee (PR #94).
- [x] **Unrealised gain always zero** — `RunFIFO`/etc. only ever set
  `GainFiat` on disposal; both `accounting.Service.Summary()` and the
  portfolio handler summed it for undisposed records too — permanently dead
  code. Now marks held lots to market against the latest `PriceSnapshot`
  (PR #95).

## Dark-first design system redesign
- [x] **Real theming, not a background swap** — Tailwind's CSS variable
  tokens existed but were never wired into Tailwind via `@theme`, so every
  page independently reached for hardcoded `slate-*`/`red-*`/`green-*`
  utilities. Rebuilt on a proper token system; dark is now the default
  identity, `.light` opts into light mode; single amber accent used
  consistently for every primary action/link/active state (PR #97).
- [x] **Composition fix** — content was capped at `max-w-3xl`/`max-w-4xl`
  leaving most of a real monitor as dead space, and the Portfolio dashboard
  was just a wallet table with nothing else on the page. Widened content
  columns, added a "Quick actions" panel (PR #97).
- [x] **Portfolio value/BTC-holdings history chart** — `GET
  /portfolio/history` reconstructs a daily cumulative sats balance from
  ledger entries and marks each day to the closest known price snapshot; a
  hand-rolled SVG area/line chart (no new dependency) renders it with a
  30/90/180/365-day range toggle and hover crosshair + tooltip (PR #97).
- [x] **Critical CSS layering bug** — `index.css`'s `* { margin: 0; padding:
  0 }` reset was unlayered, so per the CSS Cascade Layers spec it silently
  beat every Tailwind spacing utility site-wide (`p-8`, `px-4`, `gap-4`, ...)
  regardless of specificity. Found via screenshotting the live app for docs,
  not by build/lint/test (all stayed green throughout). Fixed by wrapping the
  reset in `@layer base` (PR #99).

## Docs gaps — closed, then re-reviewed and hardened
- [x] **CODE_OF_CONDUCT.md** — Contributor Covenant v2.1 (PR #99).
- [x] **`docs/deployment.md`** — production deployment guide (PR #99), then
  a brutally-honest self-review caught real gaps in the first pass and PR
  #100 fixed them: `docker-compose.yml` bound the port to all interfaces by
  default (now `127.0.0.1`-bound, since the guide had framed exposure as an
  opt-in step that in fact wasn't); the documented `ENV` var turned out to be
  dead code (dropped); asymmetric TLS guidance (Nginx had no real certbot
  path, Caddy did); missing logging/backup-verification/single-writer/
  disk-growth guidance; no privacy-preserving (Tailscale/WireGuard)
  alternative to public exposure despite the project's own stated posture.
- [x] **README screenshots** — six real screenshots against seeded demo data,
  dark mode: Portfolio (with the new chart), Transactions, Accounting,
  Reports (PR #99), then Import (drag-and-drop) and Settings — the two most
  onboarding-critical flows that were missing from the first pass (PR #100).

---

# Second Batch — ✅ all complete (retained for history)
- [x] Electrum float→sats parsing fix.
- [x] Docker hardening — non-root `USER`, `HEALTHCHECK`, `.dockerignore`, limits.
- [x] Governance docs — `SECURITY.md`, `CONTRIBUTING.md`, `CHANGELOG.md`,
  `.github/ISSUE_TEMPLATE/`, PR template, `CODEOWNERS`.
- [x] Dependabot config + grouping.
- [x] Frontend UX — toasts + confirm dialog, 404 route, table search/sort/filter
  (Wallets/Prices in-memory; Transactions server-side).
- [x] ~~Dark mode + accessibility pass~~ + responsive sidebar. **Correction
  (2026-07-13):** this line was an overclaim. "Dark mode" only ever toggled
  `background`/`foreground` — every other color stayed on light-mode values
  until the Fourth Batch's redesign actually wired Tailwind's token system up
  correctly. "Accessibility" was never separately audited (aria/keyboard nav)
  — that's now tracked as its own open item above instead of implied done here.
- [x] `API_TOKEN` → web UI wiring (PR #51).
- [x] WalletPage refactor into per-tab files; `useDialog` / `usePoll` hooks.
- [x] Import/Sync poll-to-completion tests.

---

# Release & Housekeeping — ✅ complete
- [x] **Version wiring** — `var version = "dev"` in `main.go`; ldflags
  `-X main.version=<tag>` now bakes the version into the binary (PR #71).
- [x] **Remote branch cleanup** — all 52 merged/closed stale branches deleted.
- [x] **Signed AppImage releases** — `v0.1.0` → `v0.1.1` → `v0.1.2` tagged and
  published on GitHub Releases (each with cosign signature + checksums); see
  `CHANGELOG.md` for what shipped in each. `v0.1.2` is current.
- [x] **Drag-and-drop import** — drop zone added to Import tab; upload starts on drop.
- [x] **Exchange account imports** — five exchange importers (Bitvavo, Bitonic, Kraken,
  Coinbase, Strike); `exchange_trades` table (migration 006); `ExchangeTrade` domain
  entity; `RunExchangeFIFO` accounting; Add Wallet dialog wallet-type toggle (PR #88).

Note: a stray `v1.0.0` tag also exists in the repo with **no GitHub Release**
attached — a leftover placeholder from before versioning was rethought,
pointing to a commit well behind `main` (predates the LIFO fix and security
hardening). It does not appear on the Releases page (GitHub correctly shows
`v0.1.2` as latest) and is not linked from any doc. Deleting it requires a
local git credential (tag deletion 403s from CI, same as tag push) — left as
a manual housekeeping item for the maintainer.

---

# Original post-Phase-7 backlog — ✅ all complete

## 1. Ledger & UTXO endpoints — ✅ done
## 2. Transaction metadata editing — ✅ done
## 3. Additional cost basis methods (LIFO, HIFO, SpecificID, Section 104) — ✅ done
## 4. Multi-wallet portfolio view — ✅ done
## 5. Tax report templates (NL / DE / UK / US) — ✅ done
## 6. Label & POST label endpoint + BIP329 export — ✅ done
## 7. Linux AppImage packaging + GitHub Release CI — ✅ done
