package util

/*
   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at
   http://www.apache.org/licenses/LICENSE-2.0
   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

import (
	"bytes"
	"errors"
	"fmt"
	"unicode"
)

// BoolExprMaxExpressionDepth is the maximum depth of nested parentheses in a BoolExpr.
// If an expression has more nested parentheses than this, it will fail to parse.
// This is primarily designed to prevent stack overflow from too much recursion, in order to make it safe to parse potentially malicious user input.
const BoolExprMaxExpressionDepth = 20

// EvalBoolExpr returns whether the boolean expression expr is true with given true values vals.
// This is a convenience for ParseBoolExpr.Eval.
// The expr is a boolean expression DSL. For the DSL description, see ParseBoolExpr.
// The vals is a map of symbols in boolExpr which should be evaluated as true. All other symbols in the expression are evaluated as false.
func EvalBoolExpr(expr string, vals map[string]struct{}) (bool, error) {
	boolExpr, err := ParseBoolExpr(expr)
	if err != nil {
		return false, errors.New("parsing: " + err.Error())
	}
	return boolExpr.Eval(vals), nil
}

// ParseBoolExpr takes a string which is a boolean expression, and parses it into an object.
// Returns the parsed BoolExpr, and any parse error.
//
// The xpr is a DSL supporting:
//   - parentheses: `(` and `)`
//   - logical and: `&`
//   - logical or: `|`
//   - operator names
//   - comments: everything after a `#` on a line is ignored.
//   - whitespace: ignored
//   - there is no operator precedence: different operators must be parenthesized.
//     - for example, "FOO & BAR & BAZ" is ok, but "FOO & BAR | BAZ" is invalid, and must be rewritten either "FOO & (BAR | BAZ)" or "(FOO & BAR) | BAZ)".
//
// Example:
//   APE & (CAT | DOG)
//
// Example:
//   APE            # allow apes
//   | BAT          # allow bats
//   | (CAT & !DOG) # allow a cat, but only if a dog doesn't exist.
//
func ParseBoolExpr(xpr string) (BoolExpr, error) {
	expr, newPos, err := parseBoolExpr(xpr, 0, 0)
	if err != nil {
		return nil, err
	}
	if newPos != len(xpr) {
		return nil, fmt.Errorf("malformed expression, read as far as character %v an got confused", newPos) // TODO verify for certain this isn't a security risk to let attackers know.
	}
	return expr, nil
}

func parseBoolExpr(str string, pos int, depth int) (BoolExpr, int, error) {
	depth++
	if depth > BoolExprMaxExpressionDepth {
		// This is for security, to prevent attackers causing an overflow.
		return nil, 0, errors.New("Too many parentheses. The electrons are easily confused.")
	}

	expr := BoolExpr(nil)
	op := boolExprOpTypeNone
	inNot := false
	for {
		if pos >= len(str) {
			if op != boolExprOpTypeNone {
				return nil, 0, errors.New("malformed expression, operator without a following symbol")
			}
			if expr == nil {
				// if the entire expression was empty/whitespace/comments
				expr = boolExprTrue{}
			}
			return expr, pos, nil
		}

		c := str[pos]
		switch {
		case unicode.IsSpace(rune(c)):
			pos++
			continue
		case c == ')':
			if op != boolExprOpTypeNone {
				return nil, 0, errors.New("malformed expression, got closing parenthesis after operator but no final symbol")
			}
			return expr, pos, nil
		case c == '&':
			if op != boolExprOpTypeNone {
				return nil, 0, errors.New("malformed expression, and got multiple operators with no symbol in between")
			}
			if expr == nil {
				return nil, 0, errors.New("malformed expression, got & with no preceding symbol")
			}
			op = boolExprOpTypeAnd
			pos++
		case c == '|':
			if op != boolExprOpTypeNone {
				return nil, 0, errors.New("malformed expression, or got multiple operators with no symbol in between")
			}
			if expr == nil {
				return nil, 0, errors.New("malformed expression, got | with no preceding symbol")
			}
			op = boolExprOpTypeOr
			pos++
		case c == '(':
			if op == boolExprOpTypeNone && expr != nil {
				return nil, 0, errors.New("malformed expression, got opening parenthesis for an expression after another expression, with no operator in-between")
			}
			pos++ // increment past the ( for the recursive eval
			parenExpr, newPos, err := parseBoolExpr(str, pos, depth)
			if err != nil {
				return nil, 0, errors.New("malformed parenthesized expression: " + err.Error())
			}
			if newPos >= len(str) || str[newPos] != ')' {
				return nil, 0, errors.New("malformed expression, got opening parenthesis but no matching closing parenthesis")
			}
			newPos++ // increment past the closing )
			if inNot {
				parenExpr = boolExprNot{ne: parenExpr}
				inNot = false
			}
			parenExpr = boolExprParen{pe: parenExpr}
			newExpr, err := makeBoolExprOpExpr(expr, parenExpr, op)
			if err != nil {
				return nil, 0, errors.New("making expression: " + err.Error())
			}
			expr = newExpr
			op = boolExprOpTypeNone
			pos = newPos
		case c == '!':
			if op == boolExprOpTypeNone && expr != nil {
				return nil, 0, errors.New("malformed expression, got not after a symbol with no operator")
			}
			inNot = !inNot // by flipping it, we allow multiple !! in a row, and just cancel them out.
			pos++
		case c == '#':
			// comment, read and discard everything up to the next newline
			for pos < len(str) && str[pos] != '\n' {
				pos++
			}
			if pos < len(str) {
				pos++ // move one past the newline
			}
		default:
			// if it's not an operator or parenthesis, it must be a symbol
			if op == boolExprOpTypeNone && expr != nil {
				return nil, 0, errors.New("malformed expression, got symbol after an expression, with no operator in-between")
			}
			symExpr, newPos, _ := parseBoolExprSymbol(str, pos)
			// if err != nil {
			// 	return nil, 0, errors.New("malformed symbol: " + err.Error()) // should never happen
			// }
			if inNot {
				symExpr = boolExprNot{ne: symExpr}
				inNot = false
			}
			newExpr, err := makeBoolExprOpExpr(expr, symExpr, op)
			if err != nil {
				return nil, 0, errors.New("making expression: " + err.Error())
			}
			expr = newExpr
			op = boolExprOpTypeNone
			pos = newPos
		}
	}
}

func makeBoolExprOpExpr(expr1 BoolExpr, expr2 BoolExpr, op boolExprOpType) (BoolExpr, error) {
	if op == boolExprOpTypeNone {
		if expr1 != nil {
			return nil, errors.New("got op type 'none', but expr1 is not nil!") // should never happen
		} else if expr2 == nil {
			return nil, errors.New("got op type 'none', but expr2 is nil!") // should never happen
		}
		return expr2, nil
	}
	if expr1 == nil || expr2 == nil {
		return nil, errors.New("got not-none op type, but expr is nil!") // should never happen
	}

	switch op {
	case boolExprOpTypeAnd:
		if _, ok := expr1.(boolExprOr); ok {
			return nil, errors.New("'and' operator after a different operator without parenthesis; ambiguity is not allowed, use parenthesis.")
		}
		return boolExprAnd{andExpr1: expr1, andExpr2: expr2}, nil
	case boolExprOpTypeOr:
		if _, ok := expr1.(boolExprAnd); ok {
			return nil, errors.New("'or' operator after a different operator without parenthesis; ambiguity is not allowed, use parenthesis.")
		}
		return boolExprOr{orExpr1: expr1, orExpr2: expr2}, nil
	}
	return nil, fmt.Errorf("unknown op type %v", op) // should never happen
}

type boolExprOpType int

const (
	boolExprOpTypeNone boolExprOpType = iota
	boolExprOpTypeOr
	boolExprOpTypeAnd
)

func parseBoolExprSymbol(expr string, pos int) (BoolExpr, int, error) {
	buf := bytes.Buffer{}

	if pos >= len(expr) {
		return nil, 0, errors.New("initial position beyond expression length") // should never happen
	}

	if isBoolExprToken(expr[pos]) {
		return nil, 0, errors.New("no symbol found") // should never happen
	}

	for {
		if pos >= len(expr) {
			return boolExprSymbol{symbol: buf.String()}, pos, nil
		}
		if isBoolExprToken(expr[pos]) {
			return boolExprSymbol{symbol: buf.String()}, pos, nil
		}
		_ = buf.WriteByte(expr[pos]) // bytes.Buffer.WriteByte is documented to always return a nil error.
		pos++
	}
}

func isBoolExprToken(r byte) bool {
	return r == '(' || r == ')' || r == '&' || r == '|' || r == '!' || r == '#' || unicode.IsSpace(rune(r))
}

// BoolExpr is an expression.
//
// An expression is either:
// - an expression, a binary operator, and an expression (e.g. "FOO & BAR")
// - a unary operator and an expression (e.g. "!FOO")
// - a variable (e.g. "FOO")
//
// Noting that more than two operators become a binary tree internally.
//
// For example, APE & (BAT | CAT) & DOG  turns into the following BoolExpr tree:
//
//           OP{&}
//           /  \
//          /    \
//  Expr{APE}    OP{&}
//               /   \
//              /     \
//             /       \
//          OP{|}       \
//          /  \         \
//         /    \         \
//  Expr{BAT}   EXPR{CAT}  \
//                          \
//                       EXPR{DOG}
//
//
type BoolExpr interface {
	// Eval evaluates the expression with the given vals, and returns (with the given values as true and all other values in the expression as false) whether the expression is true.
	Eval(vals map[string]struct{}) bool
	// Symbols returns all the symbols in this expression.
	Symbols() map[string]struct{}
}

type boolExprSymbol struct {
	symbol string
}

func (e boolExprSymbol) Eval(vals map[string]struct{}) bool {
	if vals == nil {
		return false
	}
	_, ok := vals[e.symbol]
	return ok
}

func (e boolExprSymbol) Symbols() map[string]struct{} {
	return map[string]struct{}{e.symbol: {}}
}

type boolExprAnd struct {
	andExpr1 BoolExpr
	andExpr2 BoolExpr
}

func (e boolExprAnd) Eval(vals map[string]struct{}) bool {
	return e.andExpr1.Eval(vals) && e.andExpr2.Eval(vals)
}

func (e boolExprAnd) Symbols() map[string]struct{} {
	e1Syms := e.andExpr1.Symbols()
	e2Syms := e.andExpr2.Symbols()
	for sym, _ := range e2Syms {
		e1Syms[sym] = struct{}{}
	}
	return e1Syms
}

type boolExprOr struct {
	orExpr1 BoolExpr
	orExpr2 BoolExpr
}

func (e boolExprOr) Eval(vals map[string]struct{}) bool {
	return e.orExpr1.Eval(vals) || e.orExpr2.Eval(vals)
}

func (e boolExprOr) Symbols() map[string]struct{} {
	e1Syms := e.orExpr1.Symbols()
	e2Syms := e.orExpr2.Symbols()
	for sym, _ := range e2Syms {
		e1Syms[sym] = struct{}{}
	}
	return e1Syms
}

// boolExprParen only serves to provide a wrapper, so we can detect "AndExpr | FOO" as invalid.
// Without this, "(A & B) | C" would be detected as "AndExpr | C". This gives us a different type.
type boolExprParen struct {
	pe BoolExpr
}

func (e boolExprParen) Eval(vals map[string]struct{}) bool {
	return e.pe.Eval(vals)
}

func (e boolExprParen) Symbols() map[string]struct{} {
	return e.pe.Symbols()
}

type boolExprNot struct {
	ne BoolExpr
}

func (e boolExprNot) Eval(vals map[string]struct{}) bool {
	return !e.ne.Eval(vals)
}

func (e boolExprNot) Symbols() map[string]struct{} {
	return e.ne.Symbols()
}

type boolExprTrue struct{}

func (e boolExprTrue) Eval(vals map[string]struct{}) bool { return true }

func (e boolExprTrue) Symbols() map[string]struct{} { return map[string]struct{}{} }
