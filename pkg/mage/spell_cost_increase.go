package mage

import (
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type spellCostTax struct {
	sourceID, incarnation uuid.UUID
	filter                SpellPredicate
	increase              core.ManaCost
	controllerOnly        bool
}

func (r spellCostTax) applies(g *Game, caster uuid.UUID, card Card) bool {
	source := g.FindPermanent(r.sourceID)
	if source == nil || source.incarnationID != r.incarnation || source.printedAbilitiesSuppressed {
		return false
	}
	if r.controllerOnly && source.ControllerID() != caster {
		return false
	}
	return r.filter != nil && r.filter(g, caster, card, r.sourceID)
}

// IncreaseSpellCostStatic adds fixed mana to each matching spell's total cost.
// With controllerOnly, the source's current controller must cast the spell.
func IncreaseSpellCostStatic(filter SpellPredicate, increase core.ManaCost, controllerOnly bool) ContinuousEffect {
	increase.Hybrid = slices.Clone(increase.Hybrid)
	return &spellCostTaxEffect{filter: filter, increase: increase, controllerOnly: controllerOnly}
}

type spellCostTaxEffect struct {
	effectSource
	filter         SpellPredicate
	increase       core.ManaCost
	controllerOnly bool
}

func (*spellCostTaxEffect) GetLayer() core.Layer       { return core.LayerAbility }
func (*spellCostTaxEffect) GetDuration() core.Duration { return core.WhileOnBattlefield }
func (e *spellCostTaxEffect) IsActive(g *Game) bool    { return g.FindPermanent(e.SourceID()) != nil }
func (*spellCostTaxEffect) Text() string               { return "increase spell costs" }
func (*spellCostTaxEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}
func (e *spellCostTaxEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "spell-cost-increase", Text: e.Text(), battlefieldSource: true}
	if e.filter == nil {
		n.problem = "supply a spell filter"
	}
	c := e.increase
	if c.HasX || c.XCount != 0 || c.Generic < 0 || c.White < 0 || c.Blue < 0 || c.Black < 0 || c.Red < 0 || c.Green < 0 || len(c.Hybrid) != 0 {
		n.problem = "supply nonnegative fixed generic and colored mana"
	}
	return n
}
func (e *spellCostTaxEffect) ApplyContinuous(ctx *EffectContext) error {
	g := ctx.Game
	source := g.FindPermanent(e.SourceID())
	if source == nil {
		return nil
	}
	g.effects.Rules.spellCostTaxes = append(g.effects.Rules.spellCostTaxes, spellCostTax{
		sourceID: source.ID(), incarnation: source.incarnationID, filter: e.filter,
		increase: e.increase, controllerOnly: e.controllerOnly,
	})
	return nil
}
