package mage

import (
	"slices"

	"github.com/google/uuid"
)

type waitingPreventionReferences interface {
	waitingPreventionPermanents() []capturedPermanent
}

func (r *preventionShieldReplacement) waitingPreventionPermanents() []capturedPermanent {
	return []capturedPermanent{{r.targetID, r.targetIncarnation}}
}

func (r *sourcePreventionReplacement) waitingPreventionPermanents() []capturedPermanent {
	return []capturedPermanent{{r.dmgSource, r.incarnation}}
}

func (r *damagePreventionRuleReplacement) waitingPreventionPermanents() []capturedPermanent {
	return []capturedPermanent{r.recipientPermanent}
}

func (r *combatSourcePreventionReplacement) waitingPreventionPermanents() []capturedPermanent {
	return []capturedPermanent{{r.preventedID, r.incarnation}}
}

func (r *creatureCombatDamagePreventionReplacement) waitingPreventionPermanents() []capturedPermanent {
	return []capturedPermanent{{r.creatureID, r.incarnation}}
}

// WaitingPreventionPermanentSources reads exact references in active prevention effects.
// A returned object must not replace information for the saved incarnation.
func (g *Game) WaitingPreventionPermanentSources() []DamageSource {
	var sources []DamageSource
	for _, list := range [2][]ReplacementEffect{g.effects.replacements, g.effects.cycleReplacements} {
		for _, replacement := range list {
			refs, ok := replacement.(waitingPreventionReferences)
			if !ok || !replacement.IsActive(g) {
				continue
			}
			for _, ref := range refs.waitingPreventionPermanents() {
				if ref.id == uuid.Nil || ref.incarnation == uuid.Nil {
					continue
				}
				saved := DamageSource{id: ref.id, incarnation: ref.incarnation}
				source := g.damageReferenceInformation(saved)
				if source.Valid() && source.SameObject(saved) && !slices.ContainsFunc(sources, source.SameObject) {
					sources = append(sources, cloneDamageSource(source))
				}
			}
		}
	}
	return sources
}

// ChooseDamageSourceFromWaitingPreventionReferences uses retained permanent references.
// Other legal source categories require separate enumeration services.
func ChooseDamageSourceFromWaitingPreventionReferences(player PlayerRef, filter DamageSourceFilter, binding *DamageSourceBinding) ResolvingEffect {
	return &damageSourceChoice{player: player, filter: filter, binding: binding, waitingPrevention: true}
}
