# conditions

A fast, embeddable condition evaluator for Go. Parse logical expressions once, evaluate them against data many times.

```go
expr, _ := conditions.Parse(`{age} > 18 AND {status} == "active"`)
ok, _ := conditions.Evaluate(expr, map[string]interface{}{"age": 25, "status": "active"})
// ok == true
```

## Install

```
go get github.com/zhouzhuojie/conditions
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/zhouzhuojie/conditions"
)

func main() {
    expr, err := conditions.Parse(`({foo} > 0.45) AND ({bar} == "ON" OR {baz} IN ["ACTIVE", "CLEAR"])`)
    if err != nil {
        panic(err)
    }

    data := map[string]interface{}{
        "foo": 0.62,
        "bar": "ON",
        "baz": "ACTIVE",
    }

    result, err := conditions.Evaluate(expr, data)
    if err != nil {
        panic(err)
    }

    fmt.Println(result) // true
}
```

Parse once, evaluate many times — the expression is safe to reuse across goroutines:

```go
expr, _ := conditions.Parse(`{price} * {qty} > 1000`)

// Concurrent evaluation
for _, order := range orders {
    go func(o Order) {
        ok, _ := conditions.Evaluate(expr, map[string]interface{}{
            "price": o.Price,
            "qty":   o.Quantity,
        })
        if ok {
            flagForReview(o)
        }
    }(order)
}
```

## Language Syntax

### Literals

| Type | Syntax | Examples |
|------|--------|---------|
| Boolean | `true`, `false` | `true`, `false` |
| Number | digits with optional decimal | `42`, `3.14`, `-100` |
| String | double-quoted | `"hello"`, `"ON"` |
| String array | `[...]` | `["a", "b", "c"]` |
| Number array | `[...]` | `[1, 2, 3]` |

### Variables

Variables are wrapped in `{curly braces}` and resolved from the `args` map at evaluation time.

| Syntax | Resolves To | Args Key |
|--------|-------------|----------|
| `{foo}` | `foo` | `"foo"` |
| `{foo}{bar}` | `foo.bar` | `"foo.bar"` |
| `{foo}{bar}{baz}` | `foo.bar.baz` | `"foo.bar.baz"` |
| `{@prefix}{key}` | `@prefix.key` | `"@prefix.key"` |
| `{my-var}` | `my-var` (hyphens allowed) | `"my-var"` |

```go
expr, _ := conditions.Parse(`{user}{name} == "Alice"`)
ok, _ := conditions.Evaluate(expr, map[string]interface{}{
    "user.name": "Alice",
})
// ok == true
```

### Operators

#### Logical

| Operator | Description | Example |
|----------|-------------|---------|
| `AND` | Both sides must be true | `{a} AND {b}` |
| `OR` | Either side must be true | `{a} OR {b}` |
| `XOR` | Exactly one side true | `{a} XOR {b}` |
| `NAND` | Not both sides true | `{a} NAND {b}` |

All logical operators are case-insensitive (`and`, `AND`, `And` all work).

`AND` and `OR` short-circuit: if the left side determines the result, the right side is not evaluated. This means missing variables on the right side won't cause errors:

```go
expr, _ := conditions.Parse(`{enabled} AND {missing_var}`)

// {enabled} is false → short-circuits, {missing_var} never evaluated
ok, _ := conditions.Evaluate(expr, map[string]interface{}{"enabled": false})
// ok == false, err == nil
```

#### Comparison

| Operator | Description | Example |
|----------|-------------|---------|
| `==` | Equal | `{x} == 10` |
| `!=` | Not equal | `{x} != 0` |
| `>` | Greater than | `{x} > 10` |
| `>=` | Greater or equal | `{x} >= 10` |
| `<` | Less than | `{x} < 100` |
| `<=` | Less or equal | `{x} <= 100` |

Numbers use epsilon-based equality (default `1e-6`) to handle floating-point imprecision:

```go
expr, _ := conditions.Parse(`{value} == 0.1`)
ok, _ := conditions.Evaluate(expr, map[string]interface{}{"value": 0.100000000001})
// ok == true (within epsilon)
```

Change epsilon with `SetDefaultEpsilon`:

```go
conditions.SetDefaultEpsilon(1e-9) // tighter tolerance
```

#### Pattern Matching

| Operator | Description | Example |
|----------|-------------|---------|
| `=~` | Matches regex | `{status} =~ /^5\d\d$/` |
| `!~` | Does not match regex | `{path} !~ /\.json$/` |

Regex patterns can use `/pattern/` syntax or double-quoted strings. Patterns are compiled once and cached:

```go
expr, _ := conditions.Parse(`{email} =~ "^[a-z]+@[a-z]+\\.com$"`)
ok, _ := conditions.Evaluate(expr, map[string]interface{}{"email": "alice@example.com"})
// ok == true
```

#### Membership

| Operator | Description | Example |
|----------|-------------|---------|
| `IN` | Value is in array | `{color} IN ["red", "green", "blue"]` |
| `NOT IN` | Value is not in array | `{color} NOT IN ["banned"]` |
| `CONTAINS` | Array contains value | `{tags} CONTAINS "urgent"` |
| `NOT CONTAINS` | Array does not contain value | `{tags} NOT CONTAINS "spam"` |

`IN` checks if a scalar is in an array. `CONTAINS` checks if an array contains a scalar — it's the reverse of `IN`:

