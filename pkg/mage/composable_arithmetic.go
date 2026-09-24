package mage

type arithmeticOperation uint8

const (
	valueSubtract arithmeticOperation = iota
	valueMin
	valueMax
)

type arithmeticValue struct {
	left, right ResolutionValue
	operation   arithmeticOperation
}

// SubtractValues subtracts the right value from the left value.
func SubtractValues(left, right ResolutionValue) ResolutionValue {
	return arithmeticValue{left, right, valueSubtract}
}

// MinValue supplies the smaller of two values.
func MinValue(left, right ResolutionValue) ResolutionValue {
	return arithmeticValue{left, right, valueMin}
}

// MaxValue supplies the larger of two values.
func MaxValue(left, right ResolutionValue) ResolutionValue {
	return arithmeticValue{left, right, valueMax}
}

func (v arithmeticValue) operationName() string {
	switch v.operation {
	case valueSubtract:
		return "subtract"
	case valueMin:
		return "minimum"
	default:
		return "maximum"
	}
}

func (v arithmeticValue) Text() string {
	return v.operationName() + "(" + v.left.Text() + ", " + v.right.Text() + ")"
}

func (v arithmeticValue) resolveValue(ctx *EffectContext) int {
	left, right := v.left.resolveValue(ctx), v.right.resolveValue(ctx)
	switch v.operation {
	case valueSubtract:
		return left - right
	case valueMin:
		return min(left, right)
	default:
		return max(left, right)
	}
}

func (v arithmeticValue) inspectDefinition() definitionNode {
	return definitionNode{
		Role: "arithmetic-value", Text: v.operationName(),
		children: []definitionEdge{{"left", v.left}, {"right", v.right}},
	}
}
