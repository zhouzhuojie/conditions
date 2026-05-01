package conditions

import (
	"fmt"
	"strings"
	"testing"
)

func BenchmarkParser(b *testing.B) {
	cond := "({foo}{dfs}{a} == true AND {bar} == true) AND false"
	args := map[string]interface{}{"foo.dfs.a": true, "bar": true, "something": 1.0}
	p := NewParser(strings.NewReader(cond))
	expr, _ := p.Parse()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Evaluate(expr, args)
	}
}

func BenchmarkParserWithShortCircuit(b *testing.B) {
	cond := "false AND {foo} > 100"
	args := map[string]interface{}{"foo": 42}
	p := NewParser(strings.NewReader(cond))
	expr, _ := p.Parse()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Evaluate(expr, args)
	}
}

func BenchmarkLongSliceString(b *testing.B) {
	items := []string{}
	for i := 0; i <= 10000; i++ {
		items = append(items, fmt.Sprintf(`"%v"`, i))
	}

	cond := fmt.Sprintf(`{foo} IN [%s]`, strings.Join(items, ","))
	args := map[string]interface{}{"foo": "123"}

	p := NewParser(strings.NewReader(cond))
	expr, _ := p.Parse()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Evaluate(expr, args)
	}
}

func BenchmarkLongSliceStringMiss(b *testing.B) {
	items := []string{}
	for i := 0; i <= 10000; i++ {
		items = append(items, fmt.Sprintf(`"%v"`, i))
	}

	cond := fmt.Sprintf(`{foo} IN [%s]`, strings.Join(items, ","))
	args := map[string]interface{}{"foo": "notfound"}

	p := NewParser(strings.NewReader(cond))
	expr, _ := p.Parse()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Evaluate(expr, args)
	}
}

func BenchmarkParseOnly(b *testing.B) {
	cond := "({foo}{dfs}{a} == true AND {bar} == true) AND false"

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		NewParser(strings.NewReader(cond)).Parse()
	}
}

func BenchmarkRegexMatch(b *testing.B) {
	cond := `{status} =~ /^5\d\d/`
	args := map[string]interface{}{"status": "500"}
	p := NewParser(strings.NewReader(cond))
	expr, _ := p.Parse()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Evaluate(expr, args)
	}
}

func BenchmarkSimpleComparison(b *testing.B) {
	cond := `{foo} == "hello"`
	args := map[string]interface{}{"foo": "hello"}
	p := NewParser(strings.NewReader(cond))
	expr, _ := p.Parse()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Evaluate(expr, args)
	}
}

func BenchmarkNumericComparison(b *testing.B) {
	cond := `{foo} > 100 AND {foo} < 200`
	args := map[string]interface{}{"foo": 150}
	p := NewParser(strings.NewReader(cond))
	expr, _ := p.Parse()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Evaluate(expr, args)
	}
}

func BenchmarkBooleanOperators(b *testing.B) {
	cond := `{a} AND {b} OR {c}`
	args := map[string]interface{}{"a": true, "b": false, "c": true}
	p := NewParser(strings.NewReader(cond))
	expr, _ := p.Parse()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Evaluate(expr, args)
	}
}

func BenchmarkShortSliceIN(b *testing.B) {
	cond := `{foo} in ["alpha", "beta", "gamma", "delta", "epsilon"]`
	args := map[string]interface{}{"foo": "gamma"}
	p := NewParser(strings.NewReader(cond))
	expr, _ := p.Parse()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Evaluate(expr, args)
	}
}

func BenchmarkNumberSliceIN(b *testing.B) {
	cond := `{foo} in [1,2,3,4,5,6,7,8,9,10]`
	args := map[string]interface{}{"foo": 7}
	p := NewParser(strings.NewReader(cond))
	expr, _ := p.Parse()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Evaluate(expr, args)
	}
}

func BenchmarkContains(b *testing.B) {
	cond := `{foo} contains "target"`
	args := map[string]interface{}{"foo": []string{"a", "b", "target", "c", "d"}}
	p := NewParser(strings.NewReader(cond))
	expr, _ := p.Parse()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Evaluate(expr, args)
	}
}

func BenchmarkVariables(b *testing.B) {
	cond := `{a} > 1 AND {b} == "test" OR {c} < 100 AND {d} in ["x","y","z"]`
	p := NewParser(strings.NewReader(cond))
	expr, _ := p.Parse()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Variables(expr)
	}
}
