package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// ControllersSacrificeEach has each current controller sacrifice selected subjects.
// All replacement and graveyard order choices precede simultaneous movement.
func ControllersSacrificeEach(subjects PermanentSet) ResolvingEffect {
	return &controllersSacrificeEachEffect{subjects: subjects}
}

type controllersSacrificeEachEffect struct{ subjects PermanentSet }

func (*controllersSacrificeEachEffect) Text() string { return "controllers sacrifice each permanent" }
func (*controllersSacrificeEachEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment, Mass: true}
}
func (e *controllersSacrificeEachEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"subjects", e.subjects}}}
}
func (e *controllersSacrificeEachEffect) refersToSourcePermanent() bool {
	return permanentSetReadsSourcePermanent(e.subjects)
}
func (e *controllersSacrificeEachEffect) Apply(ctx *EffectContext) error {
	ctx.Game.controllersSacrificePermanents(e.subjects.resolvePermanents(ctx))
	return nil
}

func (g *Game) controllersSacrificePermanents(subjects []*Permanent) {
	var selected []*Permanent
	seen := make(map[uuid.UUID]bool, len(subjects))
	for _, p := range subjects {
		if p == nil || seen[p.ID()] {
			continue
		}
		current := g.FindPermanent(p.ID())
		if current == nil || current.PhasedOut || current.incarnationID != p.incarnationID || g.GetPlayer(current.ControllerID()) == nil {
			continue
		}
		seen[p.ID()] = true
		selected = append(selected, current)
	}
	if len(selected) == 0 {
		return
	}
	g.withZoneChangeBatch(func() {
		var prepared []*SacrificePermanentAction
		for _, player := range []Player{g.ActivePlayerObj(), g.NonActivePlayerObj()} {
			for _, p := range selected {
				if p.ControllerID() != player.PlayerID() {
					continue
				}
				if result := g.prepareSacrifice(p); result != nil {
					prepared = append(prepared, result)
				}
			}
		}
		g.orderSacrificeGraveyards(prepared)
		for _, result := range prepared {
			g.performSacrifice(result)
		}
	})
}

func (g *Game) orderSacrificeGraveyards(actions []*SacrificePermanentAction) {
	for _, owner := range []Player{g.ActivePlayerObj(), g.NonActivePlayerObj()} {
		var cards []Card
		var positions []int
		byID := make(map[uuid.UUID]*SacrificePermanentAction)
		for index, action := range actions {
			p := g.FindPermanent(action.permanentID)
			if p == nil || p.PhasedOut || p.incarnationID != action.incarnation || p.IsToken || action.destination != core.ZoneGraveyard {
				continue
			}
			ownerID := p.Card.Owner()
			if ownerID == uuid.Nil {
				ownerID = p.ControllerID()
			}
			if ownerID != owner.PlayerID() {
				continue
			}
			cards = append(cards, cardForBattlefieldDeparture(p.Card))
			positions = append(positions, index)
			byID[p.ID()] = action
		}
		for index, card := range g.orderCardsForGraveyard(owner, cards) {
			actions[positions[index]] = byID[card.ID()]
		}
	}
}
