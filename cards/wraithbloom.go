package cards

import "github.com/mage/mage"

func init() {
	registerWraithbloom()
}

func registerWraithbloom() {
	mage.Register("Wraithbloom Cultivator", func() mage.Card {
		c := mage.NewCreature("Wraithbloom Cultivator", "{1}{B}{G}", "Elf", "Shaman")
		c.Power_ = 2
		c.Toughness_ = 3

		// Whenever another creature you control dies, you gain 1 life and
		// put a +1/+1 counter on Wraithbloom Cultivator.
		c.AddAbility(
			mage.DiesCreatureTrigger(
				mage.CompositeEffects("gain 1 life and put a +1/+1 counter on this",
					mage.GainLife(1),
					mage.AddCountersToSource(mage.P1P1, 1),
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
			).AddCost(mage.TapSourceCost()).
				AddCost(mage.RemoveCountersCost(mage.P1P1, 3)).
				AddTarget(mage.TargetCreatureInYourGraveyard()),
		)

		return c
	})
}
