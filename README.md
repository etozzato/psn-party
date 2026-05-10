# PSN Add

Ultra-minimal PSN friend-list helper. A user creates a group, shares the group URL, and everyone adds their PSN ID. The app checks whether each PlayStation profile appears public and renders either a PlayStation profile QR/link or a safe fallback QR with the handle.

## What Exists

- `GET /new` creates a uniquely named group.
- `GET /g/:slug` shows the group list with `A-Z`, `RECENT`, and `NEW` controls.
- `POST /g/:slug/entries` adds an optional name plus PSN ID and returns an entry admin link on screen.
- `GET|POST /g/:slug/upload` lets a group admin batch import `Name (optional),PSN-ID` CSV rows.
- `GET /g/:slug/export.csv` lets a group admin download all active entries as `group-name.csv`.
- `GET /g/:slug/:online_id` shows one QR card.
- `POST /g/:slug/:online_id/pull` re-fetches PlayStation profile visibility.
- `POST /g/:slug/:online_id/remove` removes an entry.
- `POST /g/:slug/:online_id/ban` removes and blocks the PSN ID for the group.
- `cmd/admin` is the emergency text admin for listing, showing, renaming, deleting groups, and banning IDs.

Admin access is token based. Group admins can manage every entry in the group. Entry admins can manage only the entry they created. The service does not send email; creation screens include an `EMAIL YOURSELF` mailto button so users can back up admin links in their own configured email client.

Groups can be `PUBLIC` or `PRIVATE` at creation. Private groups generate a 5-digit PIN that visitors must enter before viewing or adding entries. Group admins bypass the PIN and can make the group public, make it private again, or rotate the PIN from the group page.

The homepage shows live tallies for parties and total PSN IDs. Each party page shows its own current PSN ID count in the sticky header.

## Local Run

```bash
cp config/.env.example config/.env
docker compose up --build
open http://localhost:8890/new
```

The dev compose stack runs Postgres and the Go API. The app runs migrations at startup.

## Batch Upload

Open a group with its group admin URL, then use `UPLOAD`. The importer accepts a CSV file or pasted CSV with exactly two columns:

```csv
Name (optional),PSN-ID
Emanuele,psychic-disco2
,SomeHandle123
```

`NAME` is optional, but the comma is required. Each imported row receives its own entry admin link and `EMAIL` action in the result list. Duplicate, blocked, or invalid PSN IDs are reported per row without stopping the rest of the import. The slow part is the PlayStation public/private check; the app shows a `CHECKING PSN` loader and runs batch checks with bounded concurrency.

Group admins can also use `CSV` to download the active list as `group-name.csv` using the same two-column format.

To use the text admin:

```bash
bin/admin
```

Useful admin commands:

```text
list
show <group-id-or-slug>
rename <group-id-or-slug> <new name>
delete <group-id-or-slug>
ban <group-id-or-slug> <psn-id>
```

## Configuration

The server loads `.env` and `config/.env`.

| Variable | Required | Default | Notes |
| --- | --- | --- | --- |
| `PUBLIC_BASE_URL` | yes | `http://localhost:8890` | Used when printing group and admin links. Set to the HTTPS URL in production. |
| `DATABASE_URL` | yes | none | Postgres connection URL. |
| `PORT` | no | `8890` | Container and local HTTP port. |
| `BIND_ADDRESS` | no | `127.0.0.1` | Use `0.0.0.0` in Docker. |
| `PROFILE_TIMEOUT_SECONDS` | no | `4` | Timeout for profile.playstation.com checks. |
| `LOG_LEVEL` | no | `info` | `debug`, `info`, `warn`, or `error`. |

## Production Shape

For a laptop server:

1. Copy `config/.env.example` to `config/.env`.
2. Set `PUBLIC_BASE_URL=https://your.domain`, `POSTGRES_PASSWORD`, `DOMAIN`, and `CERT_NAME`.
3. Run `docker compose -f docker-compose.prod.yml up -d --build`.
4. Put Nginx in front of `127.0.0.1:${HOST_PORT:-8890}` using `config/nginx.conf.example`.
5. Issue the certificate with Certbot, then reload Nginx.

The app can terminate TLS itself only if you add a TLS-aware wrapper, but the intended deployment is Nginx SSL termination in front of the Dockerized Go service.

## Tailscale Funnel

If the laptop already uses port `443` for another public Funnel, expose PSN Add on a second allowed Funnel port. Tailscale Funnel supports public ports `443`, `8443`, and `10000`, so use `8443` when `443` is occupied.

Set the public URL before creating groups, because generated admin links and `EMAIL YOURSELF` bodies use `PUBLIC_BASE_URL`:

```dotenv
PUBLIC_BASE_URL=https://YOUR-LAPTOP.YOUR-TAILNET.ts.net:8443
HOST_PORT=8890
POSTGRES_PASSWORD=change-me
```

Run the app:

```bash
docker compose -f docker-compose.prod.yml up -d --build
```

Expose it through Funnel:

```bash
sudo tailscale funnel --bg --https=8443 http://127.0.0.1:8890
tailscale funnel status
curl -I https://YOUR-LAPTOP.YOUR-TAILNET.ts.net:8443/ping
```

Public entry point:

```text
https://YOUR-LAPTOP.YOUR-TAILNET.ts.net:8443/new
```

To keep the Funnel after reboot, create `/etc/systemd/system/psn-add-funnel.service`:

```ini
[Unit]
Description=PSN Add Tailscale Funnel
After=tailscaled.service docker.service
Requires=tailscaled.service

[Service]
Type=oneshot
RemainAfterExit=yes
ExecStart=/usr/bin/tailscale funnel --bg --https=8443 http://127.0.0.1:8890
ExecStop=/usr/bin/tailscale funnel --https=8443 off

[Install]
WantedBy=multi-user.target
```

Enable it:

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now psn-add-funnel.service
```

## Data Model

- `groups`: name, public slug, group admin token hash.
- `entries`: optional display name, PSN ID, entry admin token hash, cached public/private flag, timestamps for removal/ban.
- `blocked_entries`: group-specific block list.

Raw admin tokens are never stored. Only SHA-256 hashes are kept in Postgres.
