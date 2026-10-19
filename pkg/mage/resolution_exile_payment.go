package mage

import "slices"

// PayChosenExileOrElse offers one optional exile payment from a permanent set.
// The paid branch reads the actual exile count. Zero is a valid payment.
func PayChosenExileOrElse(player PlayerRef, subjects PermanentSet, result *NumberBinding, paid, unpaid ResolvingEffect) ResolvingEffect {
	if !nilDefinition(paid) {
		paid = Sequence(&resolutionExileResult{result}, paid)
	}
	return &resolutionExilePayment{player, subjects, result, paid, unpaid}
}

type resolutionExilePayment struct {
	player       PlayerRef
	subjects     PermanentSet
	result       *NumberBinding
	paid, unpaid ResolvingEffect
}

func (*resolutionExilePayment) Text() string { return "offer a chosen exile payment" }
func (e *resolutionExilePayment) Properties() EffectProperties {
	props := EffectProperties{Mass: true}
	if !nilDefinition(e.paid) {
		mergeEffectProperties(&props, e.paid.Properties())
	}
	if !nilDefinition(e.unpaid) {
		mergeEffectProperties(&props, e.unpaid.Properties())
	}
	return props
}
func (e *resolutionExilePayment) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player) || permanentSetReadsSourcePermanent(e.subjects) || effectReadsSourcePermanent(e.paid) || effectReadsSourcePermanent(e.unpaid)
}
func (e *resolutionExilePayment) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolution-exile-payment", children: []definitionEdge{{"player", e.player}, {"subjects", e.subjects}, {"result", e.result}}}
	if !nilDefinition(e.paid) {
		n.children = append(n.children, definitionEdge{"paid", e.paid})
	}
	if !nilDefinition(e.unpaid) {
		n.children = append(n.children, definitionEdge{"unpaid", e.unpaid})
	}
	if nilDefinition(e.paid) && nilDefinition(e.unpaid) {
		n.problem = "supply at least one exile payment branch"
	}
	return n
}
func (e *resolutionExilePayment) Apply(ctx *EffectContext) error {
	id, ok := e.player.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	player := ctx.Game.GetPlayer(id)
	if player == nil {
		return nil
	}
	unpaid := func() error {
		if nilDefinition(e.unpaid) {
			return nil
		}
		return e.unpaid.Apply(ctx)
	}
	if !player.ChooseMayAbility("exile any number of matching permanents") {
		return unpaid()
	}
	candidates := []*Permanent{}
	for _, p := range e.subjects.resolvePermanents(ctx) {
		if p == nil {
			continue
		}
		current := ctx.Game.FindPermanent(p.ID())
		if current != nil && !current.PhasedOut && current.incarnationID == p.incarnationID && !slices.Contains(candidates, current) {
			candidates = append(candidates, current)
		}
	}
	count := 0
	if len(candidates) > 0 {
		count = min(len(candidates), max(0, player.ChooseNumber(0, len(candidates), "Choose how many permanents to exile")))
	}
	selected := choosePermanentSubset(player, candidates, count, "Choose permanents to exile", ctx.Game)
	// A failed selection must not cause a partial payment.
	eligible := e.subjects.resolvePermanents(ctx)
	for _, p := range selected {
		current := ctx.Game.FindPermanent(p.ID())
		if current == nil || current.PhasedOut || current.incarnationID != p.incarnationID || !slices.ContainsFunc(eligible, func(other *Permanent) bool {
			return other != nil && other.ID() == p.ID() && other.incarnationID == p.incarnationID
		}) {
			return unpaid()
		}
	}
	amount := len(ctx.Game.exilePermanents(selected))
	if nilDefinition(e.paid) {
		return nil
	}
	values := ctx.Game.resolution.numberBindings
	if values == nil {
		values = make(map[*NumberBinding]int)
		ctx.Game.resolution.numberBindings = values
	}
	previous, hadPrevious := values[e.result]
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

type resolutionExileResult struct{ binding *NumberBinding }

func (*resolutionExileResult) Text() string                 { return "paid exile count" }
func (*resolutionExileResult) Properties() EffectProperties { return EffectProperties{} }
func (*resolutionExileResult) Apply(*EffectContext) error   { return nil }
func (e *resolutionExileResult) inspectDefinition() definitionNode {
	return definitionNode{Role: "paid-exile-result", bindingWrite: e.binding, children: []definitionEdge{{"result", e.binding}}}
}
