package conditions

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"sync"
)

var (
	// Boolean singletons to avoid allocations in negate and short-circuit paths
	trueExpr  = &BooleanLiteral{Val: true}
	falseExpr = &BooleanLiteral{Val: false}

	// Epsilon for float comparison. Set before first use via SetDefaultEpsilon.
	defaultEpsilon = 1e-6
)

// SetDefaultEpsilon sets the epsilon used for floating-point equality comparisons.
// Call this before any concurrent Evaluate calls if you need a non-default value.
func SetDefaultEpsilon(ep float64) {
	defaultEpsilon = ep
}

// regexCache caches compiled regex patterns to avoid recompilation on every call.
var regexCache = struct {
	sync.RWMutex
	m map[string]*regexp.Regexp
}{m: make(map[string]*regexp.Regexp)}

func getCompiledRegexp(pattern string) (*regexp.Regexp, error) {
	regexCache.RLock()
	re, ok := regexCache.m[pattern]
	regexCache.RUnlock()
	if ok {
		return re, nil
	}

	regexCache.Lock()
	defer regexCache.Unlock()
	// Double-check after acquiring write lock
	if re, ok := regexCache.m[pattern]; ok {
		return re, nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	regexCache.m[pattern] = re
	return re, nil
}

// Evaluate takes an expr and evaluates it using given args.
func Evaluate(expr Expr, args map[string]interface{}) (bool, error) {
	if expr == nil {
		return false, fmt.Errorf("provided expression is nil")
	}

	result, err := evaluate(expr, args)
	if err != nil {
		return false, err
	}
	if b, ok := result.(*BooleanLiteral); ok {
		return b.Val, nil
	}
	return false, fmt.Errorf("unexpected result of the root expression: %#v", result)
}

// evaluate recursively evaluates an expression tree.
func evaluate(expr Expr, args map[string]interface{}) (Expr, error) {
	switch n := expr.(type) {
	case *ParenExpr:
		return evaluate(n.Expr, args)

	case *BinaryExpr:
		return evalBinary(n, args)

	case *VarRef:
		return resolveVar(n.Val, args)

	default:
		// Literal — return as-is
		return expr, nil
	}
}

// evalBinary handles short-circuit logic for AND/OR, then delegates to applyOperator.
func evalBinary(n *BinaryExpr, args map[string]interface{}) (Expr, error) {
	if n.Op == AND {
		lv, err := evaluate(n.LHS, args)
		if err != nil {
			return falseExpr, err
		}
		lb, err := getBoolean(lv)
		if err != nil {
			return nil, err
		}
		if !lb {
			return falseExpr, nil
		}
		rv, err := evaluate(n.RHS, args)
		if err != nil {
			return falseExpr, err
		}
		rb, err := getBoolean(rv)
		if err != nil {
			return nil, err
		}
		return boolExpr(rb), nil
	}

	if n.Op == OR {
		lv, err := evaluate(n.LHS, args)
		if err != nil {
			return falseExpr, err
		}
		lb, err := getBoolean(lv)
		if err != nil {
			return nil, err
		}
		if lb {
			return trueExpr, nil
		}
		rv, err := evaluate(n.RHS, args)
		if err != nil {
			return falseExpr, err
		}
		rb, err := getBoolean(rv)
		if err != nil {
			return nil, err
		}
		return boolExpr(rb), nil
	}

	lv, err := evaluate(n.LHS, args)
	if err != nil {
		return falseExpr, err
	}
	rv, err := evaluate(n.RHS, args)
	if err != nil {
		return falseExpr, err
	}
	return applyOperator(n.Op, lv, rv)
}

// boolExpr returns the singleton boolean literal.
func boolExpr(v bool) *BooleanLiteral {
	if v {
		return trueExpr
	}
	return falseExpr
}

// resolveVar looks up a variable in args and converts it to the appropriate literal.
func resolveVar(name string, args map[string]interface{}) (Expr, error) {
	val, ok := args[name]
	if !ok {
		return falseExpr, fmt.Errorf("argument: %v not found", name)
	}
	if val == nil {
		return falseExpr, fmt.Errorf("unsupported argument nil type for %s", name)
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
		return &SliceNumberLiteral{Val: v}, nil
	case []int:
		return &SliceNumberLiteral{Val: toFloat64Slice(v)}, nil
	case []int32:
		return &SliceNumberLiteral{Val: toFloat64Slice(v)}, nil
	case []int64:
		return &SliceNumberLiteral{Val: toFloat64Slice(v)}, nil
	case []float32:
		return &SliceNumberLiteral{Val: toFloat64Slice(v)}, nil
	case []json.Number:
		return convertJSONNumberSlice(v)

	// []interface{} from JSON unmarshaling
	case []interface{}:
		return convertInterfaceSlice(v)
	}

	return falseExpr, fmt.Errorf("unsupported argument %s type: %T", name, val)
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
	return &SliceNumberLiteral{Val: out}, nil
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
		return &SliceNumberLiteral{Val: any(out).([]float64)}, nil
	}
	return falseExpr, fmt.Errorf("unsupported slice element type: %T", zero)
}

// --- Operator dispatch ---

