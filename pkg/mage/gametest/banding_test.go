package gametest

import (
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

func init() {
	registerBandingTestCards()
}

func registerBandingTestCards() {
	if !mage.CardRegistered("Benalish Hero") {
		mage.Register("Benalish Hero", func() mage.Card {
			return mage.NewCreature("Benalish Hero", "{W}", 1, 1,
				mage.WithSubTypes("Human", "Soldier"),
				mage.WithKeyword(core.Banding),
			)
		})
	}
	if !mage.CardRegistered("Mesa Pegasus") {
		mage.Register("Mesa Pegasus", func() mage.Card {
			return mage.NewCreature("Mesa Pegasus", "{1}{W}", 1, 1,
				mage.WithSubTypes("Pegasus"),
				mage.WithKeyword(core.Flying),
				mage.WithKeyword(core.Banding),
			)
		})
	}
	if !mage.CardRegistered("Timber Wolves") {
		mage.Register("Timber Wolves", func() mage.Card {
			return mage.NewCreature("Timber Wolves", "{G}", 1, 1,
				mage.WithSubTypes("Wolf"),
				mage.WithKeyword(core.Banding),
			)
		})
	}
	if !mage.CardRegistered("Grizzly Bears") {
		mage.Register("Grizzly Bears", func() mage.Card {
			return mage.NewCreature("Grizzly Bears", "{1}{G}", 2, 2, mage.WithSubTypes("Bear"))
		})
	}
	if !mage.CardRegistered("Hill Giant") {
		mage.Register("Hill Giant", func() mage.Card {
			return mage.NewCreature("Hill Giant", "{3}{R}", 3, 3, mage.WithSubTypes("Giant"))
		})
	}
	if !mage.CardRegistered("Gray Ogre") {
		mage.Register("Gray Ogre", func() mage.Card {
			return mage.NewCreature("Gray Ogre", "{2}{R}", 2, 2, mage.WithSubTypes("Ogre"))
		})
	}
	if !mage.CardRegistered("Craw Wurm") {
		mage.Register("Craw Wurm", func() mage.Card {
			return mage.NewCreature("Craw Wurm", "{4}{G}{G}", 6, 4, mage.WithSubTypes("Wurm"))
		})
	}
	if !mage.CardRegistered("War Mammoth") {
		mage.Register("War Mammoth", func() mage.Card {
			return mage.NewCreature("War Mammoth", "{3}{G}", 3, 3,
				mage.WithSubTypes("Elephant"),
				mage.WithKeyword(core.Trample),
			)
		})
	}
	if !mage.CardRegistered("Black Knight") {
		mage.Register("Black Knight", func() mage.Card {
			return mage.NewCreature("Black Knight", "{B}{B}", 2, 2,
				mage.WithSubTypes("Human", "Knight"),
				mage.WithKeyword(core.FirstStrike),
				mage.WithAbility(mage.ProtectionFromColor(core.White)),
			)
		})
	}
	if !mage.CardRegistered("Helm of Chatzuk") {
		mage.Register("Helm of Chatzuk", func() mage.Card {
			return mage.NewArtifact("Helm of Chatzuk", "{1}",
				mage.WithActivatedAbility(
					mage.GrantKeyword(core.Banding),
					mage.GenericCost(1),
					mage.WithCost(mage.TapSourceCost()),
					mage.WithTarget(mage.TargetCreature()),
				),
			)
		})
	}
	if !mage.CardRegistered("Camel") {
		mage.Register("Camel", func() mage.Card {
			return mage.NewCreature("Camel", "{W}", 0, 1,
				mage.WithSubTypes("Camel"),
				mage.WithKeyword(core.Banding),
			)
		})
	}
}

func TestBandFormation(t *testing.T) {
	t.Run("two_banding_creatures_form_band", func(t *testing.T) {
		g := NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, PlayerA, "Benalish Hero")
		g.AddCard(core.ZoneBattlefield, PlayerA, "Timber Wolves")
		g.FormBand(1, PlayerA, "Benalish Hero", "Timber Wolves")
		g.Attack(1, PlayerA, "Benalish Hero", "Timber Wolves")
		g.StopAt(1, core.DeclareBlockers)
		g.Execute()
		g.AssertBanded(PlayerA, "Benalish Hero", PlayerA, "Timber Wolves", true)
	})

	t.Run("three_banding_creatures_form_band", func(t *testing.T) {
		g := NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, PlayerA, "Benalish Hero")
		g.AddCard(core.ZoneBattlefield, PlayerA, "Mesa Pegasus")
		g.AddCard(core.ZoneBattlefield, PlayerA, "Timber Wolves")
		g.FormBand(1, PlayerA, "Benalish Hero", "Mesa Pegasus", "Timber Wolves")
		g.Attack(1, PlayerA, "Benalish Hero", "Mesa Pegasus", "Timber Wolves")
		g.StopAt(1, core.DeclareBlockers)
		g.Execute()
		g.AssertBanded(PlayerA, "Benalish Hero", PlayerA, "Mesa Pegasus", true)
		g.AssertBanded(PlayerA, "Benalish Hero", PlayerA, "Timber Wolves", true)
		g.AssertBanded(PlayerA, "Mesa Pegasus", PlayerA, "Timber Wolves", true)
	})

	t.Run("banding_plus_one_non_banding", func(t *testing.T) {
		g := NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, PlayerA, "Benalish Hero")
		g.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears")
		g.FormBand(1, PlayerA, "Benalish Hero", "Grizzly Bears")
		g.Attack(1, PlayerA, "Benalish Hero", "Grizzly Bears")
		g.StopAt(1, core.DeclareBlockers)
		g.Execute()
		g.AssertBanded(PlayerA, "Benalish Hero", PlayerA, "Grizzly Bears", true)
	})

	t.Run("non_banding_creatures_dont_band", func(t *testing.T) {
		g := NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, PlayerA, "Hill Giant")
		g.Attack(1, PlayerA, "Grizzly Bears", "Hill Giant")
		g.StopAt(1, core.DeclareBlockers)
		g.Execute()
		g.AssertBanded(PlayerA, "Grizzly Bears", PlayerA, "Hill Giant", false)
	})
}

