package expression

// ExtractExpressions will extract all the expressions from the provided string.
func ExtractExpressions(expression string) []Expression {
	expressions := []Expression{}

	currentExpression := ""

	inPotentialExpression := false
	capturingJsonPointer := false

	captureExpression := func(s string) {
		if s != "" {
			expressions = append(expressions, Expression(s))
		}

		currentExpression = ""
		inPotentialExpression = false
		capturingJsonPointer = false
	}

	for i, r := range expression {
		switch r {
		case '{':
			if inPotentialExpression && !capturingJsonPointer {
				inPotentialExpression = false
				capturingJsonPointer = false
				currentExpression = ""
			} else if !inPotentialExpression && len(expression) > i+1 && expression[i+1] == '$' {
				inPotentialExpression = true
			}
		case '$':
			if currentExpression == "" && !inPotentialExpression {
				inPotentialExpression = true
			} else if inPotentialExpression && i != 0 && expression[i-1] != '{' {
				inPotentialExpression = false
				capturingJsonPointer = false
				currentExpression = ""
			}
		case '}':
			if inPotentialExpression && !capturingJsonPointer && len(expression) > i+1 && expression[i+1] == '#' {
				capturingJsonPointer = true
			} else if inPotentialExpression && !capturingJsonPointer {
				currentExpression += string(r)
				captureExpression(currentExpression)
			}
		case '#':
			if inPotentialExpression && !capturingJsonPointer {
				capturingJsonPointer = true
			}
		case ' ':
			if inPotentialExpression && capturingJsonPointer {
				captureExpression(currentExpression)
			} else {
				inPotentialExpression = false
				capturingJsonPointer = false
				currentExpression = ""
			}
			continue
		}

		if inPotentialExpression {
			currentExpression += string(r)
		}
	}

	if inPotentialExpression {
		captureExpression(currentExpression)
	}

	return expressions
}
