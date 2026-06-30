# Abacus — Backlog

## Status

**Phase 0–7 and the original post-Phase-7 backlog (items 1–7 below) are complete and merged.**
A subsequent independent product audit drove a round of remediation (reports/tax
tests + fixes, sync concurrency, upload bound, privacy notice, dead-config
cleanup, a full frontend test suite, and API auth + rate limiting — all merged).
The **Second Batch** section captures what remains.

---

# Second Batch (post-audit)

Prioritised. Effort/risk noted. Reprioritise as needed.

## Manual / requires maintainer action (not automatable from CI)
- [ ] **`v1.0.0` / AppImage release** — push a `v*` tag from a local machine; the
  release workflow then builds and publishes the AppImage. The CI token cannot
  push tags (403), so this must be done by the maintainer.
- [ ] **Remote branch cleanup** — ~24 merged branches remain on the remote;
  deleting refs returns 403 from CI. Run `git push origin --delete <branches>`
  locally, or enable **Settings → General → Automatically delete head branches**.

## High value, self-contained (best ROI)
- [ ] **CI hardening** — add `golangci-lint` (Go currently only runs `go vet`),
  `gosec` SAST, `dependabot`, and a test-coverage gate. One workflow PR; no
  product risk.
- [ ] **Electrum float→sats bug** — `internal/sync/electrum` converts amounts via
  `int64(btc * 1e8)`, the last violation of the satoshis-only/no-floats
  invariant. Parse strings instead. Small, with a test.
- [ ] **Docker hardening** — non-root `USER`, `HEALTHCHECK`, `.dockerignore`,
  resource limits; the image currently runs as root. Optionally publish it.
- [ ] **Governance docs** — `SECURITY.md`, `CONTRIBUTING.md`, `CHANGELOG.md`,
  `.github/ISSUE_TEMPLATE/`, PR template, `CODEOWNERS`, third-party `NOTICE`.

## Medium (UX / maintainability — the audit's weakest scores)
- [ ] **Frontend UX regressions vs. legacy Desktop** — replace blocking
  `alert()`/`confirm()` with toasts + a confirm dialog; add a 404 route; then
  search / sort / filter on the tables.
- [ ] **Dark mode + accessibility pass** (0 aria attributes today) + responsive
  sidebar (fixed `w-48`, breaks on mobile).
- [ ] **`API_TOKEN` → web UI wiring** — so enabling bearer auth doesn't break the
  bundled UI (closes the caveat from the auth PR).
- [ ] **WalletPage refactor** — split the 516-line monolith into per-tab files;
  extract `useDialog` / `usePoll` hooks; delete dead `web/src/App.css` and unused
  `web/src/assets/`; `npm prune` extraneous deps.
- [ ] **Import/Sync poll-to-completion tests** — the fake-timer follow-up deferred
  from the WalletPage test batch (the 2s job polling loops).

## Lower / larger
- [ ] **Tax constants by-year audit** — NL Box 3 methodology, UK annual exempt
  amounts, German loss carry-forward (Verlustvortrag, currently out of scope).
  Accuracy work; needs legal care.
- [ ] **Release engineering** — artifact signing (cosign/gpg) + checksums, Docker
  image publish, Windows/macOS and arm64 builds.
- [ ] **Performance** — UTXO endpoint pagination, frontend code-splitting
  (~419 KB bundle), memoisation; review indexes for large wallets.
- [ ] **Opportunities** — portfolio dashboard with charts, journal diff/audit
  viewer over the immutable ledger.

> After this batch lands, commission a fresh **independent** audit (clean-eyes
> review, not against our own change memory) to catch blind spots.

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
