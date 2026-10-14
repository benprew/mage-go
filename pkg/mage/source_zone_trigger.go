package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type sourceCardZoneOption struct {
	zone    core.Zone
	binding *CardBinding
}

// WithSourceCardInZone saves the source visit when a zone trigger occurs.
// The source must be in its graveyard or face up in exile.
func WithSourceCardInZone(zone core.Zone, binding *CardBinding) TriggerOption {
	return sourceCardZoneOption{zone, binding}
}

func (sourceCardZoneOption) triggerOption() {}
func (o sourceCardZoneOption) applyTrigger(trigger *GenericTriggered) {
	trigger.sourceCardZoneOptions++
	trigger.sourceZone = o.zone
	trigger.sourceCardBinding = o.binding
}

func (o sourceCardZoneOption) inspectDefinition() definitionNode {
	n := definitionNode{Role: "source-zone-event", bindingWrite: o.binding, children: []definitionEdge{{"source", o.binding}}}
	if o.zone != core.ZoneGraveyard && o.zone != core.ZoneExile {
		n.problem = "source card event requires graveyard or exile"
	}
	if o.binding == nil {
		n.problem = "source card event requires a binding"
	}
	return n
}

func (g *Game) detectCardZoneTriggers(event *core.GameEvent, card Card, controller uuid.UUID, zone core.Zone) {
	for _, ability := range card.Abilities() {
		trigger, ok := UnwrapAbility(ability).(*GenericTriggered)
		if !ok || trigger.IsStateTrigger() || !trigger.FunctionsInZone(zone) || !trigger.CheckEventType(event.Type) {
			continue
		}
		bound := *trigger
		bound.SetSource(card.ID())
		bound.SetController(controller)
		if !bound.CheckTrigger(event, g) {
			continue
		}
		captures := captureDetectedTriggerEvent(&bound, event, g, g, card.ID())
		if bound.sourceCardBinding != nil {
			visit := g.graveyardVisit(card.ID())
			if zone == core.ZoneExile {
				object := g.FindExiledCard(card.ID())
				if object == nil || object.FaceDown {
					continue
				}
				visit = object.Visit
			}
			if visit == uuid.Nil {
				continue
			}
			captures = append(captures, &eventCardCapture{bound.sourceCardBinding, capturedCard{id: card.ID(), visit: visit, zone: zone}})
		}
		g.queuePendingTrigger(&pendingTrigger{
			ability: &bound, event: event, sourceID: card.ID(), controller: controller, eventCaptures: captures,
		})
	}
}

// ExiledCardFrom selects one current exile visit through a typed card reference.
func ExiledCardFrom(card CardRef) ExiledCardSet { return referencedExiledCard{card} }

type referencedExiledCard struct{ card CardRef }

func (referencedExiledCard) exiledCardSet() {}
func (s referencedExiledCard) inspectDefinition() definitionNode {
	return definitionNode{Role: "exiled-card-set", children: []definitionEdge{{"card", s.card}}}
}
func (s referencedExiledCard) resolveExiledCards(ctx *EffectContext) []capturedCard {
	card, ok := s.card.resolveCard(ctx)
	if !ok {
		return nil
	}
	object := ctx.Game.FindExiledCard(card.ID())
	if object == nil {
		return nil
	}
	return []capturedCard{{id: card.ID(), visit: object.Visit, zone: core.ZoneExile}}
}
