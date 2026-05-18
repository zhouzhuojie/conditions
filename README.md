# conditions

A fast, embeddable condition evaluator for Go. **Parse once, evaluate many times.**

```go
expr, _ := conditions.Parse(`{age} > 18 AND {status} == "active"`)
ok, _ := conditions.Evaluate(expr, map[string]interface{}{"age": 25, "status": "active"})
// ok == true
```

---

## Install

```bash
go get github.com/zhouzhuojie/conditions
```

No external runtime dependencies — only the Go standard library.

---

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/zhouzhuojie/conditions"
)

func main() {
    // Parse once — use {field.subfield} for nested data, {field}{subfield} for flat
    expr, err := conditions.Parse(`{product.price} > 100 AND {product.in_stock}`)
    if err != nil {
        panic(err)
    }

    // Evaluate many times — thread-safe, reuse across goroutines
    // Data can use nested maps (from JSON, API responses, etc.)
    data := map[string]interface{}{
        "product": map[string]interface{}{
            "price":    150.00,
            "in_stock": true,
        },
    }

    result, err := conditions.Evaluate(expr, data)
    fmt.Println(result) // true
}
```

**Parse once, evaluate many times** — the parsed AST is immutable and safe to share:

```go
expr, _ := conditions.Parse(`{price} > 1000`)

for _, order := range orders {
    go func(o Order) {
        ok, _ := conditions.Evaluate(expr, map[string]interface{}{
            "price": o.Price,
        })
        if ok {
            flagForReview(o)
        }
    }(order)
}
```

---

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

Variables reference values from the `args` map using `{curly braces}`.

**Simple reference** — a single brace group maps directly to a map key:

```
{foo}   → args["foo"]
{count} → args["count"]
```

```go
expr, _ := conditions.Parse(`{status} == "active"`)
ok, _ := conditions.Evaluate(expr, map[string]interface{}{"status": "active"})
// ok == true
```

**Composing keys** — consecutive brace groups join with `.` into a single key:

```
{foo}{bar}      → args["foo.bar"]
{foo}{bar}{baz} → args["foo.bar.baz"]
```

This is useful for **namespacing** or grouping related values without nesting your data:

```go
expr, _ := conditions.Parse(
    `{user}{name} == "Alice" AND {user}{age} > 18`,
)
ok, _ := conditions.Evaluate(expr, map[string]interface{}{
    "user.name": "Alice",
    "user.age":  25,
})
// ok == true
```

> **How it works:** At parse time, consecutive `{...}` groups are detected and concatenated into a single dotted key. The evaluation does a flat map lookup — `args["user.name"]`.

**Nested path traversal** — use dots and brackets inside a single brace group to traverse nested maps and arrays:

```
{user.name}       → args["user"]["name"]
{users[0]}        → args["users"][0]
{data[0].name}    → args["data"][0]["name"]
{a.b.c}           → args["a"]["b"]["c"]
{users[-1]}       → args["users"][len-1]   (negative index = from end)
```

This works naturally with JSON data from external APIs:

```go
// Data from json.Unmarshal — nested maps and arrays
data := map[string]interface{}{
    "user": map[string]interface{}{
        "name": "Alice",
        "age":  25,
        "tags": []interface{}{"admin", "billing"},
    },
}

expr, _ := conditions.Parse(
    `{user.name} == "Alice" AND {user.age} > 18`,
)
ok, _ := conditions.Evaluate(expr, data)
// ok == true

