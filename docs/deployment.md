# Deployment Guide

This guide covers running Abacus as a long-lived, production deployment. For a
quick local trial, see the [Quick Start](../README.md#quick-start) in the
README — this document assumes you want it running continuously, possibly
exposed beyond `localhost`.

Abacus is a single Go binary with an embedded frontend and an embedded SQLite
migration set. All persistent state lives in one SQLite file. There is no
external database, cache, or queue to operate — and no support for running
multiple instances against the same database: SQLite is single-writer, and
Abacus is designed as one process per database file, not a horizontally
scaled service.

## Docker Compose

The included `docker-compose.yml` is deployment-ready as-is for a single-user
setup, and **binds to `127.0.0.1` by default** — it is not reachable from
your network until you deliberately open it up (see below). This matches
the rest of the app's secure-by-default posture (auth and blockchain sync
are both off until you turn them on).

```bash
git clone https://github.com/storagebirddrop/abacus
cd abacus
cp .env.example .env
docker compose up --build -d
```

It runs as a non-root user, has a healthcheck, restarts automatically
(`unless-stopped`), and caps resources (1 CPU, 512 MB). State persists in the
named volume `abacus-data`.

### Exposing it beyond localhost

If Abacus will be reachable outside your own machine, consider a private
network first — Tailscale or a WireGuard VPN gives you remote access without
ever putting the app on the public internet, in keeping with the project's
"public wallet data only, self-hosted, privacy by default" stance (see
[CLAUDE.md](../CLAUDE.md#key-invariants)). If you do want it reachable more
broadly, do all three of these:

1. **Set `API_TOKEN`** in `.env` — a strong, random bearer token
   (`openssl rand -hex 32`). Without it, every `/api/v1` route is open to
   anyone who can reach the port. The bundled web UI picks the token up
   automatically once you save it on the Settings → API access page.
2. **Remove the `127.0.0.1:` prefix** from `docker-compose.yml`'s `ports:`
   entry (or bind a different specific interface) and **put a reverse proxy
   in front of it** for TLS termination — Abacus itself only serves plain
   HTTP. A minimal Caddy example (automatic HTTPS via Let's Encrypt, renews
   itself):

   ```caddyfile
   abacus.example.com {
       reverse_proxy localhost:8080
   }
   ```

   Or Nginx, which needs an explicit certificate — [certbot](https://certbot.eff.org/)
   with the Nginx plugin obtains one and installs a systemd timer that
   renews it automatically:

   ```bash
   sudo certbot --nginx -d abacus.example.com
   ```

   ```nginx
   server {
       listen 443 ssl;
       server_name abacus.example.com;
       ssl_certificate     /etc/letsencrypt/live/abacus.example.com/fullchain.pem;
       ssl_certificate_key /etc/letsencrypt/live/abacus.example.com/privkey.pem;

       location / {
           proxy_pass http://127.0.0.1:8080;
           proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
           proxy_set_header X-Real-IP $remote_addr;
       }
   }
   ```

3. **Set `TRUST_PROXY=true`** in `.env` — but *only* once step 2 is in place.
   This tells the rate limiter to read the client IP from
   `X-Forwarded-For`/`X-Real-IP` instead of the TCP peer. Enabling it without
   a real reverse proxy in front lets any client spoof those headers and evade
   per-IP rate limiting entirely.

## AppImage as a background service

The AppImage is meant for interactive desktop use, but it's a normal binary
underneath and can run as a `systemd` user service for headless/server use:

```ini
# ~/.config/systemd/user/abacus.service
[Unit]
Description=Abacus Bitcoin Accounting Engine
After=network.target

[Service]
ExecStart=%h/Abacus-x86_64.AppImage
Restart=on-failure

[Install]
WantedBy=default.target
```

```bash
systemctl --user daemon-reload
systemctl --user enable --now abacus.service
```

No `DB_PATH`/directory setup needed here: running the AppImage always executes its bundled
`AppRun` script first, which unconditionally sets `DB_PATH` to the XDG data path below and
creates the directory if it doesn't exist — before the `abacus` binary itself ever starts.

(Use `loginctl enable-linger $USER` if you want it to keep running after you
log out.)

## Logs & monitoring

Abacus logs to stdout only — there's no log file to manage or rotate:

- **Docker Compose**: `docker compose logs -f abacus`
- **systemd**: `journalctl --user -u abacus -f`

For uptime monitoring, poll `GET /api/v1/health` (the same endpoint the
Docker healthcheck above already uses) — it returns `200 {"status":"ok"}`
with no auth required.

## Environment variables

All configuration is via environment variables (`.env` for Docker Compose, or
exported directly for the AppImage/binary). See `.env.example` for the
canonical, commented list. Summary:

| Variable | Default | Purpose |
|---|---|---|
| `PORT` | `8080` | HTTP listen port |
| `DB_PATH` | `./abacus.db` | SQLite database file. The AppImage's `AppRun` script always overrides this to `~/.local/share/abacus/abacus.db` regardless of the environment — it only applies to Docker/binary use |
| `FRONTEND_DIR` | `./web/dist` | Serve the frontend from disk instead of the embedded copy (dev only) |
| `API_TOKEN` | *(unset)* | Require `Authorization: Bearer <token>` on `/api/v1` (except health/version) — see above |
| `RATE_LIMIT_RPM` | `600` | Per-IP request cap per minute on `/api/v1`; `0` disables |
| `TRUST_PROXY` | `false` | Derive the rate-limit client IP from `X-Forwarded-For`/`X-Real-IP` — only behind a trusted reverse proxy |

**Blockchain sync (Esplora/Electrum) is not configured via environment
variables.** It's off by default and configured at runtime from the in-app
Settings page, persisted in SQLite — no restart needed.

## Backups

The entire application state is the single SQLite file at `DB_PATH`
(`abacus-data` volume under Docker Compose; `~/.local/share/abacus/abacus.db`
for the AppImage default). It only grows over time — keep an eye on
available disk space on long-running deployments. To back it up:

- **Simplest**: stop the container/process, copy the file, restart.
- **Live backup without downtime**: use SQLite's own backup command, which is
  safe to run against a database still being written to:

  ```bash
  sqlite3 /path/to/abacus.db ".backup /path/to/abacus-backup.db"
  ```

A backup you haven't restored is a backup you don't actually have — periodically
verify one actually opens cleanly, not just that the copy step succeeded:

```bash
sqlite3 /path/to/abacus-backup.db "PRAGMA integrity_check;"
```

To restore, stop Abacus, replace the live file with the backup, and restart.

## Upgrading

- **Docker Compose**: `git pull && docker compose up --build -d`.
- **AppImage**: download the new release from
  [GitHub Releases](https://github.com/storagebirddrop/Abacus/releases),
  replace the old file. Releases are signed — see the verification steps in
  the [README](../README.md#quick-start).

Database migrations run automatically on startup (`golang-migrate`, embedded
in the binary) — there is no separate migration step to run.
