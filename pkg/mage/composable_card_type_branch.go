package mage

import "github.com/benprew/mage-go/pkg/mage/core"

type cardTypeBranch struct {
	card   CardRef
	typeID core.CardType
	lacks  bool
	body   ResolvingEffect
}

// IfCardHasType runs the body if the present card has the given type.
func IfCardHasType(card CardRef, typeID core.CardType, body ResolvingEffect) ResolvingEffect {
	return &cardTypeBranch{card: card, typeID: typeID, body: body}
}

// IfCardLacksType runs the body if the present card lacks the given type.
func IfCardLacksType(card CardRef, typeID core.CardType, body ResolvingEffect) ResolvingEffect {
	return &cardTypeBranch{card: card, typeID: typeID, lacks: true, body: body}
}

func (e *cardTypeBranch) Text() string {
	condition := "if card has type"
	if e.lacks {
		condition = "if card lacks type"
	}
	if e.body == nil {
		return condition
	}
	return condition + ", " + e.body.Text()
}
func (e *cardTypeBranch) Properties() EffectProperties {
	if e.body == nil {
		return EffectProperties{}
	}
	return e.body.Properties()
}
func (e *cardTypeBranch) Apply(ctx *EffectContext) error {
	card, ok := e.card.resolveCard(ctx)
	if ok && card.HasType(e.typeID) != e.lacks {
		return e.body.Apply(ctx)
	}
	return nil
}
func (e *cardTypeBranch) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolution-branch", Text: e.Text(), children: []definitionEdge{{"card", e.card}, {"then", e.body}}}
	if e.typeID < core.TypeCreature || e.typeID > core.TypePlaneswalker {
		n.problem = "supply a valid card type"
	}
	return n
}
