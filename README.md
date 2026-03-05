# RSPS — Radically Simpler Programming System (V1 / MVP)

RSPS ist eine **metadata-driven Application Platform** in Go.  
Ein Entwickler beschreibt seine Anwendung in **einer einzigen DSL-Datei** (`app.rsps`).

Aus dieser Datei generiert RSPS automatisch:

- Datenbankschema (SQLite)
- Migrationen (safe-only)
- REST-API (CRUD)
- serverseitige HTML-UI (List/Create/Edit/Delete)
- Runtime-Metadaten für Validierung und Query-Building

Das Ziel von V1 ist bewusst: **maximal einfach, modular, nachvollziehbar, Open-Source-freundlich**.

---

## Quickstart (30 Sekunden)

```bash
git clone https://github.com/ribershamoelias/rsps.git
cd rsps
go run ./cmd/rsps run ./example/app.rsps
```

Danach im Browser öffnen:

- `http://localhost:8080/ui/project`
- `http://localhost:8080/api/project`

---

## Dokumentation

- `README.md` — Überblick, Quickstart, Features, Workflow
- `docs/architecture.md` — technischer Deep Dive (Compiler/Runtime/Migrationen)
- `CONTRIBUTING.md` — Contribution-Prozess und Engineering-Regeln
- `CHANGELOG.md` — Versionen und Änderungen
- `LICENSE` — Lizenzbedingungen

---

## Inhalt

