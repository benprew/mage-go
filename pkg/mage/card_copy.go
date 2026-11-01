package mage

import (
	"maps"
	"slices"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// CopiableValues contains card characteristics without object execution state.
// The data is private so one snapshot can be applied to more than one card.
type CopiableValues struct {
	characteristics cardCharacteristics
	capabilityAttrs map[Attr]int
	abilities       []Ability
	castTargets     []Target
	additionalCosts []Cost
	alternateCosts  []AlternateCost
	uncounterable   bool
	auraAIProfile   *AuraAIProfile
}

// CopiableValues captures this card's characteristics, including copy changes.
// Battlefield counters, status, and continuous effects are not card values.
func (c *Card) CopiableValues() CopiableValues {
	ctx := newGameCloneContext()
	values := CopiableValues{
		characteristics: *c.copyCharacteristics(),
		capabilityAttrs: maps.Clone(c.capabilityAttrs),
		abilities:       ctx.abilityList(c.abilities),
		castTargets:     ctx.targetList(c.castTargets),
		additionalCosts: slices.Clone(c.additionalCosts),
		alternateCosts:  cloneAlternateCosts(c.alternateCosts),
		uncounterable:   c.uncounterable,
	}
	if c.auraAIProfile != nil {
		profile := *c.auraAIProfile
		values.auraAIProfile = &profile
	}
	ctx.resetCopiedRules()
	return values
}

// ApplyCopiableValues changes card characteristics and creates fresh abilities.
// The card keeps its identity, owner, and original printed definition.
func (c *Card) ApplyCopiableValues(values CopiableValues) {
	ctx := newGameCloneContext()
	characteristics := values.characteristics.clone()
	if c.definition == nil {
		c.cardCharacteristics = *characteristics
	} else {
		c.characteristics = characteristics
	}
	c.capabilityAttrs = maps.Clone(values.capabilityAttrs)
	c.abilities = ctx.abilityList(values.abilities)
	c.castTargets = ctx.targetList(values.castTargets)
	c.additionalCosts = slices.Clone(values.additionalCosts)
	c.alternateCosts = cloneAlternateCosts(values.alternateCosts)
	c.uncounterable = values.uncounterable
	c.auraAIProfile = nil
	if values.auraAIProfile != nil {
		profile := *values.auraAIProfile
		c.auraAIProfile = &profile
	}
	ctx.resetCopiedRules()
}

// NewCopy creates a distinct object with fresh abilities and the same owner.
// Spell announcement choices must be copied by the stack operation.
func (c *Card) NewCopy() *Card {
	copied := &Card{id: uuid.New(), owner: c.owner, definition: c.definition}
	copied.ApplyCopiableValues(c.CopiableValues())
	return copied
}

func (ctx *gameCloneContext) resetCopiedRules() {
	for _, ability := range ctx.abilities {
		ability.SetSource(uuid.Nil)
		ability.SetController(uuid.Nil)
		if setter, ok := ability.(interface{ setAbilityID(uuid.UUID) }); ok {
			setter.setAbilityID(uuid.New())
		}
		if action, ok := ability.(interface{ ResetActivation() }); ok {
			action.ResetActivation()
		}
		if static, ok := ability.(*StaticAbilityHolder); ok {
			for _, effect := range static.Effects {
				effect.SetSourceID(uuid.Nil)
			}
		}
	}
	for _, target := range ctx.targets {
		target.Reset()
	}
}

func cloneAlternateCosts(costs []AlternateCost) []AlternateCost {
	cloned := slices.Clone(costs)
	for i := range cloned {
		cloned[i].Mana = copyPrintedManaCost(costs[i].Mana)
		cloned[i].Additional = slices.Clone(costs[i].Additional)
	}
	return cloned
}
