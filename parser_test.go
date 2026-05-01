package conditions

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// --- Invalid expressions ---

var invalidTestData = []string{
	"",
	"A",
	"{var0} == DEMO",
	"{var0} == CA",
	"{var0} == PA",
	"{var0} == 'DEMO'",
	"!{var0}",
	"{var0} <> `DEMO`",
	"{foo} in []",
	"{foo} in [foobar]",
	"{foo} in [foobar, baz]",
	"{foo} in [\"foobar\", baz]",
	"{foo} in {foobar",
	"{foo} in [foobar",
	"{foo} in ['foobar']",
	"{foo} in ['foobar'",
	"{foo} in [1, 2, \"3\"]",
	"{foo} in [\"3\", 2, 1]",
	"{foo} in [\"3\", 2, 1",
	"{foo} not in [foobar]",
}

func TestInvalid(t *testing.T) {
	for _, cond := range invalidTestData {
		t.Run(cond, func(t *testing.T) {
			p := NewParser(strings.NewReader(cond))
			expr, err := p.Parse()
			assert.Error(t, err, "Should receive error for: %s", cond)
			assert.Nil(t, expr, "Expression should be nil for: %s", cond)
		})
	}
}

// --- Valid expressions ---

var validTestData = []struct {
	cond   string
	args   map[string]interface{}
	result bool
	isErr  bool
}{
	{"true", nil, true, false},
	{"false", nil, false, false},
	{"false OR true OR false OR false OR true", nil, true, false},
	{"((false OR true) AND false) OR (false OR true)", nil, true, false},
	{"{var0}", map[string]interface{}{"var0": true}, true, false},
	{"{var0}", map[string]interface{}{"var0": false}, false, false},
	{"{var0} > true", nil, false, true},
	{"{var0} > true", map[string]interface{}{"var0": 43}, false, true},
	{"{var0} > true", map[string]interface{}{"var0": false}, false, true},
	{"{var0} and {var1}", map[string]interface{}{"var0": true, "var1": true}, true, false},
	{"{var0} AND {var1}", map[string]interface{}{"var0": true, "var1": false}, false, false},
	{"{var0} AND {var1}", map[string]interface{}{"var0": false, "var1": true}, false, false},
	{"{var0} AND {var1}", map[string]interface{}{"var0": false, "var1": false}, false, false},
	{"{var0} AND false", map[string]interface{}{"var0": true}, false, false},
	{"56.43", nil, false, true},
	{"{var5}", nil, false, true},
	{"{var0} > -100 AND {var0} < -50", map[string]interface{}{"var0": -75.4}, true, false},
	{"{var5-type-2}", nil, false, true},
	{"{var5-type-2} == 1", map[string]interface{}{"var5-type-2": 1}, true, false},
	{"{var0}", map[string]interface{}{"var0": true}, true, false},
	{"{var0}", map[string]interface{}{"var0": false}, false, false},
	{"\"OFF\"", nil, false, true},
	{"\"ON\"", nil, false, true},
	{"{var0} == \"OFF\"", map[string]interface{}{"var0": "OFF"}, true, false},

	// AND
	{"{var0} > 10 AND {var1} == \"OFF\"", map[string]interface{}{"var0": 14, "var1": "OFF"}, true, false},
	{"({var0} > 10) AND ({var1} == \"OFF\")", map[string]interface{}{"var0": 14, "var1": "OFF"}, true, false},
	{"({var0} > 10) AND ({var1} == \"OFF\") OR true", map[string]interface{}{"var0": 1, "var1": "ON"}, true, false},
	{"{foo}{dfs} == true and {bar} == true", map[string]interface{}{"foo.dfs": true, "bar": true}, true, false},
	{"{foo}{dfs}{a} == true and {bar} == true", map[string]interface{}{"foo.dfs.a": true, "bar": true}, true, false},
	{"{@foo}{a} == true and {bar} == true", map[string]interface{}{"@foo.a": true, "bar": true}, true, false},
	{"{foo}{unknow} == true and {bar} == true", map[string]interface{}{"foo.dfs": true, "bar": true}, false, true},
	{"{foo} == 123", map[string]interface{}{"foo": json.Number("123"), "bar": true}, true, false},

	// OR (short-circuit: true OR ... evaluates to true without evaluating RHS)
	{"{foo} == true OR {foo} > 1", map[string]interface{}{"foo": true}, true, false},
	{"{foo} == true OR {foo} == false", map[string]interface{}{"foo": true}, true, false},
	{"{foo} > 100 OR {foo} < 99 ", map[string]interface{}{"foo": 100}, false, false},
	{"{foo}{dfs} == true or {bar} == true", map[string]interface{}{"foo.dfs": true, "bar": true}, true, false},

	//XOR
	{"false XOR false", nil, false, false},
	{"false xor true", nil, true, false},
	{"true XOR false", nil, true, false},
	{"true xor true", nil, false, false},

	//NAND
	{"false NAND false", nil, true, false},
	{"false nand true", nil, true, false},
	{"true nand false", nil, true, false},
	{"true NAND true", nil, false, false},

	// IN
	{"{foo} in {foobar}", map[string]interface{}{"foo": "findme", "foobar": []string{"notme", "may", "findme", "lol"}}, true, false},
	{"{foo} in [123]", map[string]interface{}{"foo": json.Number("123"), "baz": true}, true, false},
	{"{foo} in [123]", map[string]interface{}{"foo": json.Number("124"), "baz": true}, false, false},

	// NOT IN
	{"{foo} not in {foobar}", map[string]interface{}{"foo": "dontfindme", "foobar": []string{"notme", "may", "findme", "lol"}}, true, false},

	// IN with array of string
	{`{foo} in ["bonjour", "le monde", "oui"]`, map[string]interface{}{"foo": "le monde"}, true, false},
	{`{foo} in ["bonjour", "le monde", "oui"]`, map[string]interface{}{"foo": "world"}, false, false},

	// NOT IN with array of string
	{`{foo} not in ["bonjour", "le monde", "oui"]`, map[string]interface{}{"foo": "le monde"}, false, false},
	{`{foo} not in ["bonjour", "le monde", "oui"]`, map[string]interface{}{"foo": "world"}, true, false},

	// IN with array of numbers
	{`{foo} in [2,3,4]`, map[string]interface{}{"foo": 4}, true, false},
	{`{foo} in [2,3,4] AND {foo} == 4`, map[string]interface{}{"foo": 4}, true, false},
	{`{foo} in [2,3,4] AND {foo} == 3`, map[string]interface{}{"foo": 4}, false, false},
	{`{foo} in [2,3,4]`, map[string]interface{}{"foo": 5}, false, false},

	//{NOT}IN with array of numbers
	{`{foo} not in [2,3,4]`, map[string]interface{}{"foo": 4}, false, false},
	{`{foo} not in [2,3,4]`, map[string]interface{}{"foo": 5}, true, false},

	//{CONTAINS}
	{`{foo} contains "2"`, map[string]interface{}{"foo": []string{"1", "2"}}, true, false},
	{`{foo} contains "2"`, map[string]interface{}{"foo": []string{}}, false, false},
	{`{foo} contains 2`, map[string]interface{}{"foo": []string{"1", "2"}}, false, true},
	{`{foo} contains "2" and {foo} contains "1"`, map[string]interface{}{"foo": []string{"1", "2"}}, true, false},
	{`{foo} contains "2" and {foo} contains "0"`, map[string]interface{}{"foo": []string{"1", "2"}}, false, false},
	{`{foo} contains "2" or {foo} contains "0"`, map[string]interface{}{"foo": []string{"1", "2"}}, true, false},
	{`{foo} contains 2 and {foo} contains 1`, map[string]interface{}{"foo": []int{1, 2}}, true, false},
	{`{foo} contains 2 and {foo} contains 1`, map[string]interface{}{"foo": []int{1, 2}}, true, false},
	{`{foo} contains "2" and {foo} contains 1`, map[string]interface{}{"foo": []int{1, 2}}, false, true},
	{`{foo} contains {bar}`, map[string]interface{}{"foo": []string{"1", "2"}, "bar": "1"}, true, false},
	{`{foo} contains {bar}`, map[string]interface{}{"foo": []int{1, 2}, "bar": int32(1)}, true, false},
	{`{foo} contains {bar}`, map[string]interface{}{"foo": []int{1, 2, 3}, "bar": float32(1.0 + 2.0)}, true, false},
	{`{foo} contains {bar}`, map[string]interface{}{"foo": []float64{0.29}, "bar": float32(29.0 / 100)}, true, false},
	{`{foo} contains 2`, map[string]interface{}{"foo": []json.Number{"2"}}, true, false},
	{`{foo} contains 2`, map[string]interface{}{"foo": []json.Number{"2", "3"}}, true, false},
	{`{foo} contains 2`, map[string]interface{}{"foo": []json.Number{"3"}}, false, false},
	{`{foo} contains 2`, map[string]interface{}{"foo": []interface{}{json.Number("2")}}, true, false},
	{`{foo} contains 2`, map[string]interface{}{"foo": []interface{}{json.Number("3")}}, false, false},

	//{NOT}CONTAINS
	{`{foo} not contains "2"`, map[string]interface{}{"foo": []string{"1", "2"}}, false, false},
	{`{foo} not contains "0"`, map[string]interface{}{"foo": []string{"1", "2"}}, true, false},
	{`{foo} not contains 0`, map[string]interface{}{"foo": []string{"1", "2"}}, false, true},
	{`{foo} not contains 0`, map[string]interface{}{"bar": []string{"1", "2"}}, false, true},

	//{=~
	{`{status} =~ /^5\d\d/`, map[string]interface{}{"status": "500"}, true, false},
	{`{status} =~ /^4\d\d/`, map[string]interface{}{"status": "500"}, false, false},
	{`{status} =~ /foo/`, map[string]interface{}{"status": "foobar"}, true, false},
	{`{status} =~ "foo"`, map[string]interface{}{"status": "foobar"}, true, false},
	{`{status} =~ "foo"`, map[string]interface{}{"status": "bar"}, false, false},

	//{!~
	{"{status} !~ /^5\\d\\d/", map[string]interface{}{"status": "500"}, false, false},
	{"{status} !~ /^4\\d\\d/", map[string]interface{}{"status": "500"}, true, false},
}

