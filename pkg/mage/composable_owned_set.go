package mage

type ownedPermanentSet struct {
	owner  PlayerRef
	filter PermanentFilter
}

// PermanentsOwnedBy selects battlefield permanents owned by one player.
func PermanentsOwnedBy(owner PlayerRef, filter PermanentFilter) PermanentSet {
	return ownedPermanentSet{owner: owner, filter: filter}
}

func (ownedPermanentSet) permanentSet() {}

func (s ownedPermanentSet) resolvePermanents(ctx *EffectContext) []*Permanent {
	if nilDefinition(s.owner) {
		return nil
	}
	owner, ok := s.owner.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	var result []*Permanent
	for _, permanent := range ctx.Game.FilterBattlefield(s.filter) {
		if permanent.Card.Owner() == owner {
			result = append(result, permanent)
		}
	}
	return result
}

func (s ownedPermanentSet) inspectDefinition() definitionNode {
	n := definitionNode{Role: "permanent-set", Text: "permanents owned by player"}
	if nilDefinition(s.owner) {
		n.problem = "owned permanent set requires a player"
	} else {
		n.children = []definitionEdge{{"owner", s.owner}}
	}
	return n
}
