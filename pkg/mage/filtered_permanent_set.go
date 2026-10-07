package mage

import "github.com/google/uuid"

type filteredPermanentSet struct {
	subjects PermanentSet
	filter   PermanentFilter
}

// FilterPermanentSet keeps selected members that match the current filter.
// The original set retains its target and incarnation checks.
func FilterPermanentSet(subjects PermanentSet, filter PermanentFilter) PermanentSet {
	return filteredPermanentSet{subjects: subjects, filter: filter}
}

func (filteredPermanentSet) permanentSet() {}

func (s filteredPermanentSet) resolvePermanents(ctx *EffectContext) []*Permanent {
	var result []*Permanent
	for _, p := range s.subjects.resolvePermanents(ctx) {
		if p != nil && !p.PhasedOut && s.filter.Match(p, ctx.Game) {
			result = append(result, p)
		}
	}
	return result
}

func (s filteredPermanentSet) inspectDefinition() definitionNode {
	return definitionNode{Role: "permanent-set", Text: s.filter.Text(), children: []definitionEdge{{"subjects", s.subjects}}}
}

type noDamagePlayers struct{}

func (noDamagePlayers) playerSet()                                {}
func (noDamagePlayers) resolvePlayers(*EffectContext) []uuid.UUID { return nil }
func (noDamagePlayers) inspectDefinition() definitionNode {
	return definitionNode{Role: "player-set", Text: "no players"}
}

// PermanentDamageRecipients uses one permanent set in the shared damage batch.
func PermanentDamageRecipients(subjects PermanentSet) DamageRecipientSet {
	return DamageRecipients(noDamagePlayers{}, subjects)
}
