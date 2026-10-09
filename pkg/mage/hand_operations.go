package mage

import "github.com/google/uuid"

type revealHandForEffect struct{ player PlayerRef }

// RevealHandFor shows the selected player's current hand to all players.
func RevealHandFor(player PlayerRef) ResolvingEffect      { return &revealHandForEffect{player} }
func (*revealHandForEffect) Text() string                 { return "reveal the selected player's hand" }
func (*revealHandForEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *revealHandForEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"player", e.player}}}
}
func (e *revealHandForEffect) Apply(ctx *EffectContext) error {
	if player := handOperationPlayer(e.player, ctx); player != nil {
		for _, card := range ctx.Game.RevealHand(ctx.Game.GetPlayer(ctx.Controller), player) {
			ctx.Game.recordPublicReveal(card)
		}
	}
	return nil
}

type discardMatchingHandEffect struct {
	player      PlayerRef
	filter      CardFilter
	nameMatcher CardNameMatcher
	matchName   bool
}

// DiscardMatchingHand discards all matching hand cards in one batch.
func DiscardMatchingHand(player PlayerRef, filter CardFilter) ResolvingEffect {
	return &discardMatchingHandEffect{player: player, filter: filter}
}

// DiscardHandMatchingName checks a saved name before the shared discard batch.
// All subjects come from the selected player's current hand.
func DiscardHandMatchingName(player PlayerRef, matcher CardNameMatcher) ResolvingEffect {
	return &discardMatchingHandEffect{player: player, nameMatcher: matcher, matchName: true}
}
func (*discardMatchingHandEffect) Text() string {
	return "discard all matching cards in the selected player's hand"
}
func (*discardMatchingHandEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}
func (e *discardMatchingHandEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"player", e.player}}}
	if e.matchName {
		n.children = append(n.children, definitionEdge{"name-matcher", e.nameMatcher})
	}
	return n
}
func (e *discardMatchingHandEffect) Apply(ctx *EffectContext) error {
	player := handOperationPlayer(e.player, ctx)
	if player == nil {
		return nil
	}
	var actions []*DiscardAction
	for _, card := range player.Hand() {
		matches := e.filter.Match(card)
		if e.matchName {
			matches = e.nameMatcher.Match([]string{card.Name()}, ctx)
		}
		if matches {
			actions = append(actions, NewDiscardAction(ctx.SourceID, player.PlayerID(), card.ID()))
		}
	}
	ctx.Game.discardActionsTogetherWithEvents(actions, false)
	return nil
}

type matchingHandCount struct {
	player PlayerRef
	filter CardFilter
}

// CountMatchingHand reads the selected player's current matching card count.
func CountMatchingHand(player PlayerRef, filter CardFilter) ResolutionValue {
	return matchingHandCount{player, filter}
}
func (matchingHandCount) Text() string {
	return "number of matching cards in the selected player's hand"
}
func (v matchingHandCount) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolution-value", Text: v.Text(), children: []definitionEdge{{"player", v.player}}}
}
func (v matchingHandCount) resolveValue(ctx *EffectContext) int {
	player := handOperationPlayer(v.player, ctx)
	if player == nil {
		return 0
	}
	count := 0
	for _, card := range player.Hand() {
		if v.filter.Match(card) {
			count++
		}
	}
	return count
}

func handOperationPlayer(ref PlayerRef, ctx *EffectContext) Player {
	if ref == nil {
		return nil
	}
	id, ok := ref.resolvePlayer(ctx)
	if !ok || id == uuid.Nil {
		return nil
	}
	return ctx.Game.GetPlayer(id)
}
