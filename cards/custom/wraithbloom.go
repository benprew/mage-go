package custom

import (
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage"
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

func init() {
	registerWraithbloom()
}

func registerWraithbloom() {
	Register("Wraithbloom Cultivator", func() Card {
		return NewCreature("Wraithbloom Cultivator", "{1}{B}{G}", 2, 3,
			WithSubTypes("Elf", "Shaman"),

			// Whenever another creature you control dies, you gain 1 life and
			// put a +1/+1 counter on Wraithbloom Cultivator.
			WithAbility(DiesCreatureTrigger(
				CompositeEffects("gain 1 life and put a +1/+1 counter on this",
					GainLife(1),
					AddCounters(P1P1, Fixed(1)).Targeting(ToSource()),
				),
				false, PermanentFilter{},
			)),

			// {2}, {T}, Remove three +1/+1 counters from Wraithbloom Cultivator:
			// Return target creature card from your graveyard to the battlefield.
			WithActivatedAbility(
				ReturnFromGraveyardToBattlefield(),
				GenericCost(2),
				WithCost(TapSourceCost()),
				WithCost(RemoveCountersCost(P1P1, 3)),
				WithTarget(TargetCreatureInYourGraveyard()),
			),
		)
	})
}
