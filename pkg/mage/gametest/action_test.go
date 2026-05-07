package gametest

import (
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

func TestUnifiedActionSpellAndActivatedAbilityResolve(t *testing.T) {
	boltName := "Unified Action Bolt"
	pingerName := "Unified Action Pinger"
	if !mage.CardRegistered(boltName) {
		mage.Register(boltName, func() mage.Card {
			return mage.NewInstant(boltName, "{R}",
				mage.NewSpell(
					mage.DealDamage(mage.Fixed(3)),
					mage.WithTarget(mage.TargetAnyTarget()),
				),
			)
		})
	}
	if !mage.CardRegistered(pingerName) {
		mage.Register(pingerName, func() mage.Card {
			return mage.NewCreature(pingerName, "{2}{U}", 1, 1,
				mage.WithAction(mage.NewActivated(
					mage.Tap(),
					mage.DealDamage(mage.Fixed(1)),
					mage.WithTarget(mage.TargetAnyTarget()),
				)),
			)
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, boltName)
	tg.AddCard(core.ZoneBattlefield, PlayerA, pingerName)
	tg.CastSpell(1, core.PrecombatMain, PlayerA, boltName, "PlayerB")
	tg.ActivateAbility(1, core.PrecombatMain, PlayerA, pingerName, "PlayerB")
	tg.StopAt(1, core.EndCombat)
	tg.Execute()

	tg.AssertLife(PlayerB, 16)
	tg.AssertTapped(PlayerA, pingerName, true)
}

func TestUnifiedActionCompatibilityWrappers(t *testing.T) {
	spellName := "Unified Compat Drain"
	abilityName := "Unified Compat Healer"
	if !mage.CardRegistered(spellName) {
		mage.Register(spellName, func() mage.Card {
			return mage.NewSorcery(spellName, "{B}",
				mage.NewTargetedSpell(mage.TargetPlayer(), mage.DealDamage(mage.Fixed(2))),
			)
		})
	}
	if !mage.CardRegistered(abilityName) {
		mage.Register(abilityName, func() mage.Card {
			return mage.NewArtifact(abilityName, "{1}",
				mage.WithActivatedAbility(
					mage.GainLife(1),
					mage.Tap(),
				),
			)
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, spellName)
	tg.AddCard(core.ZoneBattlefield, PlayerA, abilityName)
	tg.CastSpell(1, core.PrecombatMain, PlayerA, spellName, "PlayerB")
	tg.ActivateAbility(1, core.PrecombatMain, PlayerA, abilityName)
	tg.StopAt(1, core.EndCombat)
	tg.Execute()

	tg.AssertLife(PlayerA, 21)
	tg.AssertLife(PlayerB, 18)
}

func TestNewActionBuilderForSpell(t *testing.T) {
	name := "Unified NewAction Growth"
	creatureName := "Unified NewAction Bear"
	if !mage.CardRegistered(name) {
		mage.Register(name, func() mage.Card {
			return mage.NewInstant(name, "{G}",
				mage.NewAction(
					mage.ActionSpell,
					mage.WithEffect(mage.Boost(mage.Fixed(3), mage.Fixed(3))),
					mage.WithTarget(mage.TargetCreature()),
				),
			)
		})
	}
	if !mage.CardRegistered(creatureName) {
		mage.Register(creatureName, func() mage.Card {
			return mage.NewCreature(creatureName, "{1}{G}", 2, 2, mage.WithSubTypes("Bear"))
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, creatureName)
	tg.AddCard(core.ZoneHand, PlayerA, name)
	tg.CastSpell(1, core.PrecombatMain, PlayerA, name, creatureName)
	tg.StopAt(1, core.EndCombat)
	tg.Execute()

	tg.AssertPowerToughness(PlayerA, creatureName, 5, 5)
}

func TestUnifiedSpellActionPaysActionCosts(t *testing.T) {
	name := "Unified Costed Spell"
	discardName := "Unified Cost Fodder"
	if !mage.CardRegistered(name) {
		mage.Register(name, func() mage.Card {
			return mage.NewInstant(name, "{R}",
				mage.NewSpell(
					mage.DealDamage(mage.Fixed(3)),
					mage.WithTarget(mage.TargetAnyTarget()),
					mage.WithCost(mage.DiscardCost(1)),
				),
			)
		})
	}
	if !mage.CardRegistered(discardName) {
		mage.Register(discardName, func() mage.Card {
			return mage.NewCreature(discardName, "{1}", 1, 1, mage.WithSubTypes("Test"))
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, name)
	tg.AddCard(core.ZoneHand, PlayerA, discardName)
	tg.ChooseDiscard(PlayerA, discardName)
	tg.CastSpell(1, core.PrecombatMain, PlayerA, name, "PlayerB")
	tg.StopAt(1, core.EndCombat)
	tg.Execute()

	tg.AssertLife(PlayerB, 17)
	tg.AssertGraveyardCount(PlayerA, discardName, 1)
}

func TestInstantAndSorceryAcceptActionParts(t *testing.T) {
	boltName := "Unified Direct Bolt"
	wrathName := "Unified Direct Wrath"
	bearName := "Unified Direct Bear"
	if !mage.CardRegistered(boltName) {
		mage.Register(boltName, func() mage.Card {
			return mage.NewInstant(boltName, "{R}",
				mage.DealDamage(mage.Fixed(3)),
				mage.WithTarget(mage.TargetAnyTarget()),
			)
		})
	}
	if !mage.CardRegistered(wrathName) {
		mage.Register(wrathName, func() mage.Card {
			return mage.NewSorcery(wrathName, "{2}{W}{W}",
				mage.DestroyAllCreatures(),
			)
		})
	}
	if !mage.CardRegistered(bearName) {
		mage.Register(bearName, func() mage.Card {
			return mage.NewCreature(bearName, "{1}{G}", 2, 2, mage.WithSubTypes("Bear"))
		})
	}

	t.Run("instant direct parts", func(t *testing.T) {
		tg := NewTestGame(t)
		tg.AddCard(core.ZoneHand, PlayerA, boltName)
		tg.CastSpell(1, core.PrecombatMain, PlayerA, boltName, "PlayerB")
		tg.StopAt(1, core.EndCombat)
		tg.Execute()

		tg.AssertLife(PlayerB, 17)
	})

	t.Run("sorcery direct parts", func(t *testing.T) {
		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, bearName)
		tg.AddCard(core.ZoneBattlefield, PlayerB, bearName)
		tg.AddCard(core.ZoneHand, PlayerA, wrathName)
		tg.CastSpell(1, core.PrecombatMain, PlayerA, wrathName)
		tg.StopAt(1, core.EndCombat)
		tg.Execute()

		tg.AssertPermanentCount(PlayerA, bearName, 0)
		tg.AssertPermanentCount(PlayerB, bearName, 0)
	})
}
