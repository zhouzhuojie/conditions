package conditions

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

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

func TestVariables(t *testing.T) {
	cond := `{a} > 1 AND {b} == "test" OR {c} < 100 AND {d} in ["x","y","z"]`
	p := NewParser(strings.NewReader(cond))
	expr, _ := p.Parse()

	vars := Variables(expr)
	assert.Equal(t, 4, len(vars))
	assert.Contains(t, vars, "a")
	assert.Contains(t, vars, "b")
	assert.Contains(t, vars, "c")
	assert.Contains(t, vars, "d")
}

func TestVariablesWithComposedKeys(t *testing.T) {
	cond := `{user}{name} == "Alice" AND {user}{age} > 18`
	p := NewParser(strings.NewReader(cond))
	expr, err := p.Parse()
	assert.NoError(t, err)

	vars := Variables(expr)
	assert.Contains(t, vars, "user.name")
	assert.Contains(t, vars, "user.age")
	assert.Equal(t, 2, len(vars))
	assert.NotContains(t, vars, "user")
	assert.NotContains(t, vars, "name")
}

func TestVariablesThreeLevelComposed(t *testing.T) {
	cond := `{a}{b}{c} == 42`
	p := NewParser(strings.NewReader(cond))
	expr, err := p.Parse()
	assert.NoError(t, err)

	vars := Variables(expr)
	assert.Contains(t, vars, "a.b.c")
	assert.Equal(t, 1, len(vars))
}

func TestVariablesHyphenatedComposed(t *testing.T) {
	cond := `{my-var}{sub-key} == "val"`
	p := NewParser(strings.NewReader(cond))
	expr, err := p.Parse()
	assert.NoError(t, err)

	vars := Variables(expr)
	assert.Contains(t, vars, "my-var.sub-key")
	assert.Equal(t, 1, len(vars))
}
