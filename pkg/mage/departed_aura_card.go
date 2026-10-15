package mage

import (
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// WithDepartedAuraCard saves the Aura's graveyard visit permitted by CR 400.7f.
func (p creatureDiesPattern) WithDepartedAuraCard(card *CardBinding) creatureDiesPattern {
	p.departedAuraCount++
	p.departedAura = card
	return p
}

type pendingAuraCardCapture struct {
	binding    *CardBinding
	aura, host capturedPermanent
}

func (*pendingAuraCardCapture) Text() string                 { return "remember the departed Aura card" }
func (*pendingAuraCardCapture) Properties() EffectProperties { return EffectProperties{} }
func (e *pendingAuraCardCapture) Apply(ctx *EffectContext) error {
	return (&eventCardCapture{binding: e.binding}).Apply(ctx)
}
func (e *pendingAuraCardCapture) inspectDefinition() definitionNode {
	return definitionNode{Role: "event-aura-card-capture", bindingWrite: e.binding, children: []definitionEdge{{"card", e.binding}}}
}

func (g *Game) captureDepartedAuraCard(aura, host capturedPermanent, card capturedCard) {
	for _, pending := range g.triggers.pending {
		for i, effect := range pending.eventCaptures {
			capture, ok := effect.(*pendingAuraCardCapture)
			if !ok || capture.aura != aura || capture.host != host {
				continue
			}
			pending.eventCaptures = slices.Clone(pending.eventCaptures)
			pending.eventCaptures[i] = &eventCardCapture{binding: capture.binding, value: card}
		}
	}
}

func (g *Game) captureBatchAuraCards(events []core.GameEvent, cards map[capturedPermanent]capturedCard) {
	for _, event := range events {
		if event.Type != core.EvtZoneChange || event.ToZone != core.ZoneGraveyard {
			continue
		}
		aura := capturedPermanent{event.SourceID, event.SourceIncarnation}
		lki := g.zones.LKIByIncarnation(aura.incarnation)
		if lki == nil || lki.Snapshot == nil || !lki.HasSubType("Aura") {
			continue
		}
		host := capturedPermanent{lki.Snapshot.AttachedTo, lki.Snapshot.attachedIncarnation}
		for _, departure := range events {
			if departure.Type == core.EvtZoneChange && departure.ToZone == core.ZoneGraveyard && departure.SourceID == host.id && departure.SourceIncarnation == host.incarnation {
				g.captureDepartedAuraCard(aura, host, cards[aura])
				break
			}
		}
	}
}

func (p creatureDiesPattern) captureAuraCard(event *core.GameEvent, observer GameReader, source uuid.UUID) []Effect {
	if p.departedAuraCount == 0 || p.departedAura == nil {
		return nil
	}
	aura := observer.FindPermanent(source)
	if aura == nil || !aura.HasSubType("Aura") {
		return nil
	}
	return []Effect{&pendingAuraCardCapture{binding: p.departedAura, aura: capturedPermanent{source, aura.incarnationID}, host: capturedPermanent{event.SourceID, event.SourceIncarnation}}}
}
