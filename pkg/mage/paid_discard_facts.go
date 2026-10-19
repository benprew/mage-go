package mage

// CardPaymentFact saves a card filter result for later instructions.
type CardPaymentFact struct {
	Filter CardFilter
	Result *NumberBinding
}

// DiscardCardCostWithFacts saves facts about one completed discard payment.
// An unrevealed hidden destination supplies no characteristic match.
func DiscardCardCostWithFacts(facts ...CardPaymentFact) Cost {
	return &discardCost{amount: 1, facts: append([]CardPaymentFact(nil), facts...)}
}

func (c *discardCost) inspectFacts(n *definitionNode) {
	seen := make(map[*NumberBinding]bool)
	for _, fact := range c.facts {
		if fact.Filter.IsZero() || fact.Result == nil || seen[fact.Result] {
			n.problem = "supply distinct non-nil card payment fact bindings and filters"
		}
		seen[fact.Result] = true
		n.bindingWrites = append(n.bindingWrites, fact.Result)
		n.children = append(n.children, definitionEdge{"payment fact", fact.Result})
	}
	if len(c.facts) > 0 && c.amount != 1 {
		n.problem = "card payment facts require one discard"
	}
}
