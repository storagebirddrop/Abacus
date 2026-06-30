# Abacus — Backlog

## Status

**Phase 0–7 and the original post-Phase-7 backlog (items 1–7 below) are complete and merged.**
A first independent product audit drove a round of remediation, and the
**Second Batch** that followed is now almost entirely landed (Electrum float fix,
Docker hardening, governance docs, full frontend test suite, API auth + rate
limiting, dark mode + accessibility, responsive sidebar, `API_TOKEN` UI wiring,
WalletPage refactor with `useDialog`/`usePoll`, and server-side transaction
search/sort/filter — all merged).

A **second independent audit** (three parallel clean-eyes explorations: backend
correctness, frontend/UX, security + repo-health) then ran against the current
tree. Its verified findings are captured in the **Third Batch** section. The
remaining open Second Batch items are folded in there too.

---

# Third Batch (second independent audit)

Prioritised. Verified against the code where noted; false positives and
already-done items are listed at the end so they aren't re-investigated.

## Correctness (money path) — do first
- [ ] **LIFO fallback bug** — `internal/accounting/lifo.go:76`. The fallback
  `if costFiat == 0 && proceedsFiat == 0` means a disposal with proceeds but no
  matched acquisition records cost `0` → 100% phantom gain. HIFO already guards
  this correctly with `bestIdx >= 0 / else`. Mirror that; add a unit test.
  **Verified real.**

## High value, self-contained
- [ ] **Frontend swallows initial-load errors** — `AccountingTab`, `ImportTab`,
  `SyncPanel`, `WalletPage` use `.catch(() => {})`, so an API failure shows a
  blank tab with no message. Add error state + display. **Verified real.**
- [ ] **Rate limiter ignores `X-Forwarded-For`** (`internal/api/middleware.go`) —
  behind a reverse proxy all clients collapse to one IP, defeating per-IP limits.
  Add opt-in trusted-proxy handling + a SECURITY.md deployment note.
- [ ] **Release signing + checksums** (`.github/workflows/release.yml`) — AppImage
  is published unsigned with no `sha256sums.txt`; add cosign/GPG + checksums.
- [ ] **Third-party `NOTICE` file** — btcsuite, maroto, excelize, etc. carry
  MIT/Apache terms; generate via `go-licenses`.
- [ ] **CI: coverage gate + `npm audit`** — Go coverage is informational only
  (never fails); add a threshold gate and an `npm audit` step to the frontend job.
  (`golangci-lint`/`gosec` remain blocked by the `go 1.26.1` toolchain — see
  Discarded.)

## Medium (correctness consistency / robustness)
- [ ] **Sync handler status codes** — `StartSync` returns 400 for a missing wallet
  (should be 404); `ListSyncJobs` skips the wallet-existence check (returns 200 +
  empty instead of 404). Mirror the accounting handler's check.
- [ ] **`PricesPage` stale data on currency switch** — on a failed refetch the old
  currency's rows remain under the spinner; clear before refetch. *Unverified.*
- [ ] **React error boundary** — a render throw currently white-screens the whole
  app; wrap the router with a boundary + recovery UI.
- [ ] **`AbortController` on data fetches** — rapid wallet/tab navigation can let a
  stale response overwrite current state. Low impact for single-user, but real.
- [ ] **Auth-bypass hardening** — `middleware.go:34` uses
  `strings.HasSuffix(p, "/health")`; safe today given chi exact routing, but a
  future `/x/health` route would bypass auth. Use exact `==` match.
  **Verified currently-safe.**

## Lower / nice-to-have
- [ ] **Uncleared `setTimeout`s** — Toast + SettingsPage "Saved" timers aren't
  cleared on unmount (harmless at app root, but tidy with cleanup).
- [ ] **Accessibility nits** — delete buttons missing `aria-label`; timezone not
  indicated on UTC `block_time` dates; no client-side descriptor validation.
- [ ] **Test coverage gaps** — `internal/sync/service.go` (the sync loop, context
  handling, job-status transitions) and the sync handlers are untested; most
  repos lack wallet-not-found negative tests; ~10 page components untested
  (`WalletPage`, the tabs, `ExportBar`).
- [ ] **`CODEOWNERS` granularity + SECURITY.md SLA** — add per-module owners for
  `accounting`/`sync`/`importer`/`middleware`; document a vuln-response window.
