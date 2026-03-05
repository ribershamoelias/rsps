# Contributing to RSPS

Danke für dein Interesse, zu RSPS beizutragen.

RSPS ist ein metadata-driven System mit bewusst kleiner, klarer Kernarchitektur. Beiträge sollen diese Einfachheit erhalten.

## Grundprinzipien

- Eine DSL-Datei ist die Single Source of Truth.
- Keine schweren Frameworks ohne zwingenden Grund.
- Änderungen müssen die modulare Architektur respektieren.
- Fehlerbehandlung ist explizit, keine stillen Fehler.

## Lokales Setup

```bash
git clone <your-fork-url>
cd rsps
go mod tidy
go run ./cmd/rsps build ./example/app.rsps
go run ./cmd/rsps run ./example/app.rsps
```

## Entwicklungsworkflow

1. Fork erstellen und Branch anlegen:

```bash
git checkout -b feat/<kurze-beschreibung>
```

2. Änderung implementieren.
3. Bei Go-Code: formatieren und bauen.

```bash
gofmt -w $(find . -name '*.go')
go build ./...
```

4. Funktional testen (mindestens mit einem Beispiel aus `example/`).
5. Pull Request mit klarer Beschreibung erstellen.

## Commit-Empfehlung

Empfohlene Struktur:

```text
type(scope): summary
```

Beispiele:

- `feat(parser): add nullable reference support`
- `fix(migrations): reject incompatible type changes`
- `docs(readme): add quickstart and examples`

## Pull Request Checklist

- [ ] Änderung ist fokussiert und begründet.
- [ ] README / docs aktualisiert (falls Verhalten geändert).
- [ ] Build läuft lokal (`go build ./...`).
- [ ] Kein unnötiger Refactor außerhalb des Scopes.
- [ ] Breaking Change explizit dokumentiert.

## Scope-Regeln für V1

Bitte keine PRs, die V1 unnötig aufblasen (z. B. große Framework-Einführung, Plugin-System, verteilte Architektur), ohne vorherige Diskussion.

## Diskussionen und Issues

Bitte in Issues enthalten:

- erwartetes Verhalten
- aktuelles Verhalten
- reproduzierbare Schritte
- relevante DSL-Datei / Logs / Fehlermeldungen

Danke fürs Beitragen.