func TestValid(t *testing.T) {
	for i, td := range validTestData {
		t.Run(fmt.Sprintf("%d_%s", i, td.cond), func(t *testing.T) {
			p := NewParser(strings.NewReader(td.cond))
			expr, err := p.Parse()
			if err != nil {
				t.Fatalf("Unexpected error parsing expression %q: %s", td.cond, err)
			}

			r, err := Evaluate(expr, td.args)
			if err != nil {
				if td.isErr {
					return // expected error
				}
				t.Fatalf("Unexpected error evaluating %q: %s", expr, err)
			} else if td.isErr {
				t.Fatalf("Expected error but got none for: %s", expr)
			}
			assert.Equal(t, td.result, r, "Expression: %s, Args: %#v", td.cond, td.args)
		})
	}
}

// --- Variable extraction ---

func TestExpressionsVariableNames(t *testing.T) {
	cond := "{@foo}{a} == true and {bar} == true or {var9} > 10"
	p := NewParser(strings.NewReader(cond))
	expr, err := p.Parse()
	assert.Nil(t, err)

	args := Variables(expr)
	assert.Contains(t, args, "@foo.a")
	assert.Contains(t, args, "bar")
	assert.Contains(t, args, "var9")
	assert.NotContains(t, args, "foo")
	assert.NotContains(t, args, "@foo")
}

