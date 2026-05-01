package conditions

import "fmt"

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

// applyEQ compares two literals for equality. It uses direct type switches
// instead of the error-cascade pattern (getString/getNumber/getBoolean) to
// avoid allocating error objects on the hot path.
func applyEQ(l, r Expr) (*BooleanLiteral, error) {
	switch lv := l.(type) {
	case *StringLiteral:
		if rv, ok := r.(*StringLiteral); ok {
			return boolExpr(lv.Val == rv.Val), nil
		}
		return falseExpr, fmt.Errorf("cannot compare string with non-string")
	case *NumberLiteral:
		if rv, ok := r.(*NumberLiteral); ok {
			return boolExpr(float64Equal(lv.Val, rv.Val)), nil
		}
		return falseExpr, fmt.Errorf("cannot compare number with non-number")
	case *BooleanLiteral:
		if rv, ok := r.(*BooleanLiteral); ok {
			return boolExpr(lv.Val == rv.Val), nil
		}
		return falseExpr, fmt.Errorf("cannot compare boolean with non-boolean")
	}
	return falseExpr, fmt.Errorf("unsupported equality comparison for types %T and %T", l, r)
}

func applyNEQ(l, r Expr) (*BooleanLiteral, error) {
	return negate(applyEQ(l, r))
}

// --- Regex ---

func applyEREG(l, r Expr) (*BooleanLiteral, error) {
	lv, lok := l.(*StringLiteral)
	rv, rok := r.(*StringLiteral)
	if !lok || !rok {
		return nil, fmt.Errorf("regex match requires string operands, got %T and %T", l, r)
	}
	re, err := getCompiledRegexp(rv.Val)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern %q: %s", rv.Val, err)
	}
	return boolExpr(re.MatchString(lv.Val)), nil
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
