# gosq-codegen

Code generator for [gosq](https://github.com/libliflin/gosq) — introspects your PostgreSQL schema and generates type-safe Go table/field definitions.

## What it does

Instead of hand-writing:

```go
var Users      = NewTable("users")
var UsersID    = NewField("users.id")
var UsersName  = NewField("users.name")
var UsersEmail = NewField("users.email")
```

Point `gosq-codegen` at your database and it generates these declarations for every table and column automatically.

## Status

**Work in progress.** The core architecture is in place but the generator is not yet functional.

## Planned usage

```bash
go install github.com/libliflin/gosq-codegen@latest

gosq-codegen -dsn "postgres://user:pass@localhost:5432/mydb" -out schema/
```

## Architecture

- `internal/introspect` — reads schema metadata from PostgreSQL via `information_schema`
- `internal/codegen` — renders Go source files from schema metadata
- `main.go` — CLI that wires introspection to code generation

## License

Same as [gosq](https://github.com/libliflin/gosq).