func TestVariablesDeduplication(t *testing.T) {
	cond := "{foo} > 1 AND {foo} < 10 AND {bar} == true"
	p := NewParser(strings.NewReader(cond))
	expr, err := p.Parse()
	assert.NoError(t, err)

	args := Variables(expr)
	assert.Equal(t, 2, len(args))
	assert.Contains(t, args, "foo")
	assert.Contains(t, args, "bar")
}

// --- Float comparison ---

func TestFloat64Equal(t *testing.T) {
	defer SetDefaultEpsilon(1e-6)

	SetDefaultEpsilon(1e-6)
	assert.True(t, float64Equal(0.01, 0.01))
	assert.True(t, float64Equal(0.01, 0.01000001))
	assert.True(t, float64Equal(1e6, 1e6))
	assert.True(t, float64Equal(1e6, 1e6+1e-7))
	assert.True(t, float64Equal(1e10, 1e10))
	assert.False(t, float64Equal(1e10, 1e10+1))
	assert.False(t, float64Equal(0.01, 0.0100001))
	assert.False(t, float64Equal(0.0, 0.0000001))
	assert.False(t, float64Equal(0, 0.0000000000000000001))
}

func TestSetDefaultEpsilon(t *testing.T) {
	defer SetDefaultEpsilon(1e-6)

	t.Run("0.1 == 0.1", func(t *testing.T) {
		SetDefaultEpsilon(1e-6)
		p := NewParser(strings.NewReader("{foo} == 0.1"))
		expr, _ := p.Parse()
		r, err := Evaluate(expr, map[string]interface{}{"foo": 0.1})
		assert.True(t, r)
		assert.NoError(t, err)
	})

	t.Run("0.1 == 0.100000000001", func(t *testing.T) {
		SetDefaultEpsilon(1e-6)
		p := NewParser(strings.NewReader("{foo} == 0.1"))
		expr, _ := p.Parse()
		r, err := Evaluate(expr, map[string]interface{}{"foo": 0.100000000001})
		assert.True(t, r)
		assert.NoError(t, err)
	})

	t.Run("0.1 != 0.100001", func(t *testing.T) {
		SetDefaultEpsilon(1e-6)
		p := NewParser(strings.NewReader("{foo} == 0.1"))
		expr, _ := p.Parse()
		r, err := Evaluate(expr, map[string]interface{}{"foo": 0.100001})
		assert.False(t, r)
		assert.NoError(t, err)
	})

	t.Run("0.1 == 0.100001 if set epsilon to 1e-5", func(t *testing.T) {
		SetDefaultEpsilon(1e-5)
		p := NewParser(strings.NewReader("{foo} == 0.1"))
		expr, _ := p.Parse()
		r, err := Evaluate(expr, map[string]interface{}{"foo": 0.100001})
		assert.True(t, r)
		assert.NoError(t, err)
	})
}

