# Evaluate literal pool — benchmarks

Measured on **linux/arm64** with `go test -benchmem -count=3 ./...`.

## vs `master` (pre-pool)

| Benchmark | master | this branch |
|-----------|--------|-------------|
| `BenchmarkBooleanOperators` (`{a} AND {b} OR {c}`, bool args) | **3 allocs/op**, 3 B/op, ~69 ns/op | **0 allocs/op**, 0 B/op, ~46 ns/op |
| `BenchmarkSimpleComparison` (`{foo} == "hello"`) | 1 alloc/op, 16 B/op, ~40 ns/op | 1 alloc/op, 16 B/op, ~44 ns/op |
| `BenchmarkNumericComparison` | 2 allocs/op, 16 B/op, ~74 ns/op | 2 allocs/op, 24 B/op, ~91 ns/op |
| `BenchmarkEvalMultiScalarVar` (4 scalar bindings, parenthesized) | — | ~205 ns/op, 4 allocs/op, 72 B/op |

The largest win is **boolean context values**: resolving `true`/`false` from `args` no longer allocates `&BooleanLiteral{}` per variable.

Scalar string/number literals are stored in per-`Evaluate` `evalPool` slices instead of separate heap objects; remaining allocs are mostly map/interface and comparison overhead.

## Reproduce

```bash
go test -bench='Benchmark(BooleanOperators|SimpleComparison|EvalMultiScalarVar)' -benchmem -count=3 ./...
go test -run='TestEvalAlloc' -v ./...
```

Compare to `master`:

```bash
git checkout master
go test -bench=BenchmarkBooleanOperators -benchmem -count=3 ./...
git checkout perf/eval-literal-pool
```