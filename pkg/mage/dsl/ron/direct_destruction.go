package ron

import "github.com/benprew/mage-go/pkg/mage"

type directDestructionPlan struct {
	subject          *permanentReferencePlan
	subjects         permanentSetFactory
	cannotRegenerate bool
}

func (plan directDestructionPlan) build(bindings instanceBindings) mage.DirectDestructionInstruction {
	instruction := mage.DirectDestructionInstruction{CannotRegenerate: plan.cannotRegenerate}
	if plan.subject != nil {
		instruction.Subject = plan.subject.build(bindings)
	} else {
		instruction.Subjects = plan.subjects(bindings)
	}
	return instruction
}

func compileDirectDestruction(n Node, ctx scope) (effectFactory, error) {
	plan, err := compileDirectDestructionPlan(n, ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect { return plan.build(bindings).Effect() }, nil
}

func compileDirectDestructionPlan(n Node, ctx scope) (directDestructionPlan, error) {
	if err := arity(n, 1); err != nil {
		return directDestructionPlan{}, err
	}
	plan := directDestructionPlan{cannotRegenerate: n.Name == "DestroyPermanentNoRegen" || n.Name == "DestroyEachPermanentNoRegen"}
	switch n.Name {
	case "DestroyPermanent", "DestroyPermanentNoRegen":
		subject, err := compilePermanentRef(n.Args[0], ctx)
		if err != nil {
			return directDestructionPlan{}, err
		}
		plan.subject = &subject
	case "DestroyEachPermanent", "DestroyEachPermanentNoRegen":
		subjects, err := compileDestructionCollection(n.Args[0], ctx)
		if err != nil {
			return directDestructionPlan{}, err
		}
		plan.subjects = subjects
	default:
		return directDestructionPlan{}, unsupported(n, "direct destruction instruction")
	}
	return plan, nil
}
