package mage

type permanentDamageHistoryEntry struct {
	recipient capturedPermanent
	source    DamageSource
	amount    int
}

// DamageReceivedThisTurnFromOtherSources reads actual damage to one exact visit.
// Source properties come from the damage event, as required by CR 608.2i.
func DamageReceivedThisTurnFromOtherSources(subject PermanentRef, from DamageSourceFilter) ResolutionValue {
	return otherSourceDamageHistoryValue{subject: subject, from: from}
}

type otherSourceDamageHistoryValue struct {
	subject PermanentRef
	from    DamageSourceFilter
}

func (otherSourceDamageHistoryValue) Text() string {
	return "damage received this turn from other sources"
}
func (v otherSourceDamageHistoryValue) inspectDefinition() definitionNode {
	n := permanentInformationNode(v.subject, "resolution-value", v.Text())
	n.children = append(n.children, definitionEdge{"sources", v.from})
	return n
}
func (v otherSourceDamageHistoryValue) resolveValue(ctx *EffectContext) int {
	ref, ok := v.subject.(permanentInformationReference)
	if !ok || nilDefinition(ref) {
		return 0
	}
	identity, ok := ref.informationIdentity(ctx)
	if !ok {
		return 0
	}
	total := 0
	for _, entry := range ctx.Game.damage.turnPermanentDamageAmounts {
		if entry.recipient != identity || (entry.source.ID() == identity.id && entry.source.Incarnation() == identity.incarnation) {
			continue
		}
		if v.from.Match(entry.source) {
			total += entry.amount
		}
	}
	return total
}
