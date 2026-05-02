package conditions

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFloat64Equal(t *testing.T) {
	defer SetDefaultEpsilon(1e-6)

	SetDefaultEpsilon(1e-6)
	assert.True(t, float64Equal(0.01, 0.01))
	assert.True(t, float64Equal(0.01, 0.01000001))
	assert.True(t, float64Equal(1e6, 1e6))
	assert.True(t, float64Equal(1e6, 1e6+1e-7))
	assert.True(t, float64Equal(1e10, 1e10))
	assert.False(t, float64Equal(1e10, 1e10+1))
	assert.False(t, float64Equal(0.01, 0.0100001))
	assert.False(t, float64Equal(0.0, 0.0000001))
	assert.False(t, float64Equal(0, 0.0000000000000000001))
}

func TestSetDefaultEpsilon(t *testing.T) {
	defer SetDefaultEpsilon(1e-6)

	t.Run("0.1 == 0.1", func(t *testing.T) {
		SetDefaultEpsilon(1e-6)
		p := NewParser(strings.NewReader("{foo} == 0.1"))
		expr, _ := p.Parse()
		r, err := Evaluate(expr, map[string]interface{}{"foo": 0.1})
		assert.True(t, r)
		assert.NoError(t, err)
	})

	t.Run("0.1 == 0.100000000001", func(t *testing.T) {
		SetDefaultEpsilon(1e-6)
		p := NewParser(strings.NewReader("{foo} == 0.1"))
		expr, _ := p.Parse()
		r, err := Evaluate(expr, map[string]interface{}{"foo": 0.100000000001})
		assert.True(t, r)
		assert.NoError(t, err)
	})

	t.Run("0.1 != 0.100001", func(t *testing.T) {
		SetDefaultEpsilon(1e-6)
		p := NewParser(strings.NewReader("{foo} == 0.1"))
		expr, _ := p.Parse()
		r, err := Evaluate(expr, map[string]interface{}{"foo": 0.100001})
		assert.False(t, r)
		assert.NoError(t, err)
	})

	t.Run("0.1 == 0.100001 if set epsilon to 1e-5", func(t *testing.T) {
		SetDefaultEpsilon(1e-5)
		p := NewParser(strings.NewReader("{foo} == 0.1"))
		expr, _ := p.Parse()
		r, err := Evaluate(expr, map[string]interface{}{"foo": 0.100001})
		assert.True(t, r)
		assert.NoError(t, err)
	})
}

func TestEpsilonSetAndUse(t *testing.T) {
	defer SetDefaultEpsilon(1e-6)

	SetDefaultEpsilon(1e-3)
	expr, _ := Parse(`{foo} == 0.1`)
	r, err := Evaluate(expr, map[string]interface{}{"foo": 0.10005})
	assert.NoError(t, err)
	assert.True(t, r, "should be equal within epsilon 1e-3")
}

func TestFloat64EqualEdgeCases(t *testing.T) {
	defer SetDefaultEpsilon(1e-6)

	SetDefaultEpsilon(1e-6)

	t.Run("equal values", func(t *testing.T) {
		assert.True(t, float64Equal(5.0, 5.0))
	})
	t.Run("beyond epsilon", func(t *testing.T) {
		assert.False(t, float64Equal(1.0, 2.0))
	})
	t.Run("near zero both", func(t *testing.T) {
		assert.False(t, float64Equal(0.0, 1e-20))
	})
	t.Run("one zero one small", func(t *testing.T) {
		assert.False(t, float64Equal(0.0, 1e-10))
	})
	t.Run("relative error large", func(t *testing.T) {
		SetDefaultEpsilon(1e-3)
		assert.False(t, float64Equal(1.0, 1.01))
	})
}
