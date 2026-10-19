package ron

import "github.com/benprew/mage-go/pkg/mage"

type directReturnPlan struct {
	subject  *permanentReferencePlan
	subjects permanentSetFactory
}

func (plan directReturnPlan) build(bindings instanceBindings) mage.DirectReturnInstruction {
	if plan.subject != nil {
		return mage.DirectReturnInstruction{Subject: plan.subject.build(bindings)}
	}
	return mage.DirectReturnInstruction{Subjects: plan.subjects(bindings)}
}

func compileDirectReturn(n Node, ctx scope) (effectFactory, error) {
	plan, err := compileDirectReturnPlan(n, ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect { return plan.build(bindings).Effect() }, nil
}

func compileDirectReturnPlan(n Node, ctx scope) (directReturnPlan, error) {
	if err := arity(n, 1); err != nil {
		return directReturnPlan{}, err
	}
	switch n.Name {
	case "PutPermanentIntoItsOwnersHand":
		subject, err := compilePermanentRef(n.Args[0], ctx)
		if err != nil {
			return directReturnPlan{}, err
		}
		return directReturnPlan{subject: &subject}, nil
	case "PutEachPermanentIntoItsOwnersHand":
		subjects, err := compileContextualCollection(n.Args[0], ctx)
		if err != nil {
			return directReturnPlan{}, err
		}
		return directReturnPlan{subjects: subjects}, nil
	default:
		return directReturnPlan{}, unsupported(n, "direct return instruction")
	}
}
