package mage

import (
	"fmt"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type manaAmountEffect struct {
	color  core.Color
	amount ResolutionValue
}

// AddManaAmount adds mana to the resolving controller's pool.
// It reads the amount once. A nonpositive amount adds no mana.
func AddManaAmount(color core.Color, amount ResolutionValue) ResolvingEffect {
	return &manaAmountEffect{color: color, amount: amount}
}

func (e *manaAmountEffect) Text() string {
	if nilDefinition(e.amount) {
		return fmt.Sprintf("add %s mana", e.color)
	}
	return fmt.Sprintf("add %s %s mana", e.amount.Text(), e.color)
}
func (e *manaAmountEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *manaAmountEffect) inspectDefinition() definitionNode {
	node := definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"amount", e.amount}}}
	if e.color < core.Colorless || e.color >= core.AnyColor {
		node.problem = "supply a concrete mana color"
	}
	return node
}
func (e *manaAmountEffect) Apply(ctx *EffectContext) error {
	if ctx.Game.GetPlayer(ctx.Controller) == nil {
		return ErrPlayerNotFound
	}
	amount := e.amount.resolveValue(ctx)
	if amount <= 0 {
		return nil
	}
	return AddMana(e.color, amount).Apply(ctx)
}

// IsActivatedManaAbility uses the same classification as immediate resolution.
func IsActivatedManaAbility(ability Ability) bool {
	inner := UnwrapAbility(ability)
	if _, ok := inner.(*ManaAbility); ok {
		return true
	}
	activated, ok := inner.(ActivatedAbility)
	return ok && inner.Type() == core.AbilityActivated && canProduceMana(activated.Effects(), activated.Targets())
}

func canProduceMana(effects []Effect, targets []Target) bool {
	if len(targets) != 0 {
		return false
	}
	for _, effect := range effects {
		switch e := effect.(type) {
		case *manaAmountEffect:
			return true
		case *addManaEffect:
			if e.amount > 0 {
				return true
			}
		case *addAnyManaEffect:
			if e.amount > 0 {
				return true
			}
		case *sequenceEffect:
			if canProduceMana(e.effects, nil) {
				return true
			}
		case *compositeEffect:
			if canProduceMana(e.effects, nil) {
				return true
			}
		}
	}
	return false
}
