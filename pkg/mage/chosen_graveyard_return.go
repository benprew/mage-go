package mage

import "github.com/google/uuid"

// ReturnChosenGraveyardCard makes a required choice for the effect controller.
// The selected graveyard visit must still be present when the card returns.
func ReturnChosenGraveyardCard(graveyard, controller PlayerRef, filter CardFilter) ResolvingEffect {
	return &chosenGraveyardReturn{graveyard: graveyard, controller: controller, filter: filter}
}

// ReturnChosenGraveyardCardToOwner returns the choice under its owner's control.
func ReturnChosenGraveyardCardToOwner(graveyard PlayerRef, filter CardFilter) ResolvingEffect {
	return &chosenGraveyardReturn{graveyard: graveyard, filter: filter, ownerControl: true}
}

type chosenGraveyardReturn struct {
	graveyard, controller PlayerRef
	filter                CardFilter
	ownerControl          bool
}

func (*chosenGraveyardReturn) Text() string {
	return "return a chosen graveyard card to the battlefield"
}
func (*chosenGraveyardReturn) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}
func (e *chosenGraveyardReturn) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"graveyard", e.graveyard}}}
	if !e.ownerControl {
		n.children = append(n.children, definitionEdge{"controller", e.controller})
	}
	return n
}
func (e *chosenGraveyardReturn) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.graveyard) || playerRefReadsSourcePermanent(e.controller)
}
func (e *chosenGraveyardReturn) Apply(ctx *EffectContext) error {
	graveyardID, ok := e.graveyard.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	graveyard := ctx.Game.GetPlayer(graveyardID)
	chooser := ctx.Game.GetPlayer(ctx.Controller)
	if graveyard == nil || chooser == nil {
		return nil
	}
	controller := uuid.Nil
	if !e.ownerControl {
		controller, ok = e.controller.resolvePlayer(ctx)
		if !ok {
			return nil
		}
	}
	var candidates []Card
	visits := make(map[uuid.UUID]uuid.UUID)
	for _, card := range graveyard.Graveyard() {
		if e.filter.Match(card) {
			candidates = append(candidates, card)
			visits[card.ID()] = ctx.Game.graveyardVisit(card.ID())
		}
	}
	if len(candidates) == 0 {
		return nil
	}
	selected := candidates[0]
	if len(candidates) > 1 {
		chosen := chooser.ChooseCardFromGraveyard(candidates, "Choose a card to return to the battlefield", ctx.Game)
		if chosen != nil {
			for _, card := range candidates {
				if card.ID() == chosen.ID() {
					selected = card
					break
				}
			}
		}
	}
	visit := visits[selected.ID()]
	if visit == uuid.Nil || ctx.Game.graveyardVisit(selected.ID()) != visit {
		return nil
	}
	present := false
	for _, card := range graveyard.Graveyard() {
		if card.ID() == selected.ID() {
			present = true
			break
		}
	}
	if !present {
		return nil
	}
	if e.ownerControl {
		controller = selected.Owner()
	}
	if ctx.Game.GetPlayer(controller) == nil {
		return nil
	}
	returnGraveyardCardToBattlefield(ctx.Game, selected.ID(), controller)
	return nil
}
