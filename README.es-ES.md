
   *(Self-Correction/Note during generation prep)*: I need to make sure the markdown tables are perfectly aligned. I'll reproduce them carefully.
   All code blocks will be copied verbatim.
   The translation will be professional and technically accurate.
   Done. 
   Output matches the response. 
   [Output Generation] -> *Proceeds*
</think>

# condiciones

Un evaluador de condiciones rápido y embebible para Go. **Analiza una vez, evalúa muchas veces.**

```go
expr, _ := conditions.Parse(`{age} > 18 AND {status} == "active"`)
ok, _ := conditions.Evaluate(expr, map[string]interface{}{"age": 25, "status": "active"})
// ok == true
```

---

## Instalar

```bash
go get github.com/zhouzhuojie/conditions
```

Sin dependencias de tiempo de ejecución externas: solo la biblioteca estándar de Go.

---

## Inicio Rápido

```go
package main

import (
    "fmt"
    "github.com/zhouzhuojie/conditions"
)

func main() {
    // Analiza una vez — el AST es inmutable y seguro para reutilizar en múltiplesgoroutines
    expr, err := conditions.Parse(`{product.price} > 100 AND {product.in_stock}`)
    if err != nil {
        panic(err)
    }

    // Evalúa muchas veces — funciona con datos anidados (de JSON, respuestas de API, etc.)
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

### Desde una cadena JSON

```go
import (
    "encoding/json"
    "github.com/zhouzhuojie/conditions"
)

jsonStr := `{"user": {"name": "Alice", "age": 25, "tags": ["admin", "billing"]}}`

var data map[string]interface{}
json.Unmarshal([]byte(jsonStr), &data)

// Los paréntesis alrededor de CONTAINS son necesarios al encadenar después de AND
// debido a la precedencia del analizador.
expr, _ := conditions.Parse(
    `{user.name} == "Alice" AND {user.age} > 18 AND ({user.tags} CONTAINS "admin")`,
)

ok, _ := conditions.Evaluate(expr, data)
// ok == true
```

La sintaxis de ruta (`{user.name}`, `{user.tags}`) coincide con la estructura anidada que produce `json.Unmarshal`: no se necesita transformación de datos.

---

## Sintaxis

### Literales

| Tipo | Ejemplos |
|------|---------|
| Booleano | `true`, `false` |
| Número | `42`, `3.14`, `-100` |
| Cadena | `"hello"`, `"ON"` |
| Array de cadenas | `["a", "b", "c"]` |
| Array de números | `[1, 2, 3]` |

### Variables

Las variables hacen referencia a valores del mapa `args` usando `{llaves}`.

**Referencia simple** — se mapea directamente a una clave de mapa:

```
{foo}   → args["foo"]
```

```go
expr, _ := conditions.Parse(`{status} == "active"`)
ok, _ := conditions.Evaluate(expr, map[string]interface{}{"status": "active"})
// ok == true
```

**Composición de claves** (`{a}{b}`) — los grupos de llaves consecutivos se unen con `.` para formar una sola clave:

```
{foo}{bar}      → args["foo.bar"]
{foo}{bar}{baz} → args["foo.bar.baz"]
```

Útil para **organización de /espacios de nombres** sin anidar tus datos:

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

Los grupos `{...}` consecutivos se unen en tiempo de análisis. El resultado es una búsqueda en un mapa plano: `args["user.name"]`.

**Recorrido de ruta anidada** (`{a.b}`, `{users[0]}`) — usa puntos y corchetes dentro de un solo grupo de llaves:

```
{user.name}       → args["user"]["name"]
{users[0]}        → args["users"][0]
{data[0].name}    → args["data"][0]["name"]
{a.b.c}           → args["a"]["b"]["c"]
{users[-1]}       → args["users"][len-1]   (índice negativo = desde el final)
```

Funciona de manera natural con datos JSON:

```go
data := map[string]interface{}{
    "user": map[string]interface{}{
        "name": "Alice",
        "age":  25,
        "tags": []interface{}{"admin", "billing"},
    },
}

expr, _ := conditions.Parse(`{user.name} == "Alice" AND {user.age} > 18`)
ok, _ := conditions.Evaluate(expr, data)
// ok == true

