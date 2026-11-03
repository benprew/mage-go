package mage

import (
	"slices"
	"uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// CopiableValues contains card characteristics without object execution state.
// The data is private so one snapshot can be applied to more than one card.
type CopiableValues struct{ *CardDefinition }

// CopiableValues captures the current card face without execution state.
func (c *Card) CopiableValues() CopiableValues {
	if c.unflippedValues != nil {
		return *c.unflippedValues
	}
	definition := c.CardDefinition
	if definition.flipFace != nil {
		definition = definition.cloneDefinition()
		definition.flipFace = definition.flipFace.cloneDefinition()
		definition.flipFace.manaCost = copyPrintedManaCost(definition.manaCost)
		definition.flipFace.colorOverride = append([]Color{}, c.Colors()...)
	}
	return CopiableValues{CardDefinition: definition}
}

// CopiableValues captures copy changes and face-down characteristics.
// Effects from later layers are not included.
func (p *Permanent) CopiableValues() CopiableValues {
	if p.FaceDown {
		return NewCreature("", "", 2, 2).CopiableValues()
	}
	return p.Card.CopiableValues()
}

// ApplyCopiableValues changes card characteristics and creates fresh abilities.
// The card keeps its identity, owner, and original printed definition.
func (c *Card) ApplyCopiableValues(values CopiableValues) {
	c.unflippedValues = nil
	c.CardDefinition = values.CardDefinition
}

// NewCopy creates a distinct object with fresh abilities and the same owner.
// Spell announcement choices must be copied by the stack operation.
func (c *Card) NewCopy() *Card {
	copied := &Card{id: uuid.New(), owner: c.owner, definition: c.definition}
	copied.ApplyCopiableValues(c.CopiableValues())
	return copied
}

func cloneAlternateCosts(costs []AlternateCost) []AlternateCost {
	cloned := slices.Clone(costs)
	for i := range cloned {
		cloned[i].Mana = copyPrintedManaCost(costs[i].Mana)
		cloned[i].Additional = slices.Clone(costs[i].Additional)
	}
	return cloned
}

// WithFlipFace sets the alternative name, type line, abilities, and power/toughness.
// A flip card keeps its normal mana cost and color on both faces (CR 710.1c).
func WithFlipFace(face *Card) CardOption {
	values := face.CopiableValues()
	return func(card *Card) { card.mutableDefinition().flipFace = values.CardDefinition }
}

func (c *Card) selectFlippedFace() {
	if c.flipFace == nil || c.unflippedValues != nil {
		return
	}
	normal := c.CopiableValues()
	c.ApplyCopiableValues(CopiableValues{CardDefinition: normal.flipFace})
	c.mutableDefinition().flipFace = normal.flipFace
	c.unflippedValues = &normal
}

func copyValuesWithOptions(values CopiableValues, colors []Color, options CopyOptions) CopiableValues {
	values.CardDefinition = values.cloneDefinition()
	if options.KeepColor {
		values.colorOverride = append([]Color{}, colors...)
	}
	if options.AdditionalAbility != nil {
		values.abilities = append(slices.Clone(values.abilities), options.AdditionalAbility())
	}
	if values.flipFace != nil {
		flipped := copyValuesWithOptions(CopiableValues{CardDefinition: values.flipFace}, colors, options)
		values.flipFace = flipped.CardDefinition
	}
	return values
}

func (g *Game) applyPermanentCopy(perm *Permanent, values CopiableValues) {
	previous := NewPermanent(perm.Card, perm.ControllerID())
	perm.Card.ApplyCopiableValues(values)
	if perm.Flipped {
		perm.Card.selectFlippedFace()
	}
	fresh := NewPermanent(perm.Card, perm.ControllerID())
	for attr := range perm.baseAttrs {
		perm.baseAttrs[attr] += fresh.baseAttrs[attr] - previous.baseAttrs[attr]
	}
	perm.printedAbilityAttrs = fresh.printedAbilityAttrs
	perm.RuntimeAbilities = fresh.RuntimeAbilities
	perm.printedAbilitiesSuppressed = false
	if g.zones.EnteringPermanent() != perm {
		for abilityIndex, ability := range perm.RuntimeAbilities {
			if static, ok := ability.(*StaticAbilityHolder); ok {
				for effectIndex, effect := range static.Effects {
					g.effects.addPrinted(effect, perm.ID(), abilityIndex, effectIndex)
				}
			}
		}
	}

}

// FlipPermanent changes a permanent's status to flipped without a zone change.
// A permanent keeps this status even while it has no alternative face.
func (g *Game) FlipPermanent(id uuid.UUID) {
	perm := g.MutablePermanent(id)
	if perm == nil || perm.Flipped {
		return
	}
	perm.Flipped = true
	if perm.Card.flipFace != nil {
		g.applyPermanentCopy(perm, perm.Card.CopiableValues())
	}
	g.ApplyContinuousEffects()
}

// FlipSource flips the source permanent when this effect resolves.
func FlipSource() Effect {
	return FuncEffect("flip this permanent", EffectProperties{}, func(g *Game, source, _ uuid.UUID, _ []uuid.UUID) error {
		if g.ResolvingSourcePermanent(source) != nil {
			g.FlipPermanent(source)
		}
		return nil
	})
}
