package mage

import (
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type activationCostModifier struct {
	sourceID, incarnation uuid.UUID
	filter                PermanentFilter
	subject               PermanentRef
	increase              core.ManaCost
	reduction, minimum    int
}

func (r activationCostModifier) applies(g *Game, abilitySource uuid.UUID) bool {
	source := g.FindPermanent(r.sourceID)
	target := g.FindPermanent(abilitySource)
	if source == nil || target == nil || source.incarnationID != r.incarnation || source.printedAbilitiesSuppressed {
		return false
	}
	if r.subject != nil {
		p, ok := r.subject.resolvePermanent(&EffectContext{Game: g, SourceID: source.ID(), Controller: source.ControllerID()})
		return ok && p.ID() == target.ID() && p.incarnationID == target.incarnationID
	}
	return !r.filter.IsZero() && r.filter.Match(target, g)
}

// IncreaseActivationCostStatic adds fixed mana to abilities of matching permanents.
func IncreaseActivationCostStatic(filter PermanentFilter, increase core.ManaCost) ContinuousEffect {
	increase.Hybrid = slices.Clone(increase.Hybrid)
	return &activationCostModifierEffect{modifier: activationCostModifier{filter: filter, increase: increase}}
}

// ReduceActivationCostStatic reduces generic mana and preserves the specified minimum.
func ReduceActivationCostStatic(subject PermanentRef, reduction, minimum int) ContinuousEffect {
	return &activationCostModifierEffect{modifier: activationCostModifier{subject: subject, reduction: reduction, minimum: minimum}, reducing: true}
}

type activationCostModifierEffect struct {
	effectSource
	modifier activationCostModifier
	reducing bool
}

func (*activationCostModifierEffect) GetLayer() core.Layer       { return core.LayerAbility }
func (*activationCostModifierEffect) GetDuration() core.Duration { return core.WhileOnBattlefield }
func (e *activationCostModifierEffect) IsActive(g *Game) bool {
	return g.FindPermanent(e.SourceID()) != nil
}
func (*activationCostModifierEffect) Text() string                 { return "modify activation mana costs" }
func (*activationCostModifierEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *activationCostModifierEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "activation-cost-modifier", Text: e.Text(), battlefieldSource: true}
	c := e.modifier.increase
	if e.reducing {
		n.children = []definitionEdge{{"subject", e.modifier.subject}}
		if e.modifier.subject == nil || e.modifier.reduction < 0 || e.modifier.minimum < 0 {
			n.problem = "supply a subject and nonnegative reduction and minimum"
		}
	} else if e.modifier.filter.IsZero() || c.HasX || c.XCount != 0 || len(c.Hybrid) != 0 || c.Generic < 0 || c.White < 0 || c.Blue < 0 || c.Black < 0 || c.Red < 0 || c.Green < 0 {
		n.problem = "supply a permanent filter and nonnegative fixed mana"
	}
	return n
}
func (e *activationCostModifierEffect) ApplyContinuous(ctx *EffectContext) error {
	source := ctx.Game.FindPermanent(e.SourceID())
	if source == nil {
		return nil
	}
	r := e.modifier
	r.sourceID = source.ID()
	r.incarnation = source.incarnationID
	ctx.Game.effects.Rules.activationCostModifiers = append(ctx.Game.effects.Rules.activationCostModifiers, r)
	return nil
}

func (g *Game) modifyActivationManaCost(sourceID uuid.UUID, cost core.ManaCost) core.ManaCost {
	reduction := g.effects.Rules.ActivationCostReductions[sourceID]
	minimum := 0
	if reduction > 0 {
		minimum = 1
	}
	for _, r := range g.effects.Rules.activationCostModifiers {
		if r.applies(g, sourceID) {
			addActionManaCost(&cost, r.increase)
		}
	}
	for _, r := range g.effects.Rules.activationCostModifiers {
		if r.applies(g, sourceID) && r.reduction > 0 {
			reduction += r.reduction
			minimum = max(minimum, r.minimum)
		}
	}
	if reduction > 0 {
		colored := cost.White + cost.Blue + cost.Black + cost.Red + cost.Green + len(cost.Hybrid)
		floor := min(cost.Generic, max(0, minimum-colored))
		cost.Generic = max(floor, cost.Generic-reduction)
	}
	return cost
}
