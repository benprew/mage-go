package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileIntrinsicObjectText(n Node, ctx scope, subject permanentReferencePlan, parts []Node) (effectFactory, error) {
	if len(parts) == 0 {
		return nil, problem(n, "supply text substitutions")
	}
	changes := make([]textSubstitutionFactory, len(parts))
	for i, part := range parts {
		build, err := compileTextSubstitution(part, ctx)
		if err != nil {
			return nil, err
		}
		changes[i] = build
	}
	return func(bindings instanceBindings) mage.Effect {
		refs := make([]mage.TextChangeRef, len(changes))
		for i, build := range changes {
			refs[i] = build(bindings)
		}
		return mage.ChangePrintedPermanentText(subject.build(bindings), core.Indefinite, refs...)
	}, nil
}
