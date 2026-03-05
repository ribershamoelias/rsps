# RSPS Architecture Deep Dive (V1)

This document describes the implemented RSPS V1 architecture as a technical reference.

## 1. End-to-End Flow

```text
app.rsps
  -> lexer
  -> parser
  -> AST
  -> semantic validator
  -> schema generator
  -> runtime registry
  -> migrations + repository
  -> API + UI handlers
```

The core design rule is that **runtime behavior is derived from metadata**, not duplicated manually.

## 2. Module Responsibilities

### `cmd/rsps`

- CLI entrypoint
- command dispatch (`build`, `run`)
- orchestrates compile/runtime pipeline

### `internal/lexer`

- token stream with line/column tracking
- recognizes keywords (`app`, `ref`, `true`, `false`, `now`), types, symbols, attributes

### `internal/parser`

- recursive descent parser
- builds AST for application/entities/fields
- parses nullable marker, attributes, references, defaults

### `internal/ast`

- canonical compile-time model used across generator and runtime layers

### `internal/validator`

- semantic checks:
  - duplicate entities
  - duplicate fields
  - invalid field type
  - invalid/missing references
  - unsupported defaults by type

### `internal/schema`

- AST to schema model transformation
- SQL type mapping
- index and FK derivation
- deterministic SQL and schema hash generation

### `internal/migrations`

- migration state table: `rsps_migrations`
- schema-hash compare for change detection
- applies only safe changes:
  - new table
  - new nullable/defaulted column
  - new index
- rejects destructive/incompatible changes

### `internal/runtime`

- metadata registry for runtime entity/field lookups
- payload validation and type coercion

### `internal/db`

- SQLite connection + pragmas
- generic CRUD repository driven by runtime metadata

### `internal/api`

- dynamic CRUD route handling for `/api/{entity}` and `/api/{entity}/{id}`
- validation + repository integration

### `internal/ui`

- server-rendered HTML CRUD UI
- generated forms from metadata
- reference fields rendered as select lists

## 3. DSL Surface (V1)

Supported field declarations:

- `<name> <type>`
- `<name> <type>?`
- `<name> ref <entity>`
- `<name> <type> @unique`
- `<name> <type> @index`
- `<name> <type> = <default>`

Supported scalar types:

- `string`, `text`, `int`, `float`, `bool`, `date`, `datetime`, `json`

## 4. SQL Mapping Rules

- every entity => one table
- implicit primary key:

```sql
id INTEGER PRIMARY KEY AUTOINCREMENT
```

- references map to `<field>_id INTEGER` + foreign key

Type mapping:

- `string`, `text`, `date`, `datetime`, `json` => `TEXT`
- `int`, `bool`, `ref` => `INTEGER`
- `float` => `REAL`

## 5. Runtime Request Path

For API create/update:

1. route resolves entity name and optional id
2. JSON payload decode
3. metadata-driven validation
4. repository prepares SQL with placeholders
5. statement execution
6. row mapping back to JSON payload

For UI create/edit:

1. route resolves entity + mode
2. form values parsed and coerced
3. same runtime validation as API
4. same repository CRUD path
5. template render or redirect

## 6. Safety Characteristics

- no runtime-generated SQL from untrusted identifiers
- prepared statements for DML operations
- SQLite foreign keys enabled
- strict metadata validation before DB writes
- destructive migrations are blocked in V1

## 7. Extensibility Paths

The architecture intentionally leaves extension points after V1:

- authentication/authorization middleware layer
- richer query endpoints (filter/pagination/sort)
- hooks before/after CRUD operations
- additional SQL backends
- richer UI rendering strategy

## 8. Non-Goals in V1

- distributed deployment model
- plugin runtime
- realtime sync
- mobile generation
- complex frontend framework support

V1 favors clarity and deterministic behavior over feature breadth.
