package ron

import "github.com/benprew/mage-go/pkg/mage"

func compilePermanentCastingRequirement(n Node) (optionFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	restriction := n.Args[0]
	if restriction.Name != "CantBeCastUnless" {
		return nil, unsupported(restriction, "permanent casting restriction")
	}
	if err := arity(restriction, 1); err != nil {
		return nil, err
	}
	condition := restriction.Args[0]
	if condition.Name != "NoPermanentsPassFilter" {
		return nil, unsupported(condition, "permanent casting condition")
	}
	if err := arity(condition, 2); err != nil {
		return nil, err
	}
	zone := condition.Args[1]
	if zone.Kind != Symbol || zone.Name != "OnTheBattlefield" {
		return nil, unsupported(zone, "permanent casting condition zone")
	}
	filter, err := compileStaticPermanentFilter(condition.Args[0])
	if err != nil {
		return nil, err
	}
	if filter.controlled || filter.excludeSource {
		return nil, problem(condition.Args[0], "casting absence requires a reference-free permanent filter")
	}
	return func() mage.CardOption {
		return mage.WithAbility(mage.CastingRequirement(mage.BattlefieldHasNone(filter.filter)))
	}, nil
}
