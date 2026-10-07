package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileRestrictedRepeatedMana(n Node, ctx scope) ([]effectFactory, error) {
	if err := arity(n, 3); err != nil {
		return nil, err
	}
	if n.Args[0].Kind != Symbol || n.Args[0].Name != "AnyManaColor" {
		return nil, unsupported(n.Args[0], "restricted one-color mana production")
	}
	restriction := n.Args[2]
	if restriction.Name != "CanOnlySpendOnSpells" {
		return nil, unsupported(restriction, "mana spending restriction")
	}
	if err := arity(restriction, 1); err != nil {
		return nil, err
	}
	filter := restriction.Args[0]
	if filter.Name != "IsCardtype" {
		return nil, unsupported(filter, "mana spending spell filter")
	}
	if err := arity(filter, 1); err != nil {
		return nil, err
	}
	if filter.Args[0].Kind != Symbol || filter.Args[0].Name != "Creature" {
		return nil, unsupported(filter.Args[0], "mana spending spell type")
	}
	value, err := compileInstanceValue(n.Args[1], ctx)
	if err != nil {
		return nil, err
	}
	return []effectFactory{func(bindings instanceBindings) mage.Effect {
		return mage.AddRestrictedManaOfAnyOneColor(value(bindings), core.CreatureSpellsOnly{})
	}}, nil
}
