package promo

import . "github.com/benprew/mage-go/pkg/mage"

func init() {
	registerLands()
}

func registerLands() {
	// Arena
	// Land
	// {3}, {T}: Tap target creature you control and target creature of an opponent's choice they control. Those creatures fight each other. (Each deals damage equal to its power to the other.)
	Register("Arena", func() Card {
		return NewLand("Arena",
			WithActivatedAbility(
				Pipeline("tap those creatures; they fight each other",
					EffectProperties{Outcome: OutcomeDetriment, Taps: true},
					SnapshotTarget(0, "controller creature"),
					SnapshotTarget(1, "opponent creature"),
					TapGathered("controller creature"),
					TapGathered("opponent creature"),
					FightGathered("controller creature", "opponent creature"),
				),
				Tap(),
				WithCost(ManaCostOf("{3}")),
				WithTarget(TargetCreatureYouControl()),
				WithTarget(TargetOpponentChoice(TargetCreatureOpponentControls())),
			),
		)
	})
}
