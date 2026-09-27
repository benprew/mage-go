package mage

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// BlockingOrBlockedBy selects creatures currently in combat with one permanent.
func BlockingOrBlockedBy(subject PermanentRef) PermanentSet {
	return combatOpponentsSet{subject: subject}
}

type combatOpponentsSet struct{ subject PermanentRef }

func (combatOpponentsSet) permanentSet() {}

func (s combatOpponentsSet) inspectDefinition() definitionNode {
	n := definitionNode{Role: "permanent-set", Text: "creatures blocking or blocked by permanent"}
	if nilDefinition(s.subject) {
		n.problem = "combat opponents require a permanent source"
	} else {
		n.children = []definitionEdge{{"subject", s.subject}}
	}
	return n
}

func (s combatOpponentsSet) resolvePermanents(ctx *EffectContext) []*Permanent {
	if nilDefinition(s.subject) {
		return nil
	}
	source, ok := s.subject.resolvePermanent(ctx)
	if !ok || !source.HasType(TypeCreature) {
		return nil
	}
	seen := make(map[*Permanent]bool)
	var result []*Permanent
	add := func(id, incarnation uuid.UUID) {
		p := ctx.Game.FindPermanent(id)
		if p == nil || p.PhasedOut || p.incarnationID != incarnation || !p.HasType(TypeCreature) || seen[p] {
			return
		}
		seen[p] = true
		result = append(result, p)
	}
	for _, group := range ctx.Game.CombatGroups() {
		if group.AttackerID == source.ID() && ctx.Game.combat.IsAttackingIncarnation(source.ID(), source.incarnationID) {
			for i, id := range group.BlockerIDs {
				if i < len(group.BlockerIncarnations) {
					add(id, group.BlockerIncarnations[i])
				}
			}
			continue
		}
		attacker := ctx.Game.FindPermanent(group.AttackerID)
		if attacker == nil || attacker.PhasedOut || !ctx.Game.combat.IsAttackingIncarnation(attacker.ID(), attacker.incarnationID) {
			continue
		}
		for i, id := range group.BlockerIDs {
			if id == source.ID() && i < len(group.BlockerIncarnations) && group.BlockerIncarnations[i] == source.incarnationID {
				add(attacker.ID(), attacker.incarnationID)
			}
		}
	}
	return result
}
