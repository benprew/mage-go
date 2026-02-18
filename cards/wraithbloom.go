package cards

import "github.com/mage/mage"

func init() {
	registerWraithbloom()
}

func registerWraithbloom() {
	mage.Register("Wraithbloom Cultivator", func() mage.Card {
		c := mage.NewCreature("Wraithbloom Cultivator", "{1}{B}{G}", 2, 3, "Elf", "Shaman")

		// Whenever another creature you control dies, you gain 1 life and
		// put a +1/+1 counter on Wraithbloom Cultivator.
		c.AddAbility(
			mage.DiesCreatureTrigger(
				mage.CompositeEffects("gain 1 life and put a +1/+1 counter on this",
					mage.GainLife(1),
					mage.AddCounters(mage.P1P1, mage.Fixed(1), mage.SelectSource),
				),
				false, nil,
			),
		)

		// {2}, {T}, Remove three +1/+1 counters from Wraithbloom Cultivator:
		// Return target creature card from your graveyard to the battlefield.
		c.AddAbility(
			mage.NewActivatedAbility(
				mage.ReturnFromGraveyardToBattlefield(),
				mage.GenericCost(2),

				mage.WithCost(mage.TapSourceCost()),
				mage.WithCost(mage.RemoveCountersCost(mage.P1P1, 3)),
				mage.WithTarget(mage.TargetCreatureInYourGraveyard()),
			),
		)

		return c
	})
}
