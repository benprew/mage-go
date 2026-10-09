package mage

import "slices"

type randomHandDiscardEffect struct {
	player   PlayerRef
	amount   ResolutionValue
	matcher  CardNameMatcher
	filter   CardFilter
	fullHand bool
}

// RevealRandomHandAndDiscardMatchingName reveals distinct random hand cards.
// Only matching revealed cards enter the shared discard batch.
func RevealRandomHandAndDiscardMatchingName(player PlayerRef, amount ResolutionValue, matcher CardNameMatcher) ResolvingEffect {
	return &randomHandDiscardEffect{player: player, amount: amount, matcher: matcher}
}

// RevealHandAndDiscardRandomMatching reveals the hand before a random discard.
// A hand with no matching card uses no random value.
func RevealHandAndDiscardRandomMatching(player PlayerRef, filter CardFilter) ResolvingEffect {
	return &randomHandDiscardEffect{player: player, filter: filter, fullHand: true}
}

func (e *randomHandDiscardEffect) Text() string {
	if e.fullHand {
		return "reveal the hand and discard one matching card at random"
	}
	return "reveal random hand cards and discard those with the chosen name"
}
func (*randomHandDiscardEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}
func (e *randomHandDiscardEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", children: []definitionEdge{{"player", e.player}}}
	if !e.fullHand {
		n.children = append(n.children, definitionEdge{"amount", e.amount}, definitionEdge{"name-matcher", e.matcher})
	}
	return n
}
func (e *randomHandDiscardEffect) Apply(ctx *EffectContext) error {
	player := handOperationPlayer(e.player, ctx)
	if player == nil {
		return nil
	}
	hand := slices.Clone(player.Hand())
	var revealed, discarded []Card
	if e.fullHand {
		revealed = hand
		var candidates []Card
		for _, card := range hand {
			if e.filter.Match(card) {
				candidates = append(candidates, card)
			}
		}
		discarded = ctx.Game.selectRandomCards(candidates, 1)
	} else {
		revealed = ctx.Game.selectRandomCards(hand, e.amount.resolveValue(ctx))
		for _, card := range revealed {
			if e.matcher.Match([]string{card.Name()}, ctx) {
				discarded = append(discarded, card)
			}
		}
	}
	for _, card := range revealed {
		ctx.Game.recordPublicReveal(card)
	}
	actions := make([]*DiscardAction, 0, len(discarded))
	for _, card := range discarded {
		actions = append(actions, NewDiscardAction(ctx.SourceID, player.PlayerID(), card.ID()))
	}
	ctx.Game.discardActionsTogether(actions)
	return nil
}
func (e *randomHandDiscardEffect) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player) || !e.fullHand && resolutionValueReadsSourcePermanent(e.amount)
}

func (g *Game) selectRandomCards(cards []Card, amount int) []Card {
	remaining := slices.Clone(cards)
	count := min(max(0, amount), len(remaining))
	selected := make([]Card, 0, count)
	for range count {
		index := g.RandIntn(len(remaining))
		selected = append(selected, remaining[index])
		remaining = append(remaining[:index], remaining[index+1:]...)
	}
	return selected
}
