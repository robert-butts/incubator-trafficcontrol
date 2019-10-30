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
	"testing"
)

func TestEvalBoolExpr(t *testing.T) {
	type Input struct {
		Expr     string
		Vals     []string
		Expected bool
	}

	inputExpecteds := []Input{
		Input{"", []string{"FOO"}, true},
		Input{"FOO", []string{"FOO"}, true},
		Input{"BAR", []string{"FOO"}, false},
		Input{"FOO | BAR", []string{"FOO"}, true},
		Input{"FOO & BAR", []string{"FOO"}, false},
		Input{"(FOO & BAR) | BAZ", []string{"BLEE", "BAZ"}, true},
		Input{"(FOO | BAR) & BAZ", []string{"BLEE", "BAZ"}, false},
		Input{"(FOO | BAR) & BAZ", []string{"FOO", "BLEE", "BAZ"}, true},
		Input{"(FOO)", []string{"FOO"}, true},
		Input{"(FOO)", []string{"BAR"}, false},
		Input{"(FOO & BAR)", []string{"BAR"}, false},
		Input{"(FOO & BAR)", []string{"BAR", "FOO"}, true},
		Input{"FOO | (FOO & (FOO & ( FOO | (BAR & (((AB | BC) & CD)))))) | (BAZ)", []string{"CAT", "BAR"}, false},
		Input{"!(FOO & BAR)", []string{"BAR", "FOO"}, false},
	}
	for _, ie := range inputExpecteds {
		vals := map[string]struct{}{}
		for _, val := range ie.Vals {
			vals[val] = struct{}{}
		}
		actual, err := EvalBoolExpr(ie.Expr, vals)
		if err != nil {
			t.Errorf("error expected nil, actual: %v", err)
			continue
		}

		if ie.Expected != actual {
			t.Errorf("expression '%v' values '%v' expected: %v, actual: %v", ie.Expr, ie.Vals, ie.Expected, actual)
		}
	}
}

// TestEvalBoolExprOverflow verifies an error without a panic, if there are too many parens.
// This is essential to prevent malicious attackers from causing fatal errors.
func TestEvalBoolExprOverflow(t *testing.T) {
	makeOverflowExpr := func(num int) string {
		str := "(A | "
		closing := ")"
		for i := 0; i < num; i++ {
			str += "(A | "
			closing += ")"
		}
		str += "A" + closing
		return str
	}

	overflowExpr := makeOverflowExpr(BoolExprMaxExpressionDepth - 1)
	vals := map[string]struct{}{"B": {}}
	// We're as much testing for a panic or infinite loop here, as for the error.
	if _, err := EvalBoolExpr(overflowExpr, vals); err == nil {
		t.Errorf("error expected, actual: nil")
	}

	almostOverflowExpr := makeOverflowExpr(BoolExprMaxExpressionDepth - 2)
	if ok, err := EvalBoolExpr(almostOverflowExpr, vals); err != nil {
		t.Errorf("error expected nil, actual: %v", err)
	} else if ok {
		t.Errorf("expected: false, actual: true")
	}

	hasVals := map[string]struct{}{"A": {}}
	if ok, err := EvalBoolExpr(almostOverflowExpr, hasVals); err != nil {
		t.Errorf("error expected nil, actual: %v", err)
	} else if !ok {
		t.Errorf("expected: true, actual: false")
	}
}

func TestEvalBoolExprNot(t *testing.T) {
	type Input struct {
		Expr     string
		Vals     []string
		Expected bool
	}

	inputExpecteds := []Input{
		Input{"FOO", []string{"FOO"}, true},
		Input{"!FOO", []string{"FOO"}, false},
		Input{"!FOO", []string{"BAR"}, true},
		Input{"!FOO | BAR", []string{"BAR"}, true},
		Input{"!FOO | !BAR", []string{"BAR"}, true},
		Input{"!FOO | !BAR", []string{"BAR", "FOO"}, false},
		Input{"!!!!!FOO", []string{"BAR"}, true},
		Input{"!!!!FOO", []string{"BAR"}, false},
		Input{"!!!!!FOO", []string{"FOO"}, false},
		Input{"!!!!FOO", []string{"FOO"}, true},
	}
	for _, ie := range inputExpecteds {
		vals := map[string]struct{}{}
		for _, val := range ie.Vals {
			vals[val] = struct{}{}
		}
		actual, err := EvalBoolExpr(ie.Expr, vals)
		if err != nil {
			t.Errorf("error expected nil, actual: %v", err)
			continue
		}
		if ie.Expected != actual {
			t.Errorf("expression '%v' valabilities '%v' expected: %v, actual: %v", ie.Expr, ie.Vals, ie.Expected, actual)
		}
	}
}