- [ ] **`listWallets()` null handling / unused type imports** — minor; lint-level.

## Carried over from Second Batch (still open)
- [ ] **Tax constants by-year audit** — NL Box 3 methodology, UK annual exempt
  amounts, German loss carry-forward (Verlustvortrag). Needs legal care.
- [ ] **Performance** — UTXO endpoint pagination, frontend code-splitting
  (~419 KB bundle), memoisation; review indexes for large wallets.
- [ ] **Opportunities** — portfolio dashboard with charts, journal diff/audit
  viewer over the immutable ledger.
- [ ] **Cross-platform release** — Docker image publish, Windows/macOS, arm64.

## Manual / requires maintainer action (not automatable from CI)
- [ ] **`v1.0.0` / AppImage release** — push a `v*` tag from a local machine; the
  release workflow then builds and publishes the AppImage. The CI token cannot
  push tags (403). The `v1.0.0` tag currently exists with no published Release.
- [ ] **Remote branch cleanup** — merged branches remain on the remote; deleting
  refs returns 403 from CI. Run `git push origin --delete <branches>` locally, or
  enable **Settings → General → Automatically delete head branches**.

## Discarded — false positives / already done (do not re-investigate)
- **Section 104 30-day rule off-by-one** — *false positive.* `acqDay.After(deadline)`
  is false when `acqDay == dispDay+30`, so day +30 is included: a full 30-day
  window. Verified correct.
- **`API_TOKEN` not wired to the UI** — done (PR #51).
- **US holding-period precision** — non-issue; standard whole-day count.
- **`golangci-lint` / `gosec` missing** — deferred; the `go 1.26.1` toolchain
  blocks these tools (tools built against go 1.25.x refuse the module).
- **`usePoll` stale closure** — retracted by the auditor; the hook is correct.

## Second Batch — ✅ completed (retained for history)
- [x] Electrum float→sats parsing fix.
- [x] Docker hardening — non-root `USER`, `HEALTHCHECK`, `.dockerignore`, limits.
- [x] Governance docs — `SECURITY.md`, `CONTRIBUTING.md`, `CHANGELOG.md`,
  `.github/ISSUE_TEMPLATE/`, PR template, `CODEOWNERS` (granularity still open).
- [x] Dependabot config + grouping (PRs #50 and the triage round).
- [x] Frontend UX — toasts + confirm dialog (no more `alert()`/`confirm()`),
  404 route, table search/sort/filter (Wallets/Prices in-memory; Transactions
  server-side, PR #56).
- [x] Dark mode + accessibility pass + responsive sidebar.
- [x] `API_TOKEN` → web UI wiring (PR #51).
- [x] WalletPage refactor into per-tab files; `useDialog` (#55) / `usePoll` hooks;
  dead `App.css`/`assets` removed.
- [x] Import/Sync poll-to-completion tests (fake-timer follow-up).

---

# Original post-Phase-7 backlog — ✅ all complete

The items below were the initial backlog; all are implemented and merged.
Retained for history.

## 1. Ledger & UTXO endpoints — ✅ done
- `GET /api/v1/wallets/{id}/ledger` — paginated ledger entries
- `GET /api/v1/wallets/{id}/ledger/{entryID}` — single entry + journal audit trail
- `GET /api/v1/wallets/{id}/utxos` — current UTXO set

## 2. Transaction metadata editing — ✅ done
- `PATCH /api/v1/wallets/{id}/transactions/{txid}` — category, note, counterparty
- Each change appends a `JournalEntry` (immutable audit trail)

## 3. Additional cost basis methods — ✅ done
- LIFO, HIFO, Specific Identification (plus Section 104) as pure functions

## 4. Multi-wallet portfolio view — ✅ done
- `GET /api/v1/portfolio/summary` — cross-wallet aggregate

## 5. Tax report templates per jurisdiction — ✅ done
- NL (Box 3), DE (§23 EStG), UK (HMRC Section 104), US (Form 8949)

## 6. Label & POST label endpoint — ✅ done
- `POST /api/v1/wallets/{id}/labels` + BIP329 `.jsonl` export

## 7. Linux AppImage packaging — ✅ done
- `packaging/appimage/` + `Makefile` `appimage` target + release workflow
  (embedded frontend/migrations; binary is self-contained). Publishing a release
  still requires pushing a tag — see the Second Batch manual items.
