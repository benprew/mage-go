package ron

import "github.com/benprew/mage-go/pkg/mage"

type permanentTargetOptionsFactory func(instanceBindings) []mage.PermanentTargetOption

func compilePlayerRelativeTargetFilter(n Node, ctx scope) (mage.PermanentFilter, permanentTargetOptionsFactory, error) {
	if n.Name == "ControlledByPlayer" {
		if err := arity(n, 1); err != nil {
			return mage.PermanentFilter{}, nil, err
		}
		player, err := compilePlayerRef(n.Args[0], ctx)
		if err != nil {
			return mage.PermanentFilter{}, nil, err
		}
		if n.Args[0].Kind != Symbol || (n.Args[0].Name != "You" && player.defender == nil) {
			return mage.PermanentFilter{}, nil, unsupported(n.Args[0], "target controller reference")
		}
		return mage.AnyPermanent, func(bindings instanceBindings) []mage.PermanentTargetOption {
			return []mage.PermanentTargetOption{mage.TargetControlledBy(player.build(bindings))}
		}, nil
	}
	if n.Name == "And" {
		if err := arity(n, 1); err != nil {
			return mage.PermanentFilter{}, nil, err
		}
		parts, err := list(n.Args[0])
		if err != nil {
			return mage.PermanentFilter{}, nil, err
		}
		var filters []mage.PermanentFilter
		var factories []permanentTargetOptionsFactory
		for _, part := range parts {
			filter, options, err := compilePlayerRelativeTargetFilter(part, ctx)
			if err != nil {
				return mage.PermanentFilter{}, nil, err
			}
			filters = append(filters, filter)
			factories = append(factories, options)
		}
		return mage.And(filters...), func(bindings instanceBindings) []mage.PermanentTargetOption {
			options := make([]mage.PermanentTargetOption, 0, len(factories))
			for _, factory := range factories {
				options = append(options, factory(bindings)...)
			}
			return options
		}, nil
	}
	filter, options, err := compilePermanentTargetFilter(n, ctx)
	return filter, func(instanceBindings) []mage.PermanentTargetOption { return options }, err
}
