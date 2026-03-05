# Changelog

All notable changes to this project will be documented in this file.

## [0.1.0] - 2026-03-05

Initial RSPS MVP prototype.

### Added

- DSL compiler pipeline: lexer, parser, AST, semantic validator.
- Schema generation for SQLite including references and indexes.
- Migration engine with schema hashing and safe-change strategy.
- Runtime metadata registry as single execution source for API/UI/CRUD.
- Generic CRUD repository with prepared SQL statements.
- Auto-generated REST API for each entity.
- Auto-generated server-rendered HTML UI (list/create/edit/delete).
- CLI commands: `rsps build <app.rsps>` and `rsps run <app.rsps>`.
- Example DSL app: todo.
- Additional example DSL apps: blog, inventory, issue tracker.
- Project documentation for setup, architecture, and contribution workflow.
