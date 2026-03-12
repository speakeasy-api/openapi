// Package expr provides a predicate expression parser and evaluator for the oq query language.
package expr

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Value represents a typed value in the expression system.
type Value struct {
	Kind    ValueKind
	Str     string
	Int     int
	Bool    bool
	isNull  bool
}

type ValueKind int

const (
	KindString ValueKind = iota
	KindInt
	KindBool
	KindNull
)

// Row provides field access for predicate evaluation.
type Row interface {
	Field(name string) Value
}

// Expr is the interface for all expression nodes.
type Expr interface {
	Eval(row Row) Value
}

// --- Expression node types ---

type binaryExpr struct {
	op    string
	left  Expr
	right Expr
}

type notExpr struct {
	inner Expr
}

type hasExpr struct {
	field string
}

type matchesExpr struct {
	field   string
	pattern *regexp.Regexp
}

type fieldExpr struct {
	name string
}

type literalExpr struct {
	val Value
}

func (e *binaryExpr) Eval(row Row) Value {
	switch e.op {
	case "and":
		l := toBool(e.left.Eval(row))
		if !l {
			return Value{Kind: KindBool, Bool: false}
		}
		return Value{Kind: KindBool, Bool: toBool(e.right.Eval(row))}
	case "or":
		l := toBool(e.left.Eval(row))
		if l {
			return Value{Kind: KindBool, Bool: true}
		}
		return Value{Kind: KindBool, Bool: toBool(e.right.Eval(row))}
	case "==":
		return Value{Kind: KindBool, Bool: equal(e.left.Eval(row), e.right.Eval(row))}
	case "!=":
		return Value{Kind: KindBool, Bool: !equal(e.left.Eval(row), e.right.Eval(row))}
	case ">":
		return Value{Kind: KindBool, Bool: compare(e.left.Eval(row), e.right.Eval(row)) > 0}
	case "<":
		return Value{Kind: KindBool, Bool: compare(e.left.Eval(row), e.right.Eval(row)) < 0}
	case ">=":
		return Value{Kind: KindBool, Bool: compare(e.left.Eval(row), e.right.Eval(row)) >= 0}
	case "<=":
		return Value{Kind: KindBool, Bool: compare(e.left.Eval(row), e.right.Eval(row)) <= 0}
	default:
		return Value{Kind: KindNull, isNull: true}
	}
}

func (e *notExpr) Eval(row Row) Value {
	return Value{Kind: KindBool, Bool: !toBool(e.inner.Eval(row))}
}

func (e *hasExpr) Eval(row Row) Value {
	v := row.Field(e.field)
	return Value{Kind: KindBool, Bool: !v.isNull && (v.Kind != KindInt || v.Int > 0) && (v.Kind != KindBool || v.Bool)}
}

func (e *matchesExpr) Eval(row Row) Value {
	v := row.Field(e.field)
	return Value{Kind: KindBool, Bool: v.Kind == KindString && e.pattern.MatchString(v.Str)}
}

func (e *fieldExpr) Eval(row Row) Value {
	return row.Field(e.name)
}

func (e *literalExpr) Eval(_ Row) Value {
	return e.val
}

// --- Helpers ---

func toBool(v Value) bool {
	switch v.Kind {
	case KindBool:
		return v.Bool
	case KindInt:
		return v.Int != 0
	case KindString:
		return v.Str != ""
	default:
		return false
	}
}

func equal(a, b Value) bool {
	if a.Kind == KindString || b.Kind == KindString {
		return toString(a) == toString(b)
	}
	if a.Kind == KindInt && b.Kind == KindInt {
		return a.Int == b.Int
	}
	if a.Kind == KindBool && b.Kind == KindBool {
		return a.Bool == b.Bool
	}
	return false
}

func compare(a, b Value) int {
	ai := toInt(a)
	bi := toInt(b)
	if ai < bi {
		return -1
	}
	if ai > bi {
		return 1
	}
	return 0
}

func toInt(v Value) int {
	switch v.Kind {
	case KindInt:
		return v.Int
	case KindBool:
		if v.Bool {
			return 1
		}
		return 0
	case KindString:
		n, _ := strconv.Atoi(v.Str)
		return n
	default:
		return 0
	}
}

func toString(v Value) string {
	switch v.Kind {
	case KindString:
		return v.Str
	case KindInt:
		return strconv.Itoa(v.Int)
	case KindBool:
		return strconv.FormatBool(v.Bool)
	default:
		return ""
	}
}

// StringVal creates a string Value.
func StringVal(s string) Value {
	return Value{Kind: KindString, Str: s}
}

// IntVal creates an int Value.
func IntVal(n int) Value {
	return Value{Kind: KindInt, Int: n}
}

// BoolVal creates a bool Value.
func BoolVal(b bool) Value {
	return Value{Kind: KindBool, Bool: b}
}

// NullVal creates a null Value.
func NullVal() Value {
	return Value{Kind: KindNull, isNull: true}
}

// --- Parser ---

// Parse parses a predicate expression string into an Expr tree.
func Parse(input string) (Expr, error) {
	p := &parser{tokens: tokenize(input)}
	expr, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	if p.pos < len(p.tokens) {
		return nil, fmt.Errorf("unexpected token: %q", p.tokens[p.pos])
	}
	return expr, nil
}

