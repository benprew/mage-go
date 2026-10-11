package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type liveDamageHistoryEffect struct {
	source        PermanentRef
	exile         bool
	filter        PermanentFilter
	creaturesOnly bool
}

// CantRegenerateDealtDamageByThisTurn restricts this source's damage recipients.
// The rule checks later damage and keeps the source's original visit.
func CantRegenerateDealtDamageByThisTurn(source PermanentRef, filter PermanentFilter) ResolvingEffect {
	return &liveDamageHistoryEffect{source: source, filter: filter}
}

// ExileDealtDamageByThisTurn replaces deaths of this source's damage recipients.
// The rule checks later damage and keeps the source's original visit.
func ExileDealtDamageByThisTurn(source PermanentRef, filter PermanentFilter) ResolvingEffect {
	return &liveDamageHistoryEffect{source: source, exile: true, filter: filter}
}
func (e *liveDamageHistoryEffect) Text() string {
	if e.exile {
		return "exile damage recipients if they would die this turn"
	}
	return "damage recipients cannot regenerate this turn"
}
func (e *liveDamageHistoryEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment, Mass: true}
}
func (e *liveDamageHistoryEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "live-damage-history-rule", Text: e.Text(), children: []definitionEdge{{"source", e.source}}}
}
func (e *liveDamageHistoryEffect) refersToSourcePermanent() bool {
	return permanentRefReadsSourcePermanent(e.source)
}
func (e *liveDamageHistoryEffect) Apply(ctx *EffectContext) error {
	source := DamageSourceOf(e.source).resolveDamageSource(ctx)
	if source.Incarnation() == uuid.Nil {
		return nil
	}
	if e.exile {
		ctx.Game.effects.AddReplacement(&temporaryDeathExileReplacement{
			replacementBase: replacementBase{sourceID: ctx.SourceID, duration: core.EndOfTurn},
			historySource:   cloneDamageSource(source),
			historyFilter:   e.filter,
		})
	} else {
		rule := &liveDamageRegenerationRule{
			funcContinuousEffect: FuncContinuousEffect(core.LayerAbility, core.EndOfTurn, func(*Game, uuid.UUID) error { return nil }).(*funcContinuousEffect),
			source:               cloneDamageSource(source),
			filter:               e.filter,
			creaturesOnly:        e.creaturesOnly,
		}
		rule.SetSourceID(ctx.SourceID)
		ctx.Game.AddContinuousEffect(rule)
	}
	return nil
}

type liveDamageRegenerationRule struct {
	*funcContinuousEffect
	source        DamageSource
	creaturesOnly bool
	filter        PermanentFilter
}

func (*liveDamageRegenerationRule) survivesSourceExit() bool { return true }
func (r *liveDamageRegenerationRule) cloneEffect() ContinuousEffect {
	clone := *r
	inner := *r.funcContinuousEffect
	clone.funcContinuousEffect = &inner
	clone.source = cloneDamageSource(r.source)
	return &clone
}

// CanRegeneratePermanent checks current prohibitions on a present subject.
// Live damage rules must apply before the next layer pass.
func (g *Game) CanRegeneratePermanent(p *Permanent) bool {
	if p == nil || p.PhasedOut || p.HasKeyword(core.CantRegenerate) {
		return false
	}
	current := g.FindPermanent(p.ID())
	if current == nil || current.incarnationID != p.incarnationID {
		return false
	}
	for _, effect := range g.effects.effects {
		if rule, ok := effect.(*liveDamageRegenerationRule); ok && rule.IsActive(g) && rule.filter.Match(p, g) && rule.matchesDamage(g, p) {
			return false
		}
	}
	return true
}

// CantRegenerateCreaturesDealtDamageByThisTurn keeps the recipient type at damage.
// Later type changes cannot end the rule on the same visit.
func CantRegenerateCreaturesDealtDamageByThisTurn(source PermanentRef) ResolvingEffect {
	return &liveDamageHistoryEffect{source: source, filter: AnyPermanent, creaturesOnly: true}
}
func (r *liveDamageRegenerationRule) matchesDamage(g *Game, p *Permanent) bool {
	if r.creaturesOnly {
		return g.damage.turnDamageToCreatures[capturedPermanent{r.source.ID(), r.source.Incarnation()}][capturedPermanent{p.ID(), p.incarnationID}]
	}
	return g.damage.dealtDamageToPermanentThisTurn(r.source, capturedPermanent{p.ID(), p.incarnationID})
}
