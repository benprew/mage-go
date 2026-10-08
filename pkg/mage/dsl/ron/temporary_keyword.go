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
	if ability.Kind == Symbol && ability.Name == "AnyBandsWithOthers" {
		return func(bindings instanceBindings) mage.ContinuousModification {
			return mage.RemoveBandsWithOther(subject.build(bindings))
		}, nil
	}
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
	keyword    core.Attr
	rampage    int
	isRampage  bool
	protection uint8
}

func compileTemporaryAbility(n Node) (temporaryAbilityPlan, error) {
	if n.Name == "Protection" {
		if err := arity(n, 1); err != nil {
			return temporaryAbilityPlan{}, err
		}
		colors, err := protectionColors(n.Args[0])
		if err != nil {
			return temporaryAbilityPlan{}, err
		}
		if len(colors) == 0 {
			return temporaryAbilityPlan{}, problem(n, "supply at least one protection color")
		}
		for _, color := range colors {
			if color < core.White || color > core.Green {
				return temporaryAbilityPlan{}, problem(n, "supply one of the five protection colors")
			}
		}
		var mask uint8
		for _, color := range colors {
			mask |= 1 << uint(color-core.White)
		}
		return temporaryAbilityPlan{protection: mask}, nil
	}
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
	if p.protection != 0 {
		return fmt.Sprintf("protection from %v", p.protectionColors())
	}
	if p.isRampage {
		return fmt.Sprintf("rampage %d", p.rampage)
	}
	return p.keyword.String()
}

func (p temporaryAbilityPlan) modification(subject mage.PermanentRef) mage.ContinuousModification {
	if p.protection != 0 {
		return mage.GrantProtectionColors(subject, p.protectionColors()...)
	}
	if p.isRampage {
		trigger := mage.WhenEvent(mage.BecomesBlocked(mage.This), mage.RampageEffect(p.rampage)).(*mage.GenericTriggered)
		return mage.TriggeredAbilityModification(subject, trigger)
	}
	return mage.GrantKeywordModification(subject, p.keyword)
}

func (p temporaryAbilityPlan) protectionColors() []core.Color {
	var colors []core.Color
	for color := core.White; color <= core.Green; color++ {
		if p.protection&(1<<uint(color-core.White)) != 0 {
			colors = append(colors, color)
		}
	}
	return colors
}
