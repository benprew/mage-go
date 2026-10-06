package mage

import "github.com/google/uuid"

type targetCombatRelation uint8

const (
	targetBlocksSource targetCombatRelation = 1 << iota
	targetBlockedBySource
)

// TargetBlocksSource requires the target to block the original source.
func TargetBlocksSource() PermanentTargetOption {
	return targetCombatRelationOption{targetBlocksSource}
}

// TargetBlockedBySource requires the original source to block the target.
func TargetBlockedBySource() PermanentTargetOption {
	return targetCombatRelationOption{targetBlockedBySource}
}

// TargetInCombatWithSource permits either direction of the blocking relation.
func TargetInCombatWithSource() PermanentTargetOption {
	return targetCombatRelationOption{targetBlocksSource | targetBlockedBySource}
}

type targetCombatRelationOption struct{ relation targetCombatRelation }

func (targetCombatRelationOption) permanentTargetOption() {}
func (o targetCombatRelationOption) applyPermanentTarget(t *PermanentTarget) {
	t.combatRelations = append(t.combatRelations, o.relation)
}
func (targetCombatRelationOption) inspectDefinition() definitionNode {
	return definitionNode{Role: "target-combat-relation", battlefieldSource: true}
}

// BlocksIncarnations checks the current pair. A departed blocker does not match.
func (c *Combat) BlocksIncarnations(blockerID, blockerIncarnation, attackerID, attackerIncarnation uuid.UUID) bool {
	if c == nil || blockerIncarnation == uuid.Nil || !c.IsAttackingIncarnation(attackerID, attackerIncarnation) {
		return false
	}
	for _, group := range c.Groups {
		if group.AttackerID != attackerID {
			continue
		}
		for i, id := range group.BlockerIDs {
			if id == blockerID && i < len(group.BlockerIncarnations) && group.BlockerIncarnations[i] == blockerIncarnation {
				return true
			}
		}
	}
	return false
}

func (r targetCombatRelation) matches(p *Permanent, card Card, g *Game) bool {
	if card == nil || p.PhasedOut {
		return false
	}
	source := g.FindPermanent(card.ID())
	if source == nil || source.PhasedOut {
		return false
	}
	if retained, ok := card.(targetSourceLKI); ok && retained.incarnation != source.incarnationID {
		return false
	}
	return r&targetBlocksSource != 0 && g.combat.BlocksIncarnations(p.ID(), p.incarnationID, source.ID(), source.incarnationID) ||
		r&targetBlockedBySource != 0 && g.combat.BlocksIncarnations(source.ID(), source.incarnationID, p.ID(), p.incarnationID)
}
