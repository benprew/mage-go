package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileSpellAbilityExclusion(n Node, ctx scope) (mage.SpellFilter, error) {
	if err := arity(n, 1); err != nil {
		return mage.SpellFilter{}, err
	}
	if !ctx.permanent || !ctx.activation {
		return mage.SpellFilter{}, unsupported(n, "ability target exclusion context")
	}
	ability := n.Args[0]
	if ability.Name != "AbilityOfAPermanent" {
		return mage.SpellFilter{}, unsupported(ability, "stack ability source selector")
	}
	if err := arity(ability, 1); err != nil {
		return mage.SpellFilter{}, err
	}
	filter, err := compileStaticPermanentFilter(ability.Args[0])
	if err != nil {
		return mage.SpellFilter{}, err
	}
	if filter.controlled {
		return mage.SpellFilter{}, unsupported(ability.Args[0], "stack ability source controller")
	}
	return mage.SpellNotTargetedByPermanentAbility(filter.filter, filter.excludeSource), nil
}
