package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileBandsWithOtherLayer(n Node, parts []Node) (optionFactory, error) {
	if n.Name != "EachPermanentLayerEffect" {
		return nil, unsupported(n, "bands-with-other context")
	}
	subjects, err := compileStaticPermanentFilter(n.Args[0])
	if err != nil {
		return nil, err
	}
	var qualities []mage.PermanentFilter
	for _, part := range parts {
		if part.Name != "AddAbility" {
			return nil, unsupported(part, "bands-with-other layer operation")
		}
		if err := arity(part, 1); err != nil {
			return nil, err
		}
		abilities, err := list(part.Args[0])
		if err != nil {
			return nil, err
		}
		if len(abilities) == 0 {
			return nil, problem(part, "ability list must not be empty")
		}
		for _, ability := range abilities {
			if ability.Name != "BandsWithOthers" {
				return nil, unsupported(ability, "combined bands-with-other grant")
			}
			quality, err := compileBandsQuality(ability)
			if err != nil {
				return nil, err
			}
			qualities = append(qualities, quality)
		}
	}
	return func() mage.CardOption {
		modifications := make([]mage.ContinuousModification, len(qualities))
		for i, quality := range qualities {
			modifications[i] = mage.GrantEachBandsWithOther(subjects.build(), quality)
		}
		condition := mage.StateConditionFunc("static bands with other", []mage.StateDependency{mage.StateBattlefield}, func(mage.StateReader, mage.StateContext) bool { return true })
		return mage.WithAbility(mage.Static(mage.While(condition, modifications...)))
	}, nil
}

func compileBandsQuality(n Node) (mage.PermanentFilter, error) {
	if err := arity(n, 1); err != nil {
		return mage.PermanentFilter{}, err
	}
	quality, err := compileStaticPermanentFilter(n.Args[0])
	if err != nil {
		return mage.PermanentFilter{}, err
	}
	if quality.controlled || quality.excludeSource {
		return mage.PermanentFilter{}, unsupported(n.Args[0], "source-dependent bands quality")
	}
	return quality.filter, nil
}

func compileTokenBandsAbility(n Node) (func() mage.Ability, error) {
	quality, err := compileBandsQuality(n)
	if err != nil {
		return nil, err
	}
	return func() mage.Ability {
		condition := mage.StateConditionFunc("token bands with other", []mage.StateDependency{mage.StateBattlefield}, func(mage.StateReader, mage.StateContext) bool { return true })
		return mage.Static(mage.While(condition, mage.GrantBandsWithOther(mage.This, quality)))
	}, nil
}
