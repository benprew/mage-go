package mage

import (
	"maps"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type damageEventPattern struct {
	source                    PermanentRef
	player                    *PlayerBinding
	opponentOnly              bool
	controllerOnly            bool
	sourceController          *PlayerBinding
	captureSourceController   bool
	duplicateSourceController bool
	damageSource              *DamageSourceBinding
	captureDamageSource       bool
	duplicateDamageSource     bool
	amount                    *NumberBinding
	captureAmount             bool
	anyRecipient              bool
	combatOnly                bool
}

// DealsDamage matches damage from the source to any recipient.
// It saves the actual damage amount without declaring a target.
func DealsDamage(source PermanentRef, amount *NumberBinding) EventPattern {
	return damageEventPattern{source: source, amount: amount, captureAmount: true, anyRecipient: true}
}

// DealsDamageToPlayer matches damage from the source to either player.
// It saves the damaged player without making that player a target.
func DealsDamageToPlayer(source PermanentRef, player *PlayerBinding) damageEventPattern {
	return damageEventPattern{source: source, player: player}
}

// ToController matches damage to the trigger controller when damage occurs.
func (p damageEventPattern) ToController() damageEventPattern {
	p.controllerOnly = true
	return p
}

// WithSourceController saves the damage source controller when damage occurs.
func (p damageEventPattern) WithSourceController(binding *PlayerBinding) damageEventPattern {
	if p.captureSourceController {
		p.duplicateSourceController = true
	}
	p.sourceController, p.captureSourceController = binding, true
	return p
}

// WithSource saves the damage source when damage occurs.
func (p damageEventPattern) WithSource(binding *DamageSourceBinding) damageEventPattern {
	if p.captureDamageSource {
		p.duplicateDamageSource = true
	}
	p.damageSource, p.captureDamageSource = binding, true
	return p
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
		if _, ok := p.source.(attachedPermanentRef); ok {
			return NewTriggered(EvtDamageDealt, false, effect).SetConditionData(EventSourceIsAttachedTo{})
		}
		return NewTriggered(EvtDamageDealt, false, effect).SetConditionData(EventSourceIsSelf{})
	}
	if p.opponentOnly {
		return NewTriggered(EvtDamageDealt, false, effect).SetConditionData(EventSourceIsSelfDamageToOpponent{})
	}
	sourceCondition := TriggerConditionData(EventSourceIsSelf{})
	if _, ok := p.source.(attachedPermanentRef); ok {
		sourceCondition = EventSourceIsAttachedTo{}
	}
	conditions := []TriggerConditionData{sourceCondition, EventTargetIsPlayer{}}
	if p.controllerOnly {
		conditions = append(conditions, EventTargetIsController{})
	}
	return NewTriggered(EvtDamageDealt, false, effect).SetConditionData(AndTriggerCond{Conditions: conditions})
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
		switch p.source.(type) {
		case sourceRef:
		case attachedPermanentRef:
			n.Text = "attached permanent deals damage"
		default:
			n.problem = "damage event source requires This or AttachedPermanent"
		}
	}
	if !p.anyRecipient && !p.combatOnly && !p.opponentOnly {
		switch p.source.(type) {
		case sourceRef:
			n.Text = "source deals damage to a player"
		case attachedPermanentRef:
			n.Text = "attached permanent deals damage to a player"
		default:
			n.problem = "damage event source requires This or AttachedPermanent"
		}
	}
	if p.controllerOnly {
		if p.anyRecipient || p.combatOnly || p.opponentOnly {
			n.problem = "controller damage filter requires DealsDamageToPlayer"
		} else {
			n.Text += " to trigger controller"
		}
	}
	if p.captureSourceController {
		if p.anyRecipient || p.combatOnly || p.opponentOnly || p.sourceController == nil || p.duplicateSourceController {
			n.problem = "damage event requires one source controller binding on DealsDamageToPlayer"
		} else if p.sourceController == p.player {
			n.problem = "damage event has duplicate player and source controller binding"
		} else {
			n.children = append(n.children, definitionEdge{"source controller", &eventPlayerCapture{binding: p.sourceController}})
		}
	}
	if p.captureDamageSource {
		if p.anyRecipient || p.combatOnly || p.opponentOnly || p.damageSource == nil || p.duplicateDamageSource {
			n.problem = "damage event requires one source binding on DealsDamageToPlayer"
		} else {
			n.bindingWrites = append(n.bindingWrites, p.damageSource)
			n.children = append(n.children, definitionEdge{"source", &eventDamageSourceCapture{binding: p.damageSource}})
		}
	}
	if p.captureAmount {
		n.children = append(n.children, definitionEdge{"amount", &eventAmountCapture{binding: p.amount}})
	}
	return n
}
func (p damageEventPattern) captureDetectedEvent(_ *GameEvent, observer GameReader, _ uuid.UUID) []Effect {
	if !p.captureDamageSource || p.damageSource == nil {
		return nil
	}
	game, ok := observer.(*Game)
	if !ok {
		return nil
	}
	return []Effect{&eventDamageSourceCapture{binding: p.damageSource, source: cloneDamageSource(game.damageEventSource)}}
}
func (p damageEventPattern) captureEventEffects(event *GameEvent, effects []Effect) []Effect {
	var captures []ResolvingEffect
	if !p.anyRecipient {
		captures = append(captures, &eventPlayerCapture{p.player, event.TargetID})
	}
	if p.captureAmount {
		captures = append(captures, &eventAmountCapture{p.amount, event.Amount})
	}
	if p.captureSourceController {
		captures = append(captures, &eventPlayerCapture{binding: p.sourceController, player: event.SourceControllerID})
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
