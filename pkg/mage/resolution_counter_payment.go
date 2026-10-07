package mage

import (
	"github.com/benprew/mage-go/pkg/mage/core"
)

type resolutionCounterPayment struct {
	player           PlayerRef
	subject          PermanentRef
	counter          core.CounterType
	count            int
	optional, chosen bool
	result           *NumberBinding
	paid, unpaid     ResolvingEffect
}

// PayCountersOrElse removes a fixed counter payment before it runs one branch.
// An optional payment gives the player a choice to decline.
func PayCountersOrElse(player PlayerRef, subject PermanentRef, counter core.CounterType, count int, optional bool, paid, unpaid ResolvingEffect) ResolvingEffect {
	return &resolutionCounterPayment{player: player, subject: subject, counter: counter, count: count, optional: optional, paid: paid, unpaid: unpaid}
}

// PayChosenCountersOrElse saves the chosen count only for the paid branch.
func PayChosenCountersOrElse(player PlayerRef, subject PermanentRef, counter core.CounterType, result *NumberBinding, paid, unpaid ResolvingEffect) ResolvingEffect {
	if nilDefinition(paid) {
		paid = &resolutionCounterResult{result}
	} else {
		paid = Sequence(&resolutionCounterResult{result}, paid)
	}
	return &resolutionCounterPayment{player: player, subject: subject, counter: counter, optional: true, chosen: true, result: result, paid: paid, unpaid: unpaid}
}

func (*resolutionCounterPayment) Text() string { return "remove counters, then resolve one branch" }
func (e *resolutionCounterPayment) Properties() EffectProperties {
	props := EffectProperties{}
	if !nilDefinition(e.paid) {
		props = e.paid.Properties()
	}
	if !nilDefinition(e.unpaid) {
		mergeEffectProperties(&props, e.unpaid.Properties())
	}
	return props
}
func (e *resolutionCounterPayment) Apply(ctx *EffectContext) error {
	player, ok := e.player.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	subject, present := e.subject.resolvePermanent(ctx)
	paid := false
	amount := 0
	if present && e.counter >= 0 && e.counter < core.NumCounters && e.count >= 0 {
		id, incarnation := subject.ID(), subject.incarnationID
		var cost = RemoveCountersCost(e.counter, e.count)
		if e.chosen {
			cost = RemoveAnyCountersCost(e.counter, e.result)
		}
		if cost.CanPay(id, player, ctx.Game) && (!e.optional || ctx.Game.GetPlayer(player).ChooseMayAbility("pay "+cost.Text())) {
			// A choice cannot make a returned object pay for the original object.
			current, stillPresent := e.subject.resolvePermanent(ctx)
			if stillPresent && current.ID() == id && current.incarnationID == incarnation {
				tx, err := ctx.Game.prepareActionPaymentTransaction(actionPaymentSpec{SourceID: id, Controller: player, Costs: []Cost{cost}})
				paid = err == nil && tx.Commit() == nil
				if paid {
					amount = e.count
					if e.chosen {
						amount = tx.paidNumbers[e.result]
					}
				}
			}
		}
	}
	if !paid {
		if nilDefinition(e.unpaid) {
			return nil
		}
		return e.unpaid.Apply(ctx)
	}
	if nilDefinition(e.paid) {
		return nil
	}
	if !e.chosen {
		return e.paid.Apply(ctx)
	}
	values := ctx.Game.resolution.numberBindings
	previous, hadPrevious := values[e.result]
	if values == nil {
		values = make(map[*NumberBinding]int)
		ctx.Game.resolution.numberBindings = values
	}
	values[e.result] = amount
	defer func() {
		if hadPrevious {
			values[e.result] = previous
		} else {
			delete(values, e.result)
		}
	}()
	return e.paid.Apply(ctx)
}
func (e *resolutionCounterPayment) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolution-counter-payment", children: []definitionEdge{{"player", e.player}, {"subject", e.subject}}}
	if !nilDefinition(e.paid) {
		n.children = append(n.children, definitionEdge{"paid", e.paid})
	}
	if !nilDefinition(e.unpaid) {
		n.children = append(n.children, definitionEdge{"unpaid", e.unpaid})
	}
	if nilDefinition(e.paid) && nilDefinition(e.unpaid) {
		n.problem = "supply at least one counter payment branch"
	}
	if e.counter < 0 || e.counter >= core.NumCounters {
		n.problem = "invalid counter type"
	}
	if e.count < 0 {
		n.problem = "counter payment requires a nonnegative count"
	}
	if e.chosen && e.result == nil {
		n.problem = "supply a non-nil paid counter binding"
	}
	return n
}

type resolutionCounterResult struct{ binding *NumberBinding }

func (*resolutionCounterResult) Text() string                 { return "paid counter count" }
func (*resolutionCounterResult) Properties() EffectProperties { return EffectProperties{} }
func (*resolutionCounterResult) Apply(*EffectContext) error   { return nil }
func (e *resolutionCounterResult) inspectDefinition() definitionNode {
	return definitionNode{Role: "paid-counter-result", bindingWrite: e.binding, children: []definitionEdge{{"result", e.binding}}, Text: "paid counter count"}
}