```go
// IN: scalar on the left, array on the right
expr1, _ := conditions.Parse(`{role} IN ["admin", "editor"]`)

// CONTAINS: array on the left, scalar on the right
expr2, _ := conditions.Parse(`{roles} CONTAINS "admin"`)

ok1, _ := conditions.Evaluate(expr1, map[string]interface{}{"role": "admin"})
ok2, _ := conditions.Evaluate(expr2, map[string]interface{}{"roles": []string{"admin", "user"}})
// ok1 == true, ok2 == true
```

String arrays use a hash map internally — `IN` and `CONTAINS` are O(1) regardless of array size:

```go
// 10,000 element array, still fast
items := make([]string, 10000)
for i := range items { items[i] = fmt.Sprintf("item_%d", i) }
expr, _ := conditions.Parse(`{target} IN {items}`)
ok, _ := conditions.Evaluate(expr, map[string]interface{}{
    "target": "item_5000",
    "items":  items,
})
// ok == true, evaluated in ~45ns
```

#### Parentheses

Group expressions to control evaluation order:

```go
expr, _ := conditions.Parse(`({a} > 10 OR {b} > 10) AND {c} == true`)
```

## Supported Types

The `args` map accepts these Go types:

| Go Type | Treated As |
|---------|-----------|
| `int`, `int8`, `int16`, `int32`, `int64` | Number |
| `uint`, `uint8`, `uint16`, `uint32`, `uint64` | Number |
| `float32`, `float64` | Number |
| `json.Number` | Number |
| `string` | String |
| `bool` | Boolean |
| `[]string` | String array |
| `[]int`, `[]int32`, `[]int64` | Number array |
| `[]float32`, `[]float64` | Number array |
| `[]json.Number` | Number array |
| `[]interface{}` | Auto-detected (from JSON) |

## API Reference

### Parse

```go
// Convenience: parse a condition string directly
func Parse(condition string) (Expr, error)

// Or use a parser with a custom reader
func NewParser(r io.Reader) *Parser
func (p *Parser) Parse() (Expr, error)
```

### Evaluate

```go
func Evaluate(expr Expr, args map[string]interface{}) (bool, error)
```

### Epsilon

```go
// Set float comparison tolerance (default: 1e-6)
// Call before any concurrent Evaluate calls.
func SetDefaultEpsilon(ep float64)
```

### Variables

```go
// Extract deduplicated list of variable names from a parsed expression
func Variables(expression Expr) []string
```

```go
expr, _ := conditions.Parse(`{a} > 1 AND {a} < 10 AND {b} == true`)
vars := conditions.Variables(expr)
// vars == ["a", "b"]
```

### AST Inspection

Walk the parsed AST:

```go
expr, _ := conditions.Parse(`{foo} > 1 AND {bar} == "test"`)

conditions.WalkFunc(expr, func(n conditions.Node) {
    if v, ok := n.(*conditions.VarRef); ok {
        fmt.Println("variable:", v.Val)
    }
})
// variable: foo
// variable: bar
```

### Type Inspection

```go
conditions.InspectDataType(42)            // "number"
conditions.InspectDataType("hello")       // "string"
conditions.InspectDataType(true)          // "boolean"
conditions.InspectDataType(struct{}{})    // ""
```

## Performance

Benchmarked on Apple M1 Max (`go test -bench=. -benchmem`):

| Operation | Time | Memory | Allocs |
|-----------|------|---------|--------|
| Simple comparison (`{foo} == "hello"`) | 33 ns/op | 16 B/op | 1 |
| Numeric comparison (`{foo} > 100 AND < 200`) | 60 ns/op | 16 B/op | 2 |
| Boolean operators (`{a} AND {b} OR {c}`) | 57 ns/op | 3 B/op | 3 |
| Regex match (`{status} =~ /^5\d\d/`) | 80 ns/op | 16 B/op | 1 |
| String IN 5-item array | 40 ns/op | 16 B/op | 1 |
| Number IN 10-element array | 38 ns/op | 8 B/op | 1 |
| String IN 10K array (hit) | 41 ns/op | 16 B/op | 1 |
| String IN 10K array (miss) | 40 ns/op | 16 B/op | 1 |
| `CONTAINS` check | 155 ns/op | 288 B/op | 3 |
| `Variables()` extraction | 143 ns/op | 64 B/op | 1 |
| Short-circuit (`false AND ...`) | 6 ns/op | 0 B/op | 0 |
| Full expression parse | 1.1 μs/op | 1896 B/op | 28 |

**Key optimizations:**
- **String array hash map** — `IN`/`CONTAINS` on string arrays is O(1) regardless of array size (10K elements = same ~40ns)
- **Regex caching** — patterns compiled once and reused across evaluations
- **Short-circuit evaluation** — `AND`/`OR` skip the right side when the result is determined
- **Boolean singletons** — no allocations for boolean results in the hot path
- **Optimized `Variables()`** — direct AST walk with map, avoiding intermediate slices (44% faster, 75% less memory)
- **Zero-copy number parsing** — `resolveVar()` uses type switches instead of `reflect`

## Credit

Forked from [oleksandr/conditions](https://github.com/oleksandr/conditions).

Differences from the original:
- Variable syntax: `[foo]` → `{foo}`
- Added `CONTAINS` / `NOT CONTAINS` operators
- Float comparison with configurable epsilon tolerance
- Hash map optimization for long array `IN`/`CONTAINS`
- Removed redundant RWMutex
- Regex pattern caching
- Short-circuit `AND`/`OR` evaluation
- Support for `uint` types and `json.Number`
- `Parse()` convenience function
