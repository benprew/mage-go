package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type combatDamageBatchPattern struct {
	player           *PlayerBinding
	capturePlayer    bool
	duplicatePlayer  bool
	excludedSubtype  string
	filterSubtype    bool
	duplicateSubtype bool
}

// OneOrMoreCreaturesYouControlDealCombatDamageToPlayer matches one damage step group.
func OneOrMoreCreaturesYouControlDealCombatDamageToPlayer() combatDamageBatchPattern {
	return combatDamageBatchPattern{}
}

// WithPlayer saves the player who received combat damage.
func (p combatDamageBatchPattern) WithPlayer(player *PlayerBinding) combatDamageBatchPattern {
	if p.capturePlayer {
		p.duplicatePlayer = true
	}
	p.player, p.capturePlayer = player, true
	return p
}

// WithoutSubtype checks the damaging creatures when they deal damage.
func (p combatDamageBatchPattern) WithoutSubtype(subtype string) combatDamageBatchPattern {
	if p.filterSubtype {
		p.duplicateSubtype = true
	}
	p.excludedSubtype, p.filterSubtype = subtype, true
	return p
}

func (combatDamageBatchPattern) eventPattern() {}

func (p combatDamageBatchPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	conditions := []TriggerConditionData{
		EventPlayerIsController{}, EventAmountGreaterThan{N: 0},
	}
	if p.filterSubtype {
		conditions = append(conditions, combatDamageSourceWithoutSubtype{subtype: p.excludedSubtype})
	}
	return NewTriggered(core.EvtCombatDamageDealt, false, effect).SetConditionData(AndTriggerCond{Conditions: conditions})
}

type combatDamageSourceWithoutSubtype struct{ subtype string }

func (c combatDamageSourceWithoutSubtype) CheckTriggerCond(event *core.GameEvent, reader GameReader, _, _ uuid.UUID) bool {
	game, ok := reader.(*Game)
	if !ok {
		return false
	}
	for _, source := range game.damage.combatDamageSourceStatesThisStep[event.PlayerID][event.TargetID] {
		if source.HasType(core.TypeCreature) && !source.HasSubType(c.subtype) {
			return true
		}
	}
	return false
}

func (p combatDamageBatchPattern) inspectDefinition() definitionNode {
	n := definitionNode{Role: "event-pattern", Text: "one or more creatures you control deal combat damage to a player", battlefieldSource: true}
	if p.filterSubtype {
		if p.excludedSubtype == "" || p.duplicateSubtype {
			n.problem = "combat damage group: supply one creature subtype"
		} else if family, known := knownSubtypeFamily(p.excludedSubtype); known && family != SubtypeCreature {
			n.problem = "combat damage group: subtype is not a creature subtype"
		}
		n.Text = "one or more creatures you control without " + p.excludedSubtype + " deal combat damage to a player"
	}
	if p.capturePlayer {
		if p.player == nil || p.duplicatePlayer {
			n.problem = "combat damage group: supply one player binding"
		} else {
			n.bindingWrite = p.player
			n.children = append(n.children, definitionEdge{"player", &eventPlayerCapture{binding: p.player}})
		}
	}
	return n
}

func (p combatDamageBatchPattern) captureEventEffects(event *core.GameEvent, effects []Effect) []Effect {
	if !p.capturePlayer {
		return effects
	}
	return []Effect{Sequence(&eventPlayerCapture{binding: p.player, player: event.TargetID}, Sequence(effects...))}
}