type parser struct {
	tokens []string
	pos    int
}

func (p *parser) peek() string {
	if p.pos >= len(p.tokens) {
		return ""
	}
	return p.tokens[p.pos]
}

func (p *parser) next() string {
	t := p.peek()
	p.pos++
	return t
}

func (p *parser) expect(tok string) error {
	if p.next() != tok {
		return fmt.Errorf("expected %q, got %q", tok, p.tokens[p.pos-1])
	}
	return nil
}

func (p *parser) parseOr() (Expr, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for p.peek() == "or" {
		p.next()
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = &binaryExpr{op: "or", left: left, right: right}
	}
	return left, nil
}

func (p *parser) parseAnd() (Expr, error) {
	left, err := p.parseComparison()
	if err != nil {
		return nil, err
	}
	for p.peek() == "and" {
		p.next()
		right, err := p.parseComparison()
		if err != nil {
			return nil, err
		}
		left = &binaryExpr{op: "and", left: left, right: right}
	}
	return left, nil
}

func (p *parser) parseComparison() (Expr, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	switch p.peek() {
	case "==", "!=", ">", "<", ">=", "<=":
		op := p.next()
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &binaryExpr{op: op, left: left, right: right}, nil
	case "matches":
		p.next()
		patternTok := p.next()
		pattern := strings.Trim(patternTok, "\"")
		re, compileErr := regexp.Compile(pattern)
		if compileErr != nil {
			return nil, fmt.Errorf("invalid regex %q: %w", pattern, compileErr)
		}
		// left must be a field reference
		fieldRef, ok := left.(*fieldExpr)
		if !ok {
			return nil, fmt.Errorf("matches requires a field on the left side")
		}
		return &matchesExpr{field: fieldRef.name, pattern: re}, nil
	}
	return left, nil
}

func (p *parser) parseUnary() (Expr, error) {
	if p.peek() == "not" {
		p.next()
		inner, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &notExpr{inner: inner}, nil
	}
	return p.parsePrimary()
}

func (p *parser) parsePrimary() (Expr, error) {
	tok := p.peek()

	// Parenthesized expression
	if tok == "(" {
		p.next()
		expr, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if err := p.expect(")"); err != nil {
			return nil, err
		}
		return expr, nil
	}

	// Function calls
	if tok == "has" {
		p.next()
		if err := p.expect("("); err != nil {
			return nil, err
		}
		field := p.next()
		if err := p.expect(")"); err != nil {
			return nil, err
		}
		return &hasExpr{field: field}, nil
	}

	if tok == "matches" {
		p.next()
		if err := p.expect("("); err != nil {
			return nil, err
		}
		field := p.next()
		if err := p.expect(","); err != nil {
			return nil, err
		}
		patternTok := p.next()
		pattern := strings.Trim(patternTok, "\"")
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regex %q: %w", pattern, err)
		}
		if err := p.expect(")"); err != nil {
			return nil, err
		}
		return &matchesExpr{field: field, pattern: re}, nil
	}

	// String literal
	if strings.HasPrefix(tok, "\"") {
		p.next()
		return &literalExpr{val: StringVal(strings.Trim(tok, "\""))}, nil
	}

	// Boolean literals
	if tok == "true" {
		p.next()
		return &literalExpr{val: BoolVal(true)}, nil
	}
	if tok == "false" {
		p.next()
		return &literalExpr{val: BoolVal(false)}, nil
	}

	// Integer literal
	if n, err := strconv.Atoi(tok); err == nil {
		p.next()
		return &literalExpr{val: IntVal(n)}, nil
	}

	// Field reference
	if tok != "" && tok != ")" && tok != "," {
		p.next()
		return &fieldExpr{name: tok}, nil
	}

	return nil, fmt.Errorf("unexpected token: %q", tok)
}

// tokenize splits an expression into tokens.
func tokenize(input string) []string {
	var tokens []string
	i := 0
	for i < len(input) {
		ch := input[i]

		// Skip whitespace
		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
			i++
			continue
		}

		// Two-character operators
		if i+1 < len(input) {
			two := input[i : i+2]
			if two == "==" || two == "!=" || two == ">=" || two == "<=" {
				tokens = append(tokens, two)
				i += 2
				continue
			}
		}

		// Single-character tokens
		if ch == '(' || ch == ')' || ch == ',' || ch == '>' || ch == '<' {
			tokens = append(tokens, string(ch))
			i++
			continue
		}

		// Quoted string
		if ch == '"' {
			j := i + 1
			for j < len(input) && input[j] != '"' {
				if input[j] == '\\' {
					j++
				}
				j++
			}
			if j < len(input) {
				j++
			}
			tokens = append(tokens, input[i:j])
			i = j
			continue
		}

		// Word (identifier, keyword, or number)
		j := i
		for j < len(input) && input[j] != ' ' && input[j] != '\t' && input[j] != '\n' &&
			input[j] != '(' && input[j] != ')' && input[j] != ',' &&
			input[j] != '>' && input[j] != '<' && input[j] != '=' && input[j] != '!' {
			j++
		}
		if j > i {
			tokens = append(tokens, input[i:j])
			i = j
		} else {
			i++
		}
	}
	return tokens
}
