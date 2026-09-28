package mage

type controlledByPermanentSet struct {
	controller PlayerRef
	filter     PermanentFilter
}

// PermanentsControlledBy selects current battlefield permanents controlled by one player.
func PermanentsControlledBy(controller PlayerRef, filter PermanentFilter) PermanentSet {
	return controlledByPermanentSet{controller: controller, filter: filter}
}

func (controlledByPermanentSet) permanentSet() {}

func (s controlledByPermanentSet) resolvePermanents(ctx *EffectContext) []*Permanent {
	if nilDefinition(s.controller) {
		return nil
	}
	controller, ok := s.controller.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	var result []*Permanent
	for _, permanent := range ctx.Game.FilterBattlefield(s.filter) {
		if permanent.ControllerID() == controller {
			result = append(result, permanent)
		}
	}
	return result
}

func (s controlledByPermanentSet) inspectDefinition() definitionNode {
	n := definitionNode{Role: "permanent-set", Text: "permanents controlled by player"}
	if nilDefinition(s.controller) {
		n.problem = "controlled permanent set requires a player"
	} else {
		n.children = []definitionEdge{{"controller", s.controller}}
	}
	return n
}
