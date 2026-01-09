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

## Konfiguration (ENV)

- `DATA_PATH` (default: `data/store.db`)
- `OUTPUT_DIR` (default: `public`)
- `TEMPLATE_DIR` (default: `templates/site`)
- `ASSET_DIR` (default: `static/site`)
- `SESSION_TTL` (default: `24h`)
- `COOKIE_SECURE` (default: `false`)
