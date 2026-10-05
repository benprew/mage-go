package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileTemporaryKeyword(n Node) (core.Attr, error) {
	if n.Name == "Landwalk" {
		return compileLandwalkKeyword(n)
	}
	if keyword, ok := keywords[n.Name]; ok && n.Kind == Symbol {
		return keyword, nil
	}
	return 0, unsupported(n, "temporary keyword")
}

func compileTemporaryKeywordRemoval(n Node, subject permanentReferencePlan) (func(instanceBindings) mage.ContinuousModification, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	ability := n.Args[0]
	if ability.Kind == Symbol && ability.Name == "AnyLandwalk" {
		return func(bindings instanceBindings) mage.ContinuousModification {
			return mage.RemoveLandwalk(subject.build(bindings))
		}, nil
	}
	keyword, err := compileTemporaryKeyword(ability)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.ContinuousModification {
		return mage.RemoveKeyword(subject.build(bindings), keyword)
	}, nil
}
