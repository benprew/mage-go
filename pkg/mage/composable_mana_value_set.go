package mage

type manaValueAtMostSet struct {
	subjects PermanentSet
	limit    ResolutionValue
}

// ManaValueAtMost keeps set members whose mana value is at most the resolution value.
func ManaValueAtMost(subjects PermanentSet, limit ResolutionValue) PermanentSet {
	return manaValueAtMostSet{subjects: subjects, limit: limit}
}

func (manaValueAtMostSet) permanentSet() {}

func (s manaValueAtMostSet) resolvePermanents(ctx *EffectContext) []*Permanent {
	maximum := s.limit.resolveValue(ctx)
	var result []*Permanent
	for _, p := range s.subjects.resolvePermanents(ctx) {
		if p.Card.ManaCost().CMC() <= maximum {
			result = append(result, p)
		}
	}
	return result
}

func (s manaValueAtMostSet) inspectDefinition() definitionNode {
	return definitionNode{Role: "permanent-set", Text: "mana value at most", children: []definitionEdge{{"subjects", s.subjects}, {"limit", s.limit}}}
}
