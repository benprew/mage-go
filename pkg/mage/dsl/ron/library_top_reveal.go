package ron

import "github.com/benprew/mage-go/pkg/mage"

type libraryRevealPlan struct{ name string }

func (p *libraryRevealPlan) binding(bindings instanceBindings) *mage.CardBinding {
	if binding := bindings.libraryReveals[p]; binding != nil {
		return binding
	}
	binding := mage.BindCard(p.name)
	if bindings.libraryReveals != nil {
		bindings.libraryReveals[p] = binding
	}
	return binding
}

func compileLibraryTopReveal(n Node, player playerReferencePlan) (effectFactory, *libraryRevealPlan, error) {
	if n.Kind != Symbol {
		return nil, nil, problem(n, "RevealTopCardOfLibrary requires a symbol")
	}
	plan := &libraryRevealPlan{name: "revealed library top " + n.Position.String()}
	return func(bindings instanceBindings) mage.Effect {
		return mage.RevealLibraryTopInto(player.build(bindings), plan.binding(bindings))
	}, plan, nil
}
