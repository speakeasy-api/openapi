// Package expr provides a predicate expression parser and evaluator for the oq query language.
package expr

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Value represents a typed value in the expression system.
type Value struct {
	Kind ValueKind
	Str  string
	Int  int
	Bool bool
	Arr  []string // for KindArray
}

type ValueKind int

const (
	KindString ValueKind = iota
	KindInt
	KindBool
	KindNull
	KindArray
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

type arithmeticExpr struct {
	op    byte // '+', '-', '*', '/'
	left  Expr
	right Expr
}

type alternativeExpr struct {
	left  Expr
	right Expr
}

type ifExpr struct {
	cond  Expr
	then_ Expr
	else_ Expr // nil means return null
}

type interpExpr struct {
	parts []Expr
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

type containsExpr struct {
	left  Expr // evaluates to string or array
	right Expr // evaluates to string (element to find)
}

type funcCallExpr struct {
	name string
	args []Expr
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
		return Value{Kind: KindNull}
	}
}

func (e *arithmeticExpr) Eval(row Row) Value {
	l := toInt(e.left.Eval(row))
	r := toInt(e.right.Eval(row))
	switch e.op {
	case '+':
		return IntVal(l + r)
	case '-':
		return IntVal(l - r)
	case '*':
		return IntVal(l * r)
	case '/':
		if r == 0 {
			return NullVal()
		}
		return IntVal(l / r)
	default:
		return NullVal()
	}
}

func (e *notExpr) Eval(row Row) Value {
	return Value{Kind: KindBool, Bool: !toBool(e.inner.Eval(row))}
}

func (e *hasExpr) Eval(row Row) Value {
	v := row.Field(e.field)
	return Value{Kind: KindBool, Bool: v.Kind != KindNull && (v.Kind != KindInt || v.Int != 0) && (v.Kind != KindBool || v.Bool) && (v.Kind != KindString || v.Str != "") && (v.Kind != KindArray || len(v.Arr) != 0)}
}

func (e *matchesExpr) Eval(row Row) Value {
	v := row.Field(e.field)
	return Value{Kind: KindBool, Bool: v.Kind == KindString && e.pattern.MatchString(v.Str)}
}

func (e *containsExpr) Eval(row Row) Value {
	haystack := e.left.Eval(row)
	needle := e.right.Eval(row)
	needleStr := toString(needle)
	switch haystack.Kind {
	case KindArray:
		for _, item := range haystack.Arr {
			if item == needleStr {
				return BoolVal(true)
			}
		}
		return BoolVal(false)
	case KindString:
		return BoolVal(strings.Contains(haystack.Str, needleStr))
	default:
		return BoolVal(false)
	}
}

func (e *funcCallExpr) Eval(row Row) Value {
	args := make([]Value, len(e.args))
	for i, a := range e.args {
		args[i] = a.Eval(row)
	}
	return evalFunc(e.name, args)
}

func evalFunc(name string, args []Value) Value {
	switch name {
	case "lower":
		if len(args) != 1 {
			return NullVal()
		}
		return StringVal(strings.ToLower(toString(args[0])))
	case "upper":
		if len(args) != 1 {
			return NullVal()
		}
		return StringVal(strings.ToUpper(toString(args[0])))
	case "len":
		if len(args) != 1 {
			return NullVal()
		}
		v := args[0]
		switch v.Kind {
		case KindString:
			return IntVal(len(v.Str))
		case KindArray:
			return IntVal(len(v.Arr))
		default:
			return IntVal(0)
		}
	case "trim":
		if len(args) != 1 {
			return NullVal()
		}
		return StringVal(strings.TrimSpace(toString(args[0])))
	case "startswith":
		if len(args) != 2 {
			return NullVal()
		}
		return BoolVal(strings.HasPrefix(toString(args[0]), toString(args[1])))
	case "endswith":
		if len(args) != 2 {
			return NullVal()
		}
		return BoolVal(strings.HasSuffix(toString(args[0]), toString(args[1])))
	case "contains":
		if len(args) != 2 {
			return NullVal()
		}
		haystack := args[0]
		needleStr := toString(args[1])
		switch haystack.Kind {
		case KindArray:
			for _, item := range haystack.Arr {
				if item == needleStr {
					return BoolVal(true)
				}
			}
			return BoolVal(false)
		default:
			return BoolVal(strings.Contains(toString(haystack), needleStr))
		}
	case "replace":
		if len(args) != 3 {
			return NullVal()
		}
		return StringVal(strings.ReplaceAll(toString(args[0]), toString(args[1]), toString(args[2])))
	case "split":
		if len(args) < 2 || len(args) > 3 {
			return NullVal()
		}
		parts := strings.Split(toString(args[0]), toString(args[1]))
		if len(args) == 3 {
			// split(str, sep, N) → return Nth segment
			idx := toInt(args[2])
			if idx < 0 || idx >= len(parts) {
				return NullVal()
			}
			return StringVal(parts[idx])
		}
		// split(str, sep) → return array
		return ArrayVal(parts)
	case "count":
		if len(args) != 1 {
			return NullVal()
		}
		v := args[0]
		switch v.Kind {
		case KindArray:
			return IntVal(len(v.Arr))
		case KindString:
			return IntVal(len(v.Str))
		default:
			return IntVal(0)
		}
	default:
		return NullVal()
	}
}

func (e *fieldExpr) Eval(row Row) Value {
	return row.Field(e.name)
}

func (e *literalExpr) Eval(_ Row) Value {
	return e.val
}

func (e *alternativeExpr) Eval(row Row) Value {
	l := e.left.Eval(row)
	if l.Kind != KindNull && toBool(l) {
		return l
	}
	return e.right.Eval(row)
}

func (e *ifExpr) Eval(row Row) Value {
	cond := e.cond.Eval(row)
	if toBool(cond) {
		return e.then_.Eval(row)
	}
	if e.else_ != nil {
		return e.else_.Eval(row)
	}
	return Value{Kind: KindNull}
}

func (e *interpExpr) Eval(row Row) Value {
	var sb strings.Builder
	for _, part := range e.parts {
		v := part.Eval(row)
		sb.WriteString(toString(v))
	}
	return StringVal(sb.String())
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
	case KindArray:
		return len(v.Arr) > 0
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
	case KindArray:
		return strings.Join(v.Arr, ", ")
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
	return Value{Kind: KindNull}
}

// ArrayVal creates an array Value.
func ArrayVal(arr []string) Value {
	return Value{Kind: KindArray, Arr: arr}
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

func (p *parser) peekAt(offset int) string {
	idx := p.pos + offset
	if idx >= len(p.tokens) {
		return ""
	}
	return p.tokens[idx]
}

func (p *parser) next() string {
	t := p.peek()
	p.pos++
	return t
}

func (p *parser) expect(tok string) error {
	got := p.next()
	if got != tok {
		return fmt.Errorf("expected %q, got %q", tok, got)
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
	left, err := p.parseAlternative()
	if err != nil {
		return nil, err
	}
	switch p.peek() {
	case "==", "!=", ">", "<", ">=", "<=":
		op := p.next()
		right, err := p.parseAlternative()
		if err != nil {
			return nil, err
		}
		return &binaryExpr{op: op, left: left, right: right}, nil
	case "matches":
		p.next()
		patternTok := p.next()
		pattern := stripQuotes(patternTok)
		re, compileErr := regexp.Compile(pattern)
		if compileErr != nil {
			return nil, fmt.Errorf("invalid regex %q: %w", pattern, compileErr)
		}
		// left must be a field reference
		fieldRef, ok := left.(*fieldExpr)
		if !ok {
			return nil, errors.New("matches requires a field on the left side")
		}
		return &matchesExpr{field: fieldRef.name, pattern: re}, nil
	case "contains":
		p.next()
		right, err := p.parseAlternative()
		if err != nil {
			return nil, err
		}
		return &containsExpr{left: left, right: right}, nil
	}
	return left, nil
}

func (p *parser) parseAlternative() (Expr, error) {
	left, err := p.parseAddSub()
	if err != nil {
		return nil, err
	}
	for p.peek() == "//" {
		p.next()
		right, err := p.parseAddSub()
		if err != nil {
			return nil, err
		}
		left = &alternativeExpr{left: left, right: right}
	}
	return left, nil
}

func (p *parser) parseAddSub() (Expr, error) {
	left, err := p.parseMulDiv()
	if err != nil {
		return nil, err
	}
	for p.peek() == "+" || p.peek() == "-" {
		op := p.next()[0]
		right, err := p.parseMulDiv()
		if err != nil {
			return nil, err
		}
		left = &arithmeticExpr{op: op, left: left, right: right}
	}
	return left, nil
}

func (p *parser) parseMulDiv() (Expr, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	for p.peek() == "*" || p.peek() == "/" {
		op := p.next()[0]
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		left = &arithmeticExpr{op: op, left: left, right: right}
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

// knownFuncs is the set of built-in function names recognized by the parser.
var knownFuncs = map[string]bool{
	"lower": true, "upper": true, "len": true, "trim": true,
	"startswith": true, "endswith": true, "contains": true,
	"replace": true, "split": true, "count": true,
}

func (p *parser) parsePrimary() (Expr, error) {
	tok := p.peek()

	// if-then-else-end
	if tok == "if" {
		return p.parseIf()
	}

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

	// has(field) — special: takes a field name, not an expression
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

	// matches(field, pattern) — special: compiles regex at parse time
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
		pattern := stripQuotes(patternTok)
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regex %q: %w", pattern, err)
		}
		if err := p.expect(")"); err != nil {
			return nil, err
		}
		return &matchesExpr{field: field, pattern: re}, nil
	}

	// Generic function calls: lower(expr), startswith(expr, expr), etc.
	// Only treat as function call if followed by '('
	if knownFuncs[tok] && p.peekAt(1) == "(" {
		name := tok
		p.next()
		p.next() // consume '('
		var args []Expr
		for p.peek() != ")" && p.peek() != "" {
			if len(args) > 0 {
				if err := p.expect(","); err != nil {
					return nil, err
				}
			}
			arg, err := p.parseOr()
			if err != nil {
				return nil, err
			}
			args = append(args, arg)
		}
		if err := p.expect(")"); err != nil {
			return nil, err
		}
		return &funcCallExpr{name: name, args: args}, nil
	}

	// String literal (possibly with interpolation).
	// Tokens prefixed with \x00 originated from single-quoted strings and skip interpolation.
	if strings.HasPrefix(tok, "\x00\"") {
		p.next()
		inner := tok[2 : len(tok)-1] // strip \x00 prefix and quotes
		return &literalExpr{val: StringVal(inner)}, nil
	}
	if strings.HasPrefix(tok, "\"") {
		p.next()
		inner := tok[1 : len(tok)-1] // strip quotes
		if strings.Contains(inner, "\\(") {
			return parseInterpolation(inner)
		}
		return &literalExpr{val: StringVal(inner)}, nil
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

func (p *parser) parseIf() (Expr, error) {
	p.next() // consume "if"
	cond, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	if err := p.expect("then"); err != nil {
		return nil, err
	}
	then_, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	var else_ Expr
	switch p.peek() {
	case "elif":
		// elif chains into a nested ifExpr
		// Rewrite "elif" token as "if" for recursive parsing
		p.tokens[p.pos] = "if"
		else_, err = p.parseIf()
		if err != nil {
			return nil, err
		}
	case "else":
		p.next()
		else_, err = p.parseOr()
		if err != nil {
			return nil, err
		}
		if err := p.expect("end"); err != nil {
			return nil, err
		}
	case "end":
		p.next()
	default:
		return nil, fmt.Errorf("expected \"else\", \"elif\", or \"end\", got %q", p.peek())
	}
	return &ifExpr{cond: cond, then_: then_, else_: else_}, nil
}

func parseInterpolation(s string) (Expr, error) {
	var parts []Expr
	for len(s) > 0 {
		idx := strings.Index(s, "\\(")
		if idx < 0 {
			parts = append(parts, &literalExpr{val: StringVal(s)})
			break
		}
		if idx > 0 {
			parts = append(parts, &literalExpr{val: StringVal(s[:idx])})
		}
		s = s[idx+2:]
		// Find matching closing paren
		depth := 1
		end := 0
		for end < len(s) {
			if s[end] == '(' {
				depth++
			} else if s[end] == ')' {
				depth--
				if depth == 0 {
					break
				}
			}
			end++
		}
		if depth != 0 {
			return nil, errors.New("unterminated interpolation \\(")
		}
		inner := s[:end]
		e, err := Parse(inner)
		if err != nil {
			return nil, fmt.Errorf("interpolation error: %w", err)
		}
		parts = append(parts, e)
		s = s[end+1:]
	}
	if len(parts) == 1 {
		return parts[0], nil
	}
	return &interpExpr{parts: parts}, nil
}

// stripQuotes removes surrounding quotes from a token, handling both the \x00
// sentinel prefix (from single-quoted strings) and regular double-quoted strings.
func stripQuotes(tok string) string {
	if strings.HasPrefix(tok, "\x00\"") {
		return tok[2 : len(tok)-1]
	}
	return strings.Trim(tok, "\"")
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
			if two == "==" || two == "!=" || two == ">=" || two == "<=" || two == "//" {
				tokens = append(tokens, two)
				i += 2
				continue
			}
		}

		// Single-character tokens (including arithmetic operators)
		if ch == '(' || ch == ')' || ch == ',' || ch == '>' || ch == '<' ||
			ch == '+' || ch == '-' || ch == '*' {
			tokens = append(tokens, string(ch))
			i++
			continue
		}

		// '/' alone (not '//') is division
		if ch == '/' {
			tokens = append(tokens, string(ch))
			i++
			continue
		}

		// Quoted string (double or single quotes)
		if ch == '"' || ch == '\'' {
			quote := ch
			j := i + 1
			for j < len(input) && input[j] != quote {
				if input[j] == '\\' && j+1 < len(input) {
					j++
				}
				j++
			}
			if j < len(input) {
				j++
			}
			// Normalize single-quoted strings to double-quoted for downstream parsing.
			// Mark single-quoted origins with a prefix so parsePrimary skips interpolation.
			if quote == '\'' {
				end := j - 1
				if end <= i {
					end = i + 1 // unterminated quote: treat as empty string
				}
				inner := input[i+1 : end]
				tokens = append(tokens, "\x00\""+inner+"\"")
			} else {
				tokens = append(tokens, input[i:j])
			}
			i = j
			continue
		}

		// Word (identifier, keyword, or number)
		j := i
		for j < len(input) && input[j] != ' ' && input[j] != '\t' && input[j] != '\n' &&
			input[j] != '(' && input[j] != ')' && input[j] != ',' &&
			input[j] != '>' && input[j] != '<' && input[j] != '=' && input[j] != '!' &&
			input[j] != '/' && input[j] != '+' && input[j] != '-' && input[j] != '*' {
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