// Array access
expr2, _ := conditions.Parse(`{user.tags[0]} == "admin"`)
ok2, _ := conditions.Evaluate(expr2, data)
// ok2 == true
```

> **How it works:** At parse time, dots `.` and brackets `[` inside `{...}` are parsed as path traversal steps. At evaluation time, the resolver walks through nested maps (`map[string]interface{}`) and slices (`[]interface{}`) — exactly what `json.Unmarshal` produces.

**`@` prefix** — brace groups starting with `@` keep the `@` in the key:

```
{@env}{key}  → args["@env.key"]
{@env.name}  → args["@env"]["name"]
```

Useful for namespacing under a well-known prefix (e.g., environment variables).

**Hyphens** — variable names can contain hyphens:

```
{my-var}          → args["my-var"]
{my-var}{sub-key} → args["my-var.sub-key"]
```

#### Two approaches to structured data

`conditions` gives you two ways to work with structured data, and you can mix both in one expression:

| Approach | Syntax | Data shape | Example |
|---|---|---|---|
| **Composed keys** | `{a}{b}` | Flat: `{"a.b": val}` | `{user}{name} == "Alice"` |
| **Path traversal** | `{a.b}` | Nested: `{"a": {"b": val}}` | `{user.name} == "Alice"` |

**Choose based on how your data arrives:**

- **Path traversal** (`{user.name}`) — use when consuming JSON from external APIs. `json.Unmarshal` produces nested `map[string]interface{}` and `[]interface{}` which this syntax traverses directly. No data preprocessing needed.
- **Composed keys** (`{user}{name}`) — use when you control the data shape and want flat, simple maps. Best for configs, feature flags, or when you want to avoid nesting overhead.

**You can even mix both in one expression:**

```go
expr, _ := conditions.Parse(
    `{user}{age} > 18 AND {user.tags} CONTAINS "admin"`,
)
//   ↑ flat key          ↑ nested path
```

### Operators

**Logical** (case-insensitive: `AND`, `and`, `And` all work):

| Operator | Description | Example |
|----------|-------------|---------|
| `AND` | Both sides true | `{a} AND {b}` |
| `OR` | Either side true | `{a} OR {b}` |
| `XOR` | Exactly one true | `{a} XOR {b}` |
| `NAND` | Not both true | `{a} NAND {b}` |

`AND` and `OR` short-circuit — if the left side determines the result, the right side is never evaluated. This is especially useful when the right side would error on missing variables:

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
conditions.SetDefaultEpsilon(1e-9) // optional: change global tolerance
```

**Pattern Matching:**

| Operator | Description | Example |
|----------|-------------|---------|
| `=~` | Matches regex | `{status} =~ /^5\d\d$/` |
| `!~` | Does not match | `{path} !~ /\.json$/` |

Patterns are compiled once and cached automatically (thread-safe).

**Membership:**

| Operator | Description | Example |
|----------|-------------|---------|
| `IN` | Value in array | `{color} IN ["red", "green"]` |
| `NOT IN` | Value not in array | `{color} NOT IN ["banned"]` |
| `CONTAINS` | Array contains value | `{tags} CONTAINS "urgent"` |
| `NOT CONTAINS` | Array lacks value | `{tags} NOT CONTAINS "spam"` |

`IN` and `CONTAINS` are O(1) for string arrays (backed by a hash map).

### Parentheses

Group expressions to control evaluation order:

```go
expr, _ := conditions.Parse(`({a} > 10 OR {b} > 10) AND {c} == true`)
```

Without parentheses, operator precedence is: `OR`/`XOR` < `AND`/`NAND` < comparisons/membership/regex.

---

## Supported Types

The `args` map accepts these Go types:

| Go Type | Treated As |
|---------|-----------|
| `int`, `int8`–`int64` | Number |
| `uint`, `uint8`–`uint64` | Number |
| `float32`, `float64` | Number |
| `json.Number` | Number |
| `string` | String |
| `bool` | Boolean |
| `[]string` | String array |
| `[]int`, `[]int32`–`int64` | Number array |
| `[]float32`, `[]float64` | Number array |
| `[]json.Number` | Number array |
| `[]interface{}` | Auto-detected (from JSON) |

---

## API

```go
// Parse a condition string into an AST expression.
// The result is immutable and thread-safe.
func Parse(condition string) (Expr, error)

// Or use a parser with a custom io.Reader.
func NewParser(r io.Reader) *Parser
func (p *Parser) Parse() (Expr, error)

// Evaluate a parsed expression against a set of arguments.
func Evaluate(expr Expr, args map[string]interface{}) (bool, error)

// Set float comparison tolerance (default: 1e-6).
// Call before any concurrent Evaluate calls.
func SetDefaultEpsilon(ep float64)

// Extract the list of variable names referenced in an expression.
func Variables(expression Expr) []string

// Walk the AST tree with a visitor function.
func WalkFunc(expr Expr, fn func(Node))
```

---

## Performance

Benchmarked on Apple M1 Max:

| Operation | Time | Memory |
|-----------|------|---------|
| Short-circuit (`false AND ...`) | 6 ns/op | 0 B/op |
| Simple comparison (`{foo} == "hello"`) | 33 ns/op | 16 B/op |
| Boolean operators (`{a} AND {b} OR {c}`) | 57 ns/op | 3 B/op |
| Numeric comparison (`{foo} > 100 AND < 200`) | 60 ns/op | 16 B/op |
| Regex match (`{status} =~ /^5\d\d/`) | 80 ns/op | 16 B/op |
| String IN 5-element array | 40 ns/op | 16 B/op |
| String IN 10,000-element array | 41 ns/op | 16 B/op |
| `CONTAINS` check | 155 ns/op | 288 B/op |
| `Variables()` extraction | 143 ns/op | 64 B/op |
| Full expression parse | 1.1 μs/op | 1896 B/op |

**Key optimizations:**
- String array hash map — `IN`/`CONTAINS` is O(1) regardless of array size
- Regex caching — patterns compiled once, reused across evaluations
- Short-circuit evaluation — `AND`/`OR` skip unnecessary work
- Boolean singletons — no allocations for boolean results
- Optimized `Variables()` — direct AST walk (44% faster than original)

---

## Credit

Forked from [oleksandr/conditions](https://github.com/oleksandr/conditions).

Key differences from the original:
- Variable syntax: `[foo]` → `{foo}`
- Key composition: `{user}{name}` → `args["user.name"]` (consecutive brace groups join with `.`)
- Nested path traversal: `{user.name}` → `args["user"]["name"]`, `{users[0]}` → `args["users"][0]`
- Added `CONTAINS` / `NOT CONTAINS` operators
- Float comparison with configurable epsilon tolerance
- Hash map optimization for array `IN` / `CONTAINS`
- Removed redundant RWMutex, added regex caching
- Short-circuit `AND` / `OR` evaluation
- Support for `uint` types and `json.Number`
- `Parse()` convenience function
