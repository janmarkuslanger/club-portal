# club-portal

Minimal sports club portal: club admins log in, manage club details, and the public site is generated as static pages.

## Features

- Go backend using graft for auth and the admin UI
- Static site generation with ssgo
- SQLite storage via GORM

## Requirements

- Go 1.24+
- Node.js + npm (only needed to build CSS)

## Local setup

```bash
go mod download
npm install
npm run build:css
```

### Run the server

```bash
go run ./cmd/server
```

Open `http://localhost:8080` (redirects to `/login`). On first run, an example club is seeded; credentials are printed in the server log.

### Run the build worker (recommended)

```bash
go run ./cmd/worker
```

The worker processes the build queue and runs a nightly build (default `03:00`). When an admin saves changes, a build task is queued and debounced with `BUILD_DEBOUNCE` (default `2m`).

For faster local feedback:

```bash
BUILD_DEBOUNCE=0 go run ./cmd/worker
```

## One-off static build

```bash
go run ./cmd/build
```

Static pages are written to `public/` and served at `/clubs/<slug>/` when the server is running. Assets are copied to `public/assets/`.

## CSS (Tailwind + DaisyUI)

```bash
npm run build:css
```

This writes `static/admin/admin.css` and copies it to `static/site/site.css`.

## Configuration

| Variable | Default | Description |
| --- | --- | --- |
| `DATA_PATH` | `data/store.db` | SQLite database path |
| `OUTPUT_DIR` | `public` | Static site output directory |
| `TEMPLATE_DIR` | `templates/site` | Static site template directory |
| `ASSET_DIR` | `static/site` | Static site assets directory |
| `SESSION_TTL` | `24h` | Session lifetime |
| `COOKIE_SECURE` | `false` | Set `true` when serving over HTTPS |
| `BUILD_DEBOUNCE` | `2m` | Delay before a queued build runs |
| `BUILD_POLL_INTERVAL` | `5s` | Worker queue polling interval |
| `BUILD_RETRY_DELAY` | `5m` | Retry delay after a failed build |
| `BUILD_NIGHTLY_AT` | `03:00` | Nightly build time (`HH:MM`) |
