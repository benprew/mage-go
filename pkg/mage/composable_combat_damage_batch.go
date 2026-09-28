package mage

import "github.com/benprew/mage-go/pkg/mage/core"

type combatDamageBatchPattern struct {
	player          *PlayerBinding
	capturePlayer   bool
	duplicatePlayer bool
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

func (combatDamageBatchPattern) eventPattern() {}

func (combatDamageBatchPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	return NewTriggered(core.EvtCombatDamageDealt, false, effect).SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{
		EventPlayerIsController{}, EventAmountGreaterThan{N: 0},
	}})
}

func (p combatDamageBatchPattern) inspectDefinition() definitionNode {
	n := definitionNode{Role: "event-pattern", Text: "one or more creatures you control deal combat damage to a player", battlefieldSource: true}
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
