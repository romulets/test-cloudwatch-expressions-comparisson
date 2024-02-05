package cloudwatch_lep

import (
	"errors"
	"fmt"
	"strings"
)

type logicalOperator string
type comparisonOperator string

const (
	loAnd logicalOperator = "&&"
	loOr  logicalOperator = "||"

	coEqual     comparisonOperator = "="
	coNotEqual  comparisonOperator = "!="
	coNotExists comparisonOperator = "NOT EXISTS"
)

func listLogicalOperators() []logicalOperator {
	return []logicalOperator{loAnd, loOr}
}

func listComparisonOperator() []comparisonOperator {
	// This order must be kept because we need to check first different and then equals
	return []comparisonOperator{coNotExists, coNotEqual, coEqual}
}

type expression interface {
	isEquivalent(s expression) bool
}

type simpleExpression struct {
	left     string
	right    string
	operator comparisonOperator
}

func (s simpleExpression) isEquivalent(o expression) bool {
	simpleOther, ok := any(o).(simpleExpression)
	if !ok {
		return false // not a simpleExpression
	}

	if simpleOther.operator != s.operator {
		return false
	}

	if simpleOther.left == s.left && simpleOther.right == s.right {
		return true
	}

	if simpleOther.left == s.right && simpleOther.right == s.left {
		return true
	}

	return false
}

type complexExpression struct {
	operator    logicalOperator
	expressions []expression
}

func (c complexExpression) isEquivalent(o expression) bool {
	complexOther, ok := any(o).(complexExpression)
	if !ok {
		return false // not a complexExpression
	}

	if complexOther.operator != c.operator {
		return false
	}

	if len(c.expressions) != len(complexOther.expressions) {
		return false
	}

	otherExpressions := make([]expression, len(complexOther.expressions))
	copy(otherExpressions, complexOther.expressions)

	for _, exp := range c.expressions {
		if found, idx := c.findEquivalentPos(exp, otherExpressions); found {
			// Replace the found index by the last position
			otherExpressions[idx] = otherExpressions[len(otherExpressions)-1]
			// Replace the last position (now it's duplicated)
			otherExpressions = otherExpressions[:len(otherExpressions)-1]
		} else {
			// if no equivalent expression found, return falses
			return false
		}
	}

	return true
}

func (c complexExpression) findEquivalentPos(exp expression, otherExpressions []expression) (bool, int) {
	for i, expB := range otherExpressions {
		if exp.isEquivalent(expB) {
			return true, i
		}
	}

	return false, -1
}

func areCloudWatchExpressionsEquivalent(a, b string) (bool, error) {
	statementA, err := parse(a)
	if err != nil {
		return false, err
	}

	statementB, err := parse(b)
	if err != nil {
		return false, err
	}

	return statementA.isEquivalent(statementB), nil
}

func parse(s string) (expression, error) {
	// remove trailling spaces and { }
	cleanS := strings.TrimSpace(strings.TrimRight(strings.TrimLeft(strings.TrimSpace(s), "{"), "}"))
	return safeParse(cleanS, 0)
}

func safeParse(s string, depth int) (expression, error) {
	if depth > 2 {
		return simpleExpression{}, fmt.Errorf("max depth reached (%d), won't parse anymore", depth)
	}

	buf := strings.Builder{}
	buf.Grow(len(s))
	parenthesisAcc := 0

	var logicalOp logicalOperator
	expressions := make([]expression, 0, len(s))
	isInsideSubExp := false

	for i, r := range s {
		if r == '(' {
			parenthesisAcc++
			if parenthesisAcc > 1 {
				isInsideSubExp = true
			}
			if parenthesisAcc > 3 {
				return simpleExpression{}, errors.New("not supported more than 2 nested parenthesis")
			}
			continue
		}

		if r == ')' {
			parenthesisAcc--
			if parenthesisAcc < 0 {
				return simpleExpression{}, errors.New("broken parenthesis")
			}
			continue
		}

		// it's in the middle of another expression, so we just add to
		// the buffer until we find the end of this sub expression
		if isInsideSubExp {
			buf.WriteRune(r)

			// Sub expression completed!
			if parenthesisAcc == 0 {
				expStr := buf.String()
				exp, err := safeParse(expStr, depth+1)
				if err != nil {
					return simpleExpression{}, err
				}

				expressions = append(expressions, exp)
				buf.Reset()
				buf.Grow(len(s) - i)
				isInsideSubExp = false
			}
			continue
		}
		tmpString := buf.String()
		if contains, op := hasSuffixLogicalOp(tmpString); contains {
			if logicalOp == "" {
				logicalOp = op
			}

			if logicalOp != op {
				return simpleExpression{}, errors.New("not supported comparison with alternating logical operators")
			}

			exprStr := strings.TrimSuffix(tmpString, string(op))

			// if the length is zero it means we had an already processed complex  expressions (between parenthesis)
			if len(exprStr) > 0 {
				exp, err := safeParse(exprStr, depth+1)
				if err != nil {
					return simpleExpression{}, err
				}

				expressions = append(expressions, exp)
			}

			buf.Reset()
			buf.Grow(len(s) - i)
		}

		buf.WriteRune(r)
	}

	if len(expressions) == 0 { // it's a simple expression!
		return parseSimpleStatement(s)
	}

	if buf.Len() > 0 {
		exp, err := safeParse(buf.String(), depth+1)
		if err != nil {
			return simpleExpression{}, err
		}
		expressions = append(expressions, exp)
	}

	return complexExpression{operator: logicalOp, expressions: expressions}, nil
}

func parseSimpleStatement(s string) (expression, error) {
	buf := strings.Builder{}
	buf.Grow(len(s))

	var left string
	var operator comparisonOperator
	foundOp := false

	for i, r := range s {
		if buf.Len() == 0 && (r == ' ' || r == '(') { //ignore trailing spaces and (
			continue
		}

		buf.WriteRune(r)
		tmpString := buf.String()
		if contains, op := hasSuffixComparisonOp(tmpString); contains {
			if foundOp {
				return simpleExpression{}, errors.New("got multiple comparison operators")
			}

			left = strings.TrimSpace(strings.TrimSuffix(tmpString, string(op)))
			operator = op
			foundOp = true
			buf.Reset()
			buf.Grow(len(s) - i)
		}
	}

	if !foundOp {
		return simpleExpression{}, errors.New("could not find a operator for this expression")
	}

	// Trim trailing spaces and )
	right := strings.TrimSpace(strings.TrimRight(strings.TrimSpace(buf.String()), ")"))
	return simpleExpression{
		left:     left,
		operator: operator,
		right:    right,
	}, nil
}

func hasSuffixComparisonOp(s string) (bool, comparisonOperator) {
	for _, op := range listComparisonOperator() {
		if strings.HasSuffix(s, string(op)) {
			return true, op
		}
	}
	return false, ""
}

func hasSuffixLogicalOp(s string) (bool, logicalOperator) {
	for _, op := range listLogicalOperators() {
		if strings.HasSuffix(s, string(op)) {
			return true, op
		}
	}
	return false, ""
}