func TestEpsilonSetAndUse(t *testing.T) {
	defer SetDefaultEpsilon(1e-6)

	SetDefaultEpsilon(1e-3)
	expr, _ := Parse(`{foo} == 0.1`)
	r, err := Evaluate(expr, map[string]interface{}{"foo": 0.10005})
	assert.NoError(t, err)
	assert.True(t, r, "should be equal within epsilon 1e-3")
}

// --- Parse convenience ---

func TestReadmeExample(t *testing.T) {
	s := `({foo} > 0.45) AND ({bar} == "ON" OR {baz} IN ["ACTIVE", "CLEAR"])`

	p := NewParser(strings.NewReader(s))
	expr, err := p.Parse()
	assert.NoError(t, err)

	data := map[string]interface{}{"foo": 0.62, "bar": "ON", "baz": "ACTIVE"}
	r, err := Evaluate(expr, data)
	assert.NoError(t, err)
	assert.True(t, r)
}

func TestJSON(t *testing.T) {
	var tests = []struct {
		cond    string
		jsonStr string
		result  bool
		isErr   bool
	}{
		{`{foo} == 123`, `{"foo": 123}`, true, false},
		{`{foo} in [123]`, `{"foo": 123}`, true, false},
		{`{foo} in [124]`, `{"foo": 123}`, false, false},
		{`{foo} in [123]`, `{"foo": 123, "bar": "baz"}`, true, false},
		{`{foo} in [124]`, `{"foo": 123, "bar": "baz"}`, false, false},

		{`{foo} == "123"`, `{"foo": 123}`, false, true},
		{`{foo} == "123"`, `{"foo": "123"}`, true, false},
		{`{foo} not in ["123"]`, `{"foo": "123"}`, false, false},

		{`{foo} contains "123"`, `{"foo": ["123"]}`, true, false},
		{`{foo} contains 123`, `{"foo": [123]}`, true, false},
		{`{foo} contains 123`, `{"foo": ["123"]}`, false, true},
		{`{foo} not contains 123`, `{"foo": [124]}`, true, false},
		{`{foo} not contains "123"`, `{"foo": ["124"]}`, true, false},
		{`{foo} not contains "123"`, `{"foo": null}`, false, true},
		{`{foo} not contains "123"`, `{}`, false, true},
	}

	for _, test := range tests {
		t.Run(test.cond+"_"+test.jsonStr, func(t *testing.T) {
			p := NewParser(strings.NewReader(test.cond))
			expr, _ := p.Parse()
			data := make(map[string]interface{})
			json.Unmarshal([]byte(test.jsonStr), &data)
			r, err := Evaluate(expr, data)
			assert.Equal(t, test.result, r, "%s with %s", test.cond, test.jsonStr)
			if test.isErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseConvenience(t *testing.T) {
	expr, err := Parse(`{foo} > 10 AND {bar} == "hello"`)
	assert.NoError(t, err)
	assert.NotNil(t, expr)

	r, err := Evaluate(expr, map[string]interface{}{"foo": 15, "bar": "hello"})
	assert.NoError(t, err)
	assert.True(t, r)
}

func TestParseConvenienceInvalid(t *testing.T) {
	_, err := Parse(`{foo} == UNQUOTED`)
	assert.Error(t, err)
}

// --- AST utilities ---

func TestQuote(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", `"hello"`},
		{"hello\nworld", `"hello\nworld"`},
		{`hello "world"`, `"hello \"world\""`},
		{`back\slash`, `"back\\slash"`},
		{"", `""`},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, Quote(tt.input))
		})
	}
}

