package mage

import (
	"slices"
	"uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// ManaProfile contains mana output and its activation costs.
type ManaProfile struct {
	Productions     []ManaProduction
	Costs           []Cost
	Interchangeable bool
}

// ManaProducer supplies the same output metadata for all mana ability forms.
type ManaProducer interface {
	ManaProfile(GameReader, uuid.UUID) ManaProfile
}

// ManaProductionEffect supplies output metadata for a mana-producing effect.
type ManaProductionEffect interface{ ManaProductions() []ManaProduction }

func (e *addManaEffect) ManaProductions() []ManaProduction {
	return []ManaProduction{{Color: e.color, Amount: e.amount}}
}
func (e *addAnyManaEffect) ManaProductions() []ManaProduction {
	return []ManaProduction{{Color: AnyColor, Amount: e.amount}}
}

func (ma *ManaAbility) ManaProfile(g GameReader, source uuid.UUID) ManaProfile {
	return ManaProfile{Productions: slices.Clone(ma.currentProductions(g, source)), Costs: []Cost{&tap{}}, Interchangeable: ma.dynamicProductions == nil && len(ma.postProduction) == 0}
}
func (a *ActionDefinition) ManaProfile(_ GameReader, _ uuid.UUID) ManaProfile {
	if a.Type() != AbilityMana || len(a.modes) > 0 {
		return ManaProfile{}
	}
	productions, ok := activatedManaEffects(a.effects)
	if !ok {
		return ManaProfile{}
	}
	return ManaProfile{Productions: productions, Costs: a.Costs(), Interchangeable: true}
}
func (a *ActionDefinition) manaActivationAllowed(ctx AbilityContext, g *Game) bool {
	return a.canActivateWithoutPayingCosts(ctx, g, true)
}

type manaPlanningCost interface {
	manaPlanCost(uuid.UUID, *Game) (ManaCost, bool)
}

func (c *tap) manaPlanCost(_ uuid.UUID, _ *Game) (ManaCost, bool) { return ManaCost{}, true }
func (c *ManaCostPayment) manaPlanCost(source uuid.UUID, g *Game) (ManaCost, bool) {
	return c.reducedCost(source, g), false
}