func TestEvalBoolExprBadExprs(t *testing.T) {
	inputs := []string{
		"FOO!",
		"FOO!BAR",
		"FOO&&BAR",
		"FOO||BAR",
		"FOO | | BAR",
		"FOO || BAR",
		"FOO   |  |    BAR",
		"FOO | & BAR",
		"FOO & | BAR",
		"FOO &&& BAR",
		"FOO !& BAR",
		"FOO & BAR | BAZ",
		"FOO | BAR & BAZ",
		"FOO & (BAZ",
		"FOO & BAZ)",
		"FOO & (&BAZ)",
		"FOO & (BAZ&)",
		"FOO & (((BAZ))",
		"FOO & ((BAZ)))",
		"FOO & (!&((BAZ)))",
		"FOO &",
		"FOO |",
		"FOO & BAR & BAZ& ",
		"FOO & BAR & BAZ  & ",
		"&FOO & BAR & BAZ& ",
		"  & FOO & BAR & BAZ  & ",
		"  &FOO & BAR & BAZ  & ",
		"|FOO | BAR | BAZ| ",
		"  | FOO | BAR | BAZ  | ",
		"  |FOO | BAR | BAZ  | ",
		"FOO (BAR)",
		"(BAR) FOO",
		"FOO & BAR | (BAZ)",
	}

	valsArr := []string{"FOO"}
	vals := map[string]struct{}{}
	for _, val := range valsArr {
		vals[val] = struct{}{}
	}

	for _, expr := range inputs {
		if _, err := EvalBoolExpr(expr, vals); err == nil {
			t.Errorf("error expected not nil, actual: nil")
		}
	}
}

// TestMakeBoolExprOpExprInvalidInput tests invalid input for MakeOpExpr, which should never be passed by the larger funcs.
//
// DO NOT add tests here for valid input. If input can ever happen from EvalBoolExpr or ParseRequiredValabilityExpression, it should be tested there. Passing valid input directly to MakeOpExpr circumvents 'code coverage' numbers, and avoids making the call though the outer function show up as missing 'coverage.'
//
// This func serves only to increase 'code coverage' and test extra safeties which can't be tested by the outer funcs.
//
func TestMakeBoolExprOpExprInvalidInput(t *testing.T) {
	expr1 := boolExprSymbol{"FOO"}
	expr2 := boolExprSymbol{"BAR"}
	if _, err := makeBoolExprOpExpr(expr1, expr2, boolExprOpType(999)); err == nil {
		t.Errorf("invalid op type expected err, actual nil")
	}
	if _, err := makeBoolExprOpExpr(expr1, expr2, boolExprOpTypeNone); err == nil {
		t.Errorf("op type none with not-nil expr1 expected err, actual nil")
	}
	if _, err := makeBoolExprOpExpr(expr1, nil, boolExprOpTypeNone); err == nil {
		t.Errorf("op type none with nil expr2 expected err, actual nil")
	}
	if _, err := makeBoolExprOpExpr(nil, nil, boolExprOpTypeNone); err == nil {
		t.Errorf("op type none with nil expr2 (and expr1) expected err, actual nil")
	}

	if _, err := makeBoolExprOpExpr(nil, expr2, boolExprOpTypeAnd); err == nil {
		t.Errorf("op type and with nil expr expected err, actual nil")
	}
	if _, err := makeBoolExprOpExpr(expr1, nil, boolExprOpTypeAnd); err == nil {
		t.Errorf("op type and with nil expr expected err, actual nil")
	}
	if _, err := makeBoolExprOpExpr(nil, nil, boolExprOpTypeAnd); err == nil {
		t.Errorf("op type and with nil expr expected err, actual nil")
	}

	if _, err := makeBoolExprOpExpr(nil, expr2, boolExprOpTypeOr); err == nil {
		t.Errorf("op type or with nil expr expected err, actual nil")
	}
	if _, err := makeBoolExprOpExpr(expr1, nil, boolExprOpTypeOr); err == nil {
		t.Errorf("op type or with nil expr expected err, actual nil")
	}
	if _, err := makeBoolExprOpExpr(nil, nil, boolExprOpTypeOr); err == nil {
		t.Errorf("op type or with nil expr expected err, actual nil")
	}
}

// TestParseBoolExprSymbolInvalidInput tests invalid input for ParseRequiredValabilityExprSymbol, which should never be passed by the larger funcs.
//
// DO NOT add tests here for valid input. If input can ever happen from EvalBoolExpr or ParseRequiredValabilityExpression, it should be tested there. Passing valid input directly to ParseRequiredValabilityExprSymbol circumvents 'code coverage' numbers, and avoids making the call though the outer function show up as missing 'coverage.'
//
// This func serves only to increase 'code coverage' and test extra safeties which can't be tested by the outer funcs.
//
func TestParseBoolExprSymbolInvalidInput(t *testing.T) {
	if _, _, err := parseBoolExprSymbol("foo & bar", 9); err == nil {
		t.Errorf("pos beyond length of expr expected err, actual nil")
	}
	if _, _, err := parseBoolExprSymbol("foo & bar", 4); err == nil {
		t.Errorf("pos starting at operator expected err, actual nil")
	}
}