func TestQuoteIdent(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"foo", "foo"},
		{"foo.bar", "foo.bar"},
		{"foo_bar", "foo_bar"},
		{"foo-bar", `"foo-bar"`},
		{"foo bar", `"foo bar"`},
		{"123", `"123"`},
		{"", `""`},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, QuoteIdent(tt.input))
		})
	}
}

func TestInspectDataType(t *testing.T) {
	assert.Equal(t, Number, InspectDataType(float64(1.0)))
	assert.Equal(t, Boolean, InspectDataType(true))
	assert.Equal(t, String, InspectDataType("hello"))
	assert.Equal(t, Unknown, InspectDataType(42))
	assert.Equal(t, Unknown, InspectDataType(nil))
}

func TestWalkAndWalkFunc(t *testing.T) {
	expr, err := Parse(`{foo} > 1 AND ({bar} == "test" OR {baz} < 100)`)
	assert.NoError(t, err)

	var nodes []string
	WalkFunc(expr, func(n Node) {
		nodes = append(nodes, n.String())
	})

	assert.Contains(t, nodes, "foo")
	assert.Contains(t, nodes, "1.000")
	assert.Contains(t, nodes, "bar")
	assert.Contains(t, nodes, "\"test\"")
	assert.Contains(t, nodes, "baz")
	assert.Contains(t, nodes, "100.000")
}

func TestWalkVisitor(t *testing.T) {
	expr, err := Parse(`{foo} > 1 AND {bar} == true`)
	assert.NoError(t, err)

	varCount := 0
	WalkFunc(expr, func(n Node) {
		if _, ok := n.(*VarRef); ok {
			varCount++
		}
	})

	assert.Equal(t, 2, varCount)
}

func TestStringMethods(t *testing.T) {
	t.Run("VarRef.String", func(t *testing.T) {
		v := &VarRef{Val: "foo"}
		assert.Equal(t, "foo", v.String())
	})
	t.Run("NumberLiteral.String", func(t *testing.T) {
		n := &NumberLiteral{Val: 3.14}
		assert.Equal(t, "3.140", n.String())
	})
	t.Run("StringLiteral.String", func(t *testing.T) {
		s := &StringLiteral{Val: "hello"}
		assert.Equal(t, `"hello"`, s.String())
	})
	t.Run("BooleanLiteral.String true", func(t *testing.T) {
		b := &BooleanLiteral{Val: true}
		assert.Equal(t, "true", b.String())
	})
	t.Run("BooleanLiteral.String false", func(t *testing.T) {
		b := &BooleanLiteral{Val: false}
		assert.Equal(t, "false", b.String())
	})
	t.Run("BinaryExpr.String", func(t *testing.T) {
		e := &BinaryExpr{
			Op:  GT,
			LHS: &VarRef{Val: "x"},
			RHS: &NumberLiteral{Val: 10},
		}
		assert.Equal(t, `x > 10.000`, e.String())
	})
	t.Run("ParenExpr.String", func(t *testing.T) {
		e := &ParenExpr{Expr: &BooleanLiteral{Val: true}}
		assert.Equal(t, "(true)", e.String())
	})
	t.Run("SliceStringLiteral.String", func(t *testing.T) {
		s := NewSliceStringLiteral([]string{"a", "b"})
		assert.Contains(t, s.String(), "a")
		assert.Contains(t, s.String(), "b")
	})
	t.Run("SliceNumberLiteral.String", func(t *testing.T) {
		s := &SliceNumberLiteral{Val: []float64{1, 2}}
		assert.Contains(t, s.String(), "1")
		assert.Contains(t, s.String(), "2")
	})
}

func TestArgsMethods(t *testing.T) {
	assert.Nil(t, (&NumberLiteral{}).Args())
	assert.Nil(t, (&StringLiteral{}).Args())
	assert.Nil(t, (&BooleanLiteral{}).Args())
	assert.Nil(t, (&SliceStringLiteral{}).Args())
	assert.Nil(t, (&SliceNumberLiteral{}).Args())
	assert.Equal(t, []string{"foo"}, (&VarRef{Val: "foo"}).Args())

	be := &BinaryExpr{
		Op:  GT,
		LHS: &VarRef{Val: "a"},
		RHS: &VarRef{Val: "b"},
	}
	assert.Equal(t, []string{"a", "b"}, be.Args())

	pe := &ParenExpr{Expr: &VarRef{Val: "x"}}
	assert.Equal(t, []string{"x"}, pe.Args())
}

// --- Error handling ---

func TestNilExprEvaluation(t *testing.T) {
	_, err := Evaluate(nil, map[string]interface{}{"foo": 1})
	assert.Error(t, err)
}

