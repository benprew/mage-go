package mage

type permanentCountValue struct {
	subjects PermanentSet
}

// CountPermanents reads collection membership when the value is used.
func CountPermanents(subjects PermanentSet) ResolutionValue {
	return permanentCountValue{subjects: subjects}
}

func (permanentCountValue) Text() string { return "number of permanents in collection" }

func (v permanentCountValue) resolveValue(ctx *EffectContext) int {
	return len(v.subjects.resolvePermanents(ctx))
}

func (v permanentCountValue) inspectDefinition() definitionNode {
	return definitionNode{Role: "permanent-count", Text: v.Text(), children: []definitionEdge{{"subjects", v.subjects}}}
}