expr2, _ := conditions.Parse(`{user.tags[0]} == "admin"`)
ok2, _ := conditions.Evaluate(expr2, data)
// ok2 == true
```

Los puntos y corchetes dentro de `{...}` se analizan como pasos de recorrido. El resolvedor recorre `map[string]interface{}` y `[]interface{}`: exactamente lo que produce `json.Unmarshal`.

**Prefijo `@`** — los grupos de llaves que comienzan con `@` mantienen el `@` en la clave (para organización de /espacios de nombres al estilo de variables de entorno):

```
{@env}{key}  → args["@env.key"]
{@env.name}  → args["@env"]["name"]
```

**Guiones** — los nombres de variables pueden contener guiones:

```
{my-var}          → args["my-var"]
{my-var}{sub-key} → args["my-var.sub-key"]
```

#### Dos enfoques para datos estructurados

| Enfoque | Sintaxis | Forma de datos | Ejemplo |
|---|---|---|---|
| **Claves compuestas** | `{a}{b}` | Plano: `{"a.b": val}` | `{user}{name} == "Alice"` |
| **Recorrido de ruta** | `{a.b}` | Anidado: `{"a": {"b": val}}` | `{user.name} == "Alice"` |

- **Recorrido de ruta** (`{user.name}`) — úsalo al consumir JSON. `json.Unmarshal` produce `map[string]interface{}` / `[]interface{}` anidados que el resolvedor recorre directamente. Sin preprocesamiento de datos.
- **Claves compuestas** (`{user}{name}`) — úsalo cuando controlas la forma de los datos. Los mapas planos son más simples y admiten más tipos de Go (`[]int`, `json.Number`, etc.) directamente.

**Combina ambos en una sola expresión** (usa paréntesis al encadenar `AND` con `CONTAINS`/`IN`/`=~`):

```go
expr, _ := conditions.Parse(
    `{user}{age} > 18 AND ({user.tags} CONTAINS "admin")`,
)
//   ↑ clave plana          ↑ ruta anidada
```

### Operadores

**Lógicos** (insensibles a mayúsculas/minúsculas: `AND`, `and`, `And` funcionan todos):

| Operador | Descripción | Ejemplo |
|----------|-------------|---------|
| `AND` | Ambos lados verdaderos | `{a} AND {b}` |
| `OR` | Al menos un lado verdadero | `{a} OR {b}` |
| `XOR` | Exactamente uno verdadero | `{a} XOR {b}` |
| `NAND` | No ambos verdaderos | `{a} NAND {b}` |

`AND` y `OR` tienen evaluación cortocircuitada: si el lado izquierdo determina el resultado, el lado derecho se omite. Esto previene errores por variables faltantes:

```go
expr, _ := conditions.Parse(`{enabled} AND {missing_var}`)

// {enabled} es false → se cortocircuita, sin error por variable faltante
ok, err := conditions.Evaluate(expr, map[string]interface{}{"enabled": false})
// ok == false, err == nil
```

**Comparación:**

| Operador | Descripción | Ejemplo |
|----------|-------------|---------|
| `==` | Igual | `{x} == 10` |
| `!=` | Diferente | `{x} != 0` |
| `>` | Mayor que | `{x} > 10` |
| `>=` | Mayor o igual | `{x} >= 10` |
| `<` | Menor que | `{x} < 100` |
| `<=` | Menor o igual | `{x} <= 100` |

Los números usan igualdad basada en epsilon (por defecto `1e-6`) para tolerancia de punto flotante:

```go
conditions.SetDefaultEpsilon(1e-9) // cambia la tolerancia global
```

**Coincidencia de patrones:**

| Operador | Descripción | Ejemplo |
|----------|-------------|---------|
| `=~` | Coincide con regex | `{status} =~ /^5\d\d$/` |
| `!~` | No coincide | `{path} !~ /\.json$/` |

Los patrones se compilan una vez y se almacenan en caché automáticamente (seguro para subprocesos).

**Pertenencia/Contenido:**

| Operador | Descripción | Ejemplo |
|----------|-------------|---------|
| `IN` | Valor en array | `{color} IN ["red", "green"]` |
| `NOT IN` | Valor no en array | `{color} NOT IN ["banned"]` |
| `CONTAINS` | El array contiene valor | `{tags} CONTAINS "urgent"` |
| `NOT CONTAINS` | El array carece de valor | `{tags} NOT CONTAINS "spam"` |

`IN` y `CONTAINS` son O(1) para arrays de cadenas (mapa hash en el literal del array — en la regla analizada para `IN ["a","b"]`, o construido al vincular `[]string` desde `args` en cada evaluación).

### Paréntesis

Agrupa expresiones para controlar el orden de evaluación:

```go
expr, _ := conditions.Parse(`({a} > 10 OR {b} > 10) AND {c} == true`)
```

Sin paréntesis, la precedencia de operadores es: `OR`/`XOR` < `AND`/`NAND` < comparaciones/pertenencia/expresiones regulares.

---

## Tipos Admitidos

| Tipo Go | Tratado como |
|---------|-----------|
| `int`, `int8`–`int64` | Número |
| `uint`, `uint8`–`uint64` | Número |
| `float32`, `float64` | Número |
| `json.Number` | Número |
| `string` | Cadena |
| `bool` | Booleano |
| `[]string` | Array de cadenas |
| `[]int`, `[]int32`–`int64` | Array de números |
| `[]float32`, `[]float64` | Array de números |
| `[]json.Number` | Array de números |
| `[]interface{}` | Detección automática (desde JSON) |


---

## API

```go
// Analiza una cadena de condición en una expresión AST (inmutable, segura para subprocesos).
func Parse(condition string) (Expr, error)

