package mage

import (
	"slices"

	"github.com/google/uuid"
)

type waitingPreventionReferences interface {
	waitingPreventionPermanents(*Game) []capturedPermanent
}

func (r *preventionShieldReplacement) waitingPreventionPermanents(*Game) []capturedPermanent {
	return []capturedPermanent{{r.targetID, r.targetIncarnation}}
}

func (r *sourcePreventionReplacement) waitingPreventionPermanents(*Game) []capturedPermanent {
	if r.continuedSource.Valid() {
		return []capturedPermanent{{r.continuedSource.ID(), r.continuedSource.Incarnation()}}
	}
	if r.exactSource.Valid() {
		return []capturedPermanent{{r.exactSource.ID(), r.exactSource.Incarnation()}}
	}
	return []capturedPermanent{{r.dmgSource, r.incarnation}}
}

func (r *damagePreventionRuleReplacement) waitingPreventionPermanents(*Game) []capturedPermanent {
	return []capturedPermanent{r.recipientPermanent}
}

func (r *combatSourcePreventionReplacement) waitingPreventionPermanents(*Game) []capturedPermanent {
	return []capturedPermanent{{r.preventedID, r.incarnation}}
}

func (r *creatureCombatDamagePreventionReplacement) waitingPreventionPermanents(*Game) []capturedPermanent {
	return []capturedPermanent{{r.creatureID, r.incarnation}}
}

func (r *counterDamagePreventionReplacement) waitingPreventionPermanents(*Game) []capturedPermanent {
	return []capturedPermanent{{r.sourceID, r.incarnation}}
}

func (r *staticDamagePreventionReplacement) waitingPreventionPermanents(g *Game) []capturedPermanent {
	subject, ok := r.currentSubject(g)
	if !ok {
		return nil
	}
	return []capturedPermanent{{subject.ID(), subject.incarnationID}}
}

func (g *Game) legacyPreventionSourceIncarnation(sourceID uuid.UUID) uuid.UUID {
	if source := g.FindPermanent(sourceID); source != nil {
		return source.incarnationID
	}
	return uuid.Nil
}

func (r *sourcePreventionShieldReplacement) waitingPreventionPermanents(*Game) []capturedPermanent {
	return []capturedPermanent{{r.dmgSource, r.incarnation}}
}

func (r *halfDamagePreventionShieldReplacement) waitingPreventionPermanents(*Game) []capturedPermanent {
	return []capturedPermanent{{r.dmgSource, r.incarnation}}
}

func (r *reverseDamageReplacement) waitingPreventionPermanents(*Game) []capturedPermanent {
	return []capturedPermanent{{r.dmgSource, r.incarnation}}
}

// WaitingPreventionPermanentSources reads exact references in active prevention effects.
// A returned object must not replace information for the saved incarnation.
func (g *Game) WaitingPreventionPermanentSources() []DamageSource {
	return g.waitingReplacementPermanentSources(func(replacement ReplacementEffect) []capturedPermanent {
		refs, ok := replacement.(waitingPreventionReferences)
		if !ok {
			return nil
		}
		return refs.waitingPreventionPermanents(g)
	})
}

func (g *Game) waitingReplacementPermanentSources(references func(ReplacementEffect) []capturedPermanent) []DamageSource {
	var sources []DamageSource
	for _, list := range [2][]ReplacementEffect{g.effects.replacements, g.effects.cycleReplacements} {
		for _, replacement := range list {
			if !replacement.IsActive(g) {
				continue
			}
			for _, ref := range references(replacement) {
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
