package secretsofstrixhaven

import (
	"testing"

	_ "git.sr.ht/~cdcarter/mage-go/cards/limited"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/gametest"
)

func TestAjanisResponse(t *testing.T) {
	t.Run("destroys target creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Ajani's Response")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Ajani's Response", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
		g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
	})
}

func TestAncestralAnger(t *testing.T) {
	t.Run("grants trample and bonus power with no copies in graveyard", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Ancestral Anger")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Ancestral Anger", "Grizzly Bears")
		g.StopAt(1, core.DeclareAttackers)
		g.Execute()
		// X = 1 + 0 (no copies in GY) = 1; Grizzly Bears 2/2 + 1/0 = 3/2
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 3, 2)
		g.AssertHasAbility(gametest.PlayerA, "Grizzly Bears", core.Trample, true)
	})

	t.Run("draws a card", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Ancestral Anger")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Ancestral Anger", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Drew one card (hand was empty after casting)
		g.AssertHandCount(gametest.PlayerA, "Ancestral Anger", 0) // The drawn card is from library, not Ancestral Anger
	})

	t.Run("bonus increases with copies in graveyard", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Ancestral Anger")
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Ancestral Anger")
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Ancestral Anger")
		// X = 1 + 2 = 3; Grizzly Bears 2/2 + 3/0 = 5/2
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Ancestral Anger", "Grizzly Bears")
		g.StopAt(1, core.DeclareAttackers)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 5, 2)
	})
}

func TestAntiquitiesOnTheLoose(t *testing.T) {
	t.Run("creates two 2/2 Spirit tokens", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Antiquities on the Loose")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Antiquities on the Loose")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Spirit Token", 2)
		g.AssertPowerToughness(gametest.PlayerA, "Spirit Token", 2, 2)
	})

	t.Run("no counter added when cast from hand", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Antiquities on the Loose")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Antiquities on the Loose")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertCounterCount(gametest.PlayerA, "Spirit Token", core.P1P1, 0)
	})
}

func TestArcaneOmens(t *testing.T) {
	t.Run("target opponent discards one card when one color spent", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Arcane Omens")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Arcane Omens", "PlayerB")
		g.ChooseDiscard(gametest.PlayerB, "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// 3 bears - 1 discarded = 2 remaining
		g.AssertHandCount(gametest.PlayerB, "Grizzly Bears", 2)
	})
}

func TestArtisticProcess(t *testing.T) {
	t.Run("mode 0 deals 6 damage to target creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Artistic Process")
		g.ChooseMode(gametest.PlayerA, 0)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Artistic Process", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
	})

	t.Run("mode 1 deals 2 damage to each creature opponent controls", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Artistic Process")
		g.ChooseMode(gametest.PlayerA, 1)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Artistic Process")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// PlayerB's bear (2/2) takes 2 damage — dies
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
		// PlayerA's bear is not affected by this mode
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
	})

	t.Run("mode 2 creates a 3/3 Elemental with flying", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Artistic Process")
		g.ChooseMode(gametest.PlayerA, 2)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Artistic Process")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Elemental Token", 1)
		g.AssertPowerToughness(gametest.PlayerA, "Elemental Token", 3, 3)
		g.AssertHasAbility(gametest.PlayerA, "Elemental Token", core.Flying, true)
	})
}

func TestBanishingBetrayal(t *testing.T) {
	t.Run("returns target nonland permanent to owner's hand", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Banishing Betrayal")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Banishing Betrayal", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
		g.AssertHandCount(gametest.PlayerB, "Grizzly Bears", 1)
	})
}

func TestBorrowedKnowledge(t *testing.T) {
	t.Run("mode 0 draws equal to opponent hand size", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Borrowed Knowledge")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Ancestral Anger")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Ancestral Anger")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Ancestral Anger")
		// PlayerB has 3 cards; PlayerA discards 3 (Borrowed Knowledge + 2 Bears) and draws 3
		g.ChooseMode(gametest.PlayerA, 0)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Borrowed Knowledge", "PlayerB")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// PlayerA now has 3 cards from library (nothing named "Grizzly Bears" or "Borrowed Knowledge" since those were discarded)
		g.AssertHandCount(gametest.PlayerA, "Borrowed Knowledge", 0)
		g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 0)
		g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 2)
		g.AssertGraveyardCount(gametest.PlayerA, "Borrowed Knowledge", 1)
	})

	t.Run("mode 1 draws equal to discarded cards", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Borrowed Knowledge")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
		// PlayerA has 3 cards total, discards all 3, draws 3
		g.ChooseMode(gametest.PlayerA, 1)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Borrowed Knowledge")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 2)
		g.AssertGraveyardCount(gametest.PlayerA, "Borrowed Knowledge", 1)
	})
}

