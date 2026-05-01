package conditions

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// DataType represents the primitive data types available in InfluxQL.
type DataType string

const (
	Unknown  = DataType("")
	Number   = DataType("number")
	Boolean  = DataType("boolean")
	String   = DataType("string")
	Time     = DataType("time")
	Duration = DataType("duration")
)

// InspectDataType returns the data type of a given value.
func InspectDataType(v interface{}) DataType {
	switch v.(type) {
	case float64:
		return Number
	case bool:
		return Boolean
	case string:
		return String
	case time.Time:
		return Time
	case time.Duration:
		return Duration
	default:
		return Unknown
	}
}

// Node represents a node in the conditions abstract syntax tree.
type Node interface {
	node()
	String() string
}

func (_ *VarRef) node()             {}
func (_ *NumberLiteral) node()      {}
func (_ *StringLiteral) node()      {}
func (_ *BooleanLiteral) node()     {}
func (_ *BinaryExpr) node()         {}
func (_ *ParenExpr) node()          {}
func (_ *SliceStringLiteral) node() {}
func (_ *SliceNumberLiteral) node() {}

// Expr represents an expression that can be evaluated to a value.
type Expr interface {
	Node
	expr()
	Args() []string
}

func (_ *VarRef) expr()             {}
func (_ *NumberLiteral) expr()      {}
func (_ *StringLiteral) expr()      {}
func (_ *BooleanLiteral) expr()     {}
func (_ *BinaryExpr) expr()         {}
func (_ *ParenExpr) expr()          {}
func (_ *SliceStringLiteral) expr() {}
func (_ *SliceNumberLiteral) expr() {}

// VarRef represents a reference to a variable.
type VarRef struct {
	Val string
}

// String returns a string representation of the variable reference.
func (r *VarRef) String() string { return QuoteIdent(r.Val) }

func (r *VarRef) Args() []string {
	return []string{r.Val}
}

// NumberLiteral represents a numeric literal.
type NumberLiteral struct {
	Val float64
}

// String returns a string representation of the literal.
func (l *NumberLiteral) String() string { return strconv.FormatFloat(l.Val, 'f', 3, 64) }

func (n *NumberLiteral) Args() []string { return nil }

func NewSliceStringLiteral(val []string) *SliceStringLiteral {
	ssl := &SliceStringLiteral{}
	ssl.Val = val
	ssl.m = make(map[string]struct{}, len(val))
	for _, item := range ssl.Val {
		ssl.m[item] = struct{}{}
	}
	return ssl
}

type SliceStringLiteral struct {
	Val []string
	m   map[string]struct{}
}

// String returns a string representation of the literal.
func (l *SliceStringLiteral) String() string {
	return fmt.Sprint(l.Val)
}

func (l *SliceStringLiteral) Args() []string { return nil }

type SliceNumberLiteral struct {
	Val []float64
}

// String returns a string representation of the literal.
func (l *SliceNumberLiteral) String() string {
	return fmt.Sprint(l.Val)
}

func (l *SliceNumberLiteral) Args() []string { return nil }

// BooleanLiteral represents a boolean literal.
type BooleanLiteral struct {
	Val bool
}

// String returns a string representation of the literal.
func (l *BooleanLiteral) String() string {
	if l.Val {
		return "true"
	}
	return "false"
}

func (l *BooleanLiteral) Args() []string { return nil }

// StringLiteral represents a string literal.
type StringLiteral struct {
	Val string
}

// String returns a string representation of the literal.
func (l *StringLiteral) String() string { return Quote(l.Val) }

func (l *StringLiteral) Args() []string { return nil }

// BinaryExpr represents an operation between two expressions.
type BinaryExpr struct {
	Op  Token
	LHS Expr
	RHS Expr
}

// String returns a string representation of the binary expression.
func (e *BinaryExpr) String() string {
	return fmt.Sprintf("%s %s %s", e.LHS.String(), e.Op, e.RHS.String())
}

func (e *BinaryExpr) Args() []string {
	args := e.LHS.Args()
	args = append(args, e.RHS.Args()...)
	return args
}

// ParenExpr represents a parenthesized expression.
type ParenExpr struct {
	Expr Expr
}

// String returns a string representation of the parenthesized expression.
func (e *ParenExpr) String() string { return fmt.Sprintf("(%s)", e.Expr.String()) }

func (p *ParenExpr) Args() []string {
	return p.Expr.Args()
}

// Visitor can be called by Walk to traverse an AST hierarchy.
// The Visit() function is called once per node.
type Visitor interface {
	Visit(Node) Visitor
}

// Walk traverses a node hierarchy in depth-first order.
func Walk(v Visitor, node Node) {
	if v = v.Visit(node); v == nil {
		return
	}

	switch n := node.(type) {
	case *BinaryExpr:
		Walk(v, n.LHS)
		Walk(v, n.RHS)

	case *ParenExpr:
		Walk(v, n.Expr)
	}
}

// WalkFunc traverses a node hierarchy in depth-first order.
func WalkFunc(node Node, fn func(Node)) {
	Walk(walkFuncVisitor(fn), node)
}

type walkFuncVisitor func(Node)

func (fn walkFuncVisitor) Visit(n Node) Visitor { fn(n); return fn }

// quoteIdentRe is pre-compiled to avoid recompilation on every call.
var quoteIdentRe = regexp.MustCompile(`[^a-zA-Z_.]`)

// quoteReplacer is pre-allocated to avoid creating a new Replacer on every Quote call.
var quoteReplacer = strings.NewReplacer("\n", `\n`, `\`, `\\`, `"`, `\"`)

// Quote returns a quoted string.
func Quote(s string) string {
	return `"` + quoteReplacer.Replace(s) + `"`
}

// QuoteIdent returns a quoted identifier if the identifier requires quoting.
// Otherwise returns the original string passed in.
func QuoteIdent(s string) string {
	if s == "" || quoteIdentRe.MatchString(s) {
		return Quote(s)
	}
	return s
}

// FormatDuration formats a duration to a string.
func FormatDuration(d time.Duration) string {
	if d%(7*24*time.Hour) == 0 {
		return fmt.Sprintf("%dw", d/(7*24*time.Hour))
	} else if d%(24*time.Hour) == 0 {
		return fmt.Sprintf("%dd", d/(24*time.Hour))
	} else if d%time.Hour == 0 {
		return fmt.Sprintf("%dh", d/time.Hour)
	} else if d%time.Minute == 0 {
		return fmt.Sprintf("%dm", d/time.Minute)
	} else if d%time.Second == 0 {
		return fmt.Sprintf("%ds", d/time.Second)
	} else if d%time.Millisecond == 0 {
		return fmt.Sprintf("%dms", d/time.Millisecond)
	} else {
		return fmt.Sprintf("%d", d/time.Microsecond)
	}
}
