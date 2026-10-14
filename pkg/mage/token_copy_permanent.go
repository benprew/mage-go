package mage

import "fmt"

// CreateTokenCopiesOfPermanent copies one present subject before batch entry.
// Each token receives fresh rules from the subject's copiable values.
func CreateTokenCopiesOfPermanent(player PlayerRef, count ResolutionValue, subject PermanentRef) ResolvingEffect {
	return &createPermanentTokenCopiesEffect{player: player, count: count, subject: subject}
}

// CreateTokenCopiesOfPermanentLinked retains the producing source's exact visit.
// Entry observations can read the link on every accepted token.
func CreateTokenCopiesOfPermanentLinked(player PlayerRef, count ResolutionValue, subject PermanentRef, link *CreatedTokenLink) ResolvingEffect {
	return &createPermanentTokenCopiesEffect{player: player, count: count, subject: subject, link: link, linked: true}
}

type createPermanentTokenCopiesEffect struct {
	player  PlayerRef
	count   ResolutionValue
	subject PermanentRef
	link    *CreatedTokenLink
	linked  bool
}

func (*createPermanentTokenCopiesEffect) Text() string { return "create tokens that copy a permanent" }
func (*createPermanentTokenCopiesEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}
func (e *createPermanentTokenCopiesEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"player", e.player}, {"count", e.count}, {"subject", e.subject}}}
	if e.linked {
		n.children = append(n.children, definitionEdge{"link", e.link}, definitionEdge{"source", This})
	}
	return n
}
func (e *createPermanentTokenCopiesEffect) refersToSourcePermanent() bool {
	return e.linked || playerRefReadsSourcePermanent(e.player) || resolutionValueReadsSourcePermanent(e.count) || permanentRefReadsSourcePermanent(e.subject)
}
func (e *createPermanentTokenCopiesEffect) Apply(ctx *EffectContext) error {
	player, ok := e.player.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	count := e.count.resolveValue(ctx)
	if count <= 0 {
		return nil
	}
	subject, ok := e.subject.resolvePermanent(ctx)
	if !ok {
		return nil
	}
	var source capturedPermanent
	if e.linked {
		source, ok = This.informationIdentity(ctx)
		if !ok {
			return nil
		}
	}
	values, err := ctx.Game.CaptureCopiableValues(subject)
	if err != nil {
		return fmt.Errorf("token copy subject: %w", err)
	}
	cards := make([]Card, 0, count)
	for range count {
		card, err := values.Build()
		if err != nil {
			return fmt.Errorf("token copy definition: %w", err)
		}
		card.(*BaseCard).isToken = true
		card.SetOwner(player)
		cards = append(cards, card)
	}
	ctx.Game.putTokensOnBattlefieldTogetherWithLink(cards, player, e.link, source)
	return nil
}
