package conditions

// Variables returns the deduplicated list of variable names referenced in the expression.
func Variables(expression Expr) []string {
	return removeDuplicates(expression.Args())
}

func removeDuplicates(a []string) []string {
	if len(a) == 0 {
		return nil
	}
	result := make([]string, 0, len(a))
	seen := make(map[string]struct{}, len(a))
	for _, val := range a {
		if _, ok := seen[val]; !ok {
			result = append(result, val)
			seen[val] = struct{}{}
		}
	}
	return result
}
