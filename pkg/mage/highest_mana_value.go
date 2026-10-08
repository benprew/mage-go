package mage

// HighestManaValue keeps all members tied for the highest current mana value.
// Face-down permanents have no mana cost. An empty set stays empty.
func HighestManaValue(subjects PermanentSet) PermanentSet {
	return highestManaValueSet{subjects: subjects}
}

type highestManaValueSet struct{ subjects PermanentSet }

func (highestManaValueSet) permanentSet() {}

func (s highestManaValueSet) resolvePermanents(ctx *EffectContext) []*Permanent {
	members := s.subjects.resolvePermanents(ctx)
	var result []*Permanent
	highest := -1
	for _, permanent := range members {
		value := 0
		if !permanent.FaceDown {
			value = permanent.Card.ManaCost().CMC()
		}
		if value > highest {
			highest = value
			result = result[:0]
		}
		if value == highest {
			result = append(result, permanent)
		}
	}
	return result
}

func (s highestManaValueSet) inspectDefinition() definitionNode {
	return definitionNode{Role: "highest-mana-value-set", children: []definitionEdge{{"subjects", s.subjects}}}
}
