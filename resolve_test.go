package conditions

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveVarAllTypes(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		cond  string
	}{
		{"int", int(1), `{x} == 1`},
		{"int8", int8(1), `{x} == 1`},
		{"int16", int16(1), `{x} == 1`},
		{"int32", int32(1), `{x} == 1`},
		{"int64", int64(1), `{x} == 1`},
		{"uint", uint(1), `{x} == 1`},
		{"uint8", uint8(1), `{x} == 1`},
		{"uint16", uint16(1), `{x} == 1`},
		{"uint32", uint32(1), `{x} == 1`},
		{"uint64", uint64(1), `{x} == 1`},
		{"float32", float32(1.0), `{x} == 1`},
		{"float64", float64(1.0), `{x} == 1`},
		{"string", "hello", `{x} == "hello"`},
		{"bool", true, `{x} == true`},
		{"json.Number", json.Number("42"), `{x} == 42`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := Parse(tt.cond)
			assert.NoError(t, err)
			r, err := Evaluate(expr, map[string]interface{}{"x": tt.value})
			assert.NoError(t, err)
			assert.True(t, r)
		})
	}
}

func TestResolveVarSlices(t *testing.T) {
	t.Run("[]int32", func(t *testing.T) {
		expr, _ := Parse(`{x} contains 1`)
		r, err := Evaluate(expr, map[string]interface{}{"x": []int32{1, 2, 3}})
		assert.NoError(t, err)
		assert.True(t, r)
	})
	t.Run("[]int64", func(t *testing.T) {
		expr, _ := Parse(`{x} contains 1`)
		r, err := Evaluate(expr, map[string]interface{}{"x": []int64{1, 2, 3}})
		assert.NoError(t, err)
		assert.True(t, r)
	})
	t.Run("[]float32", func(t *testing.T) {
		expr, _ := Parse(`{x} contains 1`)
		r, err := Evaluate(expr, map[string]interface{}{"x": []float32{1, 2, 3}})
		assert.NoError(t, err)
		assert.True(t, r)
	})
	t.Run("[]json.Number", func(t *testing.T) {
		expr, _ := Parse(`{x} contains 1`)
		r, err := Evaluate(expr, map[string]interface{}{"x": []json.Number{"1", "2", "3"}})
		assert.NoError(t, err)
		assert.True(t, r)
	})
}

func TestResolveVarErrors(t *testing.T) {
	t.Run("nil value", func(t *testing.T) {
		expr, _ := Parse(`{x} == 1`)
		_, err := Evaluate(expr, map[string]interface{}{"x": nil})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nil")
	})
	t.Run("unsupported type", func(t *testing.T) {
		expr, _ := Parse(`{x} == 1`)
		_, err := Evaluate(expr, map[string]interface{}{"x": struct{}{}})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported")
	})
	t.Run("bad json.Number in slice", func(t *testing.T) {
		expr, _ := Parse(`{x} contains 1`)
		_, err := Evaluate(expr, map[string]interface{}{"x": []json.Number{"not_a_number"}})
		assert.Error(t, err)
	})
	t.Run("bad json.Number scalar", func(t *testing.T) {
		expr, _ := Parse(`{x} == 1`)
		_, err := Evaluate(expr, map[string]interface{}{"x": json.Number("not_a_number")})
		assert.Error(t, err)
	})
}

func TestResolveVarUintTypes(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
	}{
		{"uint", uint(42)},
		{"uint8", uint8(42)},
		{"uint16", uint16(42)},
		{"uint32", uint32(42)},
		{"uint64", uint64(42)},
		{"int8", int8(42)},
		{"int16", int16(42)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, _ := Parse(fmt.Sprintf(`{%s} == 42`, tt.name))
			r, err := Evaluate(expr, map[string]interface{}{tt.name: tt.value})
			assert.NoError(t, err)
			assert.True(t, r)
		})
	}
}

func TestConvertInterfaceSliceErrors(t *testing.T) {
	t.Run("mixed types in interface slice", func(t *testing.T) {
		items := []interface{}{"hello", 42}
		_, err := convertInterfaceSlice(items)
		assert.Error(t, err)
	})
	t.Run("unsupported element type", func(t *testing.T) {
		items := []interface{}{true, false}
		_, err := convertInterfaceSlice(items)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported slice element type")
	})
}

func TestConvertInterfaceSliceEmpty(t *testing.T) {
	items := []interface{}{}
	_, err := convertInterfaceSlice(items)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty slice")
}

func TestConvertInterfaceSliceUnsupportedElement(t *testing.T) {
	items := []interface{}{struct{}{}}
	_, err := convertInterfaceSlice(items)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported slice element type")
}

func TestConvertInterfaceSliceJSONNumberError(t *testing.T) {
	items := []interface{}{json.Number("1"), "not_a_number"}
	_, err := convertInterfaceSlice(items)
	assert.Error(t, err)
}

func TestConvertTypedSliceUnsupportedType(t *testing.T) {
	items := []interface{}{true, false}
	_, err := convertTypedSlice(items, func(v interface{}) (bool, bool) {
		b, ok := v.(bool)
		return b, ok
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported slice element type")
}

func TestToFloat64SliceAllTypes(t *testing.T) {
	t.Run("[]int", func(t *testing.T) {
		result := toFloat64Slice([]int{1, 2, 3})
		assert.Equal(t, []float64{1, 2, 3}, result)
	})
	t.Run("[]int8", func(t *testing.T) {
		result := toFloat64Slice([]int8{1, 2, 3})
		assert.Equal(t, []float64{1, 2, 3}, result)
	})
	t.Run("[]int16", func(t *testing.T) {
		result := toFloat64Slice([]int16{1, 2, 3})
		assert.Equal(t, []float64{1, 2, 3}, result)
	})
	t.Run("[]uint", func(t *testing.T) {
		result := toFloat64Slice([]uint{1, 2, 3})
		assert.Equal(t, []float64{1, 2, 3}, result)
	})
	t.Run("[]uint8", func(t *testing.T) {
		result := toFloat64Slice([]uint8{1, 2, 3})
		assert.Equal(t, []float64{1, 2, 3}, result)
	})
	t.Run("[]uint16", func(t *testing.T) {
		result := toFloat64Slice([]uint16{1, 2, 3})
		assert.Equal(t, []float64{1, 2, 3}, result)
	})
	t.Run("[]uint32", func(t *testing.T) {
		result := toFloat64Slice([]uint32{1, 2, 3})
		assert.Equal(t, []float64{1, 2, 3}, result)
	})
	t.Run("[]uint64", func(t *testing.T) {
		result := toFloat64Slice([]uint64{1, 2, 3})
		assert.Equal(t, []float64{1, 2, 3}, result)
	})
}
