package promo

import . "github.com/benprew/mage-go/pkg/mage/dsl"

func init() {
	registerLands()
}

func registerLands() {
	// Arena
	// Land
	// {3}, {T}: Tap target creature you control and target creature of an opponent's choice they control. Those creatures fight each other. (Each deals damage equal to its power to the other.)
	Register("Arena", func() Card {
		first := TargetPermanent("controller creature", IsCreature, TargetYouControl())
		second := TargetPermanent("opponent creature", IsCreature, TargetOpponentControls(), TargetChosenByOpponent())
		return NewLand("Arena",
			WithAbility(Activate(Costs(ManaCostOf("{3}"), TapCost(This)),
				Pipeline("tap those creatures; they fight each other",
					EffectProperties{Outcome: OutcomeDetriment, Taps: true},
					SnapshotTarget(0, "controller creature"),
					SnapshotTarget(1, "opponent creature"),
					Tap(first),
					Tap(second),
					FightGathered("controller creature", "opponent creature"),
				),
				WithTargets(first, second),
			)),
		)
	})
}
