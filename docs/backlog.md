# Abacus — Backlog

## Status

**Phase 0–7 and the original post-Phase-7 backlog (items 1–7 below) are complete and merged.**
A first independent product audit drove a round of remediation, and the
**Second Batch** that followed is now entirely landed. A **second independent audit**
drove the **Third Batch**, which is also now largely complete. The remaining open
items are listed below.

---

# Open Items

## Medium / nice-to-have
- [ ] **`PricesPage` stale data on currency switch** — on a failed refetch the old
  currency's rows remain under the spinner; clear before refetch. *Unverified.*
- [ ] **`AbortController` on data fetches** — rapid wallet/tab navigation can let a
  stale response overwrite current state. Low impact for single-user, but real.
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

## Carried over
- [ ] **Tax constants by-year audit** — NL Box 3 methodology, UK annual exempt
  amounts, German loss carry-forward (Verlustvortrag). Needs legal care.
- [ ] **Performance** — UTXO endpoint pagination, frontend code-splitting
  (~419 KB bundle), memoisation; review indexes for large wallets.
- [ ] **Opportunities** — portfolio dashboard with charts, journal diff/audit
  viewer over the immutable ledger.
- [ ] **Cross-platform release** — Docker image publish, Windows/macOS, arm64.

## Manual / requires maintainer action
- [ ] **`v0.1.0` AppImage release** — push the `v0.1.0` tag from a local machine;
  the release workflow builds and publishes the signed AppImage. The CI token
  cannot push tags (403).

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

# Second Batch — ✅ all complete (retained for history)
- [x] Electrum float→sats parsing fix.
- [x] Docker hardening — non-root `USER`, `HEALTHCHECK`, `.dockerignore`, limits.
- [x] Governance docs — `SECURITY.md`, `CONTRIBUTING.md`, `CHANGELOG.md`,
  `.github/ISSUE_TEMPLATE/`, PR template, `CODEOWNERS`.
- [x] Dependabot config + grouping.
- [x] Frontend UX — toasts + confirm dialog, 404 route, table search/sort/filter
  (Wallets/Prices in-memory; Transactions server-side).
- [x] Dark mode + accessibility pass + responsive sidebar.
- [x] `API_TOKEN` → web UI wiring (PR #51).
- [x] WalletPage refactor into per-tab files; `useDialog` / `usePoll` hooks.
- [x] Import/Sync poll-to-completion tests.

---

# Release & Housekeeping — ✅ complete
- [x] **Version wiring** — `var version = "dev"` in `main.go`; ldflags
  `-X main.version=<tag>` now bakes the version into the binary (PR #71).
- [x] **Remote branch cleanup** — all 52 merged/closed stale branches deleted.
- [x] **`v0.1.0` tag** — pending local push to trigger signed AppImage release.

---

# Original post-Phase-7 backlog — ✅ all complete

## 1. Ledger & UTXO endpoints — ✅ done
## 2. Transaction metadata editing — ✅ done
## 3. Additional cost basis methods (LIFO, HIFO, SpecificID, Section 104) — ✅ done
## 4. Multi-wallet portfolio view — ✅ done
## 5. Tax report templates (NL / DE / UK / US) — ✅ done
## 6. Label & POST label endpoint + BIP329 export — ✅ done
## 7. Linux AppImage packaging + GitHub Release CI — ✅ done
