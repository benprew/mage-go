package mage

// WaitingPersistentReplacementPermanentSources reads explicit sources of active rules.
// General filters do not supply references to all objects that can match.
func (g *Game) WaitingPersistentReplacementPermanentSources() []DamageSource {
	return g.waitingReplacementPermanentSources(func(replacement ReplacementEffect) []capturedPermanent {
		switch rule := replacement.(type) {
		case *nextDestructionReplacement:
			return []capturedPermanent{{rule.subject, rule.incarnation}}
		case *lichLifeGainReplacement:
			return []capturedPermanent{{rule.sourceID, rule.incarnation}}
		case *islandSanctuaryReplacement:
			return []capturedPermanent{{rule.sourceID, rule.incarnation}}
		case *counterDoublerReplacement:
			return []capturedPermanent{{rule.sourceID, rule.incarnation}}
		default:
			return nil
		}
	})
}

// ChooseDamageSourceFromWaitingPersistentReplacementReferences uses explicit rule sources.
// Other legal source categories need separate enumeration services.
func ChooseDamageSourceFromWaitingPersistentReplacementReferences(player PlayerRef, filter DamageSourceFilter, binding *DamageSourceBinding) ResolvingEffect {
	return &damageSourceChoice{player: player, filter: filter, binding: binding, waitingPersistentReplacements: true}
}