func TestBandingBlocking(t *testing.T) {
	t.Run("blocking_one_member_blocks_entire_band", func(t *testing.T) {
		g := NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, PlayerA, "Benalish Hero")
		g.AddCard(core.ZoneBattlefield, PlayerA, "Timber Wolves")
		g.AddCard(core.ZoneBattlefield, PlayerB, "Gray Ogre")
		g.FormBand(1, PlayerA, "Benalish Hero", "Timber Wolves")
		g.Attack(1, PlayerA, "Benalish Hero", "Timber Wolves")
		g.Block(1, PlayerB, "Gray Ogre", "Timber Wolves")
		g.ChooseBandingDistribution(PlayerA, map[string]int{"Benalish Hero": 1, "Timber Wolves": 1})
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		g.AssertLife(PlayerB, 20)
	})
}

func TestBandingAttackDamage(t *testing.T) {
	t.Run("controller_distributes_blocker_damage_saves_creature", func(t *testing.T) {
		g := NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, PlayerA, "Benalish Hero")
		g.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, PlayerB, "Hill Giant")
		g.FormBand(1, PlayerA, "Benalish Hero", "Grizzly Bears")
		g.Attack(1, PlayerA, "Benalish Hero", "Grizzly Bears")
		g.Block(1, PlayerB, "Hill Giant", "Benalish Hero")
		g.ChooseBandingDistribution(PlayerA, map[string]int{"Benalish Hero": 3, "Grizzly Bears": 0})
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		g.AssertGraveyardCount(PlayerA, "Benalish Hero", 1)
		g.AssertPermanentCount(PlayerA, "Grizzly Bears", 1)
		g.AssertGraveyardCount(PlayerB, "Hill Giant", 1)
	})

	t.Run("without_banding_normal_damage", func(t *testing.T) {
		g := NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, PlayerB, "Hill Giant")
		g.Attack(1, PlayerA, "Grizzly Bears")
		g.Block(1, PlayerB, "Hill Giant", "Grizzly Bears")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		g.AssertGraveyardCount(PlayerA, "Grizzly Bears", 1)
		g.AssertPermanentCount(PlayerB, "Hill Giant", 1)
	})
}