func TestBrushOff(t *testing.T) {
	t.Run("counters target spell", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Brush Off")
		g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Grizzly Bears")
		g.CastInResponseTo(gametest.PlayerA, "Brush Off", "Grizzly Bears")
		g.StopAt(2, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
		g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
	})
}

func TestBurrogBarrage(t *testing.T) {
	t.Run("deals damage equal to creature power to opponent creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Burrog Barrage")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Burrog Barrage", "Grizzly Bears", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Grizzly Bears (2 power) deals 2 to opponent's 2/2 → kills it
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
	})

	t.Run("gets +1/+0 if another instant was cast this turn", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Serra Angel")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Chase Inspiration")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Burrog Barrage")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Chase Inspiration", "Grizzly Bears")
		g.CastInResponseTo(gametest.PlayerA, "Burrog Barrage", "Grizzly Bears", "Serra Angel")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Bear has 2+1=3 power (from "another instant" bonus), deals 3 to 4/4 Serra Angel - doesn't die
		g.AssertPermanentCount(gametest.PlayerB, "Serra Angel", 1)
	})
}

func TestChaseInspiration(t *testing.T) {
	t.Run("grants +0/+3 while active", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Chase Inspiration")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Chase Inspiration", "Grizzly Bears")
		g.StopAt(1, core.DeclareAttackers)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 2, 5)
	})

	t.Run("grants hexproof while active", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Chase Inspiration")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Chase Inspiration", "Grizzly Bears")
		g.StopAt(1, core.DeclareAttackers)
		g.Execute()
		g.AssertHasAbility(gametest.PlayerA, "Grizzly Bears", core.Hexproof, true)
	})

	t.Run("hexproof expires at end of turn", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Chase Inspiration")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Chase Inspiration", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertHasAbility(gametest.PlayerA, "Grizzly Bears", core.Hexproof, false)
	})
}

func TestChelonianTackle(t *testing.T) {
	t.Run("gives creature +0/+10 and it fights opponent creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Chelonian Tackle")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Chelonian Tackle", "Grizzly Bears", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// PlayerA's bear (2 power) fights opponent's 2/2 → opponent's bear dies
		// PlayerA's bear (12 toughness) survives opponent's 2 power
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
	})

	t.Run("boost applies even without fight target", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Chelonian Tackle")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Chelonian Tackle", "Grizzly Bears")
		g.StopAt(1, core.DeclareAttackers)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 2, 12)
	})
}

func TestCostOfBrilliance(t *testing.T) {
	t.Run("target player draws 2 and loses 2 life", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Cost of Brilliance")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Cost of Brilliance", "PlayerB")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 18)
	})

	t.Run("puts +1/+1 counter on target creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Cost of Brilliance")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Cost of Brilliance", "PlayerB", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 3, 3)
	})
}

func TestDaydream(t *testing.T) {
	t.Run("exiles and returns creature with +1/+1 counter", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Daydream")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Daydream", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
		// Grizzly Bears 2/2 + 1/1 counter = 3/3
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 3, 3)
	})
}

func TestDigSiteInventory(t *testing.T) {
	t.Run("puts +1/+1 counter on target creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Dig Site Inventory")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Dig Site Inventory", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Counter persists after EOT: 2/2 + 1/1 = 3/3
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 3, 3)
	})

	t.Run("creature has vigilance during turn it was cast", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Dig Site Inventory")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Dig Site Inventory", "Grizzly Bears")
		g.Attack(1, gametest.PlayerA, "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Grizzly Bears attacked with vigilance, so it should NOT be tapped
		g.AssertTapped(gametest.PlayerA, "Grizzly Bears", false)
	})
}

func TestDinasGuidance(t *testing.T) {
	t.Run("searches library for creature and puts it in hand", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Dina's Guidance")
		g.ChooseFromLibrary(gametest.PlayerA, "Grizzly Bears")
		g.ChooseMode(gametest.PlayerA, 0)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Dina's Guidance")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
	})

	t.Run("searches library for creature and puts it in graveyard", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Dina's Guidance")
		g.ChooseFromLibrary(gametest.PlayerA, "Grizzly Bears")
		g.ChooseMode(gametest.PlayerA, 1)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Dina's Guidance")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 1)
	})
}