func TestNilArgsMap(t *testing.T) {
	t.Run("boolean literal with nil args", func(t *testing.T) {
		expr, _ := Parse("true")
		r, err := Evaluate(expr, nil)
		assert.NoError(t, err)
		assert.True(t, r)
	})
	t.Run("var ref with nil args", func(t *testing.T) {
		expr, _ := Parse("{foo}")
		_, err := Evaluate(expr, nil)
		assert.Error(t, err)
	})
}

func TestRegexError(t *testing.T) {
	t.Run("invalid regex pattern", func(t *testing.T) {
		expr, err := Parse(`{status} =~ /[/`)
		if err == nil {
			_, err = Evaluate(expr, map[string]interface{}{"status": "test"})
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid regex")
		}
	})
}

func TestUnsupportedOperator(t *testing.T) {
	_, err := applyOperator(ILLEGAL, &BooleanLiteral{Val: true}, &BooleanLiteral{Val: false})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported operator")
}

func TestDeeplyNestedParentheses(t *testing.T) {
	depth := 50
	expr := "{foo}"
	for i := 0; i < depth; i++ {
		expr = "(" + expr + ")"
	}
	p := NewParser(strings.NewReader(expr))
	ast, err := p.Parse()
	assert.NoError(t, err)

	r, err := Evaluate(ast, map[string]interface{}{"foo": true})
	assert.NoError(t, err)
	assert.True(t, r)
}

