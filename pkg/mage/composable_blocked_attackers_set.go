package mage

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type blockedAttackersOfSet struct{ blocker PermanentRef }

// BlockedAttackersOf selects attackers that the referenced creature is blocking now.
func BlockedAttackersOf(blocker PermanentRef) PermanentSet {
	return blockedAttackersOfSet{blocker: blocker}
}

func (blockedAttackersOfSet) permanentSet() {}

func (s blockedAttackersOfSet) inspectDefinition() definitionNode {
	n := definitionNode{Role: "permanent-set", Text: "attackers blocked by creature"}
	if nilDefinition(s.blocker) {
		n.problem = "blocked attackers require a blocker"
	} else {
		n.children = []definitionEdge{{"blocker", s.blocker}}
	}
	return n
}

func (s blockedAttackersOfSet) resolvePermanents(ctx *EffectContext) []*Permanent {
	if nilDefinition(s.blocker) || ctx == nil || ctx.Game == nil || ctx.Game.combat == nil {
		return nil
	}
	blocker, ok := s.blocker.resolvePermanent(ctx)
	if !ok || !blocker.HasType(TypeCreature) || !ctx.Game.combat.IsBlockingIncarnation(blocker.ID(), blocker.incarnationID) {
		return nil
	}
	var result []*Permanent
	seen := make(map[uuid.UUID]bool)
	for _, group := range ctx.Game.CombatGroups() {
		if seen[group.AttackerID] {
			continue
		}
		linked := false
		for i, id := range group.BlockerIDs {
			if id == blocker.ID() && i < len(group.BlockerIncarnations) && group.BlockerIncarnations[i] == blocker.incarnationID {
				linked = true
				break
			}
		}
		if !linked {
			continue
		}
		attacker := ctx.Game.FindPermanent(group.AttackerID)
		if attacker == nil || attacker.PhasedOut || !attacker.HasType(TypeCreature) || !ctx.Game.combat.IsAttackingIncarnation(attacker.ID(), attacker.incarnationID) {
			continue
		}
		seen[attacker.ID()] = true
		result = append(result, attacker)
	}
	return result
}
