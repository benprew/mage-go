package ron

import "github.com/benprew/mage-go/pkg/mage"

type tapInstructionPlan struct {
	subject   *permanentReferencePlan
	subjects  permanentSetFactory
	operation mage.TapOperationKind
}

func (plan tapInstructionPlan) build(bindings instanceBindings) mage.TapInstruction {
	instruction := mage.TapInstruction{Operation: plan.operation}
	if plan.subject != nil {
		instruction.Subject = plan.subject.build(bindings)
	} else {
		instruction.Subjects = plan.subjects(bindings)
	}
	return instruction
}

func compileTapAction(n Node, ctx scope) (effectFactory, error) {
	plan, err := compileTapInstructionPlan(n, ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect { return plan.build(bindings).Effect() }, nil
}

func compileTapInstructionPlan(n Node, ctx scope) (tapInstructionPlan, error) {
	if err := arity(n, 1); err != nil {
		return tapInstructionPlan{}, err
	}
	plan := tapInstructionPlan{operation: mage.TapOperation}
	switch n.Name {
	case "TapPermanent", "UntapPermanent", "TapOrUntapPermanent":
		subject, err := compilePermanentRef(n.Args[0], ctx)
		if err != nil {
			return tapInstructionPlan{}, err
		}
		plan.subject = &subject
		if n.Name == "UntapPermanent" {
			plan.operation = mage.UntapOperation
		}
		if n.Name == "TapOrUntapPermanent" {
			plan.operation = mage.MayTapOrUntapOperation
		}
	case "TapEachPermanent", "UntapEachPermanent":
		subjects, err := compileContextualCollection(n.Args[0], ctx)
		if err != nil {
			return tapInstructionPlan{}, err
		}
		plan.subjects = subjects
		if n.Name == "UntapEachPermanent" {
			plan.operation = mage.UntapOperation
		}
	default:
		return tapInstructionPlan{}, unsupported(n, "tap instruction")
	}
	return plan, nil
}
