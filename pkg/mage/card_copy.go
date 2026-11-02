package mage

import (
	"maps"
	"slices"
	"uuid"

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
	flipFace        *CopiableValues
}

// CopiableValues captures this card's characteristics, including copy changes.
// Battlefield counters, status, and continuous effects are not card values.
func (c *Card) CopiableValues() CopiableValues {
	if c.unflippedValues != nil {
		return *c.unflippedValues
	}
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
	if c.flipFace != nil {
		flipped := *c.flipFace
		flipped.characteristics.manaCost = copyPrintedManaCost(values.characteristics.manaCost)
		flipped.characteristics.colorOverride = append([]Color{}, c.Colors()...)
		values.flipFace = &flipped
	}
	ctx.resetCopiedRules()
	return values
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
	c.flipFace = values.flipFace
	c.unflippedValues = nil
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
		ability.SetSource(uuid.Nil())
		ability.SetController(uuid.Nil())
		if setter, ok := ability.(interface{ setAbilityID(uuid.UUID) }); ok {
			setter.setAbilityID(uuid.New())
		}
		if action, ok := ability.(interface{ ResetActivation() }); ok {
			action.ResetActivation()
		}
		if static, ok := ability.(*StaticAbilityHolder); ok {
			for _, effect := range static.Effects {
				effect.SetSourceID(uuid.Nil())
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

// WithFlipFace sets the alternative name, type line, abilities, and power/toughness.
// A flip card keeps its normal mana cost and color on both faces (CR 710.1c).
func WithFlipFace(face *Card) CardOption {
	values := face.CopiableValues()
	return func(card *Card) { card.flipFace = &values }
}

func (c *Card) selectFlippedFace() {
	if c.flipFace == nil || c.unflippedValues != nil {
		return
	}
	normal := c.CopiableValues()
	c.ApplyCopiableValues(*normal.flipFace)
	c.flipFace = normal.flipFace
	c.unflippedValues = &normal
}

func copyValuesWithOptions(values CopiableValues, colors []Color, options CopyOptions) CopiableValues {
	if options.KeepColor {
		values.characteristics.colorOverride = append([]Color{}, colors...)
	}
	if options.AdditionalAbility != nil {
		values.abilities = append(slices.Clone(values.abilities), options.AdditionalAbility())
	}
	if values.flipFace != nil {
		flipped := copyValuesWithOptions(*values.flipFace, colors, options)
		values.flipFace = &flipped
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
		for _, ability := range perm.RuntimeAbilities {
			if static, ok := ability.(*StaticAbilityHolder); ok {
				for _, effect := range static.Effects {
					effect.SetSourceID(perm.ID())
					g.effects.Add(&printedAbilityContinuousEffect{ContinuousEffect: effect})
				}
			}
		}
	}
	g.syncAbilityContext(perm)
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