func TestStringComparisons(t *testing.T) {
	tests := []struct {
		cond   string
		args   map[string]interface{}
		result bool
	}{
		{`{a} == {b}`, map[string]interface{}{"a": "hello", "b": "hello"}, true},
		{`{a} == {b}`, map[string]interface{}{"a": "hello", "b": "world"}, false},
		{`{a} != {b}`, map[string]interface{}{"a": "hello", "b": "world"}, true},
		{`{a} != {b}`, map[string]interface{}{"a": "hello", "b": "hello"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.cond, func(t *testing.T) {
			p := NewParser(strings.NewReader(tt.cond))
			expr, err := p.Parse()
			assert.NoError(t, err)
			r, err := Evaluate(expr, tt.args)
			assert.NoError(t, err)
			assert.Equal(t, tt.result, r)
		})
	}
}

func TestNumberComparisons(t *testing.T) {
	tests := []struct {
		cond   string
		args   map[string]interface{}
		result bool
	}{
		{`{a} > {b}`, map[string]interface{}{"a": 10, "b": 5}, true},
		{`{a} > {b}`, map[string]interface{}{"a": 5, "b": 10}, false},
		{`{a} >= {b}`, map[string]interface{}{"a": 10, "b": 10}, true},
		{`{a} >= {b}`, map[string]interface{}{"a": 10, "b": 5}, true},
		{`{a} < {b}`, map[string]interface{}{"a": 5, "b": 10}, true},
		{`{a} < {b}`, map[string]interface{}{"a": 10, "b": 5}, false},
		{`{a} <= {b}`, map[string]interface{}{"a": 10, "b": 10}, true},
		{`{a} <= {b}`, map[string]interface{}{"a": 5, "b": 10}, true},
	}
	for _, tt := range tests {
		t.Run(tt.cond, func(t *testing.T) {
			p := NewParser(strings.NewReader(tt.cond))
			expr, err := p.Parse()
			assert.NoError(t, err)
			r, err := Evaluate(expr, tt.args)
			assert.NoError(t, err)
			assert.Equal(t, tt.result, r)
		})
	}
}

// --- Short-circuit ---

func TestShortCircuitAND(t *testing.T) {
	expr, err := Parse(`{flag} AND {missing}`)
	assert.NoError(t, err)
	r, evalErr := Evaluate(expr, map[string]interface{}{"flag": false})
	assert.NoError(t, evalErr)
	assert.False(t, r)
}

func TestShortCircuitOR(t *testing.T) {
	expr, err := Parse(`{flag} OR {missing}`)
	assert.NoError(t, err)
	r, evalErr := Evaluate(expr, map[string]interface{}{"flag": true})
	assert.NoError(t, evalErr)
	assert.True(t, r)
}

// --- Token tests ---

func TestTokenString(t *testing.T) {
	assert.Equal(t, "AND", AND.String())
	assert.Equal(t, "OR", OR.String())
	assert.Equal(t, "==", EQ.String())
	assert.Equal(t, "!=", NEQ.String())
	assert.Equal(t, "<", LT.String())
	assert.Equal(t, "<=", LTE.String())
	assert.Equal(t, ">", GT.String())
	assert.Equal(t, ">=", GTE.String())
	assert.Equal(t, "IN", IN.String())
	assert.Equal(t, "NOT IN", NOTIN.String())
	assert.Equal(t, "CONTAINS", CONTAINS.String())
	assert.Equal(t, "NOT CONTAINS", NOTCONTAINS.String())
	assert.Equal(t, "=~", EREG.String())
	assert.Equal(t, "!~", NEREG.String())
	assert.Equal(t, "XOR", XOR.String())
	assert.Equal(t, "NAND", NAND.String())
	assert.Equal(t, "(", LPAREN.String())
	assert.Equal(t, ")", RPAREN.String())
	assert.Equal(t, "", Token(999).String())
}

func TestTokenPrecedence(t *testing.T) {
	assert.Equal(t, 1, OR.Precedence())
	assert.Equal(t, 1, XOR.Precedence())
	assert.Equal(t, 2, AND.Precedence())
	assert.Equal(t, 2, NAND.Precedence())
	assert.Equal(t, 3, EQ.Precedence())
	assert.Equal(t, 3, NEQ.Precedence())
	assert.Equal(t, 3, GT.Precedence())
	assert.Equal(t, 3, GTE.Precedence())
	assert.Equal(t, 3, LT.Precedence())
	assert.Equal(t, 3, LTE.Precedence())
	assert.Equal(t, 3, IN.Precedence())
	assert.Equal(t, 3, NOTIN.Precedence())
	assert.Equal(t, 3, EREG.Precedence())
	assert.Equal(t, 3, NEREG.Precedence())
	assert.Equal(t, 3, CONTAINS.Precedence())
	assert.Equal(t, 3, NOTCONTAINS.Precedence())
	assert.Equal(t, 0, ILLEGAL.Precedence())
}

func TestTokenIsOperator(t *testing.T) {
	assert.True(t, AND.isOperator())
	assert.True(t, OR.isOperator())
	assert.True(t, EQ.isOperator())
	assert.True(t, IN.isOperator())
	assert.True(t, CONTAINS.isOperator())
	assert.False(t, ILLEGAL.isOperator())
	assert.False(t, EOF.isOperator())
	assert.False(t, LPAREN.isOperator())
}

// --- Type mismatch errors ---

func TestEvaluateTypeMismatchErrors(t *testing.T) {
	t.Run("string vs number", func(t *testing.T) {
		expr, _ := Parse(`{foo} == "hello"`)
		_, err := Evaluate(expr, map[string]interface{}{"foo": 42})
		assert.Error(t, err)
	})
	t.Run("number vs string", func(t *testing.T) {
		expr, _ := Parse(`{foo} == 42`)
		_, err := Evaluate(expr, map[string]interface{}{"foo": "hello"})
		assert.Error(t, err)
	})
	t.Run("boolean vs number", func(t *testing.T) {
		expr, _ := Parse(`{foo} > 10`)
		_, err := Evaluate(expr, map[string]interface{}{"foo": true})
		assert.Error(t, err)
	})
}

func TestRemoveDuplicates(t *testing.T) {
	assert.Nil(t, removeDuplicates(nil))
	assert.Nil(t, removeDuplicates([]string{}))
	assert.Equal(t, []string{"a"}, removeDuplicates([]string{"a", "a", "a"}))
	assert.Equal(t, []string{"a", "b"}, removeDuplicates([]string{"a", "b", "a", "b"}))
}

func TestRemoveEOFHandling(t *testing.T) {
	t.Run("unclosed variable brace", func(t *testing.T) {
		p := NewParser(strings.NewReader("{foo"))
		_, err := p.Parse()
		assert.Error(t, err)
	})
}

func TestNewSliceStringLiteralPreallocatesMap(t *testing.T) {
	vals := make([]string, 1000)
	for i := range vals {
		vals[i] = fmt.Sprintf("item_%d", i)
	}
	ssl := NewSliceStringLiteral(vals)
	assert.Equal(t, 1000, len(ssl.m))
	assert.Equal(t, 1000, len(ssl.Val))
	_, ok := ssl.m["item_500"]
	assert.True(t, ok)
}

// --- NEW: Fix coverage tests ---

func TestShortCircuitANDNonBooleanPanic(t *testing.T) {
	// AND with non-boolean LHS should return error, not panic
	t.Run("number AND bool", func(t *testing.T) {
		expr, _ := Parse(`{foo} AND true`)
		_, err := Evaluate(expr, map[string]interface{}{"foo": 42})
		assert.Error(t, err, "should error when LHS is not boolean")
		assert.Contains(t, err.Error(), "boolean")
	})
	t.Run("string AND bool", func(t *testing.T) {
		expr, _ := Parse(`{foo} AND true`)
		_, err := Evaluate(expr, map[string]interface{}{"foo": "hello"})
		assert.Error(t, err)
	})
	t.Run("bool AND number", func(t *testing.T) {
		expr, _ := Parse(`true AND {foo}`)
		_, err := Evaluate(expr, map[string]interface{}{"foo": 42})
		assert.Error(t, err, "should error when RHS is not boolean")
	})
}

func TestShortCircuitORNonBooleanPanic(t *testing.T) {
	// OR with non-boolean LHS should return error, not panic
	t.Run("number OR bool", func(t *testing.T) {
		expr, _ := Parse(`{foo} OR true`)
		_, err := Evaluate(expr, map[string]interface{}{"foo": 42})
		assert.Error(t, err, "should error when LHS is not boolean")
	})
	t.Run("string OR bool", func(t *testing.T) {
		expr, _ := Parse(`{foo} OR true`)
		_, err := Evaluate(expr, map[string]interface{}{"foo": "hello"})
		assert.Error(t, err)
	})
	t.Run("bool OR number", func(t *testing.T) {
		expr, _ := Parse(`false OR {foo}`)
		_, err := Evaluate(expr, map[string]interface{}{"foo": 42})
		assert.Error(t, err, "should error when RHS is not boolean")
	})
}

func TestApplyEQUnsupportedTypes(t *testing.T) {
	// Comparing two unsupported types should return an error
	t.Run("slice vs slice", func(t *testing.T) {
		ssl1 := NewSliceStringLiteral([]string{"a"})
		ssl2 := NewSliceStringLiteral([]string{"b"})
		_, err := applyEQ(ssl1, ssl2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported equality comparison")
	})
	t.Run("number literal vs slice", func(t *testing.T) {
		nl := &NumberLiteral{Val: 1}
		ssl := NewSliceStringLiteral([]string{"a"})
		_, err := applyEQ(nl, ssl)
		assert.Error(t, err)
	})
}

func TestRegexCacheDoubleCheckLocking(t *testing.T) {
	// Clear cache
	regexCache.Lock()
	regexCache.m = make(map[string]*regexp.Regexp)
	regexCache.Unlock()

	t.Run("concurrent access", func(t *testing.T) {
		done := make(chan struct{})
		for i := 0; i < 10; i++ {
			go func() {
				re, err := getCompiledRegexp(`^test\d+`)
				assert.NoError(t, err)
				assert.NotNil(t, re)
				done <- struct{}{}
			}()
		}
		for i := 0; i < 10; i++ {
			<-done
		}
		// Verify it's cached
		regexCache.RLock()
		assert.Equal(t, 1, len(regexCache.m))
		regexCache.RUnlock()
	})
}

func TestApplyINUnsupportedType(t *testing.T) {
	// Boolean literal as LHS of IN should return error
	_, err := applyIN(&BooleanLiteral{Val: true}, NewSliceStringLiteral([]string{"a"}))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "IN/CONTAINS not supported for type")
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

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		d        time.Duration
		expected string
	}{
		{7 * 24 * time.Hour, "1w"},
		{14 * 24 * time.Hour, "2w"},
		{24 * time.Hour, "1d"},
		{48 * time.Hour, "2d"},
		{time.Hour, "1h"},
		{2 * time.Hour, "2h"},
		{time.Minute, "1m"},
		{5 * time.Minute, "5m"},
		{time.Second, "1s"},
		{30 * time.Second, "30s"},
		{time.Millisecond, "1ms"},
		{100 * time.Millisecond, "100ms"},
		{time.Microsecond, "1"},
		{500 * time.Microsecond, "500"},
	}
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, FormatDuration(tt.d))
		})
	}
}

func TestBoolExprSingletons(t *testing.T) {
	// Verify boolExpr returns singletons (same pointer)
	assert.Same(t, boolExpr(true), boolExpr(true))
	assert.Same(t, boolExpr(false), boolExpr(false))
	assert.NotSame(t, boolExpr(true), boolExpr(false))
}

func TestNegateWithSingletons(t *testing.T) {
	// negate should return singletons
	result, err := negate(trueExpr, nil)
	assert.NoError(t, err)
	assert.Same(t, falseExpr, result)

	result, err = negate(falseExpr, nil)
	assert.NoError(t, err)
	assert.Same(t, trueExpr, result)

	// negate propagates errors
	_, err = negate(nil, fmt.Errorf("test error"))
	assert.Error(t, err)
}

// --- BENCHMARKS ---

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
