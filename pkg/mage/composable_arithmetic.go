package mage

type arithmeticOperation uint8

const (
	valueSubtract arithmeticOperation = iota
	valueMin
	valueMax
	valueMultiply
	valueDivideDown
	valueDivideUp
)

type arithmeticValue struct {
	left, right ResolutionValue
	operation   arithmeticOperation
}

// SubtractValues subtracts the right value from the left value.
func SubtractValues(left, right ResolutionValue) ResolutionValue {
	return arithmeticValue{left, right, valueSubtract}
}

// MultiplyValues supplies the product of two values.
func MultiplyValues(left, right ResolutionValue) ResolutionValue {
	return arithmeticValue{left, right, valueMultiply}
}

// MinValue supplies the smaller of two values.
func MinValue(left, right ResolutionValue) ResolutionValue {
	return arithmeticValue{left, right, valueMin}
}

// MaxValue supplies the larger of two values.
func MaxValue(left, right ResolutionValue) ResolutionValue {
	return arithmeticValue{left, right, valueMax}
}

// DivideRoundDown divides two values and rounds the result down.
func DivideRoundDown(numerator, denominator ResolutionValue) ResolutionValue {
	return arithmeticValue{numerator, denominator, valueDivideDown}
}

// DivideRoundUp divides two values and rounds the result up.
func DivideRoundUp(numerator, denominator ResolutionValue) ResolutionValue {
	return arithmeticValue{numerator, denominator, valueDivideUp}
}

func (v arithmeticValue) operationName() string {
	switch v.operation {
	case valueMultiply:
		return "multiply"
	case valueSubtract:
		return "subtract"
	case valueMin:
		return "minimum"
	case valueMax:
		return "maximum"
	case valueDivideDown:
		return "divide rounded down"
	case valueDivideUp:
		return "divide rounded up"
	default:
		return "invalid arithmetic"
	}
}

func (v arithmeticValue) Text() string {
	left, right := "?", "?"
	if !nilDefinition(v.left) {
		left = v.left.Text()
	}
	if !nilDefinition(v.right) {
		right = v.right.Text()
	}
	return v.operationName() + "(" + left + ", " + right + ")"
}

func (v arithmeticValue) resolveValue(ctx *EffectContext) int {
	left, right := v.left.resolveValue(ctx), v.right.resolveValue(ctx)
	switch v.operation {
	case valueMultiply:
		return left * right
	case valueSubtract:
		return left - right
	case valueMin:
		return min(left, right)
	case valueMax:
		return max(left, right)
	case valueDivideDown, valueDivideUp:
		if right == 0 {
			return 0
		}
		quotient, remainder := left/right, left%right
		if remainder == 0 {
			return quotient
		}
		if v.operation == valueDivideDown && (left < 0) != (right < 0) {
			return quotient - 1
		}
		if v.operation == valueDivideUp && (left < 0) == (right < 0) {
			return quotient + 1
		}
		return quotient
	default:
		return 0
	}
}

func (v arithmeticValue) inspectDefinition() definitionNode {
	n := definitionNode{
		Role: "arithmetic-value", Text: v.operationName(),
		children: []definitionEdge{{"left", v.left}, {"right", v.right}},
	}
	if (v.operation == valueDivideDown || v.operation == valueDivideUp) && !nilDefinition(v.right) {
		if fixed, ok := v.right.(fixedValue); ok && fixed.n == 0 {
			n.problem = "fixed divisor must not be zero"
		}
	}
	return n
}
