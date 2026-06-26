# sqllint

A fast, extensible SQL linter written in Go.

## Installation

```sh
# Install latest release via go install
go install github.com/ryanxiao/sqllint/cmd/sqllint@latest

# Or download a pre-built binary from the Releases page
```

## Quick Start

```sh
# Lint a file
sqllint query.sql

# Lint from stdin (pipe-friendly)
cat query.sql | sqllint

# Auto-fix violations
sqllint --fix query.sql

# JSON output for CI
sqllint --format json query.sql

# SARIF output (for GitHub Code Scanning)
sqllint --format sarif query.sql

# Lint multiple files / glob
sqllint migrations/*.sql
```

## Rules

| Rule ID                 | Severity | Description                                                     |
|-------------------------|----------|-----------------------------------------------------------------|
| `select-star`           | warning  | Flags `SELECT *` usage                                          |
| `missing-where`         | error    | Flags `DELETE`/`UPDATE` without `WHERE`                         |
| `keyword-casing`        | warning  | Flags mixed-case SQL keywords                                   |
| `trailing-semicolon`    | warning  | Flags missing `;` on last statement                             |
| `leading-wildcard`      | warning  | Flags `LIKE '%foo'` patterns that can't use an index            |
| `implicit-join`         | warning  | Flags comma-style joins (`FROM a, b`) — prefer explicit `JOIN`  |
| `not-in-nullable`       | warning  | Flags `NOT IN` with a subquery that may return `NULL`           |
| `unused-alias`          | warning  | Flags table aliases that are never referenced                   |
| `missing-group-by-col`  | error    | Flags columns in `SELECT` not in `GROUP BY` or an aggregate     |

Rules marked **error** cause exit code 1; **warning** rules cause exit code 2.

### How rules work

Most rules are **AST-based**: the SQL is parsed into a syntax tree and the rule
inspects nodes (e.g. `implicit-join` flags a `FROM` clause with more than one
top-level item; `leading-wildcard` reads the actual `LIKE` pattern constant).
This avoids the false positives of naive text matching — commas inside function
calls or `IN (...)` lists, `%` inside unrelated string literals, and multi-line
clauses are all handled correctly.

A small number of rules are **lexical** (`keyword-casing`, `trailing-semicolon`)
and operate on the raw source text by design: keyword casing and a trailing `;`
are properties the parser normalizes away, so they can only be checked against
the text itself.

The bundled parser does not emit source offsets, so AST rules resolve line
numbers with a comment- and string-aware text search (see
`internal/rules/ast_helpers.go`) — matches inside comments or string literals
are ignored so violations point at the real offending line.

### Limitations

The AST-based rules parse **PostgreSQL** syntax (via
[`pgplex/pgparser`](https://github.com/pgplex/pgparser)). Dialect-specific
syntax from other databases may not parse; when parsing fails, sqllint falls
back to the lexical rules only and prints a warning.

## Auto-fix (`--fix`)

`--fix` rewrites files in place (atomic write) for mechanically fixable violations.
Currently fixes: **keyword-casing** and **trailing-semicolon**.

```sh
# Before
echo "select id from users where active = 1" | sqllint --fix
# After (stdout)
SELECT id FROM users WHERE active = 1;

# Fix files in place
sqllint --fix migrations/*.sql
# fixed: migrations/001_init.sql
```

`--fix` and `--format` are mutually exclusive.

## Configuration (`.sqllint.yaml`)

```yaml
rules:
  select-star:
    enabled: false
  missing-where:
    severity: error
```

## CI Integration (GitHub Actions)

```yaml
- name: Lint SQL
  uses: ryanxiao/sqllint/.github/workflows/sqllint.yml@main
  with:
    path: "migrations/*.sql"
```

Or run directly:

```yaml
- run: go install github.com/ryanxiao/sqllint/cmd/sqllint@latest
- run: sqllint --format sarif migrations/*.sql > results.sarif
- uses: github/codeql-action/upload-sarif@v3
  with:
    sarif_file: results.sarif
```

## Development

```sh
# Run all tests
go test ./...

# Build
go build ./cmd/sqllint

# GoReleaser snapshot (requires goreleaser)
goreleaser release --snapshot --clean
```

## Suppressing violations

Add `-- sqllint:ignore` to any line to suppress all violations on that line:

```sql
SELECT * FROM users -- sqllint:ignore
```
