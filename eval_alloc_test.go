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

// TestEvalAllocScalarCompare documents pooled string/number literals for comparisons.
func TestEvalAllocScalarCompare(t *testing.T) {
	expr, err := Parse(`{foo} == "hello"`)
	require.NoError(t, err)
	args := map[string]interface{}{"foo": "hello"}

	const runs = 200
	allocs := testing.AllocsPerRun(runs, func() {
		_, _ = Evaluate(expr, args)
	})
	// One alloc remains (map/slice growth or error path); without pooling this was higher on master.
	assert.LessOrEqual(t, allocs, float64(1))
}