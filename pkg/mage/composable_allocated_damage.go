package mage

import "reflect"

// DealAllocatedDamage requires shares to be chosen before payment.
// Each spell, activation, or selected mode needs its own target group.
func DealAllocatedDamage(source DamageSourceRef, recipients *DamageTargetsBinding, total int) ResolvingEffect {
	return &allocatedDamageEffect{source: source, recipients: recipients, total: total}
}

type allocatedDamageEffect struct {
	source     DamageSourceRef
	recipients *DamageTargetsBinding
	total      int
}

func (*allocatedDamageEffect) Text() string { return "deal damage divided as you choose" }
func (e *allocatedDamageEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment, DamageValue: Fixed(e.total), Mass: true}
}
func (e *allocatedDamageEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"source", damageSourceDefinition{e.source}}, {"recipients", e.recipients}, {"total", Fixed(e.total)}}}
	if e.total <= 0 {
		n.problem = "supply a positive fixed allocation total"
	} else if e.recipients == nil || e.recipients.targetDeclaration == nil {
		n.problem = "supply a damage target declaration"
	} else if e.recipients.Min() > e.total {
		n.problem = "allocation total must supply at least one damage per required target"
	}
	return n
}
func (e *allocatedDamageEffect) Apply(ctx *EffectContext) error {
	source := e.source.resolveDamageSource(ctx)
	if !source.Valid() {
		return nil
	}
	assignments := e.recipients.resolveDamageRecipients(ctx, 1, source)
	shares := ctx.DamageDistribution
	for i := range assignments {
		id := assignments[i].request.playerID
		if permanent := assignments[i].request.permanent; permanent != nil {
			id = permanent.ID()
		}
		assignments[i].request.amount = shares[id]
	}
	ctx.Game.DealSimultaneousDamage(assignments...)
	return nil
}

// One stack allocation must not serve two damage instructions.
func singleAllocatedDamage(effect Effect) *allocatedDamageEffect {
	var allocation *allocatedDamageEffect
	count := 0
	active := make(map[Effect]bool)
	var visit func(Effect, bool)
	visit = func(effect Effect, allowed bool) {
		if nilDefinition(effect) {
			return
		}
		if reflect.ValueOf(effect).Comparable() {
			if active[effect] {
				count++
				return
			}
			active[effect] = true
			defer delete(active, effect)
		}
		switch e := effect.(type) {
		case *allocatedDamageEffect:
			if allowed {
				allocation = e
			}
			count++
		case *dealDividedDamageEffect:
			count++
		case *sequenceEffect:
			for _, child := range e.effects {
				visit(child, allowed)
			}
		default:
			for _, edge := range describeDefinition(effect).children {
				if child, ok := edge.value.(Effect); ok {
					visit(child, false)
				}
			}
		}
	}
	visit(effect, true)
	if count != 1 {
		return nil
	}
	return allocation
}

func allocatedBodyEdge(effects []Effect, targets []Target, child any) bool {
	if len(effects) != 1 || len(targets) != 1 {
		return false
	}
	allocation := singleAllocatedDamage(effects[0])
	return allocation != nil && allocation.recipients != nil && child == effects[0] && targets[0] == allocation.recipients.targetDeclaration
}
