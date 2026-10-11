package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileTemporaryDeathExile(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 3); err != nil {
		return nil, err
	}
	event := n.Args[0]
	if event.Name != "APermanentWouldDie" {
		return nil, unsupported(event, "temporary death replacement event")
	}
	if err := arity(event, 1); err != nil {
		return nil, err
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
		return nil, err
	}
	if ref.creaturesOnly {
		return nil, problem(subject, "death replacement requires a live recipient filter")
	}
	ops, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(ops) != 1 || ops[0].Kind != Symbol || ops[0].Name != "ExileItInstead" {
		return nil, problem(n.Args[1], "supply one ExileItInstead symbol")
	}
	if n.Args[2].Kind != Symbol || n.Args[2].Name != "UntilEndOfTurn" {
		return nil, unsupported(n.Args[2], "temporary death replacement duration")
	}
	return func(bindings instanceBindings) mage.Effect {
		if history {
			return mage.ExileDealtDamageByThisTurn(ref.build(bindings), filter)
		}
		return mage.ExileIfWouldDieThisTurn(ref.build(bindings))
	}, nil
}
