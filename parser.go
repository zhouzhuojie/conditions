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
	// Temporary buffer for path expression segments, set by scanArg when
	// parsing {foo.bar[0]}-style nested references.
	pathBuf []PathStep
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
		varName, segments, err := p.scanArg()
		if err != nil {
			tok = ILLEGAL
		} else if len(segments) > 0 {
			tok = PATH
			tt = varName
			p.pathBuf = segments
		} else {
			tok = IDENT
			tt = varName
		}
	case '[':
		var err error
		_, tt, err = p.scanArray()
		if err == nil {
			tok = ARRAY
		} else {
			tok = ILLEGAL
		}
	case '!':
		t, tt = p.scan()

		switch t {
		case '=':
			tok = NEQ
			tt = "!="
		case '~':
			tok = NEREG
			tt = "!~"
		default:
			tok = ILLEGAL
		}
	case '>':
		t, _ = p.scan()

		if t == '=' {
			tok = GTE
			tt = ">="
		} else {
			tok = GT
			tt = ">"
			p.unscan()
		}
	case '<':
		t, _ = p.scan()

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

		switch t {
		case '=':
			tok = EQ
			tt = "=="
		case '~':
			tok = EREG
			tt = "=~"
		default:
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

		switch ttU {
		case "AND":
			tok = AND
		case "OR":
			tok = OR
		case "XOR":
			tok = XOR
		case "NAND":
			tok = NAND
		case "IN":
			tok = IN
		case "NOT":
			_, tmp := p.scan()
			switch strings.ToUpper(tmp) {
			case "IN":
				tok = NOTIN
				tt = "NOT IN"
			case "CONTAINS":
				tok = NOTCONTAINS
				tt = "NOT CONTAINS"
			default:
				p.unscan()
				tok = ILLEGAL
			}
		case "TRUE":
			tok = TRUE
		case "FALSE":
			tok = FALSE
		case "CONTAINS":
			tok = CONTAINS
		default:
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
	case PATH:
		ref := &PathRef{Root: lit, Steps: p.pathBuf}
		p.pathBuf = nil
		return ref, nil
	case STRING:
		if len(lit) < 2 {
			return nil, fmt.Errorf("invalid string literal: %s", lit)
		}
		// String literals ("..."): properly interpret Go escape sequences
		// so that \" and \\ are correctly unescaped to " and \.
		if lit[0] == '"' && lit[len(lit)-1] == '"' {
			unquoted, err := strconv.Unquote(lit)
			if err != nil {
				// Invalid Go escape sequences (e.g. \d, \.) — fall back
				// to raw content for backward compatibility.
				return &StringLiteral{Val: lit[1 : len(lit)-1]}, nil
			}
			return &StringLiteral{Val: unquoted}, nil
		}
		// Regex literals (/.../): use raw content between delimiters.
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
			return NewSliceNumberLiteral(values), nil
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
// For path expressions like {foo.bar} or {users[0]}, it returns
// the root name and populates p.pathBuf with traversal steps.
func (p *Parser) scanArg() (string, []PathStep, error) {
	var builder strings.Builder
	sep := ""
	first := true

	for {
		t, ttTmp := p.scan()
		if t == scanner.EOF {
			return builder.String(), nil, fmt.Errorf("unexpected EOF in variable reference, missing }")
		}
		builder.WriteString(sep)
		builder.WriteString(ttTmp)

		if t == '@' {
			// @ is a prefix character — keep reading the actual variable name
			continue
		}
		t, _ = p.scan()
		if t == scanner.EOF {
			return builder.String(), nil, fmt.Errorf("unexpected EOF in variable reference, missing }")
		}

		// If this is the first ident and the next char is '.' or '[',
		// parse as a path expression (e.g. {foo.bar}, {users[0]}).
		if first && (t == '.' || t == '[') {
			root := builder.String()
			// The delimiter (. or [) was already consumed by the ahead read.
			// Pass it so scanPathSegments knows which type to expect first.
			segments, err := p.scanPathSegments(t)
			if err != nil {
				return "", nil, err
			}
			return root, segments, nil
		}
		first = false

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
			return builder.String(), nil, nil
		}

		if t != '}' {
			return builder.String(), nil, fmt.Errorf("args error")
		}
	}
}

// scanPathSegments parses path segments starting after the root identifier
// in a path expression. It is called when scanArg detects '.' or '[' after
// the first identifier. The first delimiter ('.' or '[') was already consumed
// by scanArg and passed as `delim`.
//
// It handles chained segments: .key, [index], .key[index].key, etc.
func (p *Parser) scanPathSegments(delim rune) ([]PathStep, error) {
	var segments []PathStep

	// Parse the first segment introduced by delim.
	switch delim {
	case '.':
		tKey, key := p.scan()
		if tKey == scanner.EOF {
			return nil, fmt.Errorf("unexpected EOF in path expression")
		}
		if tKey != scanner.Ident {
			return nil, fmt.Errorf("expected identifier after '.', got %q", key)
		}
		segments = append(segments, PathStep{Key: key})

	case '[':
		idx, err := p.scanIndex()
		if err != nil {
			return nil, err
		}
		segments = append(segments, PathStep{IsIndex: true, Index: idx})
	}

	// Parse any remaining segments.
	for {
		t, tt := p.scan()
		if t == scanner.EOF {
			return nil, fmt.Errorf("unexpected EOF in path expression")
		}

		switch {
		case t == '.':
			tKey, key := p.scan()
			if tKey == scanner.EOF {
				return nil, fmt.Errorf("unexpected EOF in path expression")
			}
			if tKey != scanner.Ident {
				return nil, fmt.Errorf("expected identifier after '.', got %q", key)
			}
			segments = append(segments, PathStep{Key: key})

		case t == '[':
			idx, err := p.scanIndex()
			if err != nil {
				return nil, err
			}
			segments = append(segments, PathStep{IsIndex: true, Index: idx})

		case t == '}':
			return segments, nil

		default:
			return nil, fmt.Errorf("unexpected token in path expression: %q", tt)
		}
	}
}

// scanIndex reads an integer array index inside brackets, e.g. "0" or "-1".
// The opening '[' must already have been consumed.
func (p *Parser) scanIndex() (int, error) {
	var idxStr strings.Builder
	for {
		tIdx, idxPart := p.scan()
		if tIdx == scanner.EOF {
			return 0, fmt.Errorf("unexpected EOF in array index")
		}
		if tIdx == ']' {
			break
		}
		idxStr.WriteString(idxPart)
	}
	if idxStr.Len() == 0 {
		return 0, fmt.Errorf("empty array index")
	}
	idx, err := strconv.Atoi(idxStr.String())
	if err != nil {
		return 0, fmt.Errorf("invalid array index %q", idxStr.String())
	}
	return idx, nil
}