// O usa un analizador con un io.Reader personalizado.
func NewParser(r io.Reader) *Parser
func (p *Parser) Parse() (Expr, error)

// Evalúa una expresión analizada contra un conjunto de argumentos.
func Evaluate(expr Expr, args map[string]interface{}) (bool, error)

// Establece la tolerancia de comparación de flotantes (por defecto: 1e-6).
// Llama antes de cualquier llamada a Evaluate concurrente.
func SetDefaultEpsilon(ep float64)

// Extrae los nombres de variables referenciados en una expresión.
func Variables(expression Expr) []string

// Recorre el árbol AST con una función visitante.
func WalkFunc(expr Expr, fn func(Node))
```

---

## Rendimiento

Benchmarkeados en **linux/arm64** (analiza una vez por benchmark, luego `Evaluate` en bucle; `go test -benchmem`):

| Operación | Tiempo | Memoria |
|-----------|------|--------|
| Cortocircuito (`false AND ...`) | 7 ns/op | 0 B/op |
| Comparación simple (`{foo} == "hello"`) | 44 ns/op | 16 B/op |
| Operadores booleanos (`{a} AND {b} OR {c}`) | 46 ns/op | 0 B/op |
| Comparación numérica (`{foo} > 100 AND < 200`) | 74 ns/op | 16 B/op |
| Coincidencia regex (`{status} =~ /^5\d\d/`) | 101 ns/op | 16 B/op |
| Cadena IN array de 5 elementos | 48 ns/op | 16 B/op |
| Cadena IN array de 10,000 elementos | 49 ns/op | 16 B/op |
| Verificación `CONTAINS` (`[]string` desde args) | 200 ns/op | 288 B/op |
| Ruta anidada (`{user.name} == "Alice"`) | 50 ns/op | 16 B/op |
| Extracción de `Variables()` | 172 ns/op | 64 B/op |
| Análisis de expresión completa | 1.5 μs/op | 1896 B/op |

**Optimizaciones clave:**
- Mapa hash de arrays de cadenas — `IN`/`CONTAINS` es O(1) independientemente del tamaño del array
- Caché de expresiones regulares — los patrones se compilan una vez, se reutilizan entre evaluaciones
- Evaluación cortocircuitada — `AND`/`OR` omiten trabajo innecesario
- Singleton booleanos — sin asignaciones de memoria para resultados booleanos
- `Variables()` optimizado — recorrido directo del AST (44% más rápido que el original)

---

## Créditos

Forkeado de [oleksandr/conditions](https://github.com/oleksandr/conditions).

Diferencias clave respecto al original:
- Sintaxis de variable: `[foo]` → `{foo}`
- Composición de claves: `{user}{name}` → `args["user.name"]` (los grupos de llaves consecutivos se unen con `.`)
- Recorrido de ruta anidada: `{user.name}` → `args["user"]["name"]`, `{users[0]}` → `args["users"][0]`
- Se añadieron los operadores `CONTAINS` / `NOT CONTAINS`
- Comparación de flotantes con tolerancia de epsilon configurable
- Optimización de mapa hash para `IN` / `CONTAINS` de arrays
- Se eliminó el RWMutex redundante, se añadió caché de regex
- Evaluación cortocircuitada de `AND` / `OR`
- Soporte para tipos `uint` y `json.Number`
- Función de conveniencia `Parse()`
