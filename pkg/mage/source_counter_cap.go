package mage

import "github.com/benprew/mage-go/pkg/mage/core"

type sourceCounterCap struct {
	effectSource
	counter core.CounterType
	limit   int
}

// SourceCounterCap limits one counter type on the source while it has this ability.
// The prohibition applies after replacements, because a permission cannot override it.
func SourceCounterCap(counter core.CounterType, limit int) ContinuousEffect {
	return &sourceCounterCap{counter: counter, limit: limit}
}
func (*sourceCounterCap) Text() string { return "limit counters on this permanent" }
func (*sourceCounterCap) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeUnknown}
}
func (*sourceCounterCap) GetLayer() core.Layer                 { return core.LayerAbility }
func (*sourceCounterCap) GetDuration() core.Duration           { return core.WhileOnBattlefield }
func (e *sourceCounterCap) IsActive(g *Game) bool              { return g.FindPermanent(e.sourceID) != nil }
func (*sourceCounterCap) ApplyContinuous(*EffectContext) error { return nil }
func (e *sourceCounterCap) cloneEffect() ContinuousEffect      { clone := *e; return &clone }
func (e *sourceCounterCap) inspectDefinition() definitionNode {
	n := definitionNode{Role: "static-rule", Text: e.Text()}
	if e.counter < 0 || e.counter >= core.NumCounters || e.limit < 0 {
		n.problem = "supply a valid counter type and a nonnegative limit"
	}
	return n
}

func sourceCounterLimit(p *Permanent, counter core.CounterType) (int, bool) {
	limit, found := 0, false
	for _, ability := range p.RuntimeAbilities {
		holder, ok := UnwrapAbility(ability).(*StaticAbilityHolder)
		if !ok {
			continue
		}
		for _, effect := range holder.Effects {
			if rule, ok := effect.(*sourceCounterCap); ok && rule.counter == counter {
				if !found || rule.limit < limit {
					limit = rule.limit
				}
				found = true
			}
		}
	}
	return limit, found
}

func cappedCounterAmount(p *Permanent, counter core.CounterType, amount int) int {
	if limit, found := sourceCounterLimit(p, counter); found {
		amount = min(amount, max(limit-p.Counters[counter], 0))
	}
	return max(amount, 0)
}
