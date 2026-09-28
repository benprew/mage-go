package mage

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type blockersOfSet struct{ attacker PermanentRef }

// BlockersOf selects creatures that currently block the chosen attacker.
func BlockersOf(attacker PermanentRef) PermanentSet {
	return blockersOfSet{attacker: attacker}
}

func (blockersOfSet) permanentSet() {}

func (s blockersOfSet) inspectDefinition() definitionNode {
	n := definitionNode{Role: "permanent-set", Text: "creatures blocking attacker"}
	if nilDefinition(s.attacker) {
		n.problem = "blockers require an attacker"
	} else {
		n.children = []definitionEdge{{"attacker", s.attacker}}
	}
	return n
}

func (s blockersOfSet) resolvePermanents(ctx *EffectContext) []*Permanent {
	if nilDefinition(s.attacker) || ctx == nil || ctx.Game == nil || ctx.Game.combat == nil {
		return nil
	}
	attacker, ok := s.attacker.resolvePermanent(ctx)
	if !ok || !attacker.HasType(TypeCreature) || !ctx.Game.combat.IsAttackingIncarnation(attacker.ID(), attacker.incarnationID) {
		return nil
	}
	var result []*Permanent
	seen := make(map[uuid.UUID]bool)
	for _, group := range ctx.Game.CombatGroups() {
		if group.AttackerID != attacker.ID() {
			continue
		}
		for i, id := range group.BlockerIDs {
			if i >= len(group.BlockerIncarnations) || seen[id] {
				continue
			}
			blocker := ctx.Game.FindPermanent(id)
			if blocker == nil || blocker.PhasedOut || blocker.incarnationID != group.BlockerIncarnations[i] || !blocker.HasType(TypeCreature) {
				continue
			}
			seen[id] = true
			result = append(result, blocker)
		}
	}
	return result
}
