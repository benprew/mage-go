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
			if err := arity(ability, 1); err != nil {
				return nil, err
			}
			quality, err := compileStaticPermanentFilter(ability.Args[0])
			if err != nil {
				return nil, err
			}
			if quality.controlled || quality.excludeSource {
				return nil, unsupported(ability.Args[0], "source-dependent bands quality")
			}
			qualities = append(qualities, quality.filter)
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
