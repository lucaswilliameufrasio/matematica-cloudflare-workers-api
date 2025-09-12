package eval

import (
	"errors"
	"math"
	"strconv"
	"strings"
	"unicode"
)

var (
	ErrSyntax          = errors.New("invalid expression syntax")
	ErrDivideByZero    = errors.New("division by zero")
	ErrInvalidFunction = errors.New("invalid function usage")
)

// Evaluate parses and evaluates the supported expression subset and returns a decimal string.
func Evaluate(input string) (string, error) {
	t := newTokenizer(input)
	p := parser{tz: t}
	val, err := p.parseExpr()
	if err != nil {
		return "", err
	}
	if p.tz.peek().kind != tokEOF {
		return "", ErrSyntax
	}
	return format(val), nil
}

// ---- Lexer ----

type tokenKind int

const (
	tokEOF tokenKind = iota
	tokNum
	tokPlus
	tokMinus
	tokStar
	tokSlash
	tokLPar
	tokRPar
	tokIdent
)

type token struct {
	kind tokenKind
	lit  string
}

type tokenizer struct {
	s   string
	i   int
	cur token
}

func newTokenizer(s string) *tokenizer {
	tz := &tokenizer{s: strings.TrimSpace(s)}
	tz.next()
	return tz
}

func (t *tokenizer) next() token {
	t.cur = t.scan()
	return t.cur
}

func (t *tokenizer) peek() token { return t.cur }

func (t *tokenizer) scan() token {
	s := t.s
	n := len(s)
	// skip spaces
	for t.i < n && unicode.IsSpace(rune(s[t.i])) {
		t.i++
	}
	if t.i >= n {
		return token{kind: tokEOF}
	}
	ch := s[t.i]
	switch ch {
	case '+':
		t.i++
		return token{kind: tokPlus, lit: "+"}
	case '-':
		t.i++
		return token{kind: tokMinus, lit: "-"}
	case '*':
		t.i++
		return token{kind: tokStar, lit: "*"}
	case '/':
		t.i++
		return token{kind: tokSlash, lit: "/"}
	case '(':
		t.i++
		return token{kind: tokLPar, lit: "("}
	case ')':
		t.i++
		return token{kind: tokRPar, lit: ")"}
	}
	// number: [0-9]* ('.' [0-9]+)? | [0-9]+ ('.' [0-9]*)?
	if ch == '.' || (ch >= '0' && ch <= '9') {
		start := t.i
		if ch == '.' {
			t.i++
			for t.i < n && isDigit(s[t.i]) {
				t.i++
			}
		} else {
			for t.i < n && isDigit(s[t.i]) {
				t.i++
			}
			if t.i < n && s[t.i] == '.' {
				t.i++
				for t.i < n && isDigit(s[t.i]) {
					t.i++
				}
			}
		}
		return token{kind: tokNum, lit: s[start:t.i]}
	}
	// ident
	if isAlpha(ch) {
		start := t.i
		for t.i < n && (isAlpha(s[t.i]) || isDigit(s[t.i])) {
			t.i++
		}
		return token{kind: tokIdent, lit: s[start:t.i]}
	}
	return token{kind: tokEOF}
}

func isDigit(b byte) bool { return b >= '0' && b <= '9' }
func isAlpha(b byte) bool { return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') }

// ---- Pratt Parser ----

type parser struct{ tz *tokenizer }

// expr := sum
func (p *parser) parseExpr() (float64, error) { return p.parseSum() }

// sum := prod (('+' | '-') prod)*
func (p *parser) parseSum() (float64, error) {
	left, err := p.parseProd()
	if err != nil {
		return 0, err
	}
	for {
		tok := p.tz.peek()
		switch tok.kind {
		case tokPlus:
			p.tz.next()
			r, err := p.parseProd()
			if err != nil {
				return 0, err
			}
			left += r
		case tokMinus:
			p.tz.next()
			r, err := p.parseProd()
			if err != nil {
				return 0, err
			}
			left -= r
		default:
			return left, nil
		}
	}
}

// prod := unary (('*' | '/') unary)*
func (p *parser) parseProd() (float64, error) {
	left, err := p.parseUnary()
	if err != nil {
		return 0, err
	}
	for {
		tok := p.tz.peek()
		switch tok.kind {
		case tokStar:
			p.tz.next()
			r, err := p.parseUnary()
			if err != nil {
				return 0, err
			}
			left *= r
		case tokSlash:
			p.tz.next()
			r, err := p.parseUnary()
			if err != nil {
				return 0, err
			}
			if r == 0 {
				return 0, ErrDivideByZero
			}
			left /= r
		default:
			return left, nil
		}
	}
}

// unary := ('-' unary) | primary | funcall
func (p *parser) parseUnary() (float64, error) {
	tok := p.tz.peek()
	if tok.kind == tokMinus {
		p.tz.next()
		v, err := p.parseUnary()
		if err != nil {
			return 0, err
		}
		return -v, nil
	}
	if tok.kind == tokIdent {
		return p.parseFuncall()
	}
	return p.parsePrimary()
}

// funcall := ident '(' expr ')'
func (p *parser) parseFuncall() (float64, error) {
	name := p.tz.peek().lit
	p.tz.next() // ident
	if p.tz.peek().kind != tokLPar {
		return 0, ErrInvalidFunction
	}
	p.tz.next() // '('
	arg, err := p.parseExpr()
	if err != nil {
		return 0, err
	}
	if p.tz.peek().kind != tokRPar {
		return 0, ErrSyntax
	}
	p.tz.next() // ')'
	switch strings.ToLower(name) {
	case "sqrt":
		if arg < 0 {
			return math.NaN(), nil
		}
		return math.Sqrt(arg), nil
	default:
		return 0, ErrInvalidFunction
	}
}

// primary := number | '(' expr ')'
func (p *parser) parsePrimary() (float64, error) {
	tok := p.tz.peek()
	switch tok.kind {
	case tokNum:
		p.tz.next()
		// parse decimal
		if strings.Count(tok.lit, ".") > 1 || tok.lit == "." {
			return 0, ErrSyntax
		}
		v, err := strconv.ParseFloat(tok.lit, 64)
		if err != nil {
			return 0, ErrSyntax
		}
		return v, nil
	case tokLPar:
		p.tz.next()
		v, err := p.parseExpr()
		if err != nil {
			return 0, err
		}
		if p.tz.peek().kind != tokRPar {
			return 0, ErrSyntax
		}
		p.tz.next()
		return v, nil
	default:
		return 0, ErrSyntax
	}
}

func format(v float64) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "nan"
	}
	// Format with up to 10 decimals, trim trailing zeros and dot
	s := strconv.FormatFloat(v, 'f', 10, 64)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	if s == "-0" {
		s = "0"
	}
	return s
}
