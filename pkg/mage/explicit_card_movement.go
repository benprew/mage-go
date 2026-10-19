package mage

import (
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type explicitCardMovement struct {
	card           CardRef
	player         PlayerRef
	physicalSource bool
}

// PutHandCardInHandFor moves one exact hand visit to the selected player's hand.
// The owner's corresponding zone takes precedence under CR 400.3.
func PutHandCardInHandFor(card CardRef, player PlayerRef) ResolvingEffect {
	return &explicitCardMovement{card: card, player: player}
}

// PutSourceCardInGraveyardFromAnywhere finds the physical source across zones.
// This permission does not extend the lifetime of ordinary card references.
func PutSourceCardInGraveyardFromAnywhere(player PlayerRef) ResolvingEffect {
	return &explicitCardMovement{player: player, physicalSource: true}
}

func (*explicitCardMovement) Text() string                 { return "move the card to the selected zone" }
func (*explicitCardMovement) Properties() EffectProperties { return EffectProperties{} }
func (e *explicitCardMovement) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", children: []definitionEdge{{"player", e.player}}}
	if !e.physicalSource {
		n.children = append(n.children, definitionEdge{"card", e.card})
	}
	return n
}
func (e *explicitCardMovement) refersToSourcePermanent() bool {
	return e.physicalSource || playerRefReadsSourcePermanent(e.player)
}
func (e *explicitCardMovement) Apply(ctx *EffectContext) error {
	g := ctx.Game
	player, ok := e.player.resolvePlayer(ctx)
	if !ok || g.GetPlayer(player) == nil {
		return nil
	}
	var card Card
	destination := core.ZoneHand
	if e.physicalSource {
		card = g.FindCardAnywhere(ctx.SourceID)
		destination = core.ZoneGraveyard
	} else {
		card, ok = e.card.resolveCard(ctx)
		if !ok || g.physicalCardZone(card.ID()) != core.ZoneHand {
			return nil
		}
	}
	if card == nil || card.IsToken() || g.GetPlayer(card.Owner()) == nil {
		return nil
	}
	from := g.physicalCardZone(card.ID())
	if from == core.ZoneBattlefield {
		g.PutPermanentIntoGraveyard(g.FindPermanent(card.ID()))
		return nil
	}
	visit, storage := g.cardMovementVisit(card.ID(), from)
	if visit == uuid.Nil {
		return nil
	}
	if from == destination && storage == card.Owner() {
		return nil
	}
	proposal := &CardZoneMoveAction{source: ctx.SourceID, card: card.ID(), visit: visit, from: from, destination: destination, owner: card.Owner()}
	result, ok := g.effects.ApplyReplacements(proposal, g).(*CardZoneMoveAction)
	if !ok || result == nil {
		return nil
	}
	g.performCardZoneMove(result)
	return nil
}

// CardZoneMoveAction retains one nonbattlefield visit through replacements.
// Battlefield movement uses the existing permanent movement actions.
type CardZoneMoveAction struct {
	source, card, visit, owner, exileSource uuid.UUID
	from, destination                       core.Zone
}

func (a *CardZoneMoveAction) ActionSource() uuid.UUID { return a.source }
func (a *CardZoneMoveAction) CardID() uuid.UUID       { return a.card }
func (a *CardZoneMoveAction) PlayerID() uuid.UUID     { return a.owner }
func (a *CardZoneMoveAction) FromZone() core.Zone     { return a.from }
func (a *CardZoneMoveAction) Destination() core.Zone  { return a.destination }

// WithExile replaces the destination without changing the saved subject.
func (a *CardZoneMoveAction) WithExile(source uuid.UUID) *CardZoneMoveAction {
	clone := *a
	clone.destination, clone.exileSource = core.ZoneExile, source
	return &clone
}

func (g *Game) physicalCardZone(id uuid.UUID) core.Zone {
	if g.FindPermanentIncludingPhased(id) != nil {
		return core.ZoneBattlefield
	}
	for _, zone := range []core.Zone{core.ZoneHand, core.ZoneLibrary, core.ZoneGraveyard, core.ZoneExile, core.ZoneStack, core.ZoneAnte, core.ZoneCommand} {
		if visit, _ := g.cardMovementVisit(id, zone); visit != uuid.Nil {
			return zone
		}
	}
	return core.ZoneAny
}

