package conditions

import "fmt"

// Boolean singletons to avoid allocations in negate and short-circuit paths
var (
	trueExpr  = &BooleanLiteral{Val: true}
	falseExpr = &BooleanLiteral{Val: false}
)

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

	case *PathRef:
		return resolvePathRef(n, args)

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

// negate inverts a boolean result, propagating errors.
func negate(result *BooleanLiteral, err error) (*BooleanLiteral, error) {
	if err != nil {
		return nil, err
	}
	return boolExpr(!result.Val), nil
}
