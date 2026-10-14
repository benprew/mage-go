package mage

import (
	"fmt"
	"slices"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type copySpellColorsEffect struct {
	spell  StackObjectRef
	colors []core.Color
}

// CopyInstantOrSorceryWithColors copies an instant or sorcery with fixed colors.
// The copy uses these colors before it can choose targets.
// An empty list makes the copy colorless. Other color changes are not copied.
func CopyInstantOrSorceryWithColors(spell StackObjectRef, colors ...core.Color) ResolvingEffect {
	return &copySpellColorsEffect{spell: spell, colors: slices.Clone(colors)}
}

func (*copySpellColorsEffect) Text() string {
	return "copy spell with fixed colors and optional new targets"
}
func (*copySpellColorsEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *copySpellColorsEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"spell", e.spell}}}
	for _, color := range e.colors {
		switch color {
		case core.White, core.Blue, core.Black, core.Red, core.Green:
		default:
			n.problem = "supply Magic colors; use an empty list for colorless"
		}
	}
	return n
}

func (e *copySpellColorsEffect) Apply(ctx *EffectContext) error {
	obj, ok := e.spell.resolveStackObject(ctx)
	if !ok || (!obj.Card.HasType(core.TypeInstant) && !obj.Card.HasType(core.TypeSorcery)) {
		return nil
	}
	card, ok := obj.Card.Copy().(*BaseCard)
	if !ok {
		return fmt.Errorf("spell color copy requires a base card")
	}
	colors := slices.Clone(e.colors)
	if colors == nil {
		colors = []core.Color{}
	}
	modification := copyRetainedColors{colors: colors}
	if err := modification.validateCopyModification(); err != nil {
		return err
	}
	modification = CopyRetainColors(colors...).(copyRetainedColors)
	rebindCopyTextDefinition(card, []CopyModification{modification})
	if card.copiableValues != nil {
		values, err := card.copiableValues.WithModifications(modification)
		if err != nil {
			return err
		}
		card.copiableValues = values
		card.definitionPlan = values.definition
	}
	modification.applyCopyModification(card)
	card.SetOwner(ctx.Controller)
	ctx.Game.copyStackObjectWithCard(obj, ctx.Controller, true, card)
	return nil
}
