package conditions

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// testCase represents a single evaluation test.
type testCase struct {
	cond   string
	args   map[string]interface{}
	result bool
	isErr  bool
}

// runTestCases runs a table of evaluation tests as subtests named by expression.
func runTestCases(t *testing.T, cases []testCase) {
	for _, td := range cases {
		t.Run(td.cond, func(t *testing.T) {
			p := NewParser(strings.NewReader(td.cond))
			expr, err := p.Parse()
			if err != nil {
				if td.isErr {
					return
				}
				t.Fatalf("Unexpected parse error for %q: %s", td.cond, err)
			}

			r, err := Evaluate(expr, td.args)
			if err != nil {
				if td.isErr {
					return
				}
				t.Fatalf("Unexpected eval error for %q: %s", td.cond, err)
			}
			if td.isErr {
				t.Fatalf("Expected error but got none for: %s", td.cond)
			}
			assert.Equal(t, td.result, r, "Expression: %s", td.cond)
		})
	}
}

// runJSONTests runs a table of JSON unmarshal + evaluation tests.
func runJSONTests(t *testing.T, tests []struct {
	cond    string
	jsonStr string
	result  bool
	isErr   bool
}) {
	for _, test := range tests {
		t.Run(test.cond+" | "+test.jsonStr, func(t *testing.T) {
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

// ---------------------------------------------------------------------------
// Invalid expressions — expect parse errors
// ---------------------------------------------------------------------------

var invalidExpressions = []string{
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

func TestParseErrors(t *testing.T) {
	for _, cond := range invalidExpressions {
		t.Run(cond, func(t *testing.T) {
			p := NewParser(strings.NewReader(cond))
			expr, err := p.Parse()
			assert.Error(t, err, "Should receive error for: %s", cond)
			assert.Nil(t, expr, "Expression should be nil for: %s", cond)
		})
	}
}

// ---------------------------------------------------------------------------
// Literals
// ---------------------------------------------------------------------------

func TestBooleanLiterals(t *testing.T) {
	runTestCases(t, []testCase{
		{cond: "true", result: true},
		{cond: "false", result: false},
	})
}

func TestLiteralAsExpressionErrors(t *testing.T) {
	// Standalone non-boolean literals are errors — root must be boolean.
	runTestCases(t, []testCase{
		{cond: "56.43", isErr: true},
		{cond: `"OFF"`, isErr: true},
		{cond: `"ON"`, isErr: true},
	})
}

// ---------------------------------------------------------------------------
// Variable references — {foo}
// ---------------------------------------------------------------------------

func TestSimpleVariableRef(t *testing.T) {
	runTestCases(t, []testCase{
		{cond: "{var0}", args: map[string]interface{}{"var0": true}, result: true},
		{cond: "{var0}", args: map[string]interface{}{"var0": false}, result: false},
	})
}

func TestMissingVariableError(t *testing.T) {
	runTestCases(t, []testCase{
		{cond: "{var5}", isErr: true},
		{cond: "{var5}", args: map[string]interface{}{"var5-type-2": 1}, isErr: true},
	})
}

func TestHyphenatedVarNames(t *testing.T) {
	runTestCases(t, []testCase{
		{cond: "{var5-type-2} == 1", args: map[string]interface{}{"var5-type-2": 1}, result: true},
	})
}

// ---------------------------------------------------------------------------
// Logical operators — AND / OR / XOR / NAND
// ---------------------------------------------------------------------------

func TestLogicalAND(t *testing.T) {
	runTestCases(t, []testCase{
		{cond: "{var0} and {var1}", args: map[string]interface{}{"var0": true, "var1": true}, result: true},
		{cond: "{var0} AND {var1}", args: map[string]interface{}{"var0": true, "var1": false}, result: false},
		{cond: "{var0} AND {var1}", args: map[string]interface{}{"var0": false, "var1": true}, result: false},
		{cond: "{var0} AND {var1}", args: map[string]interface{}{"var0": false, "var1": false}, result: false},
		{cond: "{var0} AND false", args: map[string]interface{}{"var0": true}, result: false},
		{cond: "false OR true OR false OR false OR true", result: true},
		{cond: "((false OR true) AND false) OR (false OR true)", result: true},
	})
}

func TestLogicalOR(t *testing.T) {
	runTestCases(t, []testCase{
		{cond: "{foo} == true OR {foo} > 1", args: map[string]interface{}{"foo": true}, result: true},
		{cond: "{foo} == true OR {foo} == false", args: map[string]interface{}{"foo": true}, result: true},
		{cond: "{foo} > 100 OR {foo} < 99 ", args: map[string]interface{}{"foo": 100}, result: false},
	})
}

func TestLogicalXOR(t *testing.T) {
	runTestCases(t, []testCase{
		{cond: "false XOR false", result: false},
		{cond: "false xor true", result: true},
		{cond: "true XOR false", result: true},
		{cond: "true xor true", result: false},
	})
}

func TestLogicalNAND(t *testing.T) {
	runTestCases(t, []testCase{
		{cond: "false NAND false", result: true},
		{cond: "false nand true", result: true},
		{cond: "true nand false", result: true},
		{cond: "true NAND true", result: false},
	})
}

// ---------------------------------------------------------------------------
// Comparison operators — ==, !=, >, >=, <, <=
// ---------------------------------------------------------------------------

func TestComparisonNumber(t *testing.T) {
	runTestCases(t, []testCase{
		{cond: "{var0} > -100 AND {var0} < -50", args: map[string]interface{}{"var0": -75.4}, result: true},
	})
}

func TestComparisonWithParens(t *testing.T) {
	runTestCases(t, []testCase{
		{cond: `{var0} > 10 AND {var1} == "OFF"`, args: map[string]interface{}{"var0": 14, "var1": "OFF"}, result: true},
		{cond: `({var0} > 10) AND ({var1} == "OFF")`, args: map[string]interface{}{"var0": 14, "var1": "OFF"}, result: true},
		{cond: `({var0} > 10) AND ({var1} == "OFF") OR true`, args: map[string]interface{}{"var0": 1, "var1": "ON"}, result: true},
	})
}

func TestStringEquality(t *testing.T) {
	runTestCases(t, []testCase{
		{cond: `{var0} == "OFF"`, args: map[string]interface{}{"var0": "OFF"}, result: true},
	})
}

func TestTypeMismatchErrors(t *testing.T) {
	runTestCases(t, []testCase{
		{cond: "{var0} > true", isErr: true},
		{cond: "{var0} > true", args: map[string]interface{}{"var0": 43}, isErr: true},
		{cond: "{var0} > true", args: map[string]interface{}{"var0": false}, isErr: true},
		{cond: `{foo} == "123"`, args: map[string]interface{}{"foo": 123}, isErr: true},
	})
}

// ---------------------------------------------------------------------------
// Pattern matching — =~, !~
// ---------------------------------------------------------------------------

func TestRegexMatch(t *testing.T) {
	runTestCases(t, []testCase{
		{cond: `{status} =~ /^5\d\d/`, args: map[string]interface{}{"status": "500"}, result: true},
		{cond: `{status} =~ /^4\d\d/`, args: map[string]interface{}{"status": "500"}, result: false},
		{cond: `{status} =~ /foo/`, args: map[string]interface{}{"status": "foobar"}, result: true},
		{cond: `{status} =~ "foo"`, args: map[string]interface{}{"status": "foobar"}, result: true},
		{cond: `{status} =~ "foo"`, args: map[string]interface{}{"status": "bar"}, result: false},
	})
}

func TestRegexNoMatch(t *testing.T) {
	runTestCases(t, []testCase{
		{cond: `{status} !~ /^5\d\d/`, args: map[string]interface{}{"status": "500"}, result: false},
		{cond: `{status} !~ /^4\d\d/`, args: map[string]interface{}{"status": "500"}, result: true},
	})
}

// ---------------------------------------------------------------------------
// Membership — IN, NOT IN, CONTAINS, NOT CONTAINS
// ---------------------------------------------------------------------------

func TestIN(t *testing.T) {
	runTestCases(t, []testCase{
		{cond: `{foo} in {foobar}`, args: map[string]interface{}{"foo": "findme", "foobar": []string{"notme", "may", "findme", "lol"}}, result: true},
		{cond: `{foo} in [123]`, args: map[string]interface{}{"foo": json.Number("123")}, result: true},
		{cond: `{foo} in [123]`, args: map[string]interface{}{"foo": json.Number("124")}, result: false},
		{cond: `{foo} in ["bonjour", "le monde", "oui"]`, args: map[string]interface{}{"foo": "le monde"}, result: true},
		{cond: `{foo} in ["bonjour", "le monde", "oui"]`, args: map[string]interface{}{"foo": "world"}, result: false},
		{cond: `{foo} in [2,3,4]`, args: map[string]interface{}{"foo": 4}, result: true},
		{cond: `{foo} in [2,3,4] AND {foo} == 4`, args: map[string]interface{}{"foo": 4}, result: true},
		{cond: `{foo} in [2,3,4] AND {foo} == 3`, args: map[string]interface{}{"foo": 4}, result: false},
		{cond: `{foo} in [2,3,4]`, args: map[string]interface{}{"foo": 5}, result: false},
	})
}

func TestNOTIN(t *testing.T) {
	runTestCases(t, []testCase{
		{cond: `{foo} not in {foobar}`, args: map[string]interface{}{"foo": "dontfindme", "foobar": []string{"notme", "may", "findme", "lol"}}, result: true},
		{cond: `{foo} not in ["bonjour", "le monde", "oui"]`, args: map[string]interface{}{"foo": "le monde"}, result: false},
		{cond: `{foo} not in ["bonjour", "le monde", "oui"]`, args: map[string]interface{}{"foo": "world"}, result: true},
		{cond: `{foo} not in [2,3,4]`, args: map[string]interface{}{"foo": 4}, result: false},
		{cond: `{foo} not in [2,3,4]`, args: map[string]interface{}{"foo": 5}, result: true},
	})
}

func TestCONTAINS(t *testing.T) {
	runTestCases(t, []testCase{
		{cond: `{foo} contains "2"`, args: map[string]interface{}{"foo": []string{"1", "2"}}, result: true},
		{cond: `{foo} contains "2"`, args: map[string]interface{}{"foo": []string{}}, result: false},
		{cond: `{foo} contains "2" and {foo} contains "1"`, args: map[string]interface{}{"foo": []string{"1", "2"}}, result: true},
		{cond: `{foo} contains "2" and {foo} contains "0"`, args: map[string]interface{}{"foo": []string{"1", "2"}}, result: false},
		{cond: `{foo} contains "2" or {foo} contains "0"`, args: map[string]interface{}{"foo": []string{"1", "2"}}, result: true},
		{cond: `{foo} contains 2 and {foo} contains 1`, args: map[string]interface{}{"foo": []int{1, 2}}, result: true},
		{cond: `{foo} contains "2" and {foo} contains 1`, args: map[string]interface{}{"foo": []int{1, 2}}, isErr: true},
		{cond: `{foo} contains {bar}`, args: map[string]interface{}{"foo": []string{"1", "2"}, "bar": "1"}, result: true},
		{cond: `{foo} contains {bar}`, args: map[string]interface{}{"foo": []int{1, 2}, "bar": int32(1)}, result: true},
		{cond: `{foo} contains {bar}`, args: map[string]interface{}{"foo": []int{1, 2, 3}, "bar": float32(1.0 + 2.0)}, result: true},
		{cond: `{foo} contains {bar}`, args: map[string]interface{}{"foo": []float64{0.29}, "bar": float32(29.0 / 100)}, result: true},
		{cond: `{foo} contains 2`, args: map[string]interface{}{"foo": []json.Number{"2"}}, result: true},
		{cond: `{foo} contains 2`, args: map[string]interface{}{"foo": []json.Number{"2", "3"}}, result: true},
		{cond: `{foo} contains 2`, args: map[string]interface{}{"foo": []json.Number{"3"}}, result: false},
		{cond: `{foo} contains 2`, args: map[string]interface{}{"foo": []interface{}{json.Number("2")}}, result: true},
		{cond: `{foo} contains 2`, args: map[string]interface{}{"foo": []interface{}{json.Number("3")}}, result: false},
	})
}

func TestNOTCONTAINS(t *testing.T) {
	runTestCases(t, []testCase{
		{cond: `{foo} not contains "2"`, args: map[string]interface{}{"foo": []string{"1", "2"}}, result: false},
		{cond: `{foo} not contains "0"`, args: map[string]interface{}{"foo": []string{"1", "2"}}, result: true},
		{cond: `{foo} not contains 0`, args: map[string]interface{}{"foo": []string{"1", "2"}}, isErr: true},
		{cond: `{foo} not contains 0`, args: map[string]interface{}{"bar": []string{"1", "2"}}, isErr: true},
	})
}

// ---------------------------------------------------------------------------
// Key composition — {a}{b} → flat key "a.b"
// ---------------------------------------------------------------------------

func TestKeyComposition(t *testing.T) {
	runTestCases(t, []testCase{
		// Two-level
		{cond: "{foo}{dfs} == true and {bar} == true",
			args: map[string]interface{}{"foo.dfs": true, "bar": true}, result: true},
		// Three-level
		{cond: "{foo}{dfs}{a} == true and {bar} == true",
			args: map[string]interface{}{"foo.dfs.a": true, "bar": true}, result: true},
		// @ prefix
		{cond: "{@foo}{a} == true and {bar} == true",
			args: map[string]interface{}{"@foo.a": true, "bar": true}, result: true},
		// Missing key error
		{cond: "{foo}{unknow} == true and {bar} == true",
			args: map[string]interface{}{"foo.dfs": true, "bar": true}, isErr: true},
		// OR with composed key
		{cond: "{foo}{dfs} == true or {bar} == true",
			args: map[string]interface{}{"foo.dfs": true, "bar": true}, result: true},
	})
}

func TestKeyCompositionEdgeCases(t *testing.T) {
	runTestCases(t, []testCase{
		// Number comparison
		{cond: "{user}{age} > 18", args: map[string]interface{}{"user.age": 25.0}, result: true},
		{cond: "{user}{age} > 18", args: map[string]interface{}{"user.age": 15.0}, result: false},

		// IN
		{cond: `{user}{role} IN ["admin", "moderator"]`, args: map[string]interface{}{"user.role": "admin"}, result: true},
		{cond: `{user}{role} IN ["admin", "moderator"]`, args: map[string]interface{}{"user.role": "viewer"}, result: false},
		{cond: `{user}{role} NOT IN ["banned"]`, args: map[string]interface{}{"user.role": "admin"}, result: true},

		// CONTAINS
		{cond: `{user}{tags} CONTAINS "urgent"`, args: map[string]interface{}{"user.tags": []string{"urgent", "billing"}}, result: true},
		{cond: `{user}{tags} NOT CONTAINS "spam"`, args: map[string]interface{}{"user.tags": []string{"urgent", "billing"}}, result: true},

		// Regex
		{cond: `{user}{status} =~ /^5\d\d/`, args: map[string]interface{}{"user.status": "500"}, result: true},
		{cond: `{user}{status} =~ /^5\d\d/`, args: map[string]interface{}{"user.status": "400"}, result: false},

		// Hyphenated
		{cond: `{my-var}{sub-key} == "val"`, args: map[string]interface{}{"my-var.sub-key": "val"}, result: true},
		{cond: `{my-var}{sub-key} == "wrong"`, args: map[string]interface{}{"my-var.sub-key": "val"}, result: false},

		// Both sides
		{cond: `{a}{x} == {b}{y}`, args: map[string]interface{}{"a.x": "hello", "b.y": "hello"}, result: true},
		{cond: `{a}{x} == {b}{y}`, args: map[string]interface{}{"a.x": "hello", "b.y": "world"}, result: false},

		// Parens
		{cond: `({user}{age} > 18)`, args: map[string]interface{}{"user.age": 25.0}, result: true},

		// Four-level
		{cond: "{a}{b}{c}{d} == true", args: map[string]interface{}{"a.b.c.d": true}, result: true},

		// Short-circuit
		{cond: "false AND {missing}{key} == true", result: false},
		{cond: "true OR {missing}{key} == true", result: true},
	})
}

func TestKeyCompositionCompound(t *testing.T) {
	runTestCases(t, []testCase{
		// Multiple operators; parens needed around regex for correct precedence
		{cond: `{user}{age} > 18 AND {user}{role} IN ["admin"] AND ({user}{status} =~ /^A/)`,
			args: map[string]interface{}{"user.age": 25.0, "user.role": "admin", "user.status": "Active"},
			result: true},
	})
}

func TestKeyCompositionErrors(t *testing.T) {
	runTestCases(t, []testCase{
		// Missing key
		{cond: "{user}{missing} == true", args: map[string]interface{}{"user.name": "Alice"}, isErr: true},
		// Partial match
		{cond: "{a}{b} == true", args: map[string]interface{}{"a.x": true}, isErr: true},
		// Nested map does NOT work with flat-key syntax
		{cond: "{user}{status} == 100",
			args: map[string]interface{}{"user": map[string]interface{}{"status": 100}}, isErr: true},
	})
}

// ---------------------------------------------------------------------------
// Nested path traversal — {foo.bar}, {users[0]}
// ---------------------------------------------------------------------------

func TestNestedPathDotAccess(t *testing.T) {
	runTestCases(t, []testCase{
		{cond: `{user.name} == "Alice"`,
			args: map[string]interface{}{"user": map[string]interface{}{"name": "Alice"}}, result: true},
		{cond: `{user.name} == "Bob"`,
			args: map[string]interface{}{"user": map[string]interface{}{"name": "Alice"}}, result: false},
		{cond: `{user.age} > 18`,
			args: map[string]interface{}{"user": map[string]interface{}{"age": 25.0}}, result: true},
		{cond: `{user.age} > 18`,
			args: map[string]interface{}{"user": map[string]interface{}{"age": 15.0}}, result: false},
		{cond: `{a.b.c} == 42`,
			args: map[string]interface{}{"a": map[string]interface{}{"b": map[string]interface{}{"c": 42.0}}}, result: true},
	})
}

func TestNestedPathArrayAccess(t *testing.T) {
	runTestCases(t, []testCase{
		// Simple index
		{cond: "{users[0]} == 42",
			args: map[string]interface{}{"users": []interface{}{42, 43, 44}}, result: true},
		{cond: "{users[1]} == 43",
			args: map[string]interface{}{"users": []interface{}{42, 43, 44}}, result: true},
		{cond: "{users[2]} == 99",
			args: map[string]interface{}{"users": []interface{}{42, 43, 44}}, result: false},
		// Negative index (from end)
		{cond: "{users[-1]} == 44",
			args: map[string]interface{}{"users": []interface{}{42, 43, 44}}, result: true},
		// Array of maps
		{cond: `{users[0].name} == "Alice"`,
			args: map[string]interface{}{"users": []interface{}{
				map[string]interface{}{"name": "Alice"},
				map[string]interface{}{"name": "Bob"},
			}}, result: true},
		{cond: `{users[1].name} == "Bob"`,
			args: map[string]interface{}{"users": []interface{}{
				map[string]interface{}{"name": "Alice"},
				map[string]interface{}{"name": "Bob"},
			}}, result: true},
		// Deeply nested arrays
		{cond: "{matrix[0][1]} == 2",
			args: map[string]interface{}{"matrix": []interface{}{
				[]interface{}{1, 2, 3},
				[]interface{}{4, 5, 6},
			}}, result: true},
		{cond: "{matrix[1][0]} == 4",
			args: map[string]interface{}{"matrix": []interface{}{
				[]interface{}{1, 2, 3},
				[]interface{}{4, 5, 6},
			}}, result: true},
	})
}

func TestNestedPathChained(t *testing.T) {
	runTestCases(t, []testCase{
		// Dot + bracket mixed
		{cond: `{data[0].name} == "foo"`,
			args: map[string]interface{}{
				"data": []interface{}{map[string]interface{}{"name": "foo"}},
			}, result: true},
		// Bracket + dot + bracket + dot
		{cond: "{a.b[0].c.d} == 1",
			args: map[string]interface{}{
				"a": map[string]interface{}{
					"b": []interface{}{
						map[string]interface{}{
							"c": map[string]interface{}{"d": 1.0},
						},
					},
				},
			}, result: true},
		// Multiple arrays
		{cond: `{stores[0].items[1].price} > 10`,
			args: map[string]interface{}{
				"stores": []interface{}{
					map[string]interface{}{
						"items": []interface{}{
							map[string]interface{}{"price": 5.0},
							map[string]interface{}{"price": 15.0},
						},
					},
				},
			}, result: true},
		{cond: `{stores[0].items[0].price} > 10`,
			args: map[string]interface{}{
				"stores": []interface{}{
					map[string]interface{}{
						"items": []interface{}{
							map[string]interface{}{"price": 5.0},
							map[string]interface{}{"price": 15.0},
						},
					},
				},
			}, result: false},
	})
}

func TestNestedPathOperators(t *testing.T) {
	runTestCases(t, []testCase{
		// IN with path
		{cond: `{user.role} IN ["admin", "moderator"]`,
			args: map[string]interface{}{"user": map[string]interface{}{"role": "admin"}}, result: true},

		// CONTAINS with path
		{cond: `{user.tags} CONTAINS "urgent"`,
			args: map[string]interface{}{"user": map[string]interface{}{"tags": []string{"urgent", "billing"}}}, result: true},

		// Regex with path
		{cond: `{user.status} =~ /^5\d\d/`,
			args: map[string]interface{}{"user": map[string]interface{}{"status": "500"}}, result: true},

		// Path on both sides of comparison
		{cond: `{a.x} == {b.y}`,
			args: map[string]interface{}{
				"a": map[string]interface{}{"x": "hello"},
				"b": map[string]interface{}{"y": "hello"},
			}, result: true},
		{cond: `{a.x} == {b.y}`,
			args: map[string]interface{}{
				"a": map[string]interface{}{"x": "hello"},
				"b": map[string]interface{}{"y": "world"},
			}, result: false},

		// @ prefix with path
		{cond: `{@user.name} == "Alice"`,
			args: map[string]interface{}{"@user": map[string]interface{}{"name": "Alice"}}, result: true},
	})
}

func TestNestedPathParens(t *testing.T) {
	runTestCases(t, []testCase{
		{cond: `({user.age} > 18)`,
			args: map[string]interface{}{"user": map[string]interface{}{"age": 25.0}}, result: true},
		{cond: `(({user.age} > 18))`,
			args: map[string]interface{}{"user": map[string]interface{}{"age": 25.0}}, result: true},
	})
}

func TestNestedPathShortCircuit(t *testing.T) {
	runTestCases(t, []testCase{
		{cond: "false AND {missing.deep.key} == true", result: false},
		{cond: "true OR {missing.deep.key} == true", result: true},
	})
}

func TestNestedPathErrors(t *testing.T) {
	runTestCases(t, []testCase{
		// Missing key in nested map
		{cond: "{user.missing} == true",
			args: map[string]interface{}{"user": map[string]interface{}{"name": "Alice"}}, isErr: true},
		// Index out of bounds
		{cond: "{users[999]} == 42",
			args: map[string]interface{}{"users": []interface{}{1, 2, 3}}, isErr: true},
		// Access key on string (non-map)
		{cond: "{user.name.nested} == true",
			args: map[string]interface{}{"user": map[string]interface{}{"name": "Alice"}}, isErr: true},
		// Access key on array (should use index)
		{cond: "{users.foo} == 42",
			args: map[string]interface{}{"users": []interface{}{1, 2, 3}}, isErr: true},
		// Root not found
		{cond: "{missing.key} == true", args: map[string]interface{}{"foo": "bar"}, isErr: true},
	})
}

// ---------------------------------------------------------------------------
// Mixed flat-key + nested-path in one expression
// ---------------------------------------------------------------------------

func TestMixedKeyCompositionAndPath(t *testing.T) {
	runTestCases(t, []testCase{
		// Flat {user}{age} + nested {user.tags} in same expression
		{cond: `{user}{age} > 18 AND {user.tags} CONTAINS "admin"`,
			args: map[string]interface{}{
				"user.age": 25.0,
				"user":     map[string]interface{}{"tags": []string{"admin"}},
			}, result: true},
		// Same, false case
		{cond: `{user}{age} > 18 AND {user.tags} CONTAINS "admin"`,
			args: map[string]interface{}{
				"user.age": 15.0,
				"user":     map[string]interface{}{"tags": []string{"admin"}},
			}, result: false},
		// Both sides use different approaches
		{cond: `{a}{flat} == "yes" AND {b.nested} == "ok"`,
			args: map[string]interface{}{
				"a.flat":   "yes",
				"b":        map[string]interface{}{"nested": "ok"},
			}, result: true},
	})
}

// ---------------------------------------------------------------------------
// JSON interoperability
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// JSON-based tests — real json.Unmarshal roundtrip
// ---------------------------------------------------------------------------

func TestJSONInterop(t *testing.T) {
	runJSONTests(t, []struct {
		cond    string
		jsonStr string
		result  bool
		isErr   bool
	}{
		// Basic flat keys
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

		// Composed keys (flat dotted)
		{`{user}{name} == "Alice"`, `{"user.name": "Alice"}`, true, false},
		{`{user}{age} > 18`, `{"user.age": 25}`, true, false},
		{`{user}{role} IN ["admin"]`, `{"user.role": "admin"}`, true, false},
		{`{user}{role} IN ["admin"]`, `{"user.role": "viewer"}`, false, false},

		// Nested path traversal
		{`{user.name} == "Alice"`, `{"user": {"name": "Alice"}}`, true, false},
		{`{user.age} > 18`, `{"user": {"age": 25}}`, true, false},
		{`{users[0]} == 42`, `{"users": [42, 43]}`, true, false},
		{`{data[0].name} == "foo"`, `{"data": [{"name": "foo"}]}`, true, false},
		{`{data[0].items[1]} == 200`, `{"data": [{"items": [100, 200]}]}`, true, false},
	})
}

func TestJSONComprehensive(t *testing.T) {
	runJSONTests(t, []struct {
		cond    string
		jsonStr string
		result  bool
		isErr   bool
	}{
		// ── Flat key types from JSON ──────────────────────────────────────

		{`{bool_val} == true`, `{"bool_val": true}`, true, false},
		{`{bool_val} == false`, `{"bool_val": true}`, false, false},
		{`{str_val} == "hello"`, `{"str_val": "hello"}`, true, false},
		{`{num_val} > 10`, `{"num_val": 15}`, true, false},
		{`{num_val} >= 10`, `{"num_val": 10}`, true, false},
		{`{num_val} == 0`, `{"num_val": 0}`, true, false},
		{`{num_val} == -5`, `{"num_val": -5}`, true, false},

		// ── Nested objects (dot path) ────────────────────────────────────

		{`{a.b} == 42`, `{"a": {"b": 42}}`, true, false},
		{`{a.b.c} == 42`, `{"a": {"b": {"c": 42}}}`, true, false},
		{`{a.b.c.d} == 42`, `{"a": {"b": {"c": {"d": 42}}}}`, true, false},
		{`{a.b} == 99`, `{"a": {"b": 42}}`, false, false},

		// ── Array access ─────────────────────────────────────────────────

		{`{arr[0]} == 1`, `{"arr": [1, 2, 3]}`, true, false},
		{`{arr[1]} == 2`, `{"arr": [1, 2, 3]}`, true, false},
		{`{arr[2]} == 3`, `{"arr": [1, 2, 3]}`, true, false},
		{`{arr[-1]} == 3`, `{"arr": [1, 2, 3]}`, true, false},
		{`{arr[-2]} == 2`, `{"arr": [1, 2, 3]}`, true, false},
		{`{arr[-3]} == 1`, `{"arr": [1, 2, 3]}`, true, false},

		// ── Array of objects ─────────────────────────────────────────────

		{`{data[0].id} == 1`,
			`{"data": [{"id": 1, "name": "Alice"}, {"id": 2, "name": "Bob"}]}`,
			true, false},
		{`{data[1].name} == "Bob"`,
			`{"data": [{"id": 1, "name": "Alice"}, {"id": 2, "name": "Bob"}]}`,
			true, false},
		{`{data[0].name} == "Bob"`,
			`{"data": [{"id": 1, "name": "Alice"}, {"id": 2, "name": "Bob"}]}`,
			false, false},

		// ── Nested arrays (matrix) ───────────────────────────────────────

		{`{matrix[0][0]} == 1`, `{"matrix": [[1, 2], [3, 4]]}`, true, false},
		{`{matrix[0][1]} == 2`, `{"matrix": [[1, 2], [3, 4]]}`, true, false},
		{`{matrix[1][0]} == 3`, `{"matrix": [[1, 2], [3, 4]]}`, true, false},
		{`{matrix[1][1]} == 4`, `{"matrix": [[1, 2], [3, 4]]}`, true, false},

		// ── Mixed nested structures ──────────────────────────────────────

		// Object → array → object → field
		{`{store.employees[0].name} == "Alice"`,
			`{"store": {"employees": [{"name": "Alice", "role": "dev"}, {"name": "Bob", "role": "qa"}]}}`,
			true, false},
		// Object → array → object → field (second element)
		{`{store.employees[1].role} == "qa"`,
			`{"store": {"employees": [{"name": "Alice", "role": "dev"}, {"name": "Bob", "role": "qa"}]}}`,
			true, false},
		// Multiple array levels — access 2nd team's 2nd lead
		{`{org.departments[0].teams[1].leads[1]} == "Carol"`,
			`{"org": {"departments": [{"name": "Eng", "teams": [{"leads": ["Alice"]}, {"leads": ["Bob", "Carol"]}]}]}}`,
			true, false},
		{`{org.departments[0].teams[1].leads[0]} == "Bob"`,
			`{"org": {"departments": [{"name": "Eng", "teams": [{"leads": ["Alice"]}, {"leads": ["Bob", "Carol"]}]}]}}`,
			true, false},

		// ── Boolean in nested object ─────────────────────────────────────

		{`{meta.enabled} == true`,
			`{"meta": {"enabled": true}}`,
			true, false},
		{`{meta.enabled}`, // bare boolean ref works
			`{"meta": {"enabled": true}}`,
			true, false},

		// ── Nested path across various operators ─────────────────────────

		{`{user.age} > 18 AND {user.name} == "Alice"`,
			`{"user": {"name": "Alice", "age": 25}}`,
			true, false},
		{`{user.age} > 18 AND {user.name} == "Bob"`,
			`{"user": {"name": "Alice", "age": 25}}`,
			false, false},
		{`{user.role} IN ["admin", "moderator"]`,
			`{"user": {"role": "admin", "name": "Alice"}}`,
			true, false},
		{`{user.tags} CONTAINS "urgent"`,
			`{"user": {"tags": ["urgent", "billing"]}}`,
			true, false},
		{`{user.tags} CONTAINS "spam"`,
			`{"user": {"tags": ["urgent", "billing"]}}`,
			false, false},
		{`{user.status} =~ /^A/`,
			`{"user": {"status": "Active"}}`,
			true, false},
		{`{user.score} >= 1000`,
			`{"user": {"score": 1500}}`,
			true, false},

		// ── Error cases with JSON ────────────────────────────────────────

		// Root key missing entirely
		{`{missing.key} == true`, `{}`, false, true},
		// Nested key missing
		{`{user.missing} == true`, `{"user": {"name": "Alice"}}`, false, true},
		// Access key on string (not a map)
		{`{user.name.nested} == true`, `{"user": {"name": "Alice"}}`, false, true},
		// Access key on array (should use index)
		{`{users.key} == 1`, `{"users": [1, 2, 3]}`, false, true},
		// Array index out of bounds
		{`{users[999]} == 1`, `{"users": [1, 2, 3]}`, false, true},
		// Larger negative than length
		{`{users[-10]} == 1`, `{"users": [1, 2, 3]}`, false, true},
		// Null value as root
		{`{foo} == true`, `{"foo": null}`, false, true},
		// Null as intermediate value
		{`{user.name} == "Alice"`, `{"user": null}`, false, true},
	})
}

// ---------------------------------------------------------------------------
// Short-circuit evaluation
// ---------------------------------------------------------------------------

func TestShortCircuit(t *testing.T) {
	t.Run("AND short-circuits on false LHS", func(t *testing.T) {
		expr, err := Parse(`{flag} AND {missing}`)
		assert.NoError(t, err)
		r, evalErr := Evaluate(expr, map[string]interface{}{"flag": false})
		assert.NoError(t, evalErr)
		assert.False(t, r)
	})

	t.Run("OR short-circuits on true LHS", func(t *testing.T) {
		expr, err := Parse(`{flag} OR {missing}`)
		assert.NoError(t, err)
		r, evalErr := Evaluate(expr, map[string]interface{}{"flag": true})
		assert.NoError(t, evalErr)
		assert.True(t, r)
	})
}

// ---------------------------------------------------------------------------
// Misc: convenience functions, edge cases
// ---------------------------------------------------------------------------

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

func TestParseConvenience(t *testing.T) {
	expr, err := Parse(`{foo} > 10 AND {bar} == "hello"`)
	assert.NoError(t, err)
	assert.NotNil(t, expr)

	r, err := Evaluate(expr, map[string]interface{}{"foo": 15, "bar": "hello"})
	assert.NoError(t, err)
	assert.True(t, r)

	_, err = Parse(`{foo} == UNQUOTED`)
	assert.Error(t, err)
}

func TestNilExprEvaluation(t *testing.T) {
	_, err := Evaluate(nil, map[string]interface{}{"foo": 1})
	assert.Error(t, err)
}

func TestNilArgsMap(t *testing.T) {
	t.Run("boolean literal", func(t *testing.T) {
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

func TestJSONNumberTypes(t *testing.T) {
	runTestCases(t, []testCase{
		{cond: "{foo} == 123", args: map[string]interface{}{"foo": json.Number("123")}, result: true},
	})
}

func TestUnsupportedOperator(t *testing.T) {
	_, err := applyOperator(Token(999), &BooleanLiteral{Val: true}, &BooleanLiteral{Val: false})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported operator")
}