func (g *Game) cardMovementVisit(id uuid.UUID, zone core.Zone) (visit, storage uuid.UUID) {
	for _, p := range g.players {
		switch zone {
		case core.ZoneHand:
			if visitor, ok := p.(interface{ handVisit(uuid.UUID) uuid.UUID }); ok && visitor.handVisit(id) != uuid.Nil {
				return visitor.handVisit(id), p.PlayerID()
			}
		case core.ZoneGraveyard:
			if visitor, ok := p.(interface{ graveyardVisit(uuid.UUID) uuid.UUID }); ok && visitor.graveyardVisit(id) != uuid.Nil {
				return visitor.graveyardVisit(id), p.PlayerID()
			}
		case core.ZoneLibrary:
			if visitor, ok := p.(interface{ libraryVisit(uuid.UUID) uuid.UUID }); ok && visitor.libraryVisit(id) != uuid.Nil {
				return visitor.libraryVisit(id), p.PlayerID()
			}
		case core.ZoneAnte:
			if visitor, ok := p.(interface{ anteVisit(uuid.UUID) uuid.UUID }); ok && visitor.anteVisit(id) != uuid.Nil {
				return visitor.anteVisit(id), p.PlayerID()
			}
		}
	}
	if zone == core.ZoneExile {
		if c := g.FindExiledCard(id); c != nil {
			return c.Visit, c.Card.Owner()
		}
	}
	if zone == core.ZoneCommand {
		for _, c := range g.zones.commandObjects {
			if c.card.ID() == id {
				return c.visit, c.card.Owner()
			}
		}
	}
	if zone == core.ZoneStack {
		for _, obj := range append(slices.Clone(g.stack.objects), g.stack.resolvingReferenceObjects...) {
			if !obj.leftStack && !obj.IsAbility && obj.Card != nil && obj.Card.ID() == id {
				return obj.ID, obj.Card.Owner()
			}
		}
	}
	return uuid.Nil, uuid.Nil
}

func (g *Game) performCardZoneMove(a *CardZoneMoveAction) {
	for _, event := range g.performCardZoneMoveEvents(a) {
		g.FireEvent(event)
	}
}

func (g *Game) performCardZoneMoveEvents(a *CardZoneMoveAction) []core.GameEvent {
	visit, storage := g.cardMovementVisit(a.card, a.from)
	if visit != a.visit || g.physicalCardZone(a.card) != a.from {
		return nil
	}
	card := g.FindCardAnywhere(a.card)
	if card == nil || card.IsToken() {
		return nil
	}
	owner := g.GetPlayer(card.Owner())
	if owner == nil {
		return nil
	}
	if a.destination != core.ZoneHand && a.destination != core.ZoneGraveyard && a.destination != core.ZoneExile && a.destination != core.ZoneLibrary {
		return nil
	}
	if a.from == a.destination && storage == owner.PlayerID() {
		return nil
	}
	var moved Card
	switch a.from {
	case core.ZoneStack:
		for _, obj := range append(slices.Clone(g.stack.objects), g.stack.resolvingReferenceObjects...) {
			if obj.ID != a.visit || obj.leftStack || obj.IsAbility || obj.Card == nil || obj.Card.ID() != a.card {
				continue
			}
			g.stack.RemoveByID(obj.ID)
			g.stack.saveSpellInformation(obj)
			obj.leftStack = true
			moved = g.revealFaceDownCard(obj.Card)
			break
		}
	case core.ZoneGraveyard:
		moved, _ = g.GetPlayer(storage).RemoveFromGraveyard(a.card)
	case core.ZoneAnte:
		moved, _ = g.removeAnteVisit(capturedCard{id: a.card, visit: a.visit, zone: core.ZoneAnte})
	case core.ZoneCommand:
		moved, _ = g.RemoveCommandObject(a.visit)
	default:
		moved, _, _, _ = g.removeCardForAnte(a.card, a.from)
	}
	if moved == nil {
		return nil
	}
	switch a.destination {
	case core.ZoneLibrary:
		owner.SetLibrary(append([]Card{moved}, owner.Library()...))
	case core.ZoneHand:
		owner.AddToHand(moved)
	case core.ZoneGraveyard:
		owner.AddToGraveyard(moved)
	case core.ZoneExile:
		g.ExileCard(moved, a.exileSource)
	}
	events := []core.GameEvent{{Type: core.EvtZoneChange, SourceID: a.card, PlayerID: owner.PlayerID(), FromZone: a.from, ToZone: a.destination}}
	if a.from == core.ZoneGraveyard {
		events = append(events, core.GameEvent{Type: core.EvtCardsLeftGraveyard, PlayerID: storage, Amount: 1})
	}
	return events
}
