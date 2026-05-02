package conditions

// Variables returns the deduplicated list of variable names referenced in the expression.
// It traverses the AST directly, collecting variable names into a map to avoid
// intermediate slice allocations.
func Variables(expression Expr) []string {
	if expression == nil {
		return nil
	}
	seen := make(map[string]struct{})
	collectVars(expression, seen)
	if len(seen) == 0 {
		return nil
	}
	result := make([]string, 0, len(seen))
	for v := range seen {
		result = append(result, v)
	}
	return result
}

// collectVars walks the AST and collects variable names into seen.
func collectVars(n Node, seen map[string]struct{}) {
	switch node := n.(type) {
	case *VarRef:
		seen[node.Val] = struct{}{}
	case *BinaryExpr:
		collectVars(node.LHS, seen)
		collectVars(node.RHS, seen)
	case *ParenExpr:
		collectVars(node.Expr, seen)
	}
	// All other node types have no variable references
}

// removeDuplicates is used by tests to verify deduplication logic.
func removeDuplicates(a []string) []string {
	if len(a) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(a))
	result := make([]string, 0, len(a))
	for _, val := range a {
		if _, ok := seen[val]; !ok {
			result = append(result, val)
			seen[val] = struct{}{}
		}
	}
	return result
}
