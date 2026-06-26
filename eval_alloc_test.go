package conditions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEvalAllocBooleanArgs documents that bool context values reuse singletons
// (no per-eval BooleanLiteral heap allocation).
func TestEvalAllocBooleanArgs(t *testing.T) {
	expr, err := Parse(`{a} AND {b} OR {c}`)
	require.NoError(t, err)
	args := map[string]interface{}{"a": true, "b": false, "c": true}

	const runs = 200
	allocs := testing.AllocsPerRun(runs, func() {
		_, _ = Evaluate(expr, args)
	})
	assert.Equal(t, float64(0), allocs, "bool-only resolve should not allocate on eval hot path")
}

// TestEvalAllocScalarCompare documents pooled string literals for a simple compare.
// On master and this branch, 1 alloc/op remains (interface/map overhead).
func TestEvalAllocScalarCompare(t *testing.T) {
	expr, err := Parse(`{foo} == "hello"`)
	require.NoError(t, err)
	args := map[string]interface{}{"foo": "hello"}

	const runs = 200
	allocs := testing.AllocsPerRun(runs, func() {
		_, _ = Evaluate(expr, args)
	})
	assert.Equal(t, float64(1), allocs)
}

// TestEvalMultiScalarVar matches BenchmarkEvalMultiScalarVar (parentheses required).
func TestEvalMultiScalarVar(t *testing.T) {
	expr, err := Parse(`({a} == "x" AND {b} == 1) AND ({c} == "y" AND {d} == 2)`)
	require.NoError(t, err)
	ok, err := Evaluate(expr, map[string]interface{}{
		"a": "x", "b": 1, "c": "y", "d": 2,
	})
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestEvalAllocMultiScalarVar(t *testing.T) {
	expr, err := Parse(`({a} == "x" AND {b} == 1) AND ({c} == "y" AND {d} == 2)`)
	require.NoError(t, err)
	args := map[string]interface{}{"a": "x", "b": 1, "c": "y", "d": 2}

	const runs = 100
	allocs := testing.AllocsPerRun(runs, func() {
		_, _ = Evaluate(expr, args)
	})
	// Four pooled scalars + residual eval overhead; guard against large regressions.
	assert.LessOrEqual(t, allocs, float64(6))
}