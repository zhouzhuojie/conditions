package conditions

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringMethods(t *testing.T) {
	t.Run("VarRef.String", func(t *testing.T) {
		v := &VarRef{Val: "foo"}
		assert.Equal(t, "foo", v.String())
	})
	t.Run("NumberLiteral.String", func(t *testing.T) {
		n := &NumberLiteral{Val: 3.14}
		assert.Equal(t, "3.140", n.String())
	})
	t.Run("StringLiteral.String", func(t *testing.T) {
		s := &StringLiteral{Val: "hello"}
		assert.Equal(t, `"hello"`, s.String())
	})
	t.Run("BooleanLiteral.String true", func(t *testing.T) {
		b := &BooleanLiteral{Val: true}
		assert.Equal(t, "true", b.String())
	})
	t.Run("BooleanLiteral.String false", func(t *testing.T) {
		b := &BooleanLiteral{Val: false}
		assert.Equal(t, "false", b.String())
	})
	t.Run("BinaryExpr.String", func(t *testing.T) {
		e := &BinaryExpr{
			Op:  GT,
			LHS: &VarRef{Val: "x"},
			RHS: &NumberLiteral{Val: 10},
		}
		assert.Equal(t, `x > 10.000`, e.String())
	})
	t.Run("ParenExpr.String", func(t *testing.T) {
		e := &ParenExpr{Expr: &BooleanLiteral{Val: true}}
		assert.Equal(t, "(true)", e.String())
	})
	t.Run("SliceStringLiteral.String", func(t *testing.T) {
		s := NewSliceStringLiteral([]string{"a", "b"})
		assert.Contains(t, s.String(), "a")
		assert.Contains(t, s.String(), "b")
	})
	t.Run("SliceNumberLiteral.String", func(t *testing.T) {
		s := &SliceNumberLiteral{Val: []float64{1, 2}}
		assert.Contains(t, s.String(), "1")
		assert.Contains(t, s.String(), "2")
	})
}

func TestArgsMethods(t *testing.T) {
	assert.Nil(t, (&NumberLiteral{}).Args())
	assert.Nil(t, (&StringLiteral{}).Args())
	assert.Nil(t, (&BooleanLiteral{}).Args())
	assert.Nil(t, (&SliceStringLiteral{}).Args())
	assert.Nil(t, (&SliceNumberLiteral{}).Args())
	assert.Equal(t, []string{"foo"}, (&VarRef{Val: "foo"}).Args())

	be := &BinaryExpr{
		Op:  GT,
		LHS: &VarRef{Val: "a"},
		RHS: &VarRef{Val: "b"},
	}
	assert.Equal(t, []string{"a", "b"}, be.Args())

	pe := &ParenExpr{Expr: &VarRef{Val: "x"}}
	assert.Equal(t, []string{"x"}, pe.Args())
}

func TestNewSliceStringLiteralPreallocatesMap(t *testing.T) {
	vals := make([]string, 1000)
	for i := range vals {
		vals[i] = fmt.Sprintf("item_%d", i)
	}
	ssl := NewSliceStringLiteral(vals)
	assert.Equal(t, 1000, len(ssl.m))
	assert.Equal(t, 1000, len(ssl.Val))
	_, ok := ssl.m["item_500"]
	assert.True(t, ok)
}