func TestEvalBoolExprComments(t *testing.T) {
	type Input struct {
		Expr     string
		Vals     []string
		Expected bool
	}

	// TODO test empty input

	inputExpecteds := []Input{
		Input{"FOO # ignore me whatever !&&||!|!", []string{"FOO"}, true},
		Input{"FOO # ignore me whatever BAR !&&||!|!", []string{"FOO"}, true},
		Input{"#FOO", []string{"BAR"}, true},
		Input{"FOO#", []string{"FOO"}, true},
		Input{"FOO #", []string{"FOO"}, true},
		Input{"FOO# ", []string{"FOO"}, true},
		Input{"FOO # ", []string{"FOO"}, true},
		Input{"FOO#BAR", []string{"BAR"}, false},
		Input{"FOO #BAR", []string{"BAR"}, false},
		Input{"FOO # BAR", []string{"BAR"}, false},
		Input{"FOO # BAR # FOO # BAR", []string{"BAR"}, false},

		Input{`
FOO # BAR
| BAR
`, []string{"BAR"}, true},

		Input{`
FOO # BAR
| BAZ
`, []string{"BAR"}, false},

		Input{`
FOO # BAR
| BAZ
`, []string{"BAZ"}, true},

		Input{`
FOO # BAR
& BAZ
`, []string{"FOO", "BAR"}, false},

		Input{`
FOO # BAR
& BAZ
`, []string{"BAZ", "BAR"}, false},

		Input{`
FOO # BAR
& BAZ
`, []string{"BAZ", "FOO"}, true},
	}

	for _, ie := range inputExpecteds {
		vals := map[string]struct{}{}
		for _, val := range ie.Vals {
			vals[val] = struct{}{}
		}
		actual, err := EvalBoolExpr(ie.Expr, vals)
		if err != nil {
			t.Errorf("error expected nil, actual: %v", err)
			continue
		}
		if ie.Expected != actual {
			t.Errorf("expression '%v' valabilities '%v' expected: %v, actual: %v", ie.Expr, ie.Vals, ie.Expected, actual)
		}
	}

	{
		// test complex expression

		complexExpr := `
ALLIGATOR   # must be an alligator
& ( BEAR    # and, either a bear
    | CAT   # or a cat
  )
& ( ! BEAR  # but not _both_ a bear
    | !CAT) # _and_ a cat

`
		type InputValExpected struct {
			Vals     []string
			Expected bool
		}

		complexExprExpected := []InputValExpected{
			{[]string{"ALLIGATOR"}, false},
			{[]string{"ALLIGATOR", "BEAR"}, true},
			{[]string{"ALLIGATOR", "CAT"}, true},
			{[]string{"ALLIGATOR", "BEAR", "CAT"}, false},
			{[]string{"BEAR", "CAT"}, false},
			{[]string{"BEAR"}, false},
			{[]string{"CAT"}, false},
			{[]string{"FOO"}, false},
		}

		for _, ie := range complexExprExpected {
			vals := map[string]struct{}{}
			for _, val := range ie.Vals {
				vals[val] = struct{}{}
			}
			actual, err := EvalBoolExpr(complexExpr, vals)
			if err != nil {
				t.Errorf("error expected nil, actual: %v", err)
				continue
			}
			if ie.Expected != actual {
				t.Errorf("expression '%v' valabilities '%v' expected: %v, actual: %v", complexExpr, ie.Vals, ie.Expected, actual)
			}
		}
	}
}

func TestEvalBoolExprSymbols(t *testing.T) {
	exprStr := `
ALLIGATOR   # must be an alligator
& ( BEAR    # and, either a bear
    | CAT   # or a cat
  )
& ( ! BEAR  # but not _both_ a bear
    | !CAT) # _and_ a cat

`

	expecteds := map[string]struct{}{
		"ALLIGATOR": {},
		"BEAR":      {},
		"CAT":       {},
	}

	expr, err := ParseBoolExpr(exprStr)
	if err != nil {
		t.Fatalf("error expected nil, actual: %v", err)
	}

	actual := expr.Symbols()
	for expected, _ := range expecteds {
		if _, ok := actual[expected]; !ok {
			t.Errorf("expected %v, actual: missing", expected)
		} else {
			delete(actual, expected)
		}
	}
	if len(actual) > 0 {
		t.Errorf("expected: only listed values, actual: %+v", actual)
	}

	expr, err = ParseBoolExpr("")
	if err != nil {
		t.Fatalf("error expected nil, actual: %v", err)
	}
	if len(expr.Symbols()) > 0 {
		t.Errorf("expected: no values for empty expression, actual: %+v", actual)
	}

}
