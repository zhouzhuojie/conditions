# Evaluate literal pool — benchmarks

Compare allocation behavior on `master` vs `perf/eval-literal-pool`:

```bash
# Current branch
go test -bench='Benchmark(BooleanOperators|SimpleComparison|NumericComparison|EvalMultiScalarVar)' -benchmem -count=3 ./...

# Baseline (master)
git stash -q 2>/dev/null; git checkout master
go test -bench='Benchmark(BooleanOperators|SimpleComparison|NumericComparison)' -benchmem -count=3 ./...
git checkout -
git stash pop -q 2>/dev/null
```

## What improved

| Benchmark | Expected gain |
|-----------|----------------|
| `BenchmarkBooleanOperators` | **0 allocs/op** — context `bool` values use `trueExpr`/`falseExpr` instead of `&BooleanLiteral{}` |
| `BenchmarkSimpleComparison` / path access | Fewer tiny heap allocs for resolved `string`/`number` (pooled in `evalPool` slices); `B/op` may still show string/map overhead |
| `BenchmarkEvalMultiScalarVar` | Several resolved scalars per eval — compare `allocs/op` vs pre-pool master (rebuild benchmark on master or use `testing.AllocsPerRun` in `eval_alloc_test.go`) |

Run `go test -run='TestEvalAlloc'` for regression guards on allocs.