1. [Quickstart (30 Sekunden)](#quickstart-30-sekunden)
2. [Dokumentation](#dokumentation)
3. [Projektstatus](#projektstatus)
4. [Philosophie](#philosophie)
5. [Features V1](#features-v1)
6. [Out of Scope in V1](#out-of-scope-in-v1)
7. [Technologie-Stack](#technologie-stack)
8. [Projektstruktur](#projektstruktur)
9. [Architektur im Detail](#architektur-im-detail)
10. [DSL-Spezifikation](#dsl-spezifikation)
11. [Build- und Run-Workflow](#build--und-run-workflow)
12. [API-Spezifikation](#api-spezifikation)
13. [UI-Spezifikation](#ui-spezifikation)
14. [Migrationen & Schema-Hash](#migrationen--schema-hash)
15. [Beispielprojekt (Todo)](#beispielprojekt-todo)
16. [Weitere Beispielprojekte](#weitere-beispielprojekte)
17. [Entwicklung & lokale Nutzung](#entwicklung--lokale-nutzung)
18. [Fehlerbehandlung & Troubleshooting](#fehlerbehandlung--troubleshooting)
19. [Sicherheit, Grenzen, Performance](#sicherheit-grenzen-performance)
20. [Roadmap nach V1](#roadmap-nach-v1)
21. [Beitragen](#beitragen)

---

## Projektstatus

Dieses Repository enthält einen **funktionalen RSPS V1-Prototypen** mit:

- eigenem Lexer + Parser (recursive descent)
- AST-Modell + semantischer Validierung
- Schema-Generator + Migrationslogik
- generischer metadata-getriebener CRUD-Repository-Schicht
- REST-API-Generator
- HTML-UI-Generator über `html/template`
- CLI mit `build` und `run`

---

## Philosophie

Moderne CRUD-Anwendungen leiden oft unter doppelter Modellierung:

- DB-Schema separat
- API-Modelle separat
- Validierungsregeln separat
- UI-Formulare separat

Das erzeugt Inkonsistenz und erhöht Aufwand. RSPS ersetzt das durch:

> **Single Source of Truth:** Die DSL beschreibt die Struktur einmal, RSPS erzeugt den Rest deterministisch.

Damit verschiebt sich die Arbeit von Infrastruktur-Wiring hin zu Domänenmodellierung.

---

## Features V1

- Deklarative App-Definition über `app.rsps`
- Entity/Field-Modell mit Typen, Nullable, Defaults, `@unique`, `@index`
- Referenzen via `ref <entity>`
- SQL-Generierung für SQLite
- Safe-Migrationen (nur additive/kompatible Änderungen)
- REST-Endpunkte pro Entity:
	- `GET /api/{entity}`
	- `GET /api/{entity}/{id}`
	- `POST /api/{entity}`
	- `PUT /api/{entity}/{id}`
	- `DELETE /api/{entity}/{id}`
- Auto-generierte HTML-UI:
	- `/ui/{entity}`
	- `/ui/{entity}/new`
	- `/ui/{entity}/{id}/edit`
	- Delete via `POST /ui/{entity}/{id}/delete`

---

## Out of Scope in V1

Bewusst **nicht** enthalten:

- Authentifizierung/Autorisierung
- verteilte Systeme/Microservices
- Realtime/WebSocket
- Plugin-Ökosystem
- komplexe Frontend-Framework-Integration
- Mobile-Generierung

Diese Reduktion hält den Kern robust und verständlich.

---

## Technologie-Stack

- **Sprache:** Go
- **HTTP:** Go Standardbibliothek (`net/http`)
- **Templating:** `html/template`
- **JSON:** `encoding/json`
- **Datenbank:** SQLite (`modernc.org/sqlite`)
- **Parser:** eigener Lexer + recursive descent parser

### Warum dieser Stack?

- minimale Abhängigkeiten
- geringe Laufzeitkomplexität
- einfache Installation und Distribution
- sehr gut für Open-Source-MVP geeignet

---

## Projektstruktur

```text
rsps/
├── cmd/
│   └── rsps/
│       └── main.go                # CLI entrypoint: build/run
├── docs/
│   └── architecture.md            # Technischer Deep Dive
├── internal/
│   ├── api/
│   │   └── handler.go             # Dynamische REST-CRUD-Generierung
│   ├── ast/
│   │   └── ast.go                 # AST-Modelle
│   ├── db/
│   │   ├── repository.go          # Generisches CRUD auf Metadatenbasis
│   │   └── sqlite.go              # SQLite-Verbindung + PRAGMAs
│   ├── lexer/
│   │   ├── lexer.go               # Tokenisierung inkl. line/column
│   │   └── token.go               # Token-Typen + Keyword-Mapping
│   ├── migrations/
│   │   └── migrations.go          # Schema-Hash + Safe-Migrationen
│   ├── parser/
│   │   ├── parse.go               # ParseString/ParseFile API
│   │   └── parser.go              # Recursive descent parser
│   ├── runtime/
│   │   ├── registry.go            # Runtime-Metadaten (Entities/Fields)
│   │   └── validation.go          # Payload-Validierung & Typ-Coercion
│   ├── schema/
│   │   └── schema.go              # AST -> SQL/Schema/Index/FK
│   ├── ui/
│   │   └── handler.go             # Server-seitige UI-Generierung
│   └── validator/
│       └── validator.go           # Semantische DSL-Validierung
├── templates/
│   ├── ui_form.html               # Create/Edit Form
│   └── ui_list.html               # Tabellenansicht mit Aktionen
├── example/
│   ├── app.rsps                   # Beispiel-Todo-App
│   ├── blog.rsps                  # Blog-Beispiel
│   ├── inventory.rsps             # Inventory-Beispiel
│   └── issue_tracker.rsps         # Issue-Tracker-Beispiel
├── .gitignore
├── CHANGELOG.md
├── CONTRIBUTING.md
├── LICENSE
├── go.mod
├── go.sum
└── README.md
```

---

## Architektur im Detail

Für den technischen Deep Dive mit Datenfluss, Schichtgrenzen und Erweiterungspunkten siehe auch `docs/architecture.md`.

### 1) Compile-Pipeline (`rsps build`)

1. DSL-Datei lesen (`parser.ParseFile`)
2. Tokenisieren (`internal/lexer`)
3. AST bauen (`internal/parser`)
4. Semantik validieren (`internal/validator`)
5. Schema erzeugen (`internal/schema`)
6. Runtime-Registry erzeugen (`internal/runtime`)
7. Artefakte in `.rsps/` schreiben

Erzeugte Artefakte:

- `schema.sql` — SQL DDL
- `schema.json` — strukturiertes Schema
- `schema.hash` — SHA256 über das Schema
- `metadata.json` — Runtime-Metadaten

Hinweis: Diese Dateien liegen immer neben der DSL-Datei, also z. B. bei `./example/.rsps/`.

### 2) Runtime-Pipeline (`rsps run`)

1. Führt intern erneut `build` aus (synchronisiert Artefakte)
2. Öffnet SQLite DB (`internal/db/sqlite.go`)
3. Aktiviert PRAGMAs:
	 - `foreign_keys = ON`
	 - `busy_timeout = 5000`
4. Wendet Migrationen an (`internal/migrations`)
5. Initialisiert Repository + API + UI Handler
6. Startet HTTP-Server (`:8080` default oder `RSPS_ADDR`)

### 3) Runtime-Kernprinzip

Die Registry (`internal/runtime/registry.go`) mappt DSL-Elemente auf Laufzeitdaten:

- Entity -> Tabelle
- Field -> Spalte
- `ref` -> Foreign-Key-Spalte (`<field>_id`)
- Attribute/Nullable/Default -> Validierung + SQL-Verhalten

API, UI und Repository nutzen diese Registry gemeinsam. Dadurch bleibt das Verhalten konsistent.

---

## DSL-Spezifikation

### Grundstruktur

```rsps
app <app_name> {
	<entity_name> {
		<field_name> <type>
	}
}
```

### Feldsyntax

- `field_name type`
- `field_name type?` (nullable)
- `field_name ref entity_name`
- `field_name type @unique`
- `field_name type @index`
- `field_name type = default`

### Unterstützte Typen

- `string`
- `text`
- `int`
- `float`
- `bool`
- `date`
- `datetime`
- `json`
- `ref <entity>` (Beziehung)

### Defaults (V1)

- String-Literale: `"..."`
- Zahlen: `123`, `1.5`
- Bool: `true` / `false`
- Zeitstempel: `now` (für `datetime`)

### Attribute

- `@unique` -> Unique Index
- `@index` -> Non-Unique Index

### Nullability

`?` direkt nach dem Typ/Reference-Target macht das Feld nullable.

Beispiele:

```rsps
due datetime?
project ref project?
```

---

## Typ-Mapping nach SQLite

| DSL-Typ | SQLite-Typ |
|--------|------------|
| string | TEXT |
| text | TEXT |
| int | INTEGER |
| float | REAL |
| bool | INTEGER (0/1) |
| date | TEXT |
| datetime | TEXT |
| json | TEXT |
| ref | INTEGER (`<field>_id`) |

Automatisch je Tabelle:

```sql
id INTEGER PRIMARY KEY AUTOINCREMENT
```

---

## Semantische Validierung

`internal/validator` prüft u. a.:

- doppelte Entity-Namen
- doppelte Feldnamen in einer Entity
- ungültige Feldtypen
- ungültige/missing Referenzen
- nicht vorhandene Referenz-Ziel-Entities
- unzulässige Defaults (z. B. Default auf `ref`)
- duplicate Attribute auf einem Feld

Fehler enthalten Positionsinformationen (`line:column`) aus Lexer/Parser.

---

## Build- und Run-Workflow

### Voraussetzungen

- Go Toolchain passend zu `go.mod` (aktuell `go 1.24.0`)
- Linux/macOS/Windows mit Netzwerkzugang für `go mod` beim ersten Build

### Development Commands

```bash
# 1) Build-Artefakte aus DSL erzeugen
go run ./cmd/rsps build ./example/app.rsps

# 2) Anwendung starten (inkl. Migrationen)
go run ./cmd/rsps run ./example/app.rsps
```

### Optionale Umgebungsvariablen

- `RSPS_ADDR` — HTTP-Adresse (default `:8080`)
- `RSPS_TEMPLATE_DIR` — expliziter Template-Ordner mit `ui_list.html` und `ui_form.html`

Beispiel:

```bash
RSPS_ADDR=:9090 go run ./cmd/rsps run ./example/app.rsps
```

---

## API-Spezifikation

Für jede Entity `x` werden erzeugt:

- `GET /api/x` -> Liste
- `GET /api/x/{id}` -> Einzelobjekt
- `POST /api/x` -> Erstellen
- `PUT /api/x/{id}` -> Aktualisieren
- `DELETE /api/x/{id}` -> Löschen

### API-Verhalten

- JSON Input/Output
- Feldvalidierung über Runtime-Metadaten
- Unknown fields werden abgelehnt
- Typparsing für `int/float/bool/ref`
- `@unique` / FK-Verletzungen führen zu Konfliktfehlern

### Typische HTTP-Statuscodes

- `200` OK
- `201` Created
- `400` Bad Request (Validierung)
- `404` Not Found
- `409` Conflict (Constraint/FK/Unique)
- `500` Internal Error

### Beispiel-Requests

```bash
# Create project
curl -X POST http://localhost:8080/api/project \
	-H "Content-Type: application/json" \
	-d '{"name":"Personal","color":"blue"}'

# List tasks
curl http://localhost:8080/api/task

# Update task
curl -X PUT http://localhost:8080/api/task/1 \
	-H "Content-Type: application/json" \
	-d '{"done":true}'
```

---

## UI-Spezifikation

Die UI ist serverseitig gerendert (kein SPA-Framework) und wird vollständig aus Metadaten erzeugt.

### Routen

- `/ui/{entity}` -> List View
- `/ui/{entity}/new` -> Create Form
- `/ui/{entity}/{id}/edit` -> Edit Form
- `/ui/{entity}/{id}/delete` -> Delete Action (POST)

### Feld-Widget-Mapping

- `string`, `json` -> text input
- `text` -> textarea
- `int`, `float` -> number input
- `bool` -> checkbox
- `date` -> date input
- `datetime` -> datetime-local input
- `ref` -> select (Lookup auf Referenz-Entity)

### UI-Templates

- `templates/ui_list.html`
- `templates/ui_form.html`

Die Templates sind bewusst minimal, leicht austauschbar und ohne JS-Build-Tooling.

---

## Migrationen & Schema-Hash

RSPS nutzt eine einfache, sichere Migrationsstrategie:

- Tabelle `rsps_migrations` speichert je App den letzten `schema_hash`
- Bei Start wird der neue Hash mit dem gespeicherten Hash verglichen
- Nur **safe** Änderungen werden automatisch ausgeführt

### Erlaubt (safe)

- neue Tabelle
- neue Spalte (nur wenn nullable oder mit default)
- neue Indexe

### Verboten (destructive/unsafe)

- Entfernen von Tabellen
- Entfernen von Spalten
- inkompatible Spaltentypänderung
- Hinzufügen einer PK-Spalte zu existierender Tabelle

Bei unsafe Änderungen bricht RSPS bewusst mit einer klaren Fehlermeldung ab.

---

## Beispielprojekt (Todo)

Datei: `example/app.rsps`

```rsps
app todo {
	project {
		name string @unique
		color string
	}

	task {
		title string
		done bool = false
		due datetime?
		project ref project
	}
}
```

### Was RSPS daraus generiert

- Tabellen:
	- `project(id, name, color)`
	- `task(id, title, done, due, project_id)`
- Foreign Key: `task.project_id -> project.id`
- API-Routen für `project` und `task`
- UI-Routen für `project` und `task`

---

## Weitere Beispielprojekte

Zusätzlich zur Todo-App sind weitere DSL-Beispiele enthalten:

- `example/blog.rsps`
- `example/inventory.rsps`
- `example/issue_tracker.rsps`

Diese Beispiele helfen dabei, verschiedene Domänenmodelle mit RSPS schnell zu verstehen und zu testen.

---

## Entwicklung & lokale Nutzung

### 1) Repository starten

```bash
git clone https://github.com/ribershamoelias/rsps.git
cd rsps
```

### 2) Optional: Binary bauen

```bash
go build -o bin/rsps ./cmd/rsps
```

### 3) Build + Run

```bash
./bin/rsps build ./example/app.rsps
./bin/rsps run ./example/app.rsps
```

oder direkt mit `go run`:

```bash
go run ./cmd/rsps build ./example/app.rsps
go run ./cmd/rsps run ./example/app.rsps
```

### 4) Ergebnis prüfen

- UI: `http://localhost:8080/ui/`
- API: `http://localhost:8080/api/project`

---

## Fehlerbehandlung & Troubleshooting

### Häufige Probleme

1. **`missing go.sum entry`**
	 - Lösung: `go mod tidy`

2. **Template nicht gefunden**
	 - Prüfen, ob `templates/ui_list.html` und `templates/ui_form.html` existieren
	 - Alternativ `RSPS_TEMPLATE_DIR` setzen

3. **Unsafe Migration Fehler**
	 - Schemaänderung ist destruktiv oder inkompatibel
	 - Erwartetes Verhalten in V1 (Absicherung gegen Datenverlust)

4. **`entity not found` / `unknown field`**
	 - DSL prüfen und `build` erneut ausführen

### Artefakte neu erzeugen

```bash
rm -rf ./example/.rsps
go run ./cmd/rsps build ./example/app.rsps
```

### Lokale Testdatenbank zurücksetzen

```bash
rm -f ./example/todo.sqlite
go run ./cmd/rsps run ./example/app.rsps
```

---

## Sicherheit, Grenzen, Performance

### Sicherheit

- SQL-Statements werden über prepared statements ausgeführt
- Foreign Keys sind in SQLite aktiviert
- Input-Validierung erfolgt vor CRUD-Operationen

### Grenzen V1

- keine Authentifizierung
- keine Rollen/Rechte
- keine komplexen Query-Filter
- keine Pagination/Sorting-API
- keine Custom Business Logic Hooks

### Performance-Charakteristik

- Für kleine bis mittlere CRUD-Workloads geeignet
- Single-Process + SQLite = schneller Start, sehr geringe Ops-Last
- Nicht für horizontale Skalierung in V1 ausgelegt

---

## Roadmap nach V1

Mögliche nächste Schritte:

- AuthN/AuthZ (Session/JWT + Policies)
- Realtime Events (SSE/WebSocket)
- Postgres-Backend neben SQLite
- Hook-System für Custom-Logik
- Erweiterte UI-Generierung (Filter, Pagination)

Wichtig: Die V1-Architektur (DSL -> AST -> Schema/Registry -> Runtime) bleibt der stabile Kern.

---

## Beitragen

Beiträge sind willkommen.

Empfohlener Ablauf:

1. Fork erstellen
2. Branch pro Feature/Fix
3. Kleine, fokussierte Changes einreichen
4. PR mit Begründung + Test-/Reproduktionsschritten

### Coding-Prinzipien im Projekt

- kleine, klar getrennte Pakete
- explizite Fehlerbehandlung
- keine schweren Frameworks
- Runtime-Verhalten aus Metadaten ableiten statt hardcodieren
