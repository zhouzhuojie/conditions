package conditions

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/scanner"
)

const maxArrayLen = 65536

// Parser encapsulates the scanner and responsible for returning AST
// composed from statements read from a given reader.
type Parser struct {
	// Text scanner
	s scanner.Scanner
	// Buffer to keep the read forward token
	buf struct {
		tok rune   // last read token
		tt  string // token text
		n   int    // buffer size (max=1)
	}
}

// NewParser returns a new instance of Parser.
func NewParser(r io.Reader) *Parser {
	p := &Parser{s: scanner.Scanner{}}
	p.s.Mode = scanner.ScanIdents | scanner.ScanFloats | scanner.ScanStrings
	p.s.Init(r)
	return p
}

// Parse is a convenience function that parses a condition string into an AST expression.
func Parse(condition string) (Expr, error) {
	p := NewParser(strings.NewReader(condition))
	return p.Parse()
}

// Parse starts scanning & parsing process (main entry point).
// It returns an expression (AST) which you can use for the final evaluation
// of the conditions/statements
func (p *Parser) Parse() (Expr, error) {
	return p.parseExpr()
}

// scan returns the next token from the underlying scanner.
// If a token has been unscanned then read that instead.
func (p *Parser) scan() (rune, string) {
	// If we have a token on the buffer, then return it.
	if p.buf.n != 0 {
		p.buf.n = 0
	} else {
		// Otherwise read and put into buffer in case we 'unscan' it later
		p.buf.tok, p.buf.tt = p.s.Scan(), p.s.TokenText()
	}
	return p.buf.tok, p.buf.tt
}

// scanWithMapping uses scan with buffer (supports 'unscan') and maps
// scanner's tokens to our custom tokens.
func (p *Parser) scanWithMapping() (Token, string) {
	var (
		t   rune
		tok Token
		tt  string
	)

	t, tt = p.scan()

	// Map Go's token to our Token
	switch t {
	case scanner.EOF:
		tok = EOF
	case '(':
		tok = LPAREN
	case ')':
		tok = RPAREN
	case '-':
		t, tt = p.scan()

		if t == scanner.Float || t == scanner.Int {
			tok = NUMBER
			tt = "-" + tt
		} else {
			tok = ILLEGAL
		}
	case scanner.Float, scanner.Int:
		tok = NUMBER
	case '{':
		var err error
		t, tt, err = p.scanArg()
		if err != nil {
			tok = ILLEGAL
		} else {
			tok = IDENT
		}
	case '[':
		var err error
		t, tt, err = p.scanArray()
		if err == nil {
			tok = ARRAY
		} else {
			tok = ILLEGAL
		}
	case '!':
		t, tt = p.scan()

		if t == '=' {
			tok = NEQ
			tt = "!="
		} else if t == '~' {
			tok = NEREG
			tt = "!~"
		} else {
			tok = ILLEGAL
		}
	case '>':
		t, tt = p.scan()

		if t == '=' {
			tok = GTE
			tt = ">="
		} else {
			tok = GT
			tt = ">"
			p.unscan()
		}
	case '<':
		t, tt = p.scan()

		if t == '=' {
			tok = LTE
			tt = "<="
		} else {
			tok = LT
			tt = "<"
			p.unscan()
		}
	case '=':
		t, tt = p.scan()

		if t == '=' {
			tok = EQ
			tt = "=="
		} else if t == '~' {
			tok = EREG
			tt = "=~"
		} else {
			tok = ILLEGAL
		}

	case '/':
		var builder strings.Builder
		builder.WriteString("/")
		for {
			t, ttTmp := p.scan()
			if t == scanner.EOF {
				return ILLEGAL, ""
			}
			builder.WriteString(ttTmp)
			if t == '/' {
				tok = STRING
				break
			}
		}
		tt = builder.String()

	case scanner.String:
		tok = STRING
	case scanner.Ident:
		ttU := strings.ToUpper(tt)

		if ttU == "AND" {
			tok = AND
		} else if ttU == "OR" {
			tok = OR
		} else if ttU == "XOR" {
			tok = XOR
		} else if ttU == "NAND" {
			tok = NAND
		} else if ttU == "IN" {
			tok = IN
		} else if ttU == "NOT" {
			_, tmp := p.scan()
			if strings.ToUpper(tmp) == "IN" {
				tok = NOTIN
				tt = "NOT IN"
			} else if strings.ToUpper(tmp) == "CONTAINS" {
				tok = NOTCONTAINS
				tt = "NOT CONTAINS"
			} else {
				p.unscan()
				tok = ILLEGAL
			}
		} else if ttU == "TRUE" {
			tok = TRUE
		} else if ttU == "FALSE" {
			tok = FALSE
		} else if ttU == "CONTAINS" {
			tok = CONTAINS
		} else {
			tok = ILLEGAL
		}
	}

	return tok, tt
}

// unscan pushes the previously read token back onto the buffer.
func (p *Parser) unscan() {
	p.buf.n = 1
}

