package mage

import "slices"

func (g *Game) continueSpellExileReplacement(spell DamageSource, permanent *Permanent) {
	for _, list := range [2][]ReplacementEffect{g.effects.replacements, g.effects.cycleReplacements} {
		for _, replacement := range list {
			if rule, ok := replacement.(*exileInsteadOfGraveyardReplacement); ok && rule.IsActive(g) && rule.subject.SameObject(spell) {
				rule.continued = damageSourceFromPermanent(permanent)
			}
		}
	}
}

// WaitingExileSources reads exact subjects of active exile replacements.
// Saved information keeps a departed subject available until cleanup.
func (g *Game) WaitingExileSources() []DamageSource {
	var sources []DamageSource
	for _, list := range [2][]ReplacementEffect{g.effects.replacements, g.effects.cycleReplacements} {
		for _, replacement := range list {
			if !replacement.IsActive(g) {
				continue
			}
			var subjects []DamageSource
			switch rule := replacement.(type) {
			case *exileInsteadOfGraveyardReplacement:
				subjects = []DamageSource{rule.subject, rule.continued}
			case *temporaryDeathExileReplacement:
				subjects = []DamageSource{rule.reference}
			default:
				continue
			}
			for _, saved := range subjects {
				if !saved.hasExactIdentity() {
					continue
				}
				source := g.damageReferenceInformation(saved)
				if !source.SameObject(saved) {
					source = saved
				}
				if !slices.ContainsFunc(sources, source.SameObject) {
					sources = append(sources, cloneDamageSource(source))
				}
			}
		}
	}
	return sources
}

// ChooseDamageSourceFromWaitingExileReferences uses exact exile-rule subjects.
// Other source categories need separate enumeration services.
func ChooseDamageSourceFromWaitingExileReferences(player PlayerRef, filter DamageSourceFilter, binding *DamageSourceBinding) ResolvingEffect {
	return &damageSourceChoice{player: player, filter: filter, binding: binding, waitingExile: true}
}

func (g *Game) spellMarkedExileInstead(obj *StackObject) bool {
	if obj == nil || obj.IsAbility || obj.Card == nil {
		return false
	}
	if g.zones.IsExileInstead(obj.Card.ID()) {
		return true
	}
	spell := damageSourceFromSpellObject(obj)
	for _, list := range [2][]ReplacementEffect{g.effects.replacements, g.effects.cycleReplacements} {
		for _, replacement := range list {
			if rule, ok := replacement.(*exileInsteadOfGraveyardReplacement); ok && rule.IsActive(g) && rule.subject.SameObject(spell) {
				return true
			}
		}
	}
	return false
}
