package conditions

// evalPool holds ephemeral literals for one Evaluate call. A new pool is created
// per Evaluate (stack-local); pointers into its slices are only valid until
// that call returns.
type evalPool struct {
	strings []StringLiteral
	numbers []NumberLiteral
}

// numberFrom coerces an integer or float32 context value to a pooled NumberLiteral.
func (p *evalPool) numberFrom(v any) *NumberLiteral {
	switch n := v.(type) {
	case int:
		return p.numberLit(float64(n))
	case int8:
		return p.numberLit(float64(n))
	case int16:
		return p.numberLit(float64(n))
	case int32:
		return p.numberLit(float64(n))
	case int64:
		return p.numberLit(float64(n))
	case uint:
		return p.numberLit(float64(n))
	case uint8:
		return p.numberLit(float64(n))
	case uint16:
		return p.numberLit(float64(n))
	case uint32:
		return p.numberLit(float64(n))
	case uint64:
		return p.numberLit(float64(n))
	case float32:
		return p.numberLit(float64(n))
	default:
		panic("evalPool.numberFrom: unexpected type")
	}
}

func (p *evalPool) stringLit(s string) *StringLiteral {
	p.strings = append(p.strings, StringLiteral{Val: s})
	return &p.strings[len(p.strings)-1]
}

func (p *evalPool) numberLit(f float64) *NumberLiteral {
	p.numbers = append(p.numbers, NumberLiteral{Val: f})
	return &p.numbers[len(p.numbers)-1]
}