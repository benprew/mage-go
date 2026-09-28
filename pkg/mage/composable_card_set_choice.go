package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type upToOneCardChoice struct {
	player   PlayerRef
	subjects CardSet
	filter   CardFilter
	binding  *CardBinding
}

// ChooseUpToOneCard saves an optional graveyard card choice at resolution.
func ChooseUpToOneCard(player PlayerRef, subjects CardSet, filter CardFilter, binding *CardBinding) ResolvingEffect {
	return &upToOneCardChoice{player: player, subjects: subjects, filter: filter, binding: binding}
}

func (*upToOneCardChoice) Text() string                 { return "choose up to one card" }
func (*upToOneCardChoice) Properties() EffectProperties { return EffectProperties{} }
func (e *upToOneCardChoice) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), bindingWrite: e.binding,
		children: []definitionEdge{{"player", e.player}, {"cards", e.subjects}, {"binding", e.binding}}}
}
func (e *upToOneCardChoice) Apply(ctx *EffectContext) error {
	delete(ctx.Game.resolution.cardBindings, e.binding)
	id, ok := e.player.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	player := ctx.Game.GetPlayer(id)
	if player == nil {
		return nil
	}
	var candidates []Card
	visits := make(map[uuid.UUID]uuid.UUID)
	for _, card := range e.subjects.resolveCards(ctx) {
		if e.filter.Match(card) {
			candidates = append(candidates, card)
			visits[card.ID()] = ctx.Game.graveyardVisit(card.ID())
		}
	}
	if len(candidates) == 0 || !player.ChooseMayAbility("choose "+e.binding.name+" from graveyard") {
		return nil
	}
	chosen := player.ChooseCardFromGraveyard(candidates, "choose "+e.binding.name, ctx.Game)
	if chosen == nil {
		return nil
	}
	for _, card := range candidates {
		if card.ID() != chosen.ID() {
			continue
		}
		visit := visits[card.ID()]
		if visit == uuid.Nil || ctx.Game.graveyardVisit(card.ID()) != visit {
			return nil
		}
		if ctx.Game.resolution.cardBindings == nil {
			ctx.Game.resolution.cardBindings = make(map[*CardBinding]capturedCard)
		}
		ctx.Game.resolution.cardBindings[e.binding] = capturedCard{id: card.ID(), visit: visit, zone: core.ZoneGraveyard}
		return nil
	}
	return nil
}
