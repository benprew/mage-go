package ron

import "github.com/benprew/mage-go/pkg/mage"

type temporaryDeathExilePlan struct {
	ref     damageHistorySourcePlan
	filter  mage.PermanentFilter
	history bool
}

func (plan temporaryDeathExilePlan) build(bindings instanceBindings) mage.Effect {
	if plan.history {
		return mage.ExileDealtDamageByThisTurn(plan.ref.build(bindings), plan.filter)
	}
	return mage.ExileIfWouldDieThisTurn(plan.ref.build(bindings))
}

func compileTemporaryDeathExile(n Node, ctx scope) (effectFactory, error) {
	plan, err := compileTemporaryDeathExilePlan(n, ctx)
	if err != nil {
		return nil, err
	}
	return plan.build, nil
}

func compileTemporaryDeathExilePlan(n Node, ctx scope) (temporaryDeathExilePlan, error) {
	if err := arity(n, 3); err != nil {
		return temporaryDeathExilePlan{}, err
	}
	event := n.Args[0]
	if event.Name != "APermanentWouldDie" {
		return temporaryDeathExilePlan{}, unsupported(event, "temporary death replacement event")
	}
	if err := arity(event, 1); err != nil {
		return temporaryDeathExilePlan{}, err
	}
	subject := event.Args[0]
	var ref damageHistorySourcePlan
	filter := mage.AnyPermanent
	var err error
	history := subject.Name != "SinglePermanent"
	if history {
		ref, filter, err = compileDamageHistorySubjects(subject, ctx)
	} else {
		if err = arity(subject, 1); err == nil {
			ref.permanentReferencePlan, err = compilePermanentRef(subject.Args[0], ctx)
		}
	}
	if err != nil {
		return temporaryDeathExilePlan{}, err
	}
	if ref.creaturesOnly {
		return temporaryDeathExilePlan{}, problem(subject, "death replacement requires a live recipient filter")
	}
	ops, err := list(n.Args[1])
	if err != nil {
		return temporaryDeathExilePlan{}, err
	}
	if len(ops) != 1 || ops[0].Kind != Symbol || ops[0].Name != "ExileItInstead" {
		return temporaryDeathExilePlan{}, problem(n.Args[1], "supply one ExileItInstead symbol")
	}
	if n.Args[2].Kind != Symbol || n.Args[2].Name != "UntilEndOfTurn" {
		return temporaryDeathExilePlan{}, unsupported(n.Args[2], "temporary death replacement duration")
	}
	return temporaryDeathExilePlan{ref: ref, filter: filter, history: history}, nil
}
