package mage

// WaitingDrawStepLifePermanentSources reads exact sources of active draw-step life rules.
// An inactive rule cannot supply its former source through retained information.
func (g *Game) WaitingDrawStepLifePermanentSources() []DamageSource {
	return g.waitingReplacementPermanentSources(func(replacement ReplacementEffect) []capturedPermanent {
		rule, ok := replacement.(*drawStepLifeReplacement)
		if !ok {
			return nil
		}
		return []capturedPermanent{{rule.sourceID, rule.incarnation}}
	})
}

// ChooseDamageSourceFromWaitingDrawStepLifeReferences uses active draw-step life sources.
// Other legal source categories need separate enumeration services.
func ChooseDamageSourceFromWaitingDrawStepLifeReferences(player PlayerRef, filter DamageSourceFilter, binding *DamageSourceBinding) ResolvingEffect {
	return &damageSourceChoice{player: player, filter: filter, binding: binding, waitingDrawStepLife: true}
}
