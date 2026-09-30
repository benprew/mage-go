package mage

// DealAllocatedDamage requires shares to be chosen before payment.
// Use one target group as the direct body of a spell or activation.
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
