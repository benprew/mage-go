package mage

// PowerEqualToLeastOf keeps subjects whose power equals the least comparison power.
// An empty comparison set supplies no power and keeps no subjects.
func PowerEqualToLeastOf(subjects, compared PermanentSet) PermanentSet {
	return leastPowerComparisonSet{subjects: subjects, compared: compared}
}

type leastPowerComparisonSet struct {
	subjects PermanentSet
	compared PermanentSet
}

func (leastPowerComparisonSet) permanentSet() {}

func (s leastPowerComparisonSet) resolvePermanents(ctx *EffectContext) []*Permanent {
	least := LowestPower(s.compared).resolvePermanents(ctx)
	if len(least) == 0 {
		return nil
	}
	power := least[0].CurrentPower(ctx.Game)
	var result []*Permanent
	for _, p := range s.subjects.resolvePermanents(ctx) {
		if p.CurrentPower(ctx.Game) == power {
			result = append(result, p)
		}
	}
	return result
}

func (s leastPowerComparisonSet) inspectDefinition() definitionNode {
	return definitionNode{Role: "least-power-comparison-set", children: []definitionEdge{{"subjects", s.subjects}, {"compared", s.compared}}}
}
