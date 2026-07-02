# Repository Guidelines

## Project Overview

Go library **`github.com/zhouzhuojie/conditions`**: **parse once, evaluate many** — boolean DSL → immutable `Expr` AST, then `Evaluate(expr, map[string]interface{})`. JSON/FBP-friendly; stdlib-only at runtime (`testify` tests only). No `main` or CLI.

- Docs: `README.md` · CI: `.github/workflows/ci.yml` (branch `master`)

## Architecture & Data Flow

**Single package `conditions` at repo root** — all `.go` and `*_test.go` files live here (no `src/`, `internal/`, `cmd/`).

`Parse` → `parser.go` / `token.go` → `ast.go` → `Evaluate` (`evaluator.go`, `operators.go`, `resolve.go`, `regex.go`, `float.go`).

**Public API:** `Parse`, `Evaluate`, `Variables`, `Walk`/`WalkFunc`, `SetDefaultEpsilon` (before concurrent eval).

**Variables in expressions:** `{key}` flat; `{a.b}` / `{users[0]}` nested (`resolve.go`); `{foo}{bar}` → `foo.bar`. Parsed AST is goroutine-safe; set epsilon early if needed.

## Development Commands

```bash
go test -v -race -cover ./...   # CI
go test ./...
go fmt ./...
golangci-lint run ./...         # CI lint
go test -bench=. ./...
```

Go **1.24** (`go.mod`). Format with `go fmt`; no Makefile/npm/Docker.

## Code Conventions & Patterns

- Errors via `(T, error)`; no panics on user data.
- `AND`/`OR` short-circuit in `evaluator.go`.
- `booleanFromExpr` fast path for `AND`/`OR`; `boolExpr` singletons for bool args (`evaluator.go`, `resolve.go`).
- `applyIN` uses direct type switches (`operators.go`); string `IN`/`CONTAINS` maps built at parse (AST) or per-eval from args (no global slice cache).
- Syntax/precedence changes: `parser.go`, `token.go`, `operators.go` + tests.
- Binding/coercion changes: `resolve.go` + `evaluator_test.go` / `resolve_test.go`.
- Table-driven tests; README examples in `TestReadme*` (`evaluator_test.go`).

## Testing & QA

Colocated `*_test.go`; benchmarks in `parser_bench_test.go`. CI: race + cover + golangci-lint. Extend table tests and README parity tests when behavior changes.