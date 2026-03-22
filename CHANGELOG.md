# Changelog

## [0.1.0] - 2026-03-21

### Added
- 9 lint rules: select-star, missing-where, keyword-casing, trailing-semicolon,
  leading-wildcard, implicit-join, not-in-nullable, unused-alias, missing-group-by-col
- AST-based rules powered by pgparser for accurate analysis
- `--fix` flag: auto-fixes keyword casing and trailing semicolons (atomic file rewrite)
- `--format` flag: text (default), JSON, and SARIF output
- Concurrent file linting with glob pattern support
- Exit codes: 0 = clean, 1 = errors, 2 = warnings only
- `sqllint:ignore` inline comment to suppress violations per line
- `.sqllint.yaml` config file for per-rule enable/severity overrides
- GitHub Actions workflow for CI linting with SARIF upload
- GoReleaser pipeline: Linux/macOS/Windows binaries (amd64 + arm64)
