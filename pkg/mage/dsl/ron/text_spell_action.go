package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileSpellTextAction(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	subject, err := compileSpellRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	parts, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, problem(n, "supply text substitutions")
	}
	changes := make([]textSubstitutionFactory, len(parts))
	for i, part := range parts {
		changes[i], err = compileTextSubstitution(part, ctx)
		if err != nil {
			return nil, err
		}
	}
	return func(bindings instanceBindings) mage.Effect {
		refs := make([]mage.TextChangeRef, len(changes))
		for i, build := range changes {
			refs[i] = build(bindings)
		}
		return mage.ChangePrintedSpellText(subject.build(bindings), refs...)
	}, nil
}
