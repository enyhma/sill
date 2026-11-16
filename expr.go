package sill

import (
	"fmt"
	"strconv"
)

// ── AST ──────────────────────────────────────────────────────────────────

// Expr is a node in a parsed filter/formula expression.
type Expr interface{}

type litExpr struct{ Value any }
type identExpr struct{ Name string }
type memberExpr struct {
	Target Expr
	Name   string
}
type indexExpr struct {
	Target Expr
	Key    Expr
}
type callExpr struct {
	Target Expr // nil for a bare global function call
	Name   string
	Args   []Expr
}
type unaryExpr struct {
	Op string
	X  Expr
}
type binaryExpr struct {
	Op   string
	X, Y Expr
}

// ── Lexer ────────────────────────────────────────────────────────────────

type tokKind int

const (
	tEOF tokKind = iota
	tIdent
	tNumber
	tString
	tPunct // includes operators and ( ) [ ] , .
)

type token struct {
	kind tokKind
	str  string
	num  float64
}

type lexer struct {
	src []rune
	pos int
}

func newLexer(s string) *lexer { return &lexer{src: []rune(s)} }

func (l *lexer) peekByte() rune {
	if l.pos >= len(l.src) {
		return 0
	}
	return l.src[l.pos]
}

func (l *lexer) next() (token, error) {
	for l.pos < len(l.src) && (l.src[l.pos] == ' ' || l.src[l.pos] == '\t' || l.src[l.pos] == '\n') {
		l.pos++
	}
	if l.pos >= len(l.src) {
		return token{kind: tEOF}, nil
	}
	c := l.src[l.pos]

	if c == '"' || c == '\'' {
		quote := c
		start := l.pos + 1
		i := start
		for i < len(l.src) && l.src[i] != quote {
			i++
		}
		if i >= len(l.src) {
			return token{}, fmt.Errorf("unterminated string literal")
		}
		s := string(l.src[start:i])
		l.pos = i + 1
		return token{kind: tString, str: s}, nil
	}

	if c >= '0' && c <= '9' {
		start := l.pos
		for l.pos < len(l.src) && (l.src[l.pos] >= '0' && l.src[l.pos] <= '9') {
			l.pos++
		}
		if l.pos < len(l.src) && l.src[l.pos] == '.' {
			l.pos++
			for l.pos < len(l.src) && (l.src[l.pos] >= '0' && l.src[l.pos] <= '9') {
				l.pos++
			}
		}
		n, err := strconv.ParseFloat(string(l.src[start:l.pos]), 64)
		if err != nil {
			return token{}, err
		}
		return token{kind: tNumber, num: n}, nil
	}

	if isIdentStart(c) {
		start := l.pos
		for l.pos < len(l.src) && isIdentPart(l.src[l.pos]) {
			l.pos++
		}
		return token{kind: tIdent, str: string(l.src[start:l.pos])}, nil
	}

	for _, op := range []string{"==", "!=", ">=", "<=", "&&", "||"} {
		if l.pos+2 <= len(l.src) && string(l.src[l.pos:l.pos+2]) == op {
			l.pos += 2
			return token{kind: tPunct, str: op}, nil
		}
	}
	for _, op := range []string{"(", ")", "[", "]", ",", ".", "+", "-", "*", "/", "%", ">", "<", "!"} {
		if string(c) == op {
			l.pos++
			return token{kind: tPunct, str: op}, nil
		}
	}
	return token{}, fmt.Errorf("unexpected character %q", string(c))
}

func isIdentStart(c rune) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isIdentPart(c rune) bool {
	return isIdentStart(c) || (c >= '0' && c <= '9')
}

// ── Parser ───────────────────────────────────────────────────────────────

type parser struct {
	lex *lexer
	tok token
}

