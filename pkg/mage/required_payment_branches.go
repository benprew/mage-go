package mage

import "github.com/google/uuid"

// PayRequiredOrElse selects a branch after a mandatory resolution payment.
// Discard and matching sacrifice costs use the shared payment transaction.
// The player does not receive an optional payment choice.
func PayRequiredOrElse(player PlayerRef, costs CostGroup, paid, unpaid ResolvingEffect) ResolvingEffect {
	return &unlessPaidEffect{player: player, costs: costs, effect: paid, otherwise: unpaid, ifPaid: true, twoBranches: true, required: true}
}

type discardOneBranch struct {
	player                  PlayerRef
	discarded, notDiscarded ResolvingEffect
}

// DiscardOneOrElse selects a branch from an actual effect-caused discard.
// A changed destination counts; a canceled discard does not count.
func DiscardOneOrElse(player PlayerRef, discarded, notDiscarded ResolvingEffect) ResolvingEffect {
	return &discardOneBranch{player: player, discarded: discarded, notDiscarded: notDiscarded}
}

func (*discardOneBranch) Text() string { return "discard one card, then resolve one branch" }
func (e *discardOneBranch) Properties() EffectProperties {
	props := EffectProperties{Outcome: OutcomeDetriment}
	for _, body := range []ResolvingEffect{e.discarded, e.notDiscarded} {
		if !nilDefinition(body) {
			mergeEffectProperties(&props, body.Properties())
		}
	}
	return props
}
func (e *discardOneBranch) Apply(ctx *EffectContext) error {
	id, ok := e.player.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	player := ctx.Game.GetPlayer(id)
	if player == nil {
		return nil
	}
	discarded := false
	if len(player.Hand()) > 0 {
		visits := make(map[uuid.UUID]uuid.UUID, len(player.Hand()))
		for _, card := range player.Hand() {
			visits[card.ID()] = ctx.Game.handVisit(card.ID())
		}
		chosen := player.ChooseCardsFromHand(1, "discard", ctx.Game)
		if len(chosen) == 1 && !nilDefinition(chosen[0]) {
			if visit, present := visits[chosen[0].ID()]; present && visit != uuid.Nil {
				action := NewDiscardAction(ctx.SourceID, id, chosen[0].ID())
				action.handVisit = visit
				discarded = ctx.Game.discardActionsTogether([]*DiscardAction{action}) == 1
			}
		}
	}
	body := e.notDiscarded
	if discarded {
		body = e.discarded
	}
	if nilDefinition(body) {
		return nil
	}
	return body.Apply(ctx)
}
func (e *discardOneBranch) inspectDefinition() definitionNode {
	n := definitionNode{Role: "discard-one-branch", Text: e.Text(), children: []definitionEdge{{"player", e.player}}}
	if !nilDefinition(e.discarded) {
		n.children = append(n.children, definitionEdge{"discarded", e.discarded})
	}
	if !nilDefinition(e.notDiscarded) {
		n.children = append(n.children, definitionEdge{"not discarded", e.notDiscarded})
	}
	if nilDefinition(e.discarded) && nilDefinition(e.notDiscarded) {
		n.problem = "supply at least one discard branch"
	}
	return n
}
func (e *discardOneBranch) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player) || effectReadsSourcePermanent(e.discarded) || effectReadsSourcePermanent(e.notDiscarded)
}
