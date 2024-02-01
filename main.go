package main

import (
	"fmt"
	lep "github.com/mgudov/logic-expression-parser"
	"strings"
)

func cleanExpression(exp string) string {
	exp = strings.ReplaceAll(exp, "$", "")
	exp = strings.ReplaceAll(exp, ".", "")
	exp = strings.ReplaceAll(exp, "{", "")
	exp = strings.ReplaceAll(exp, "}", "")
	exp = strings.ReplaceAll(exp, "NOT EXISTS", "= \"__NOT_EXISTS__\"")
	exp = strings.TrimSpace(exp)
	return exp
}

func compareExpressions(strA, strB string) bool {
	strA = cleanExpression(strA)
	strB = cleanExpression(strB)

	fmt.Println(strA)
	fmt.Println(strB)

	expA, err := lep.ParseExpression(strA, lep.Recover(false))
	if err != nil {
		fmt.Printf("Error %+v\n", err)
		return false
	}

	expB, err := lep.ParseExpression(strB, lep.Recover(false))
	if err != nil {
		fmt.Printf("Error %+v\n", err)
		return false
	}

	return expA.Equals(expB)
}