func TestBandingBlockDamage(t *testing.T) {
	t.Run("blocking_band_distributes_attacker_damage", func(t *testing.T) {
		g := NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, PlayerA, "Hill Giant")
		g.AddCard(core.ZoneBattlefield, PlayerB, "Benalish Hero")
		g.AddCard(core.ZoneBattlefield, PlayerB, "Timber Wolves")
		g.Attack(1, PlayerA, "Hill Giant")
		g.Block(1, PlayerB, "Benalish Hero", "Hill Giant")
		g.Block(1, PlayerB, "Timber Wolves", "Hill Giant")
		g.ChooseBandingDistribution(PlayerB, map[string]int{"Benalish Hero": 3, "Timber Wolves": 0})
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		g.AssertGraveyardCount(PlayerB, "Benalish Hero", 1)
		g.AssertPermanentCount(PlayerB, "Timber Wolves", 1)
		g.AssertPermanentCount(PlayerA, "Hill Giant", 1)
	})
}

func TestBandingTrample(t *testing.T) {
	t.Run("trample_vs_blocking_band", func(t *testing.T) {
		g := NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, PlayerA, "War Mammoth")
		g.AddCard(core.ZoneBattlefield, PlayerB, "Benalish Hero")
		g.AddCard(core.ZoneBattlefield, PlayerB, "Timber Wolves")
		g.Attack(1, PlayerA, "War Mammoth")
		g.Block(1, PlayerB, "Benalish Hero", "War Mammoth")
		g.Block(1, PlayerB, "Timber Wolves", "War Mammoth")
		g.ChooseBandingDistribution(PlayerB, map[string]int{"Benalish Hero": 1, "Timber Wolves": 1})
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		g.AssertGraveyardCount(PlayerB, "Benalish Hero", 1)
		g.AssertGraveyardCount(PlayerB, "Timber Wolves", 1)
		g.AssertLife(PlayerB, 19)
	})

	t.Run("attacking_band_with_trample_unblocked", func(t *testing.T) {
		g := NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, PlayerA, "War Mammoth")
		g.AddCard(core.ZoneBattlefield, PlayerA, "Benalish Hero")
		g.AddCard(core.ZoneBattlefield, PlayerA, "Helm of Chatzuk")
		g.ActivateAbility(1, core.PrecombatMain, PlayerA, "Helm of Chatzuk", "War Mammoth")
		g.FormBand(1, PlayerA, "War Mammoth", "Benalish Hero")
		g.Attack(1, PlayerA, "War Mammoth", "Benalish Hero")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		g.AssertLife(PlayerB, 16)
	})
}

func TestBandingEdgeCases(t *testing.T) {
	t.Run("zero_power_in_band", func(t *testing.T) {
		g := NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, PlayerA, "Camel")
		g.AddCard(core.ZoneBattlefield, PlayerA, "Benalish Hero")
		g.FormBand(1, PlayerA, "Camel", "Benalish Hero")
		g.Attack(1, PlayerA, "Camel", "Benalish Hero")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		g.AssertLife(PlayerB, 19)
	})

	t.Run("first_strike_in_band", func(t *testing.T) {
		g := NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, PlayerA, "Black Knight")
		g.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, PlayerA, "Helm of Chatzuk")
		g.AddCard(core.ZoneBattlefield, PlayerB, "Craw Wurm")
		g.ActivateAbility(1, core.PrecombatMain, PlayerA, "Helm of Chatzuk", "Black Knight")
		g.FormBand(1, PlayerA, "Black Knight", "Grizzly Bears")
		g.Attack(1, PlayerA, "Black Knight", "Grizzly Bears")
		g.Block(1, PlayerB, "Craw Wurm", "Black Knight")
		g.ChooseBandingDistribution(PlayerA, map[string]int{"Black Knight": 4, "Grizzly Bears": 2})
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		g.AssertGraveyardCount(PlayerA, "Black Knight", 1)
		g.AssertGraveyardCount(PlayerA, "Grizzly Bears", 1)
		g.AssertGraveyardCount(PlayerB, "Craw Wurm", 1)
	})
}
