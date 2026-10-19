package mage

import (
	"maps"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type optionalHandChoice struct {
	receipt           *ActivationManaBinding
	requiresReceipt   bool
	player            PlayerRef
	filter            CardFilter
	result            *CardBinding
	chosen, notChosen ResolvingEffect
}

// ChooseHandCardOrElse saves a private hand visit for the chosen branch.
// Selection does not move the card. The result is local to that branch.
func ChooseHandCardOrElse(player PlayerRef, filter CardFilter, result *CardBinding, chosen, notChosen ResolvingEffect) ResolvingEffect {
	return &optionalHandChoice{player: player, filter: filter, result: result, chosen: chosen, notChosen: notChosen}
}
func (*optionalHandChoice) Text() string { return "may choose a card from hand" }
func (e *optionalHandChoice) Properties() EffectProperties {
	var props EffectProperties
	for _, body := range []ResolvingEffect{e.chosen, e.notChosen} {
		if !nilDefinition(body) {
			mergeEffectProperties(&props, body.Properties())
		}
	}
	return props
}
func (e *optionalHandChoice) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player) || effectReadsSourcePermanent(e.chosen) || effectReadsSourcePermanent(e.notChosen)
}
func (e *optionalHandChoice) inspectDefinition() definitionNode {
	n := definitionNode{Role: "optional-hand-choice", children: []definitionEdge{{"player", e.player}, {"result", e.result}}}
	if e.requiresReceipt {
		n.children = append(n.children, definitionEdge{"receipt", activationManaRead{e.receipt}})
	}
	if !nilDefinition(e.chosen) {
		n.children = append(n.children, definitionEdge{"chosen", Sequence(&handChoiceResult{e.result}, e.chosen)})
	}
	if !nilDefinition(e.notChosen) {
		n.children = append(n.children, definitionEdge{"not-chosen", e.notChosen})
	}
	if nilDefinition(e.chosen) && nilDefinition(e.notChosen) {
		n.problem = "supply at least one hand choice branch"
	}
	return n
}

type handChoiceResult struct{ result *CardBinding }

func (*handChoiceResult) Text() string                 { return "chosen hand visit" }
func (*handChoiceResult) Properties() EffectProperties { return EffectProperties{} }
func (*handChoiceResult) Apply(*EffectContext) error   { return nil }
func (e *handChoiceResult) inspectDefinition() definitionNode {
	return definitionNode{Role: "hand-choice-result", bindingWrite: e.result, children: []definitionEdge{{"result", e.result}}}
}

func (e *optionalHandChoice) Apply(ctx *EffectContext) error {
	player := handOperationPlayer(e.player, ctx)
	if player == nil {
		return nil
	}
	saved := ctx.Game.resolution.cardBindings
	ctx.Game.resolution.cardBindings = maps.Clone(saved)
	delete(ctx.Game.resolution.cardBindings, e.result)
	defer func() { ctx.Game.resolution.cardBindings = saved }()
	branch := e.notChosen
	candidates := []Card{}
	visits := map[uuid.UUID]uuid.UUID{}
	for _, card := range player.Hand() {
		visit := ctx.Game.handVisit(card.ID())
		if card.IsToken() || visit == uuid.Nil || !e.matches(ctx, card) {
			continue
		}
		visits[card.ID()] = visit
		copied := card.Copy()
		copied.SetID(card.ID())
		copied.SetOwner(card.Owner())
		candidates = append(candidates, copied)
	}
	if len(candidates) > 0 && player.ChooseMayAbility(e.Text()) {
		selected := player.ChooseCardFromHand(candidates, e.Text(), ctx.Game)
		if selected != nil {
			visit, offered := visits[selected.ID()]
			if offered && ctx.Game.handVisit(selected.ID()) == visit {
				for _, card := range player.Hand() {
					if card.ID() != selected.ID() || card.IsToken() || !e.matches(ctx, card) {
						continue
					}
					if ctx.Game.resolution.cardBindings == nil {
						ctx.Game.resolution.cardBindings = make(map[*CardBinding]capturedCard)
					}
					ctx.Game.resolution.cardBindings[e.result] = capturedCard{id: card.ID(), visit: visit, zone: core.ZoneHand}
					branch = e.chosen
					break
				}
			}
		}
	}
	if nilDefinition(branch) {
		return nil
	}
	return branch.Apply(ctx)
}
