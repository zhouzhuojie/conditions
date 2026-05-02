# conditions

A fast, embeddable condition evaluator for Go. Parse expressions once, evaluate them repeatedly.

```go
expr, _ := conditions.Parse(`{age} > 18 AND {status} == "active"`)
ok, _ := conditions.Evaluate(expr, map[string]interface{}{"age": 25, "status": "active"})
// ok == true
```

## Install

```bash
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
    // Parse once
    expr, err := conditions.Parse(`({foo} > 0.45) AND ({bar} == "ON" OR {baz} IN ["ACTIVE", "CLEAR"])`)
    if err != nil {
        panic(err)
    }

    // Evaluate many times
    data := map[string]interface{}{
        "foo": 0.62,
        "bar": "ON",
        "baz": "ACTIVE",
    }

    result, err := conditions.Evaluate(expr, data)
    fmt.Println(result) // true
}
```

**Parse once, evaluate many times** — expressions are thread-safe and can be reused across goroutines:

```go
expr, _ := conditions.Parse(`{price} * {qty} > 1000`)

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

## Syntax

### Literals

| Type | Examples |
|------|---------|
| Boolean | `true`, `false` |
| Number | `42`, `3.14`, `-100` |
| String | `"hello"`, `"ON"` |
| String array | `["a", "b", "c"]` |
| Number array | `[1, 2, 3]` |

### Variables

Variables are wrapped in `{curly braces}` and resolved from the `args` map at evaluation time:

| Syntax | Args Key | Resolves To |
|--------|----------|-------------|
| `{foo}` | `"foo"` | `foo` |
| `{foo}{bar}` | `"foo.bar"` | `foo.bar` |
| `{foo}{bar}{baz}` | `"foo.bar.baz"` | `foo.bar.baz` |
| `{@prefix}{key}` | `"@prefix.key"` | `@prefix.key` |
| `{my-var}` | `"my-var"` | `my-var` |

```go
expr, _ := conditions.Parse(`{user}{name} == "Alice"`)
ok, _ := conditions.Evaluate(expr, map[string]interface{}{
    "user.name": "Alice",
})
// ok == true
```

### Operators

**Logical** (case-insensitive: `AND`, `and`, `And` all work):

| Operator | Description | Example |
|----------|-------------|---------|
| `AND` | Both sides true | `{a} AND {b}` |
| `OR` | Either side true | `{a} OR {b}` |
| `XOR` | Exactly one true | `{a} XOR {b}` |
| `NAND` | Not both true | `{a} NAND {b}` |

`AND` and `OR` short-circuit — if the left side determines the result, the right side is never evaluated:

```go
expr, _ := conditions.Parse(`{enabled} AND {missing_var}`)

// {enabled} is false → short-circuits, no error for missing variable
ok, err := conditions.Evaluate(expr, map[string]interface{}{"enabled": false})
// ok == false, err == nil
```

**Comparison:**

| Operator | Description | Example |
|----------|-------------|---------|
| `==` | Equal | `{x} == 10` |
| `!=` | Not equal | `{x} != 0` |
| `>` | Greater than | `{x} > 10` |
| `>=` | Greater or equal | `{x} >= 10` |
| `<` | Less than | `{x} < 100` |
| `<=` | Less or equal | `{x} <= 100` |

Numbers use epsilon-based equality (default `1e-6`) for floating-point tolerance:

```go
conditions.SetDefaultEpsilon(1e-9) // optional: change tolerance
```

**Pattern Matching:**

| Operator | Description | Example |
|----------|-------------|---------|
| `=~` | Matches regex | `{status} =~ /^5\d\d$/` |
| `!~` | Does not match | `{path} !~ /\.json$/` |

Patterns are compiled once and cached automatically.

**Membership:**

| Operator | Description | Example |
|----------|-------------|---------|
| `IN` | Value in array | `{color} IN ["red", "green"]` |
| `NOT IN` | Value not in array | `{color} NOT IN ["banned"]` |
| `CONTAINS` | Array contains value | `{tags} CONTAINS "urgent"` |
| `NOT CONTAINS` | Array lacks value | `{tags} NOT CONTAINS "spam"` |

`IN` and `CONTAINS` are O(1) for string arrays (uses hash map internally).

### Parentheses

Group expressions to control evaluation order:

```go
expr, _ := conditions.Parse(`({a} > 10 OR {b} > 10) AND {c} == true`)
```

## Supported Types

The `args` map accepts these Go types:

| Go Type | Treated As |
|---------|-----------|
| `int`, `int8-64` | Number |
| `uint`, `uint8-64` | Number |
| `float32`, `float64` | Number |
| `json.Number` | Number |
| `string` | String |
| `bool` | Boolean |
| `[]string` | String array |
| `[]int`, `[]int32-64` | Number array |
| `[]float32`, `[]float64` | Number array |
| `[]json.Number` | Number array |
| `[]interface{}` | Auto-detected (from JSON) |

## API

```go
// Parse a condition string
func Parse(condition string) (Expr, error)

// Or use a parser with custom reader
func NewParser(r io.Reader) *Parser
func (p *Parser) Parse() (Expr, error)

// Evaluate a parsed expression
func Evaluate(expr Expr, args map[string]interface{}) (bool, error)

// Set float comparison tolerance (default: 1e-6)
// Call before any concurrent Evaluate calls
func SetDefaultEpsilon(ep float64)

// Extract variable names from a parsed expression
func Variables(expression Expr) []string

// Walk the AST
func WalkFunc(expr Expr, fn func(Node))
```

## Performance

Benchmarked on Apple M1 Max:

| Operation | Time | Memory |
|-----------|------|---------|
| Simple comparison (`{foo} == "hello"`) | 33 ns/op | 16 B/op |
| Numeric comparison (`{foo} > 100 AND < 200`) | 60 ns/op | 16 B/op |
| Boolean operators (`{a} AND {b} OR {c}`) | 57 ns/op | 3 B/op |
| Regex match (`{status} =~ /^5\d\d/`) | 80 ns/op | 16 B/op |
| String IN 5-element array | 40 ns/op | 16 B/op |
| String IN 10K-element array | 41 ns/op | 16 B/op |
| `CONTAINS` check | 155 ns/op | 288 B/op |
| `Variables()` extraction | 143 ns/op | 64 B/op |
| Short-circuit (`false AND ...`) | 6 ns/op | 0 B/op |
| Full expression parse | 1.1 μs/op | 1896 B/op |

**Key optimizations:**
- String array hash map — `IN`/`CONTAINS` is O(1) regardless of array size
- Regex caching — patterns compiled once, reused across evaluations
- Short-circuit evaluation — `AND`/`OR` skip unnecessary work
- Boolean singletons — no allocations for boolean results
- Optimized `Variables()` — direct AST walk (44% faster than original)

## Credit

Forked from [oleksandr/conditions](https://github.com/oleksandr/conditions).

Differences from the original:
- Variable syntax: `[foo]` → `{foo}`
- Added `CONTAINS` / `NOT CONTAINS` operators
- Float comparison with configurable epsilon tolerance
- Hash map optimization for array `IN`/`CONTAINS`
- Removed redundant RWMutex, added regex caching
- Short-circuit `AND`/`OR` evaluation
- Support for `uint` types and `json.Number`
- `Parse()` convenience function
