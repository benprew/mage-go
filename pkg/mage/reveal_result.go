package mage

import (
	"slices"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// Saved frames share this private snapshot because its fields do not change.
type revealedCardFacts struct {
	name      string
	types     []core.CardType
	manaValue int
}

func captureRevealedCardFacts(card Card) *revealedCardFacts {
	return &revealedCardFacts{name: card.Name(), types: slices.Clone(card.Types()), manaValue: card.ManaCost().CMC()}
}

type revealedCardManaValue struct{ result *CardBinding }

// RevealedCardManaValue reads the mana value saved by a reveal operation.
// Missing reveal facts supply zero and give no permission to act on a card.
func RevealedCardManaValue(result *CardBinding) ResolutionValue {
	return revealedCardManaValue{result: result}
}
func (revealedCardManaValue) Text() string { return "mana value of the saved reveal" }
func (v revealedCardManaValue) resolveValue(ctx *EffectContext) int {
	facts := ctx.Game.resolution.cardBindings[v.result].revealed
	if facts == nil {
		return 0
	}
	return facts.manaValue
}
func (v revealedCardManaValue) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolution-value", Text: v.Text(), bindingRead: v.result, children: []definitionEdge{{"result", v.result}}}
}

// RevealedCardMatcher checks the facts saved by a reveal operation.
// A match does not permit an action on an absent card.
type RevealedCardMatcher interface {
	matchRevealed(revealedCardFacts, *EffectContext) bool
	inspectDefinition() definitionNode
}

type revealedCardTypeMatcher struct {
	kind  core.CardType
	lacks bool
}

// RevealedCardHasType checks the card type saved at the reveal.
func RevealedCardHasType(kind core.CardType) RevealedCardMatcher {
	return revealedCardTypeMatcher{kind: kind}
}

// RevealedCardLacksType checks that the revealed card did not have this type.
func RevealedCardLacksType(kind core.CardType) RevealedCardMatcher {
	return revealedCardTypeMatcher{kind: kind, lacks: true}
}
func (m revealedCardTypeMatcher) matchRevealed(facts revealedCardFacts, _ *EffectContext) bool {
	return slices.Contains(facts.types, m.kind) != m.lacks
}
func (m revealedCardTypeMatcher) inspectDefinition() definitionNode {
	n := definitionNode{Role: "revealed-card-matcher", Text: "saved reveal card type"}
	if !m.kind.IsACardType() {
		n.problem = "supply a valid card type"
	}
	return n
}

type revealedCardNameMatcher struct{ matcher CardNameMatcher }

// RevealedCardNamed checks a saved reveal name with the shared name matcher.
func RevealedCardNamed(matcher CardNameMatcher) RevealedCardMatcher {
	return revealedCardNameMatcher{matcher: matcher}
}
func (m revealedCardNameMatcher) matchRevealed(facts revealedCardFacts, ctx *EffectContext) bool {
	return m.matcher.Match([]string{facts.name}, ctx)
}
func (m revealedCardNameMatcher) inspectDefinition() definitionNode {
	return definitionNode{Role: "revealed-card-matcher", children: []definitionEdge{{"name", m.matcher}}}
}

type revealedCardBranch struct {
	result             *CardBinding
	matcher            RevealedCardMatcher
	matched, unmatched ResolvingEffect
}

// IfRevealedCardMatches checks one saved reveal and runs one result branch.
// Missing reveal facts select the unmatched branch, including for a negative type.
// A nil branch makes no change. Supply at least one branch.
func IfRevealedCardMatches(result *CardBinding, matcher RevealedCardMatcher, matched, unmatched ResolvingEffect) ResolvingEffect {
	return &revealedCardBranch{result: result, matcher: matcher, matched: matched, unmatched: unmatched}
}
func (*revealedCardBranch) Text() string { return "if the saved reveal matches" }
func (e *revealedCardBranch) Properties() EffectProperties {
	effects := make([]Effect, 0, 2)
	if e.matched != nil {
		effects = append(effects, e.matched)
	}
	if e.unmatched != nil {
		effects = append(effects, e.unmatched)
	}
	return CompositeEffects("", effects...).Properties()
}
func (e *revealedCardBranch) Apply(ctx *EffectContext) error {
	facts := ctx.Game.resolution.cardBindings[e.result].revealed
	body := e.unmatched
	if facts != nil && e.matcher.matchRevealed(*facts, ctx) {
		body = e.matched
	}
	if body == nil {
		return nil
	}
	return body.Apply(ctx)
}
func (e *revealedCardBranch) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolution-branch", bindingRead: e.result, children: []definitionEdge{{"result", e.result}, {"matcher", e.matcher}}}
	if e.matched != nil {
		n.children = append(n.children, definitionEdge{"then", e.matched})
	}
	if e.unmatched != nil {
		n.children = append(n.children, definitionEdge{"else", e.unmatched})
	}
	if e.matched == nil && e.unmatched == nil {
		n.problem = "supply at least one result branch"
	}
	return n
}

func (e *revealedCardBranch) refersToSourcePermanent() bool {
	return effectReadsSourcePermanent(e.matched) || effectReadsSourcePermanent(e.unmatched)
}
