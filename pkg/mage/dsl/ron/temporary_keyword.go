package ron

import (
	"fmt"

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

type temporaryAbilityPlan struct {
	keyword   core.Attr
	rampage   int
	isRampage bool
}

func compileTemporaryAbility(n Node) (temporaryAbilityPlan, error) {
	if n.Name == "Rampage" {
		if err := arity(n, 1); err != nil {
			return temporaryAbilityPlan{}, err
		}
		if n.Args[0].Kind != IntegerNode || n.Args[0].Integer < 0 {
			return temporaryAbilityPlan{}, problem(n.Args[0], "rampage requires a nonnegative integer")
		}
		return temporaryAbilityPlan{rampage: n.Args[0].Integer, isRampage: true}, nil
	}
	keyword, err := compileTemporaryKeyword(n)
	return temporaryAbilityPlan{keyword: keyword}, err
}

func (p temporaryAbilityPlan) label() string {
	if p.isRampage {
		return fmt.Sprintf("rampage %d", p.rampage)
	}
	return p.keyword.String()
}

func (p temporaryAbilityPlan) modification(subject mage.PermanentRef) mage.ContinuousModification {
	if p.isRampage {
		trigger := mage.WhenEvent(mage.BecomesBlocked(mage.This), mage.RampageEffect(p.rampage)).(*mage.GenericTriggered)
		return mage.TriggeredAbilityModification(subject, trigger)
	}
	return mage.GrantKeywordModification(subject, p.keyword)
}
