package conditions

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

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
	{`"OFF"`, nil, false, true},
	{`"ON"`, nil, false, true},
	{`{var0} == "OFF"`, map[string]interface{}{"var0": "OFF"}, true, false},

	// AND
	{`{var0} > 10 AND {var1} == "OFF"`, map[string]interface{}{"var0": 14, "var1": "OFF"}, true, false},
	{`({var0} > 10) AND ({var1} == "OFF")`, map[string]interface{}{"var0": 14, "var1": "OFF"}, true, false},
	{`({var0} > 10) AND ({var1} == "OFF") OR true`, map[string]interface{}{"var0": 1, "var1": "ON"}, true, false},
	{`{foo}{dfs} == true and {bar} == true`, map[string]interface{}{"foo.dfs": true, "bar": true}, true, false},
	{`{foo}{dfs}{a} == true and {bar} == true`, map[string]interface{}{"foo.dfs.a": true, "bar": true}, true, false},
	{`{@foo}{a} == true and {bar} == true`, map[string]interface{}{"@foo.a": true, "bar": true}, true, false},
	{`{foo}{unknow} == true and {bar} == true`, map[string]interface{}{"foo.dfs": true, "bar": true}, false, true},
	{`{foo} == 123`, map[string]interface{}{"foo": json.Number("123"), "bar": true}, true, false},

	// OR
	{`{foo} == true OR {foo} > 1`, map[string]interface{}{"foo": true}, true, false},
	{`{foo} == true OR {foo} == false`, map[string]interface{}{"foo": true}, true, false},
	{`{foo} > 100 OR {foo} < 99 `, map[string]interface{}{"foo": 100}, false, false},
	{`{foo}{dfs} == true or {bar} == true`, map[string]interface{}{"foo.dfs": true, "bar": true}, true, false},

	// XOR
	{"false XOR false", nil, false, false},
	{"false xor true", nil, true, false},
	{"true XOR false", nil, true, false},
	{"true xor true", nil, false, false},

	// NAND
	{"false NAND false", nil, true, false},
	{"false nand true", nil, true, false},
	{"true nand false", nil, true, false},
	{"true NAND true", nil, false, false},

	// IN
	{`{foo} in {foobar}`, map[string]interface{}{"foo": "findme", "foobar": []string{"notme", "may", "findme", "lol"}}, true, false},
	{`{foo} in [123]`, map[string]interface{}{"foo": json.Number("123"), "baz": true}, true, false},
	{`{foo} in [123]`, map[string]interface{}{"foo": json.Number("124"), "baz": true}, false, false},

	// NOT IN
	{`{foo} not in {foobar}`, map[string]interface{}{"foo": "dontfindme", "foobar": []string{"notme", "may", "findme", "lol"}}, true, false},

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

	// NOT IN with array of numbers
	{`{foo} not in [2,3,4]`, map[string]interface{}{"foo": 4}, false, false},
	{`{foo} not in [2,3,4]`, map[string]interface{}{"foo": 5}, true, false},

	// CONTAINS
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

	// NOT CONTAINS
	{`{foo} not contains "2"`, map[string]interface{}{"foo": []string{"1", "2"}}, false, false},
	{`{foo} not contains "0"`, map[string]interface{}{"foo": []string{"1", "2"}}, true, false},
	{`{foo} not contains 0`, map[string]interface{}{"foo": []string{"1", "2"}}, false, true},
	{`{foo} not contains 0`, map[string]interface{}{"bar": []string{"1", "2"}}, false, true},

	// =~
	{`{status} =~ /^5\d\d/`, map[string]interface{}{"status": "500"}, true, false},
	{`{status} =~ /^4\d\d/`, map[string]interface{}{"status": "500"}, false, false},
	{`{status} =~ /foo/`, map[string]interface{}{"status": "foobar"}, true, false},
	{`{status} =~ "foo"`, map[string]interface{}{"status": "foobar"}, true, false},
	{`{status} =~ "foo"`, map[string]interface{}{"status": "bar"}, false, false},

	// !~
	{"{status} !~ /^5\\d\\d/", map[string]interface{}{"status": "500"}, false, false},
	{"{status} !~ /^4\\d\\d/", map[string]interface{}{"status": "500"}, true, false},

	// --- Key composition (concat) edge cases ---

	// Number comparison with composed key
	{`{user}{age} > 18`, map[string]interface{}{"user.age": 25.0}, true, false},
	{`{user}{age} > 18`, map[string]interface{}{"user.age": 15.0}, false, false},

	// IN with composed key
	{`{user}{role} IN ["admin", "moderator"]`, map[string]interface{}{"user.role": "admin"}, true, false},
	{`{user}{role} IN ["admin", "moderator"]`, map[string]interface{}{"user.role": "viewer"}, false, false},

	// NOT IN with composed key
	{`{user}{role} NOT IN ["banned"]`, map[string]interface{}{"user.role": "admin"}, true, false},
	{`{user}{role} NOT IN ["banned"]`, map[string]interface{}{"user.role": "banned"}, false, false},

	// CONTAINS with composed key
	{`{user}{tags} CONTAINS "urgent"`, map[string]interface{}{"user.tags": []string{"urgent", "billing"}}, true, false},
	{`{user}{tags} CONTAINS "urgent"`, map[string]interface{}{"user.tags": []string{"spam"}}, false, false},

	// NOT CONTAINS with composed key
	{`{user}{tags} NOT CONTAINS "spam"`, map[string]interface{}{"user.tags": []string{"urgent", "billing"}}, true, false},

	// Regex with composed key
	{`{user}{status} =~ /^5\d\d/`, map[string]interface{}{"user.status": "500"}, true, false},
	{`{user}{status} =~ /^5\d\d/`, map[string]interface{}{"user.status": "400"}, false, false},

	// Hyphenated names with composition
	{`{my-var}{sub-key} == "val"`, map[string]interface{}{"my-var.sub-key": "val"}, true, false},
	{`{my-var}{sub-key} == "wrong"`, map[string]interface{}{"my-var.sub-key": "val"}, false, false},

	// Concat in both LHS and RHS
	{`{a}{x} == {b}{y}`, map[string]interface{}{"a.x": "hello", "b.y": "hello"}, true, false},
	{`{a}{x} == {b}{y}`, map[string]interface{}{"a.x": "hello", "b.y": "world"}, false, false},

	// Concat inside parentheses
	{`({user}{age} > 18)`, map[string]interface{}{"user.age": 25.0}, true, false},
	{`(({user}{age} > 18))`, map[string]interface{}{"user.age": 25.0}, true, false},

	// Four-level concatenation
	{`{a}{b}{c}{d} == true`, map[string]interface{}{"a.b.c.d": true}, true, false},

	// Concat with AND short-circuit
	{`false AND {missing}{key}` + " == true", nil, false, false},

	// Concat with OR short-circuit
	{`true OR {missing}{key}` + " == true", nil, true, false},

	// Concat in compound expression with various operators (parens required
	// for correct precedence when mixing AND with =~)
	{`{user}{age} > 18 AND {user}{role} IN ["admin"] AND ({user}{status} =~ /^A/)`,
		map[string]interface{}{"user.age": 25.0, "user.role": "admin", "user.status": "Active"},
		true, false},

	// Error: composed key missing from args
	{`{user}{missing} == true`, map[string]interface{}{"user.name": "Alice"}, false, true},

	// Error: compose with missing second part
	{`{a}{b} == true`, map[string]interface{}{"a.x": true}, false, true},

	// Error: nested map does NOT work with composed keys — {user}{status}
	// looks up args["user.status"], not args["user"]["status"]
	{`{user}{status} == 100`, map[string]interface{}{"user": map[string]interface{}{"status": 100}}, false, true},

	// --- Nested path traversal ({foo.bar}, {users[0]}) ---

	// Simple dot access into nested maps
	{`{user.name} == "Alice"`, map[string]interface{}{"user": map[string]interface{}{"name": "Alice"}}, true, false},
	{`{user.name} == "Bob"`, map[string]interface{}{"user": map[string]interface{}{"name": "Alice"}}, false, false},

	// Array index access
	{`{users[0]} == 42`, map[string]interface{}{"users": []interface{}{42, 43, 44}}, true, false},
	{`{users[1]} == 43`, map[string]interface{}{"users": []interface{}{42, 43, 44}}, true, false},
	{`{users[2]} == 99`, map[string]interface{}{"users": []interface{}{42, 43, 44}}, false, false},

	// Negative index
	{`{users[-1]} == 44`, map[string]interface{}{"users": []interface{}{42, 43, 44}}, true, false},

	// Chained dot + bracket
	{`{data[0].name} == "foo"`, map[string]interface{}{
		"data": []interface{}{map[string]interface{}{"name": "foo"}},
	}, true, false},

	// Deep nesting
	{`{a.b.c} == 42`, map[string]interface{}{"a": map[string]interface{}{"b": map[string]interface{}{"c": 42.0}}}, true, false},

	// Number comparison
	{`{user.age} > 18`, map[string]interface{}{"user": map[string]interface{}{"age": 25.0}}, true, false},
	{`{user.age} > 18`, map[string]interface{}{"user": map[string]interface{}{"age": 15.0}}, false, false},

	// IN with nested access
	{`{user.role} IN ["admin", "moderator"]`, map[string]interface{}{"user": map[string]interface{}{"role": "admin"}}, true, false},

	// CONTAINS with nested access
	{`{user.tags} CONTAINS "urgent"`, map[string]interface{}{"user": map[string]interface{}{"tags": []string{"urgent", "billing"}}}, true, false},

	// Regex with nested access
	{`{user.status} =~ /^5\d\d/`, map[string]interface{}{"user": map[string]interface{}{"status": "500"}}, true, false},

	// Path inside parentheses
	{`({user.age} > 18)`, map[string]interface{}{"user": map[string]interface{}{"age": 25.0}}, true, false},
	{`(({user.age} > 18))`, map[string]interface{}{"user": map[string]interface{}{"age": 25.0}}, true, false},

	// Path in both LHS and RHS
	{`{a.x} == {b.y}`, map[string]interface{}{
		"a": map[string]interface{}{"x": "hello"},
		"b": map[string]interface{}{"y": "hello"},
	}, true, false},
	{`{a.x} == {b.y}`, map[string]interface{}{
		"a": map[string]interface{}{"x": "hello"},
		"b": map[string]interface{}{"y": "world"},
	}, false, false},

	// @ prefix with path
	{`{@user.name} == "Alice"`, map[string]interface{}{"@user": map[string]interface{}{"name": "Alice"}}, true, false},

	// Chained: {a.b[0].c.d}
	{`{a.b[0].c.d} == 1`, map[string]interface{}{
		"a": map[string]interface{}{
			"b": []interface{}{
				map[string]interface{}{
					"c": map[string]interface{}{"d": 1.0},
				},
			},
		},
	}, true, false},

	// Path with AND short-circuit
	{`false AND {missing.deep.key} == true`, nil, false, false},

	// Path with OR short-circuit
	{`true OR {missing.deep.key} == true`, nil, true, false},

	// Error: path key not found in nested map
	{`{user.missing} == true`, map[string]interface{}{"user": map[string]interface{}{"name": "Alice"}}, false, true},

	// Error: array index out of bounds
	{`{users[999]} == 42`, map[string]interface{}{"users": []interface{}{1, 2, 3}}, false, true},

	// Error: access key on non-map value
	{`{user.name.nested} == true`, map[string]interface{}{"user": map[string]interface{}{"name": "Alice"}}, false, true},

	// Error: access key on array (should use index)
	{`{users.foo} == 42`, map[string]interface{}{"users": []interface{}{1, 2, 3}}, false, true},

	// Error: path root not found
	{`{missing.key} == true`, map[string]interface{}{"foo": "bar"}, false, true},
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
					return
				}
				t.Fatalf("Unexpected error evaluating %q: %s", expr, err)
			} else if td.isErr {
				t.Fatalf("Expected error but got none for: %s", expr)
			}
			assert.Equal(t, td.result, r, "Expression: %s, Args: %#v", td.cond, td.args)
		})
	}
}

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

		// JSON interop with composed keys
		{`{user}{name} == "Alice"`, `{"user.name": "Alice"}`, true, false},
		{`{user}{age} > 18`, `{"user.age": 25}`, true, false},
		{`{user}{role} IN ["admin"]`, `{"user.role": "admin"}`, true, false},
		{`{user}{role} IN ["admin"]`, `{"user.role": "viewer"}`, false, false},

		// JSON interop with nested path traversal
		{`{user.name} == "Alice"`, `{"user": {"name": "Alice"}}`, true, false},
		{`{user.age} > 18`, `{"user": {"age": 25}}`, true, false},
		{`{users[0]} == 42`, `{"users": [42, 43]}`, true, false},
		{`{data[0].name} == "foo"`, `{"data": [{"name": "foo"}]}`, true, false},
		{`{data[0].items[1]} == 200`, `{"data": [{"items": [100, 200]}]}`, true, false},
	}

	for _, test := range tests {
		t.Run(test.cond+"_"+test.jsonStr, func(t *testing.T) {
			p := NewParser(strings.NewReader(test.cond))
			expr, _ := p.Parse()
		data := make(map[string]interface{})
		if err := json.Unmarshal([]byte(test.jsonStr), &data); err != nil {
			t.Fatalf("failed to unmarshal json: %v", err)
		}
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

func TestUnsupportedOperator(t *testing.T) {
	_, err := applyOperator(Token(999), &BooleanLiteral{Val: true}, &BooleanLiteral{Val: false})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported operator")
}

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

func TestEvalBinaryXorNandDirect(t *testing.T) {
	t.Run("XOR", func(t *testing.T) {
		expr, _ := Parse(`true XOR false`)
		r, err := Evaluate(expr, nil)
		assert.NoError(t, err)
		assert.True(t, r)
	})
	t.Run("NAND", func(t *testing.T) {
		expr, _ := Parse(`true NAND true`)
		r, err := Evaluate(expr, nil)
		assert.NoError(t, err)
		assert.False(t, r)
	})
}
