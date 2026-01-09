# club-portal

Minimaler Start fuer ein Sportvereins-Portal: Vereinsleiter koennen sich anmelden, ihren Clubnamen + Beschreibung pflegen, und das Frontend wird statisch generiert.

## Features

- Backend mit `graft` (Login, Registrierung, Clubdaten)
- Statische Clubseiten mit `ssgo`
- GORM Storage (SQLite) fuer saubere Datenabstraktion

## Lokales Setup

```bash
go run ./cmd/server
```

Danach unter `http://localhost:8080` oeffnen.

Hinweis: Der Static-Build laeuft ueber einen Worker-Prozess (siehe unten). Fuer aktuelle Seiten sollte der Worker parallel laufen.

## Styles (DaisyUI)

```bash
npm install
npm run build:css
```

Die gebaute CSS liegt in `static/admin/admin.css` und `static/site/site.css`.

## Static Build

```bash
go run ./cmd/build
```

Die Seiten landen in `public/` und koennen lokal ueber `/clubs/<slug>/` aufgerufen werden (wenn der Server laeuft).

## Build Worker (Queue + Nightly)

```bash
go run ./cmd/worker
```

Der Worker verarbeitet die Build-Queue (debounced) und startet jede Nacht einen kompletten Build.

Wichtig:
- Es gibt keinen manuellen Build-Button mehr.
- Beim Speichern wird ein Build-Task in der DB-Queue aktualisiert.
- Der Worker baut erst nach Ablauf von `BUILD_DEBOUNCE` (default 2m).

Schneller fuer lokale Tests:
```bash
BUILD_DEBOUNCE=0 go run ./cmd/worker
```

## Konfiguration (ENV)

- `DATA_PATH` (default: `data/store.db`)
- `OUTPUT_DIR` (default: `public`)
- `TEMPLATE_DIR` (default: `templates/site`)
- `ASSET_DIR` (default: `static/site`)
- `SESSION_TTL` (default: `24h`)
- `COOKIE_SECURE` (default: `false`)
- `BUILD_DEBOUNCE` (default: `2m`)
- `BUILD_POLL_INTERVAL` (default: `5s`)
- `BUILD_RETRY_DELAY` (default: `5m`)
- `BUILD_NIGHTLY_AT` (default: `03:00`)
