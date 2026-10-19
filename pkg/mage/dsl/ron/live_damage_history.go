package ron

import "github.com/benprew/mage-go/pkg/mage"

type damageHistorySourcePlan struct {
	permanentReferencePlan
	creaturesOnly bool
}

func compileDamageHistorySource(n Node, ctx scope) (damageHistorySourcePlan, error) {
	if n.Name != "WasDealtDamageByPermanentThisTurn" {
		return damageHistorySourcePlan{}, unsupported(n, "live damage-history subject")
	}
	if n.Kind != Tuple || (len(n.Args) != 1 && len(n.Args) != 2) {
		return damageHistorySourcePlan{}, problem(n, "supply a source and optional creature-at-damage filter")
	}
	onlyCreatures := len(n.Args) == 2
	if onlyCreatures {
		filter := n.Args[1]
		if filter.Name != "IsCardtype" || filter.Kind != Tuple || len(filter.Args) != 1 || filter.Args[0].Kind != Symbol || filter.Args[0].Name != "Creature" {
			return damageHistorySourcePlan{}, unsupported(filter, "damage-time recipient filter")
		}
	}
	ref, err := compilePermanentRef(n.Args[0], ctx)
	return damageHistorySourcePlan{ref, onlyCreatures}, err
}

type collectionDamageHistoryRulePlan struct {
	ref    damageHistorySourcePlan
	filter mage.PermanentFilter
}

func (plan collectionDamageHistoryRulePlan) build(bindings instanceBindings) mage.Effect {
	if plan.ref.creaturesOnly {
		return mage.CantRegenerateCreaturesDealtDamageByThisTurn(plan.ref.build(bindings))
	}
	return mage.CantRegenerateDealtDamageByThisTurn(plan.ref.build(bindings), plan.filter)
}

func compileCollectionDamageHistoryRule(n Node, ctx scope) (effectFactory, error) {
	plan, err := compileCollectionDamageHistoryRulePlan(n, ctx)
	if err != nil {
		return nil, err
	}
	return plan.build, nil
}

func compileCollectionDamageHistoryRulePlan(n Node, ctx scope) (collectionDamageHistoryRulePlan, error) {
	if err := arity(n, 3); err != nil {
		return collectionDamageHistoryRulePlan{}, err
	}
	ref, filter, err := compileDamageHistorySubjects(n.Args[0], ctx)
	if err != nil {
		return collectionDamageHistoryRulePlan{}, err
	}
	ops, err := list(n.Args[1])
	if err != nil {
		return collectionDamageHistoryRulePlan{}, err
	}
	if len(ops) != 1 || ops[0].Kind != Symbol || ops[0].Name != "CantBeRegenerated" {
		return collectionDamageHistoryRulePlan{}, problem(n.Args[1], "supply one CantBeRegenerated symbol")
	}
	if n.Args[2].Kind != Symbol || n.Args[2].Name != "UntilEndOfTurn" {
		return collectionDamageHistoryRulePlan{}, unsupported(n.Args[2], "live damage-history duration")
	}
	return collectionDamageHistoryRulePlan{ref: ref, filter: filter}, nil
}

func compileDamageHistorySubjects(n Node, ctx scope) (damageHistorySourcePlan, mage.PermanentFilter, error) {
	if n.Name == "WasDealtDamageByPermanentThisTurn" {
		ref, err := compileDamageHistorySource(n, ctx)
		return ref, mage.AnyPermanent, err
	}
	if n.Name != "And" {
		return damageHistorySourcePlan{}, mage.PermanentFilter{}, unsupported(n, "live damage-history collection")
	}
	if err := arity(n, 1); err != nil {
		return damageHistorySourcePlan{}, mage.PermanentFilter{}, err
	}
	parts, err := list(n.Args[0])
	if err != nil {
		return damageHistorySourcePlan{}, mage.PermanentFilter{}, err
	}
	var ref damageHistorySourcePlan
	var filters []mage.PermanentFilter
	found := false
	for _, part := range parts {
		if part.Name == "WasDealtDamageByPermanentThisTurn" {
			if found {
				return ref, mage.PermanentFilter{}, problem(part, "supply one damage-history source")
			}
			ref, err = compileDamageHistorySource(part, ctx)
			found = true
		} else {
			var filter mage.PermanentFilter
			filter, err = compileFilter(part)
			filters = append(filters, filter)
		}
		if err != nil {
			return ref, mage.PermanentFilter{}, err
		}
	}
	if !found || len(filters) == 0 {
		return ref, mage.PermanentFilter{}, problem(n, "supply one history source and a permanent filter")
	}
	if ref.creaturesOnly {
		return ref, mage.PermanentFilter{}, problem(n, "damage-time creature filter cannot use a live filter")
	}
	return ref, mage.And(filters...), nil
}
