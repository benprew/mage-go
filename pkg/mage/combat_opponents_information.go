package mage

import "github.com/benprew/mage-go/pkg/mage/core"

// CombatOpponentsWithLKI selects present creatures from an exact combat visit.
// Departure information keeps the relation after the source leaves combat.
func CombatOpponentsWithLKI(subject PermanentRef) PermanentSet {
	return combatOpponentsInformationSet{subject: subject}
}

type combatOpponentsInformationSet struct{ subject PermanentRef }

func (combatOpponentsInformationSet) permanentSet() {}

func (s combatOpponentsInformationSet) inspectDefinition() definitionNode {
	return permanentInformationNode(s.subject, "permanent-set", "combat opponents with last known information")
}

func (s combatOpponentsInformationSet) resolvePermanents(ctx *EffectContext) []*Permanent {
	if ctx == nil || ctx.Game == nil || nilDefinition(s.subject) {
		return nil
	}
	source, lki := readPermanentInformation(s.subject, ctx)
	var opponents []capturedPermanent
	if lki != nil {
		opponents = lki.combatOpponents
	} else if source != nil {
		opponents = snapshotCombatOpponents(source, ctx.Game)
	}
	var result []*Permanent
	for _, ref := range opponents {
		p := ctx.Game.FindPermanent(ref.id)
		if p != nil && !p.PhasedOut && p.incarnationID == ref.incarnation && p.HasType(core.TypeCreature) {
			result = append(result, p)
		}
	}
	return result
}
