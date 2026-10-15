package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileResolvingMana(n Node, ctx scope) ([]effectFactory, error) {
	if n.Name == "AddMana" && len(n.Args) > 0 && n.Args[0].Name == "TheManaLostThisWay" {
		return compileLostManaConsumer(n, ctx)
	}

	if n.Name == "AddMana" && len(n.Args) == 1 && n.Args[0].Name == "Trigger_AnyManaTypeProduced" {
		return compilePlayerMana(n, ctx)
	}
	var dynamic valueFactory
	count := 1
	args := 1
	if n.Name == "AddManaRepeated" {
		args = 2
	}
	if err := arity(n, args); err != nil {
		return nil, err
	}
	if args == 2 {
		value := n.Args[1]
		if value.Name == "TheNumberOfCountersRemovedThisWay" {
			return compileBoundRepeatedMana(n, ctx)
		}
		if value.Name == "ManaValueOfTheSacrificedPermanent" {
			if ctx.sacrificeValue == nil {
				return nil, problem(value, "unbound sacrifice mana value")
			}
			production := n.Args[0]
			if production.Kind != Symbol || production.Name == "AnyManaColor" {
				return nil, unsupported(production, "fixed-color sacrifice mana production")
			}
			return compileBoundRepeatedMana(n, ctx)
		}
		if value.Name == "ManaValueOfSpell" {
			if ctx.enteringSource || ctx.drawReplacementBody {
				return nil, problem(value, "spell mana value production requires an ordinary resolving body")
			}
			var err error
			dynamic, err = compileSpellManaValue(value, ctx)
			if err != nil {
				return nil, err
			}
		} else if value.Name == "ManaValueOfPermanent" {
			var err error
			dynamic, err = compileInstanceValue(value, ctx)
			if err != nil {
				return nil, err
			}
		} else if value.Name != "Integer" {
			return nil, unsupported(value, "mana amount")
		}
		if dynamic == nil {
			if err := arity(value, 1); err != nil {
				return nil, err
			}
			if value.Args[0].Kind != IntegerNode || value.Args[0].Integer < 0 {
				return nil, problem(value, "mana amount must be nonnegative")
			}
			count = value.Args[0].Integer
		}
	}
	productions, err := compileMana(n.Args[0])
	if err != nil {
		return nil, err
	}
	if dynamic != nil && (len(productions) != 1 || productions[0].Color == core.AnyColor) {
		return nil, unsupported(n.Args[0], "fixed-color mana-value production")
	}
	var effects []effectFactory
	for _, production := range productions {
		amount, color := production.Amount*count, production.Color
		effects = append(effects, func(bindings instanceBindings) mage.Effect {
			if dynamic != nil {
				value := mage.MultiplyValues(mage.Fixed(production.Amount), dynamic(bindings))
				return mage.AddManaAmount(color, value)
			}
			if color == core.AnyColor {
				return mage.AddManaOfAnyOneColor(mage.Fixed(amount))
			}
			return mage.AddMana(color, amount)
		})
	}
	return effects, nil
}
