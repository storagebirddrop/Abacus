# Security Policy

## Reporting a vulnerability

Please report security issues **privately** — do not open a public issue for a
suspected vulnerability.

Use GitHub's private vulnerability reporting: the repository's **Security** tab →
**Report a vulnerability**. Include a description, affected version/commit, and
reproduction steps. We aim to acknowledge reports within a few days.

## Security model

Abacus is a self-hosted, read-only accounting tool over **public** wallet data.
Understanding the model helps scope reports:

- **No private material.** Abacus never stores or transmits private keys, seed
  phrases, or signing material. Encrypted Sparrow databases (`.mv.db`) are
  rejected on import.
- **Public data only.** It works with descriptors, xpubs, addresses, and txids.
- **Blockchain sync is opt-in** and off by default. When enabled it discloses
  wallet addresses to the configured Esplora/Electrum server; self-hosting that
  backend is recommended. (See the Settings page and `README` → Privacy.)
- **Local-first.** The default deployment assumes a single user on localhost.
  Optional `API_TOKEN` bearer auth and per-IP rate limiting (`RATE_LIMIT_RPM`)
  exist for deployments exposed beyond localhost — see `.env.example`.

## In scope

- Authentication/authorization bypass on `/api/v1` when `API_TOKEN` is set.
- Path traversal, SSRF, injection, or RCE in the server or importers.
- Any code path that could persist or leak private key material (it must not).
- Resource-exhaustion vectors not bounded by existing limits.

## Out of scope

- Issues that require an already-compromised host or a malicious operator.
- Exposing the app to the public internet without the documented auth/proxy.
- Privacy of addresses disclosed to a third-party blockchain backend the user
  explicitly configured (this is documented behaviour).

## Supported versions

Security fixes target the latest release and `main`.
