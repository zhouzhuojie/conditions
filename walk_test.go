package conditions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
	assert.Contains(t, nodes, `"test"`)
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

// nilVisitor returns nil on Visit, causing Walk to stop after the first node.
type nilVisitor struct{}

func (nilVisitor) Visit(Node) Visitor { return nil }

func TestWalkNilVisitor(t *testing.T) {
	expr, _ := Parse(`{foo} > 1`)
	Walk(nilVisitor{}, expr)
}

func TestWalkParenExpr(t *testing.T) {
	expr, _ := Parse(`({foo} > 1)`)
	var nodes []string
	WalkFunc(expr, func(n Node) {
		nodes = append(nodes, n.String())
	})
	assert.Contains(t, nodes, "foo")
	assert.Contains(t, nodes, "1.000")
}
