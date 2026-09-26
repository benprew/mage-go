package mage

import (
	"maps"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type damageEventPattern struct {
	player        *PlayerBinding
	opponentOnly  bool
	amount        *NumberBinding
	captureAmount bool
	anyRecipient  bool
	combatOnly    bool
}

// DealsDamage matches damage from the source to any recipient.
// It saves the actual damage amount without declaring a target.
func DealsDamage(_ sourceRef, amount *NumberBinding) EventPattern {
	return damageEventPattern{amount: amount, captureAmount: true, anyRecipient: true}
}

// DealsDamageToPlayer matches damage from the source to either player.
// It saves the damaged player without making that player a target.
func DealsDamageToPlayer(_ sourceRef, player *PlayerBinding) damageEventPattern {
	return damageEventPattern{player: player}
}

// DealsCombatDamageToPlayer matches combat damage from the source to a player.
func DealsCombatDamageToPlayer(_ sourceRef, player *PlayerBinding) damageEventPattern {
	return damageEventPattern{player: player, combatOnly: true}
}

// DealsDamageToOpponent matches damage to an opponent of the source controller.
// It checks the controller when damage occurs and saves the damaged player.
func DealsDamageToOpponent(_ sourceRef, player *PlayerBinding) damageEventPattern {
	return damageEventPattern{player: player, opponentOnly: true}
}

// WithAmount saves the damage event amount for the trigger body.
// It preserves the player capture and does not read later source power.
func (p damageEventPattern) WithAmount(amount *NumberBinding) EventPattern {
	p.amount, p.captureAmount = amount, true
	return p
}

func (damageEventPattern) eventPattern() {}
func (p damageEventPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	if p.combatOnly {
		return NewTriggered(EvtDamageDealt, false, effect).SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{
			EventSourceIsSelf{}, EventTargetIsPlayer{}, EventIsCombatDamage{},
		}})
	}
	if p.anyRecipient {
		return NewTriggered(EvtDamageDealt, false, effect).SetConditionData(EventSourceIsSelf{})
	}
	if p.opponentOnly {
		return NewTriggered(EvtDamageDealt, false, effect).SetConditionData(EventSourceIsSelfDamageToOpponent{})
	}
	return NewTriggered(EvtDamageDealt, false, effect).SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{
		EventSourceIsSelf{}, EventTargetIsPlayer{},
	}})
}
func (p damageEventPattern) inspectDefinition() definitionNode {
	text := "source deals damage to a player"
	if p.opponentOnly {
		text = "source deals damage to an opponent"
	}
	if p.combatOnly {
		text = "source deals combat damage to a player"
	}
	n := definitionNode{Role: "event-pattern", Text: text, battlefieldSource: true,
		bindingWrite: p.player, children: []definitionEdge{{"player", p.player}}}
	if p.anyRecipient {
		n.Text, n.bindingWrite, n.children = "source deals damage", nil, nil
	}
	if p.captureAmount {
		n.children = append(n.children, definitionEdge{"amount", &eventAmountCapture{binding: p.amount}})
	}
	return n
}
func (p damageEventPattern) captureEventEffects(event *GameEvent, effects []Effect) []Effect {
	var captures []ResolvingEffect
	if !p.anyRecipient {
		captures = append(captures, &eventPlayerCapture{p.player, event.TargetID})
	}
	if p.captureAmount {
		captures = append(captures, &eventAmountCapture{p.amount, event.Amount})
	}
	captures = append(captures, Sequence(effects...))
	return []Effect{Sequence(captures...)}
}

type eventPlayerCapture struct {
	binding *PlayerBinding
	player  uuid.UUID
}

func (*eventPlayerCapture) Text() string                 { return "remember the event player" }
func (*eventPlayerCapture) Properties() EffectProperties { return EffectProperties{} }
func (e *eventPlayerCapture) Apply(ctx *EffectContext) error {
	if ctx.Game.resolution.playerBindings == nil {
		ctx.Game.resolution.playerBindings = make(map[*PlayerBinding]uuid.UUID)
	}
	ctx.Game.resolution.playerBindings[e.binding] = e.player
	return nil
}
func (e *eventPlayerCapture) inspectDefinition() definitionNode {
	return definitionNode{Role: "event-player-capture", bindingWrite: e.binding, children: []definitionEdge{{"player", e.binding}}}
}

func captureTriggerEventEffects(pt *pendingTrigger, obj *StackObject) {
	if pt.delayed != nil && pt.delayed.eventPlayerBinding != nil && pt.event != nil {
		for i, effect := range obj.Effects {
			captured, ok := effect.(*capturedDelayedBody)
			if !ok {
				continue
			}
			body := *captured
			body.players = maps.Clone(captured.players)
			if body.players == nil {
				body.players = make(map[*PlayerBinding]uuid.UUID)
			}
			body.players[pt.delayed.eventPlayerBinding] = pt.event.PlayerID
			obj.Effects[i] = &body
		}
	}
	if len(pt.eventCaptures) != 0 {
		obj.Effects = []Effect{Sequence(Sequence(pt.eventCaptures...), Sequence(obj.Effects...))}
	}
	if pt.event == nil {
		return
	}
	trigger, ok := pt.ability.(*GenericTriggered)
	if !ok {
		return
	}
	if pattern, ok := trigger.eventDefinition.(interface {
		captureEventEffects(*GameEvent, []Effect) []Effect
	}); ok {
		obj.Effects = pattern.captureEventEffects(pt.event, obj.Effects)
	}
}