func newParser(s string) (*parser, error) {
	p := &parser{lex: newLexer(s)}
	if err := p.advance(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *parser) advance() error {
	t, err := p.lex.next()
	if err != nil {
		return err
	}
	p.tok = t
	return nil
}

func (p *parser) is(s string) bool { return p.tok.kind == tPunct && p.tok.str == s }

func (p *parser) expect(s string) error {
	if !p.is(s) {
		return fmt.Errorf("expected %q, got %q", s, p.tok.str)
	}
	return p.advance()
}

// parseExpr parses a complete expression string, e.g. a filter statement
// or a formula body.
func parseExpr(s string) (Expr, error) {
	p, err := newParser(s)
	if err != nil {
		return nil, err
	}
	e, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	if p.tok.kind != tEOF {
		return nil, fmt.Errorf("unexpected trailing input at %q", p.tok.str)
	}
	return e, nil
}

func (p *parser) parseOr() (Expr, error) {
	x, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for p.is("||") {
		if err := p.advance(); err != nil {
			return nil, err
		}
		y, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		x = &binaryExpr{Op: "||", X: x, Y: y}
	}
	return x, nil
}

func (p *parser) parseAnd() (Expr, error) {
	x, err := p.parseCmp()
	if err != nil {
		return nil, err
	}
	for p.is("&&") {
		if err := p.advance(); err != nil {
			return nil, err
		}
		y, err := p.parseCmp()
		if err != nil {
			return nil, err
		}
		x = &binaryExpr{Op: "&&", X: x, Y: y}
	}
	return x, nil
}

var cmpOps = map[string]bool{"==": true, "!=": true, ">": true, "<": true, ">=": true, "<=": true}

func (p *parser) parseCmp() (Expr, error) {
	x, err := p.parseAdd()
	if err != nil {
		return nil, err
	}
	for p.tok.kind == tPunct && cmpOps[p.tok.str] {
		op := p.tok.str
		if err := p.advance(); err != nil {
			return nil, err
		}
		y, err := p.parseAdd()
		if err != nil {
			return nil, err
		}
		x = &binaryExpr{Op: op, X: x, Y: y}
	}
	return x, nil
}

func (p *parser) parseAdd() (Expr, error) {
	x, err := p.parseMul()
	if err != nil {
		return nil, err
	}
	for p.is("+") || p.is("-") {
		op := p.tok.str
		if err := p.advance(); err != nil {
			return nil, err
		}
		y, err := p.parseMul()
		if err != nil {
			return nil, err
		}
		x = &binaryExpr{Op: op, X: x, Y: y}
	}
	return x, nil
}

func (p *parser) parseMul() (Expr, error) {
	x, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	for p.is("*") || p.is("/") || p.is("%") {
		op := p.tok.str
		if err := p.advance(); err != nil {
			return nil, err
		}
		y, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		x = &binaryExpr{Op: op, X: x, Y: y}
	}
	return x, nil
}

func (p *parser) parseUnary() (Expr, error) {
	if p.is("!") || p.is("-") {
		op := p.tok.str
		if err := p.advance(); err != nil {
			return nil, err
		}
		x, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &unaryExpr{Op: op, X: x}, nil
	}
	return p.parsePostfix()
}

func (p *parser) parsePostfix() (Expr, error) {
	x, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}
	for {
		switch {
		case p.is("."):
			if err := p.advance(); err != nil {
				return nil, err
			}
			if p.tok.kind != tIdent {
				return nil, fmt.Errorf("expected identifier after '.', got %q", p.tok.str)
			}
			name := p.tok.str
			if err := p.advance(); err != nil {
				return nil, err
			}
			if p.is("(") {
				args, err := p.parseArgs()
				if err != nil {
					return nil, err
				}
				x = &callExpr{Target: x, Name: name, Args: args}
			} else {
				x = &memberExpr{Target: x, Name: name}
			}
		case p.is("["):
			if err := p.advance(); err != nil {
				return nil, err
			}
			key, err := p.parseOr()
			if err != nil {
				return nil, err
			}
			if err := p.expect("]"); err != nil {
				return nil, err
			}
			x = &indexExpr{Target: x, Key: key}
		default:
			return x, nil
		}
	}
}

func (p *parser) parsePrimary() (Expr, error) {
	switch {
	case p.is("("):
		if err := p.advance(); err != nil {
			return nil, err
		}
		x, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if err := p.expect(")"); err != nil {
			return nil, err
		}
		return x, nil
	case p.tok.kind == tNumber:
		v := p.tok.num
		return &litExpr{Value: v}, p.advance()
	case p.tok.kind == tString:
		v := p.tok.str
		return &litExpr{Value: v}, p.advance()
	case p.tok.kind == tIdent:
		switch p.tok.str {
		case "true":
			return &litExpr{Value: true}, p.advance()
		case "false":
			return &litExpr{Value: false}, p.advance()
		case "null":
			return &litExpr{Value: nil}, p.advance()
		}
		name := p.tok.str
		if err := p.advance(); err != nil {
			return nil, err
		}
		if p.is("(") {
			args, err := p.parseArgs()
			if err != nil {
				return nil, err
			}
			return &callExpr{Target: nil, Name: name, Args: args}, nil
		}
		return &identExpr{Name: name}, nil
	default:
		return nil, fmt.Errorf("unexpected token %q", p.tok.str)
	}
}

func (p *parser) parseArgs() ([]Expr, error) {
	if err := p.expect("("); err != nil {
		return nil, err
	}
	var args []Expr
	if p.is(")") {
		return args, p.advance()
	}
	for {
		a, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		args = append(args, a)
		if p.is(",") {
			if err := p.advance(); err != nil {
				return nil, err
			}
			continue
		}
		break
	}
	return args, p.expect(")")
}
