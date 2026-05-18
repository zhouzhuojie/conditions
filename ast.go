package conditions

import (
	"fmt"
	"strconv"
)

// Node represents a node in the conditions abstract syntax tree.
type Node interface {
	node()
	String() string
}

func (*VarRef) node()             {}
func (*PathRef) node()            {}
func (*NumberLiteral) node()      {}
func (*StringLiteral) node()      {}
func (*BooleanLiteral) node()     {}
func (*BinaryExpr) node()         {}
func (*ParenExpr) node()          {}
func (*SliceStringLiteral) node() {}
func (*SliceNumberLiteral) node() {}

// Expr represents an expression that can be evaluated to a value.
type Expr interface {
	Node
	expr()
	Args() []string
}

func (*VarRef) expr()             {}
func (*PathRef) expr()            {}
func (*NumberLiteral) expr()      {}
func (*StringLiteral) expr()      {}
func (*BooleanLiteral) expr()     {}
func (*BinaryExpr) expr()         {}
func (*ParenExpr) expr()          {}
func (*SliceStringLiteral) expr() {}
func (*SliceNumberLiteral) expr() {}

// VarRef represents a reference to a variable.
type VarRef struct {
	Val string
}

// String returns a string representation of the variable reference.
func (r *VarRef) String() string { return QuoteIdent(r.Val) }

func (r *VarRef) Args() []string {
	return []string{r.Val}
}

// PathStep is one step in a nested path traversal.
// It is either an object property access (.key) or an array index access ([n]).
type PathStep struct {
	IsIndex bool   // true for [n], false for .key
	Key     string // used when IsIndex is false
	Index   int    // used when IsIndex is true
}

// PathRef represents a nested variable reference with path traversal steps.
//
//	{user.name}       → Root: "user", Steps: [{Key: "name"}]
//	{users[0]}        → Root: "users", Steps: [{Index: 0}]
//	{data[0].name}    → Root: "data", Steps: [{Index: 0}, {Key: "name"}]
type PathRef struct {
	Root  string
	Steps []PathStep
}

// String returns a string representation of the path reference.
func (r *PathRef) String() string {
	b := r.Root
	for _, s := range r.Steps {
		if s.IsIndex {
			b += fmt.Sprintf("[%d]", s.Index)
		} else {
			b += "." + s.Key
		}
	}
	return b
}

func (r *PathRef) Args() []string {
	return []string{r.Root}
}

// NumberLiteral represents a numeric literal.
type NumberLiteral struct {
	Val float64
}

// String returns a string representation of the literal.
func (l *NumberLiteral) String() string { return strconv.FormatFloat(l.Val, 'f', 3, 64) }

func (n *NumberLiteral) Args() []string { return nil }

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

func NewSliceStringLiteral(val []string) *SliceStringLiteral {
	ssl := &SliceStringLiteral{}
	ssl.Val = val
	ssl.m = make(map[string]struct{}, len(val))
	for _, item := range ssl.Val {
		ssl.m[item] = struct{}{}
	}
	return ssl
}

func NewSliceNumberLiteral(val []float64) *SliceNumberLiteral {
	return &SliceNumberLiteral{Val: val}
}
