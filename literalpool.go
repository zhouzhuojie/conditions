package conditions

// evalPool holds ephemeral literals for one Evaluate call. Pointers into its
// slices are only valid until that Evaluate returns.
type evalPool struct {
	strings []StringLiteral
	numbers []NumberLiteral
}

func (p *evalPool) reset() {
	p.strings = p.strings[:0]
	p.numbers = p.numbers[:0]
}

func (p *evalPool) stringLit(s string) *StringLiteral {
	p.strings = append(p.strings, StringLiteral{Val: s})
	return &p.strings[len(p.strings)-1]
}

func (p *evalPool) numberLit(f float64) *NumberLiteral {
	p.numbers = append(p.numbers, NumberLiteral{Val: f})
	return &p.numbers[len(p.numbers)-1]
}