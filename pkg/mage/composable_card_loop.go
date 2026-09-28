package mage

import "maps"

type savedCardLoop struct {
	cards  *CardsBinding
	member *CardBinding
	body   ResolvingEffect
}

// ForEachSavedCard runs the body once for each card saved by one instruction.
// Each member keeps its original zone visit, even if an earlier body moves it.
func ForEachSavedCard(cards *CardsBinding, member *CardBinding, body ResolvingEffect) ResolvingEffect {
	return &savedCardLoop{cards: cards, member: member, body: body}
}

func (*savedCardLoop) Text() string { return "for each saved card" }
func (e *savedCardLoop) Properties() EffectProperties {
	var props EffectProperties
	if !nilDefinition(e.body) {
		props = e.body.Properties()
	}
	props.Mass = true
	return props
}
func (e *savedCardLoop) inspectDefinition() definitionNode {
	n := definitionNode{Role: "saved-card-loop", isolatedBindings: true, children: []definitionEdge{
		{"cards", savedCardsSet{binding: e.cards}}, {"member", savedCardLoopMember{e.member}}, {"body", e.body},
	}}
	if e.cards == nil || e.member == nil || nilDefinition(e.body) {
		n.problem = "supply non-nil saved cards, member, and body"
	}
	return n
}
func (e *savedCardLoop) Apply(ctx *EffectContext) error {
	members := append([]capturedCard(nil), ctx.Game.resolution.cardsBindings[e.cards]...)
	state := &ctx.Game.resolution
	outer := *state
	defer func() {
		state.permanentBindings = outer.permanentBindings
		state.createdTokenBindings = outer.createdTokenBindings
		state.cardBindings = outer.cardBindings
		state.cardsBindings = outer.cardsBindings
		state.spellBindings = outer.spellBindings
		state.playerBindings = outer.playerBindings
		state.numberBindings = outer.numberBindings
		state.manaTypesBindings = outer.manaTypesBindings
		state.damageSourceBindings = outer.damageSourceBindings
		state.damageResultsBindings = outer.damageResultsBindings
		state.damageResultBindings = outer.damageResultBindings
	}()
	for _, captured := range members {
		state.permanentBindings = maps.Clone(outer.permanentBindings)
		state.createdTokenBindings = cloneCreatedTokenBindings(outer.createdTokenBindings)
		state.cardBindings = maps.Clone(outer.cardBindings)
		state.cardsBindings = cloneCardsBindings(outer.cardsBindings)
		state.spellBindings = maps.Clone(outer.spellBindings)
		state.playerBindings = maps.Clone(outer.playerBindings)
		state.numberBindings = maps.Clone(outer.numberBindings)
		state.manaTypesBindings = cloneManaTypesBindings(outer.manaTypesBindings)
		state.damageSourceBindings = cloneDamageSourceBindings(outer.damageSourceBindings)
		state.damageResultsBindings = cloneDamageResultsBindings(outer.damageResultsBindings)
		state.damageResultBindings = maps.Clone(outer.damageResultBindings)
		if state.cardBindings == nil {
			state.cardBindings = make(map[*CardBinding]capturedCard)
		}
		state.cardBindings[e.member] = captured
		if err := e.body.Apply(ctx); err != nil {
			return err
		}
	}
	return nil
}

type savedCardLoopMember struct{ binding *CardBinding }

func (m savedCardLoopMember) inspectDefinition() definitionNode {
	return definitionNode{Role: "saved-card-loop-member", bindingWrite: m.binding, children: []definitionEdge{{"binding", m.binding}}}
}
