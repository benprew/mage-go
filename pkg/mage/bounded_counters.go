package mage

import (
	"fmt"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type boundedCountersEffect struct {
	subject PermanentRef
	counter CounterType
	amount  ResolutionValue
	limit   int
}

// AddCountersUpTo lets the effect controller choose counters within an action limit.
// The limit also applies after replacements, because this action cannot exceed it.
func AddCountersUpTo(subject PermanentRef, counter CounterType, amount ResolutionValue, limit int) ResolvingEffect {
	return &boundedCountersEffect{subject: subject, counter: counter, amount: amount, limit: limit}
}
func (*boundedCountersEffect) Text() string {
	return "put up to the chosen number of counters within the limit"
}
func (e *boundedCountersEffect) Properties() EffectProperties {
	return AddCounters(e.counter, Fixed(1)).Properties()
}
func (e *boundedCountersEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"subject", e.subject}, {"amount", e.amount}}}
	if e.limit < 0 || e.limit > 255 || e.counter < 0 || e.counter >= NumCounters {
		n.problem = "supply a limit from zero through 255 and a valid counter type"
	}
	return n
}
func (e *boundedCountersEffect) Apply(ctx *EffectContext) error {
	p, ok := e.subject.resolvePermanent(ctx)
	if !ok {
		return nil
	}
	upper := min(e.amount.resolveValue(ctx), max(e.limit-int(p.Counters[e.counter]), 0))
	if upper <= 0 {
		return nil
	}
	player := ctx.Game.GetPlayer(ctx.Controller)
	if player == nil {
		return nil
	}
	chosen := player.ChooseNumber(0, upper, fmt.Sprintf("Put how many %s counters?", e.counter))
	if chosen <= 0 || chosen > upper {
		return nil
	}
	live := ctx.Game.FindPermanent(p.ID())
	if live == nil || live.incarnationID != p.incarnationID {
		return nil
	}
	chosen = min(chosen, max(e.limit-int(live.Counters[e.counter]), 0))
	if chosen > 0 {
		ctx.Game.addCountersWithReplacementLimit(live, e.counter, chosen, ctx.SourceID, false, &e.limit)
	}
	return nil
}