// parseExpr is an entry point to parsing
func (p *Parser) parseExpr() (Expr, error) {
	// Parse a non-binary expression type to start.
	// This variable will always be the root of the expression tree.
	expr, err := p.parseUnaryExpr()
	if err != nil {
		return nil, err
	}

	// Loop over operations and unary exprs and build a tree based on precedence.
	for {
		// If the next token is NOT an operator then return the expression.
		op, tx := p.scanWithMapping()
		if op == ILLEGAL {
			return nil, fmt.Errorf("ILLEGAL %s", tx)
		}
		if !op.isOperator() {
			p.unscan()
			return expr, nil
		}

		// Otherwise parse the next unary expression.
		rhs, err := p.parseUnaryExpr()
		if err != nil {
			return nil, err
		}

		// Assign the new root based on the precedence of the LHS and RHS operators.
		if lhs, ok := expr.(*BinaryExpr); ok && lhs.Op.Precedence() <= op.Precedence() {
			expr = &BinaryExpr{
				LHS: lhs.LHS,
				RHS: &BinaryExpr{LHS: lhs.RHS, RHS: rhs, Op: op},
				Op:  lhs.Op,
			}
		} else {
			expr = &BinaryExpr{LHS: expr, RHS: rhs, Op: op}
		}
	}
}

// parseUnaryExpr parses a non-binary expression.
func (p *Parser) parseUnaryExpr() (Expr, error) {
	// If the first token is a LPAREN then parse it as its own grouped expression.
	tok, lit := p.scanWithMapping()
	if tok == LPAREN {
		expr, err := p.parseExpr()
		if err != nil {
			return nil, err
		}

		// Expect an RPAREN at the end.
		if tok, _ := p.scanWithMapping(); tok != RPAREN {
			return nil, fmt.Errorf("missing )")
		}

		return &ParenExpr{Expr: expr}, nil
	}

	// Read next token.
	switch tok {
	case IDENT:
		return &VarRef{Val: lit}, nil
	case STRING:
		if len(lit) < 2 {
			return nil, fmt.Errorf("invalid string literal: %s", lit)
		}
		return &StringLiteral{Val: lit[1 : len(lit)-1]}, nil
	case NUMBER:
		v, err := strconv.ParseFloat(lit, 64)
		if err != nil {
			return nil, fmt.Errorf("unable to parse number: %s", lit)
		}
		return &NumberLiteral{Val: v}, nil
	case TRUE, FALSE:
		return &BooleanLiteral{Val: (tok == TRUE)}, nil
	case ARRAY:
		mapVal := []interface{}{}
		if err := json.Unmarshal([]byte(`[`+lit+`]`), &mapVal); err != nil {
			return nil, fmt.Errorf("unable to parse array: %s", err)
		}
		if len(mapVal) == 0 {
			return nil, fmt.Errorf("empty slice not castable")
		}
		switch t := mapVal[0].(type) {
		case string:
			values := make([]string, 0, len(mapVal))
			for _, v := range mapVal {
				str, ok := v.(string)
				if !ok {
					return nil, fmt.Errorf("the items in the array are not all string")
				}
				values = append(values, str)
			}
			return NewSliceStringLiteral(values), nil
		case float64:
			values := make([]float64, 0, len(mapVal))
			for _, v := range mapVal {
				f, ok := v.(float64)
				if !ok {
					return nil, fmt.Errorf("the items in the array are not all number")
				}
				values = append(values, f)
			}
			return &SliceNumberLiteral{Val: values}, nil
		default:
			return nil, fmt.Errorf("slice of unknown type %T", t)
		}

	default:
		return nil, fmt.Errorf("parsing error: tok=%v, lit=%v", tok, lit)
	}
}

// scanArray reads tokens until ']' and returns the array content as a string.
func (p *Parser) scanArray() (rune, string, error) {
	var builder strings.Builder
	for i := 0; i < maxArrayLen; i++ {
		t, ttTmp := p.scan()
		if t == scanner.EOF {
			return t, "", fmt.Errorf("unexpected EOF in array, missing ]")
		}
		if t == ']' {
			return t, builder.String(), nil
		}
		builder.WriteString(ttTmp)
	}
	return 0, "", fmt.Errorf("parsing error: no ] found in array syntax")
}

// scanArg extracts {variable} to variable,
// {variable}{key1}{key2} to variable.key1.key2,
// handles variable names starting with "@".
func (p *Parser) scanArg() (rune, string, error) {
	var builder strings.Builder
	sep := ""

	for {
		t, ttTmp := p.scan()
		if t == scanner.EOF {
			return t, builder.String(), fmt.Errorf("unexpected EOF in variable reference, missing }")
		}
		builder.WriteString(sep)
		builder.WriteString(ttTmp)
		if t == '@' {
			continue
		}
		t, _ = p.scan()
		if t == scanner.EOF {
			return t, builder.String(), fmt.Errorf("unexpected EOF in variable reference, missing }")
		}
		// Allow variables to contain "-"
		if t == '-' {
			sep = "-"
			continue
		}
		if t == '}' {
			ti, _ := p.scan()
			if ti == '{' {
				sep = "."
				continue
			}
			p.unscan()
			return t, builder.String(), nil
		}

		if t != '}' {
			return t, builder.String(), fmt.Errorf("args error")
		}
	}
}


