# AGENTS.md

## Zweck
Dieses Projekt ist ein Vereinsportal fuer Sportvereine. Admins pflegen Vereinsdaten, die oeffentliche
Seite wird als statische Seite generiert. Nutzer können gezielt nach passenden Vereinen suchen. 

## Tech-Stack
- Go Backend mit **graft** (Routing/Auth + Admin UI) — https://github.com/janmarkuslanger/graft
- Statische Site-Generierung mit **ssgo** — https://github.com/janmarkuslanger/ssgo
- SQLite via GORM
- UI mit Tailwind + **DaisyUI**

## Wichtige Pfade
- Server: `cmd/server`
- Worker: `cmd/worker`
- Statischer Build: `cmd/build`
- Admin-Templates: `templates/admin`
- Public-Templates: `templates/public`
- Clubseiten-Templates: `templates/site`
- Assets (Source): `static/admin`, `static/site`
- Output: `public/assets`, `public/clubs`
- App-Name/Copy: `internal/i18n`

## Build/Run
- CSS bauen: `npm run build:css`
- Server: `go run ./cmd/server`
- Worker: `go run ./cmd/worker`
- Einmaliger Build: `go run ./cmd/build`

## Hinweise
- Public Pages laden CSS aus `/assets/site.css` (kommt aus `static/site`).
- Neue Assets fuer die statische Seite nach `static/site` legen (werden beim Build kopiert).
