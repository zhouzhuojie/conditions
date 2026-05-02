package conditions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTokenString(t *testing.T) {
	assert.Equal(t, "AND", AND.String())
	assert.Equal(t, "OR", OR.String())
	assert.Equal(t, "==", EQ.String())
	assert.Equal(t, "!=", NEQ.String())
	assert.Equal(t, "<", LT.String())
	assert.Equal(t, "<=", LTE.String())
	assert.Equal(t, ">", GT.String())
	assert.Equal(t, ">=", GTE.String())
	assert.Equal(t, "IN", IN.String())
	assert.Equal(t, "NOT IN", NOTIN.String())
	assert.Equal(t, "CONTAINS", CONTAINS.String())
	assert.Equal(t, "NOT CONTAINS", NOTCONTAINS.String())
	assert.Equal(t, "=~", EREG.String())
	assert.Equal(t, "!~", NEREG.String())
	assert.Equal(t, "XOR", XOR.String())
	assert.Equal(t, "NAND", NAND.String())
	assert.Equal(t, "(", LPAREN.String())
	assert.Equal(t, ")", RPAREN.String())
	assert.Equal(t, "", Token(999).String())
}

func TestTokenPrecedence(t *testing.T) {
	assert.Equal(t, 1, OR.Precedence())
	assert.Equal(t, 1, XOR.Precedence())
	assert.Equal(t, 2, AND.Precedence())
	assert.Equal(t, 2, NAND.Precedence())
	assert.Equal(t, 3, EQ.Precedence())
	assert.Equal(t, 3, NEQ.Precedence())
	assert.Equal(t, 3, GT.Precedence())
	assert.Equal(t, 3, GTE.Precedence())
	assert.Equal(t, 3, LT.Precedence())
	assert.Equal(t, 3, LTE.Precedence())
	assert.Equal(t, 3, IN.Precedence())
	assert.Equal(t, 3, NOTIN.Precedence())
	assert.Equal(t, 3, EREG.Precedence())
	assert.Equal(t, 3, NEREG.Precedence())
	assert.Equal(t, 3, CONTAINS.Precedence())
	assert.Equal(t, 3, NOTCONTAINS.Precedence())
	assert.Equal(t, 0, ILLEGAL.Precedence())
}

func TestTokenIsOperator(t *testing.T) {
	assert.True(t, AND.isOperator())
	assert.True(t, OR.isOperator())
	assert.True(t, EQ.isOperator())
	assert.True(t, IN.isOperator())
	assert.True(t, CONTAINS.isOperator())
	assert.False(t, ILLEGAL.isOperator())
	assert.False(t, EOF.isOperator())
	assert.False(t, LPAREN.isOperator())
}
