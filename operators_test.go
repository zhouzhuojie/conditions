package conditions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApplyEQUnsupportedTypes(t *testing.T) {
	t.Run("slice vs slice", func(t *testing.T) {
		ssl1 := NewSliceStringLiteral([]string{"a"})
		ssl2 := NewSliceStringLiteral([]string{"b"})
		_, err := applyEQ(ssl1, ssl2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported equality comparison")
	})
	t.Run("number literal vs slice", func(t *testing.T) {
		nl := &NumberLiteral{Val: 1}
		ssl := NewSliceStringLiteral([]string{"a"})
		_, err := applyEQ(nl, ssl)
		assert.Error(t, err)
	})
}

func TestApplyEREGErrors(t *testing.T) {
	t.Run("non-string LHS", func(t *testing.T) {
		_, err := applyEREG(&NumberLiteral{Val: 1}, &StringLiteral{Val: ".*"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "string operands")
	})
	t.Run("non-string RHS", func(t *testing.T) {
		_, err := applyEREG(&StringLiteral{Val: "test"}, &NumberLiteral{Val: 1})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "string operands")
	})
}

func TestApplyINErrors(t *testing.T) {
	t.Run("boolean LHS unsupported", func(t *testing.T) {
		_, err := applyIN(&BooleanLiteral{Val: true}, NewSliceStringLiteral([]string{"a"}))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "IN/CONTAINS not supported")
	})
	t.Run("number IN non-slice", func(t *testing.T) {
		_, err := applyIN(&NumberLiteral{Val: 1}, &NumberLiteral{Val: 2})
		assert.Error(t, err)
	})
	t.Run("string IN non-slice", func(t *testing.T) {
		_, err := applyIN(&StringLiteral{Val: "a"}, &StringLiteral{Val: "b"})
		assert.Error(t, err)
	})
}

func TestGetStringError(t *testing.T) {
	_, err := getString(&NumberLiteral{Val: 1})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not a string")
}

func TestGetSliceNumberError(t *testing.T) {
	_, err := getSliceNumber(&NumberLiteral{Val: 1})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not a slice of float64")
}

func TestGetMapStringError(t *testing.T) {
	_, err := getMapString(&NumberLiteral{Val: 1})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not a slice of string")
}

func TestApplyOperatorUnsupported(t *testing.T) {
	_, err := applyOperator(Token(999), &BooleanLiteral{Val: true}, &BooleanLiteral{Val: false})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported operator")
}

func TestEvalBinaryShortCircuitErrors(t *testing.T) {
	t.Run("AND LHS error", func(t *testing.T) {
		expr, _ := Parse(`{foo} AND true`)
		_, err := Evaluate(expr, map[string]interface{}{"foo": "not_bool"})
		assert.Error(t, err)
	})
	t.Run("AND RHS error", func(t *testing.T) {
		expr, _ := Parse(`true AND {foo}`)
		_, err := Evaluate(expr, map[string]interface{}{"foo": "not_bool"})
		assert.Error(t, err)
	})
	t.Run("OR LHS error", func(t *testing.T) {
		expr, _ := Parse(`{foo} OR true`)
		_, err := Evaluate(expr, map[string]interface{}{"foo": "not_bool"})
		assert.Error(t, err)
	})
	t.Run("OR RHS error", func(t *testing.T) {
		expr, _ := Parse(`false OR {foo}`)
		_, err := Evaluate(expr, map[string]interface{}{"foo": "not_bool"})
		assert.Error(t, err)
	})
}

func TestApplyINStringNonSlice(t *testing.T) {
	_, err := applyIN(&StringLiteral{Val: "a"}, &StringLiteral{Val: "b"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not a slice of string")
}

func TestApplyINNumberNonSlice(t *testing.T) {
	_, err := applyIN(&NumberLiteral{Val: 1}, &NumberLiteral{Val: 2})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not a slice of float64")
}

func TestApplyBoolOpErrors(t *testing.T) {
	t.Run("non-boolean LHS", func(t *testing.T) {
		_, err := applyBoolOp(&NumberLiteral{Val: 1}, &BooleanLiteral{Val: true}, func(a, b bool) bool { return a && b })
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "boolean")
	})
	t.Run("non-boolean RHS", func(t *testing.T) {
		_, err := applyBoolOp(&BooleanLiteral{Val: true}, &NumberLiteral{Val: 1}, func(a, b bool) bool { return a && b })
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "boolean")
	})
}

func TestApplyEQAllBranches(t *testing.T) {
	t.Run("string == string", func(t *testing.T) {
		r, err := applyEQ(&StringLiteral{Val: "a"}, &StringLiteral{Val: "a"})
		assert.NoError(t, err)
		assert.True(t, r.Val)
	})
	t.Run("string != string", func(t *testing.T) {
		r, err := applyEQ(&StringLiteral{Val: "a"}, &StringLiteral{Val: "b"})
		assert.NoError(t, err)
		assert.False(t, r.Val)
	})
	t.Run("number == number", func(t *testing.T) {
		r, err := applyEQ(&NumberLiteral{Val: 1}, &NumberLiteral{Val: 1})
		assert.NoError(t, err)
		assert.True(t, r.Val)
	})
	t.Run("boolean == boolean", func(t *testing.T) {
		r, err := applyEQ(&BooleanLiteral{Val: true}, &BooleanLiteral{Val: true})
		assert.NoError(t, err)
		assert.True(t, r.Val)
	})
	t.Run("string vs number error", func(t *testing.T) {
		_, err := applyEQ(&StringLiteral{Val: "a"}, &NumberLiteral{Val: 1})
		assert.Error(t, err)
	})
	t.Run("number vs string error", func(t *testing.T) {
		_, err := applyEQ(&NumberLiteral{Val: 1}, &StringLiteral{Val: "a"})
		assert.Error(t, err)
	})
	t.Run("boolean vs number error", func(t *testing.T) {
		_, err := applyEQ(&BooleanLiteral{Val: true}, &NumberLiteral{Val: 1})
		assert.Error(t, err)
	})
	t.Run("unsupported type error", func(t *testing.T) {
		ssl := NewSliceStringLiteral([]string{"a"})
		_, err := applyEQ(ssl, ssl)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported equality")
	})
}

func TestNEQOperator(t *testing.T) {
	t.Run("string != string (different)", func(t *testing.T) {
		expr, _ := Parse(`{foo} != "bar"`)
		r, err := Evaluate(expr, map[string]interface{}{"foo": "baz"})
		assert.NoError(t, err)
		assert.True(t, r)
	})
	t.Run("string != string (same)", func(t *testing.T) {
		expr, _ := Parse(`{foo} != "bar"`)
		r, err := Evaluate(expr, map[string]interface{}{"foo": "bar"})
		assert.NoError(t, err)
		assert.False(t, r)
	})
	t.Run("number != number (different)", func(t *testing.T) {
		expr, _ := Parse(`{foo} != 42`)
		r, err := Evaluate(expr, map[string]interface{}{"foo": 43})
		assert.NoError(t, err)
		assert.True(t, r)
	})
	t.Run("number != number (same)", func(t *testing.T) {
		expr, _ := Parse(`{foo} != 42`)
		r, err := Evaluate(expr, map[string]interface{}{"foo": 42})
		assert.NoError(t, err)
		assert.False(t, r)
	})
	t.Run("boolean != boolean (different)", func(t *testing.T) {
		expr, _ := Parse(`{foo} != true`)
		r, err := Evaluate(expr, map[string]interface{}{"foo": false})
		assert.NoError(t, err)
		assert.True(t, r)
	})
}