func applyOperator(op Token, l, r Expr) (*BooleanLiteral, error) {
	switch op {
	case EQ:
		return applyEQ(l, r)
	case NEQ:
		return applyNEQ(l, r)
	case GT:
		return applyCmp(l, r, func(a, b float64) bool { return a > b })
	case GTE:
		return applyCmp(l, r, func(a, b float64) bool { return a > b || float64Equal(a, b) })
	case LT:
		return applyCmp(l, r, func(a, b float64) bool { return a < b })
	case LTE:
		return applyCmp(l, r, func(a, b float64) bool { return a < b || float64Equal(a, b) })
	case XOR:
		return applyBoolOp(l, r, func(a, b bool) bool { return a != b })
	case NAND:
		return applyBoolOp(l, r, func(a, b bool) bool { return !(a && b) })
	case IN:
		return applyIN(l, r)
	case NOTIN:
		return negate(applyIN(l, r))
	case EREG:
		return applyEREG(l, r)
	case NEREG:
		return negate(applyEREG(l, r))
	case CONTAINS:
		return applyIN(r, l)
	case NOTCONTAINS:
		return negate(applyIN(r, l))
	}
	return falseExpr, fmt.Errorf("unsupported operator: %s", op)
}

// negate inverts a boolean result, propagating errors.
func negate(result *BooleanLiteral, err error) (*BooleanLiteral, error) {
	if err != nil {
		return nil, err
	}
	return boolExpr(!result.Val), nil
}

// --- Boolean operators ---

func applyBoolOp(l, r Expr, fn func(a, b bool) bool) (*BooleanLiteral, error) {
	a, err := getBoolean(l)
	if err != nil {
		return nil, err
	}
	b, err := getBoolean(r)
	if err != nil {
		return nil, err
	}
	return boolExpr(fn(a, b)), nil
}

// --- Numeric comparison ---

func applyCmp(l, r Expr, cmp func(a, b float64) bool) (*BooleanLiteral, error) {
	a, err := getNumber(l)
	if err != nil {
		return nil, err
	}
	b, err := getNumber(r)
	if err != nil {
		return nil, err
	}
	return boolExpr(cmp(a, b)), nil
}

// --- Equality ---

func applyEQ(l, r Expr) (*BooleanLiteral, error) {
	// Try string comparison
	if as, err := getString(l); err == nil {
		bs, err := getString(r)
		if err != nil {
			return falseExpr, fmt.Errorf("cannot compare string with non-string")
		}
		return boolExpr(as == bs), nil
	}
	// Try number comparison
	if an, err := getNumber(l); err == nil {
		bn, err := getNumber(r)
		if err != nil {
			return falseExpr, fmt.Errorf("cannot compare number with non-number")
		}
		return boolExpr(float64Equal(an, bn)), nil
	}
	// Try boolean comparison
	if ab, err := getBoolean(l); err == nil {
		bb, err := getBoolean(r)
		if err != nil {
			return falseExpr, fmt.Errorf("cannot compare boolean with non-boolean")
		}
		return boolExpr(ab == bb), nil
	}
	return falseExpr, fmt.Errorf("unsupported equality comparison for types %T and %T", l, r)
}

func applyNEQ(l, r Expr) (*BooleanLiteral, error) {
	return negate(applyEQ(l, r))
}

// --- Regex ---

func applyEREG(l, r Expr) (*BooleanLiteral, error) {
	a, err := getString(l)
	if err != nil {
		return nil, err
	}
	b, err := getString(r)
	if err != nil {
		return nil, err
	}
	re, err := getCompiledRegexp(b)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern %q: %s", b, err)
	}
	return boolExpr(re.MatchString(a)), nil
}

// --- Membership (IN / CONTAINS) ---

func applyIN(l, r Expr) (*BooleanLiteral, error) {
	switch l.(type) {
	case *StringLiteral:
		a, err := getString(l)
		if err != nil {
			return nil, err
		}
		b, err := getMapString(r)
		if err != nil {
			return nil, err
		}
		_, found := b[a]
		return boolExpr(found), nil

	case *NumberLiteral:
		a, err := getNumber(l)
		if err != nil {
			return nil, err
		}
		b, err := getSliceNumber(r)
		if err != nil {
			return nil, err
		}
		for _, e := range b {
			if float64Equal(a, e) {
				return trueExpr, nil
			}
		}
		return falseExpr, nil
	}

	return nil, fmt.Errorf("IN/CONTAINS not supported for type %T", l)
}

// --- Type extraction helpers ---

func getBoolean(e Expr) (bool, error) {
	if b, ok := e.(*BooleanLiteral); ok {
		return b.Val, nil
	}
	return false, fmt.Errorf("literal is not a boolean: %v", e)
}

func getString(e Expr) (string, error) {
	if s, ok := e.(*StringLiteral); ok {
		return s.Val, nil
	}
	return "", fmt.Errorf("literal is not a string: %v", e)
}

func getNumber(e Expr) (float64, error) {
	if n, ok := e.(*NumberLiteral); ok {
		return n.Val, nil
	}
	return 0, fmt.Errorf("literal is not a number: %v", e)
}

func getSliceNumber(e Expr) ([]float64, error) {
	if s, ok := e.(*SliceNumberLiteral); ok {
		return s.Val, nil
	}
	return nil, fmt.Errorf("literal is not a slice of float64: %v", e)
}

func getMapString(e Expr) (map[string]struct{}, error) {
	if s, ok := e.(*SliceStringLiteral); ok {
		return s.m, nil
	}
	return nil, fmt.Errorf("literal is not a slice of string: %v", e)
}

// float64Equal compares two floats with epsilon tolerance.
func float64Equal(a, b float64) bool {
	if a == b {
		return true
	}
	diff := math.Abs(a - b)
	if diff > defaultEpsilon {
		return false
	}
	// Near-zero: use absolute tolerance scaled by smallest representable float
	if a == 0 || b == 0 {
		return diff < defaultEpsilon*math.SmallestNonzeroFloat32
	}
	// Relative error check for well-separated values
	return diff/math.Max(math.Abs(a), math.Abs(b)) < defaultEpsilon
}
