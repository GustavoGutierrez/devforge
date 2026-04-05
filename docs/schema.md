# Removed database schema

DevForge no longer includes a bundled database layer.

Removed in the utility-focused refactor:

- SQLite/libSQL schema
- FTS5 search tables
- vector/embedding storage
- runtime `devforge.db`

This file remains only to document that the schema-based subsystem was intentionally removed as a breaking change.
