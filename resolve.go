package conditions

import (
	"encoding/json"
	"fmt"
)

// resolveVar looks up a variable in args and converts it to the appropriate literal.
func resolveVar(name string, args map[string]interface{}) (Expr, error) {
	val, ok := args[name]
	if !ok {
		return falseExpr, fmt.Errorf("argument: %v not found", name)
	}
	if val == nil {
		return falseExpr, fmt.Errorf("unsupported argument nil type for %s", name)
	}
	return valueToExpr(val)
}

// resolvePathRef resolves a nested path expression by walking through
// nested maps and slices in args.
//
//	{user.name}    → args["user"]["name"]
//	{users[0]}     → args["users"][0]
//	{data[0].name} → args["data"][0]["name"]
func resolvePathRef(ref *PathRef, args map[string]interface{}) (Expr, error) {
	current, ok := args[ref.Root]
	if !ok {
		return falseExpr, fmt.Errorf("argument: %v not found", ref.Root)
	}

	for _, step := range ref.Steps {
		if current == nil {
			return falseExpr, fmt.Errorf("nil value encountered traversing %s", ref.Root)
		}

		if step.IsIndex {
			arr, ok := current.([]interface{})
			if !ok {
				return falseExpr, fmt.Errorf("cannot index non-array value traversing %s", ref.Root)
			}
			idx := step.Index
			if idx < 0 {
				idx = len(arr) + idx
			}
			if idx < 0 || idx >= len(arr) {
				return falseExpr, fmt.Errorf("index %d out of bounds traversing %s", step.Index, ref.Root)
			}
			current = arr[idx]
		} else {
			m, ok := current.(map[string]interface{})
			if !ok {
				return falseExpr, fmt.Errorf("cannot access key %q on non-map value traversing %s", step.Key, ref.Root)
			}
			v, ok := m[step.Key]
			if !ok {
				return falseExpr, fmt.Errorf("key %q not found traversing %s", step.Key, ref.Root)
			}
			current = v
		}
	}

	if current == nil {
		return falseExpr, fmt.Errorf("nil value at end of path %s", ref.Root)
	}
	return valueToExpr(current)
}

// valueToExpr converts a Go value to an AST literal expression.
func valueToExpr(val interface{}) (Expr, error) {
	if val == nil {
		return falseExpr, fmt.Errorf("unsupported nil type")
	}

	switch v := val.(type) {
	// Signed integers
	case int:
		return &NumberLiteral{Val: float64(v)}, nil
	case int8:
		return &NumberLiteral{Val: float64(v)}, nil
	case int16:
		return &NumberLiteral{Val: float64(v)}, nil
	case int32:
		return &NumberLiteral{Val: float64(v)}, nil
	case int64:
		return &NumberLiteral{Val: float64(v)}, nil

	// Unsigned integers
	case uint:
		return &NumberLiteral{Val: float64(v)}, nil
	case uint8:
		return &NumberLiteral{Val: float64(v)}, nil
	case uint16:
		return &NumberLiteral{Val: float64(v)}, nil
	case uint32:
		return &NumberLiteral{Val: float64(v)}, nil
	case uint64:
		return &NumberLiteral{Val: float64(v)}, nil

	// Floats
	case float32:
		return &NumberLiteral{Val: float64(v)}, nil
	case float64:
		return &NumberLiteral{Val: v}, nil

	// Scalars
	case string:
		return &StringLiteral{Val: v}, nil
	case bool:
		return &BooleanLiteral{Val: v}, nil
	case json.Number:
		f, err := v.Float64()
		if err != nil {
			return falseExpr, fmt.Errorf("unsupported JSON Number %v: %s", val, err)
		}
		return &NumberLiteral{Val: f}, nil

	// Typed slices — fast paths
	case []string:
		return NewSliceStringLiteral(v), nil
	case []float64:
		return NewSliceNumberLiteral(v), nil
	case []int:
		return NewSliceNumberLiteral(toFloat64Slice(v)), nil
	case []int32:
		return NewSliceNumberLiteral(toFloat64Slice(v)), nil
	case []int64:
		return NewSliceNumberLiteral(toFloat64Slice(v)), nil
	case []float32:
		return NewSliceNumberLiteral(toFloat64Slice(v)), nil
	case []json.Number:
		return convertJSONNumberSlice(v)

	// []interface{} from JSON unmarshaling
	case []interface{}:
		return convertInterfaceSlice(v)
	}

	return falseExpr, fmt.Errorf("unsupported type: %T", val)
}

// toFloat64Slice converts a numeric slice to []float64 using generics.
func toFloat64Slice[S ~[]E, E interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~float32 | ~float64
}](s S) []float64 {
	out := make([]float64, len(s))
	for i, v := range s {
		out[i] = float64(v)
	}
	return out
}

// convertJSONNumberSlice converts []json.Number to a SliceNumberLiteral.
func convertJSONNumberSlice(nums []json.Number) (Expr, error) {
	out := make([]float64, 0, len(nums))
	for _, n := range nums {
		f, err := n.Float64()
		if err != nil {
			return falseExpr, fmt.Errorf("unsupported JSON Number %v in slice: %s", n, err)
		}
		out = append(out, f)
	}
	return NewSliceNumberLiteral(out), nil
}

// convertInterfaceSlice handles []interface{} from JSON unmarshaling.
func convertInterfaceSlice(items []interface{}) (Expr, error) {
	if len(items) == 0 {
		return falseExpr, fmt.Errorf("empty slice not castable")
	}

	switch items[0].(type) {
	case string:
		return convertTypedSlice(items, func(v interface{}) (string, bool) {
			s, ok := v.(string)
			return s, ok
		})
	case float64:
		return convertTypedSlice(items, func(v interface{}) (float64, bool) {
			f, ok := v.(float64)
			return f, ok
		})
	case json.Number:
		nums := make([]json.Number, 0, len(items))
		for _, v := range items {
			jn, ok := v.(json.Number)
			if !ok {
				return falseExpr, fmt.Errorf("the items in the array are not all json.Number")
			}
			nums = append(nums, jn)
		}
		return convertJSONNumberSlice(nums)
	}

	return falseExpr, fmt.Errorf("unsupported slice element type: %T", items[0])
}

// convertTypedSlice is a generic helper that converts []interface{} to a typed slice,
// returning the appropriate literal expression.
func convertTypedSlice[T any](items []interface{}, cast func(interface{}) (T, bool)) (Expr, error) {
	out := make([]T, 0, len(items))
	for _, item := range items {
		v, ok := cast(item)
		if !ok {
			return falseExpr, fmt.Errorf("the items in the array are not all the same type")
		}
		out = append(out, v)
	}
	var zero T
	switch any(zero).(type) {
	case string:
		return NewSliceStringLiteral(any(out).([]string)), nil
	case float64:
		return NewSliceNumberLiteral(any(out).([]float64)), nil
	}
	return falseExpr, fmt.Errorf("unsupported slice element type: %T", zero)
}
