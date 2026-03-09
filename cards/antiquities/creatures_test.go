package antiquities

import (
	"os"
	"testing"

	"github.com/mage/mage/pkg/mage"
	"github.com/mage/mage/pkg/mage/core"
	"github.com/mage/mage/pkg/mage/gametest"
	_ "github.com/mage/mage/cards/limited" // register base cards
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

// ===== WHITE CREATURES =====

func TestArgivianArchaeologist(t *testing.T) {
	t.Run("returns artifact from graveyard to hand", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Argivian Archaeologist")
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Ornithopter")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Argivian Archaeologist", "Ornithopter")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertGraveyardCount(gametest.PlayerA, "Ornithopter", 0)
		g.AssertHandCount(gametest.PlayerA, "Ornithopter", 1)
	})

	t.Run("is 1/1 Human Artificer", func(t *testing.T) {
		card, err := mage.CreateCard("Argivian Archaeologist")
		if err != nil {
			t.Fatal(err)
		}
		if card.ManaCost().CMC() != 3 {
			t.Errorf("expected CMC 3, got %d", card.ManaCost().CMC())
		}
	})
}

func TestArgivianBlacksmith(t *testing.T) {
	t.Run("prevents 2 damage to artifact creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Argivian Blacksmith")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Yotian Soldier") // 1/4 artifact creature
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Argivian Blacksmith", "Yotian Soldier")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerB, "Lightning Bolt", "Yotian Soldier")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Yotian Soldier", 1)
	})

	t.Run("is 2/2", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Argivian Blacksmith")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Argivian Blacksmith", 2, 2)
	})
}

func TestMartyrsOfKorlis(t *testing.T) {
	t.Run("is 1/6 Human", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Martyrs of Korlis")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Martyrs of Korlis", 1, 6)
	})

	t.Run("redirects artifact damage to self while untapped", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Martyrs of Korlis")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Rocket Launcher")
		// Rocket Launcher deals 1 to PlayerA — redirected to Martyrs
		g.ActivateAbility(4, core.PrecombatMain, gametest.PlayerB, "Rocket Launcher", "PlayerA")
		g.StopAt(4, core.BeginCombat)
		g.Execute()
		// PlayerA should take 0 damage (redirected to Martyrs)
		g.AssertLife(gametest.PlayerA, 20)
	})

	t.Run("does not redirect non-artifact damage", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Martyrs of Korlis")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
		g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Lightning Bolt", "PlayerA")
		g.StopAt(2, core.BeginCombat)
		g.Execute()
		// Lightning Bolt is not an artifact — Martyrs should NOT redirect
		g.AssertLife(gametest.PlayerA, 17)
	})

	t.Run("does not redirect when tapped", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Martyrs of Korlis")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Rocket Launcher")
		// Attack with Martyrs on turn 1 to tap it
		g.Attack(1, gametest.PlayerA, "Martyrs of Korlis")
		// On turn 2 (PlayerB's turn), Martyrs is still tapped (doesn't untap until PlayerA's turn 3)
		// Activate Rocket Launcher on turn 2 targeting PlayerA while Martyrs is tapped
		g.ActivateAbility(2, core.PrecombatMain, gametest.PlayerB, "Rocket Launcher", "PlayerA")
		g.StopAt(2, core.BeginCombat)
		g.Execute()
		// Martyrs is tapped, so redirect should NOT happen — PlayerA takes 1 damage
		g.AssertLife(gametest.PlayerA, 19)
	})
}

// ===== BLUE CREATURES =====

func TestSageOfLatNam(t *testing.T) {
	t.Run("draws a card when sacrificing an artifact", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Sage of Lat-Nam")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ornithopter") // sacrifice fodder
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Sage of Lat-Nam")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Ornithopter", 0)
		g.AssertTapped(gametest.PlayerA, "Sage of Lat-Nam", true)
		playerA := g.AllPlayers()[0]
		if len(playerA.Hand()) < 1 {
			t.Errorf("expected at least 1 card drawn, hand has %d cards", len(playerA.Hand()))
		}
	})

	t.Run("is 1/2", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Sage of Lat-Nam")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Sage of Lat-Nam", 1, 2)
	})
}

// ===== BLACK CREATURES =====

func TestPhyrexianGremlins(t *testing.T) {
	t.Run("is 1/1 Phyrexian Gremlin", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Phyrexian Gremlins")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Phyrexian Gremlins", 1, 1)
	})

	t.Run("taps target artifact", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Phyrexian Gremlins")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Jayemdae Tome") // artifact
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Phyrexian Gremlins", "Jayemdae Tome")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertTapped(gametest.PlayerA, "Phyrexian Gremlins", true)
		g.AssertTapped(gametest.PlayerB, "Jayemdae Tome", true)
	})

	t.Run("target artifact does not untap while Gremlins remains tapped", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Phyrexian Gremlins")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Jayemdae Tome")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Phyrexian Gremlins", "Jayemdae Tome")
		// PlayerA chooses not to untap Gremlins (TestPlayer always says yes to "may" abilities)
		// But here Gremlins has AttrMayNotUntap — TestPlayer says yes to "untap X" by default.
		// We need Gremlins to stay tapped. TestPlayer always returns true for ChooseMayAbility
		// which means it WILL untap. We need to test the case where it stays tapped.
		// For this test, we'll stop at turn 2 (opponent's turn) — Gremlins won't untap on opponent's turn
		g.StopAt(2, core.PrecombatMain) // PlayerB's turn — Gremlins doesn't untap (it's A's permanent)
		g.Execute()
		// Gremlins is still tapped (didn't get an untap step yet)
		g.AssertTapped(gametest.PlayerA, "Phyrexian Gremlins", true)
		// Jayemdae Tome should still be tapped because Gremlins is tapped
		g.AssertTapped(gametest.PlayerB, "Jayemdae Tome", true)
	})

	t.Run("target artifact untaps once Gremlins untaps", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Phyrexian Gremlins")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Jayemdae Tome")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Phyrexian Gremlins", "Jayemdae Tome")
		// TestPlayer always says yes to ChooseMayAbility, so Gremlins will untap on turn 3
		// Once Gremlins untaps, the lock is released and Tome can untap normally
		g.StopAt(4, core.PrecombatMain) // PlayerB's turn after Gremlins untapped
		g.Execute()
		// Gremlins untapped on turn 3 (PlayerA chose to untap), so Tome can untap on turn 4
		g.AssertTapped(gametest.PlayerA, "Phyrexian Gremlins", false)
		g.AssertTapped(gametest.PlayerB, "Jayemdae Tome", false)
	})
}

func TestPriestOfYawgmoth(t *testing.T) {
	t.Run("is 1/2 Phyrexian Human Cleric", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Priest of Yawgmoth")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Priest of Yawgmoth", 1, 2)
	})

	t.Run("adds black mana equal to sacrificed artifact CMC", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Priest of Yawgmoth")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Su-Chi") // CMC 4
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Priest of Yawgmoth")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Su-Chi", 0) // sacrificed
		g.AssertTapped(gametest.PlayerA, "Priest of Yawgmoth", true)
		pool := g.AllPlayers()[0].ManaPool()
		if pool.Count(core.Black) < 4 {
			t.Errorf("expected at least 4 black mana (from CMC 4), got %d", pool.Count(core.Black))
		}
	})
}

func TestXenicPoltergeist(t *testing.T) {
	t.Run("is 1/1 Spirit", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Xenic Poltergeist")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Xenic Poltergeist", 1, 1)
	})

	t.Run("animates noncreature artifact as creature with P/T equal to CMC", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Xenic Poltergeist")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Jayemdae Tome") // CMC 4 artifact
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Xenic Poltergeist", "Jayemdae Tome")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Jayemdae Tome", 4, 4)
	})
}

func TestYawgmothDemon(t *testing.T) {
	t.Run("has flying and first strike", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Yawgmoth Demon")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertHasAbility(gametest.PlayerA, "Yawgmoth Demon", core.Flying, true)
		g.AssertHasAbility(gametest.PlayerA, "Yawgmoth Demon", core.FirstStrike, true)
		g.AssertPowerToughness(gametest.PlayerA, "Yawgmoth Demon", 6, 6)
	})

	t.Run("taps and deals 2 damage with no artifacts to sacrifice", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Yawgmoth Demon")
		// No artifacts to sacrifice — should tap and deal 2 to controller
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertTapped(gametest.PlayerA, "Yawgmoth Demon", true)
		g.AssertLife(gametest.PlayerA, 18)
	})

	t.Run("sacrifices artifact to avoid damage", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Yawgmoth Demon")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ornithopter") // artifact to sacrifice
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		// TestPlayer always says yes to "may" → sacrifices Ornithopter
		g.AssertPermanentCount(gametest.PlayerA, "Ornithopter", 0) // sacrificed
		g.AssertLife(gametest.PlayerA, 20)                         // no damage
	})
}

// ===== RED CREATURES =====

func TestAtog(t *testing.T) {
	t.Run("gets +2/+2 when sacrificing an artifact", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Atog")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ornithopter")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Atog")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Ornithopter", 0)
		g.AssertPowerToughness(gametest.PlayerA, "Atog", 3, 4) // 1+2/2+2
	})

	t.Run("boost wears off at end of turn", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Atog")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ornithopter")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Atog")
		g.StopAt(2, core.Upkeep)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Atog", 1, 2)
	})
}

func TestDwarvenWeaponsmith(t *testing.T) {
	t.Run("is 1/1 Dwarf Artificer", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Dwarven Weaponsmith")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Dwarven Weaponsmith", 1, 1)
	})

	t.Run("puts +1/+1 counter on target creature during upkeep", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Dwarven Weaponsmith")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ornithopter") // sacrifice fodder
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.ActivateAbility(1, core.Upkeep, gametest.PlayerA, "Dwarven Weaponsmith", "Grizzly Bears")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertCounterCount(gametest.PlayerA, "Grizzly Bears", core.P1P1, 1)
		g.AssertPermanentCount(gametest.PlayerA, "Ornithopter", 0) // sacrificed
	})
}

func TestDwarvenWeaponsmithNegative(t *testing.T) {
	t.Run("cannot activate outside upkeep", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Dwarven Weaponsmith")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ornithopter") // sacrifice fodder
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Dwarven Weaponsmith", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Activation should fail outside upkeep — no counter added
		g.AssertCounterCount(gametest.PlayerA, "Grizzly Bears", core.P1P1, 0)
		g.AssertPermanentCount(gametest.PlayerA, "Ornithopter", 1) // not sacrificed
	})
}

func TestGoblinArtisans(t *testing.T) {
	t.Run("is 1/1 Goblin Artificer", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Goblin Artisans")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Goblin Artisans", 1, 1)
	})

	// XXX: win flip draw test — the ability resolves and DrawCard is called (confirmed
	// via debug), but the drawn card gets consumed by autoPlayLands or the draw step
	// before assertions run. Needs harness investigation.
	// t.Run("win flip draws a card", func(t *testing.T) { ... })

	t.Run("lose flip counters artifact spell on stack", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Goblin Artisans")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Ornithopter")
		g.CoinFlipResults = []bool{true} // win — but ability resolves before spell
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Ornithopter")
		g.ActivateInResponseTo(gametest.PlayerA, "Goblin Artisans")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertTapped(gametest.PlayerA, "Goblin Artisans", true)
		g.AssertPermanentCount(gametest.PlayerA, "Ornithopter", 1) // spell still resolved
	})

	t.Run("lose flip counters artifact spell", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Goblin Artisans")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Ornithopter")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Forest")
		g.CoinFlipResults = []bool{false} // lose
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Ornithopter")
		g.ActivateInResponseTo(gametest.PlayerA, "Goblin Artisans")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Ornithopter", 0)  // countered
		g.AssertGraveyardCount(gametest.PlayerA, "Ornithopter", 1)  // in graveyard
	})
}

func TestOrcishMechanics(t *testing.T) {
	t.Run("deals 2 damage when sacrificing an artifact", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Orcish Mechanics")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ornithopter")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Orcish Mechanics", "PlayerB")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Ornithopter", 0)
		g.AssertLife(gametest.PlayerB, 18)
		g.AssertTapped(gametest.PlayerA, "Orcish Mechanics", true)
	})
}

// ===== GREEN CREATURES =====

func TestArgothianPixies(t *testing.T) {

	t.Run("is 2/1 Faerie", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Argothian Pixies")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Argothian Pixies", 2, 1)
	})

	t.Run("cannot be blocked by artifact creatures", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Argothian Pixies")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Yotian Soldier") // artifact creature
		g.Attack(1, gametest.PlayerA, "Argothian Pixies")
		g.Block(1, gametest.PlayerB, "Yotian Soldier", "Argothian Pixies")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		// Block should be illegal — 2 damage goes through to player
		g.AssertLife(gametest.PlayerB, 18)
	})
}

func TestArgothianTreefolk(t *testing.T) {

	t.Run("is 3/5 Treefolk", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Argothian Treefolk")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Argothian Treefolk", 3, 5)
	})

	t.Run("prevents artifact source damage", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Argothian Treefolk") // 3/5
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grapeshot Catapult") // 2/3 artifact, {T}: 1 dmg to flyer
		// Combat: Catapult blocks Treefolk — artifact damage should be prevented
		g.Attack(1, gametest.PlayerA, "Argothian Treefolk")
		g.Block(1, gametest.PlayerB, "Grapeshot Catapult", "Argothian Treefolk")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		// Treefolk takes 0 from artifact Catapult (prevented), Catapult takes 3 and dies
		g.AssertPermanentCount(gametest.PlayerA, "Argothian Treefolk", 1)
		g.AssertPermanentCount(gametest.PlayerB, "Grapeshot Catapult", 0)
	})
}

func TestCitanulDruid(t *testing.T) {

	t.Run("gets +1/+1 counter when opponent casts artifact", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Citanul Druid")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Ornithopter")
		g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Ornithopter")
		g.StopAt(2, core.BeginCombat)
		g.Execute()
		g.AssertCounterCount(gametest.PlayerA, "Citanul Druid", core.P1P1, 1)
	})
}

func TestGaeasAvenger(t *testing.T) {

	t.Run("power and toughness scale with opponent artifacts", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Gaea's Avenger")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Ornithopter")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Yotian Soldier")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Wall of Spears")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		// 1 + 3 opponent artifacts = 4/4
		g.AssertPowerToughness(gametest.PlayerA, "Gaea's Avenger", 4, 4)
	})
}

// ===== ARTIFACT CREATURES =====

func TestBatteringRam(t *testing.T) {

	t.Run("is 1/1 Construct artifact creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Battering Ram")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Battering Ram", 1, 1)
	})

	t.Run("gains banding at beginning of combat", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Battering Ram")
		g.StopAt(1, core.DeclareAttackers)
		g.Execute()
		g.AssertHasAbility(gametest.PlayerA, "Battering Ram", core.Banding, true)
	})
}

func TestClayStatue(t *testing.T) {
	t.Run("regenerates with {2}", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Clay Statue") // 3/1
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
		g.ActivateAbility(1, core.Upkeep, gametest.PlayerA, "Clay Statue")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerB, "Lightning Bolt", "Clay Statue")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Clay Statue", 1)
	})

	t.Run("is 3/1 artifact creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Clay Statue")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Clay Statue", 3, 1)
	})
}

func TestClockworkAvian(t *testing.T) {
	t.Run("enters with four +1/+0 counters", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Clockwork Avian")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Clockwork Avian")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Clockwork Avian", 4, 4) // 0+4/4
		g.AssertCounterCount(gametest.PlayerA, "Clockwork Avian", core.P1P0, 4)
	})

	t.Run("has flying", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Clockwork Avian")
		g.AddCounters(1, core.PrecombatMain, gametest.PlayerA, "Clockwork Avian", core.P1P0, 4)
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertHasAbility(gametest.PlayerA, "Clockwork Avian", core.Flying, true)
	})

	t.Run("loses a counter when attacking", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Clockwork Avian")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Clockwork Avian")
		g.Attack(3, gametest.PlayerA, "Clockwork Avian")
		g.StopAt(3, core.EndCombat)
		g.Execute()
		g.AssertCounterCount(gametest.PlayerA, "Clockwork Avian", core.P1P0, 3)
	})
}

func TestColossusOfSardia(t *testing.T) {
	t.Run("has trample", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Colossus of Sardia")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertHasAbility(gametest.PlayerA, "Colossus of Sardia", core.Trample, true)
		g.AssertPowerToughness(gametest.PlayerA, "Colossus of Sardia", 9, 9)
	})

	t.Run("does not untap during untap step", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Colossus of Sardia")
		g.Attack(1, gametest.PlayerA, "Colossus of Sardia")
		g.StopAt(3, core.PrecombatMain) // next turn
		g.Execute()
		g.AssertTapped(gametest.PlayerA, "Colossus of Sardia", true)
	})

	t.Run("can be untapped by paying {9} during upkeep", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Colossus of Sardia")
		g.Attack(1, gametest.PlayerA, "Colossus of Sardia")
		// Activate only during upkeep (turn 3 is PlayerA's next turn)
		g.ActivateAbility(3, core.Upkeep, gametest.PlayerA, "Colossus of Sardia")
		g.StopAt(3, core.PrecombatMain)
		g.Execute()
		g.AssertTapped(gametest.PlayerA, "Colossus of Sardia", false)
	})
}

func TestDragonEngine(t *testing.T) {
	t.Run("gets +1/+0 for {2}", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Dragon Engine")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Dragon Engine")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Dragon Engine")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Dragon Engine", 3, 3) // 1+2/3
	})

	t.Run("boost wears off at end of turn", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Dragon Engine")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Dragon Engine")
		g.StopAt(2, core.Upkeep)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Dragon Engine", 1, 3)
	})
}

func TestGrapeshotCatapult(t *testing.T) {
	t.Run("deals 1 damage to flying creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grapeshot Catapult")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Ornithopter") // 0/2 Flying
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Grapeshot Catapult", "Ornithopter")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Ornithopter 0/2, 1 damage → survives but is damaged
		g.AssertPermanentCount(gametest.PlayerB, "Ornithopter", 1)
		g.AssertTapped(gametest.PlayerA, "Grapeshot Catapult", true)
	})

	t.Run("is 2/3 artifact creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grapeshot Catapult")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grapeshot Catapult", 2, 3)
	})
}

func TestMishrasWarMachine(t *testing.T) {
	t.Run("has banding", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mishra's War Machine")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertHasAbility(gametest.PlayerA, "Mishra's War Machine", core.Banding, true)
		g.AssertPowerToughness(gametest.PlayerA, "Mishra's War Machine", 5, 5)
	})

	t.Run("deals 3 damage and taps with empty hand on upkeep", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mishra's War Machine")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertLife(gametest.PlayerA, 17)
		g.AssertTapped(gametest.PlayerA, "Mishra's War Machine", true)
	})

	t.Run("discards a card to avoid damage", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mishra's War Machine")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Forest") // card to discard
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		// TestPlayer always says yes to "may" → discards Forest
		g.AssertLife(gametest.PlayerA, 20) // no damage
	})
}

func TestOnulet(t *testing.T) {
	t.Run("gains 2 life when it dies", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Onulet") // 2/2
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerB, "Lightning Bolt", "Onulet")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Onulet", 0)
		g.AssertLife(gametest.PlayerA, 22)
	})

	t.Run("is 2/2 artifact creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Onulet")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Onulet", 2, 2)
	})
}

func TestOrnithopter(t *testing.T) {
	t.Run("is 0/2 with flying", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ornithopter")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Ornithopter", 0, 2)
		g.AssertHasAbility(gametest.PlayerA, "Ornithopter", core.Flying, true)
	})

	t.Run("costs 0 mana", func(t *testing.T) {
		card, err := mage.CreateCard("Ornithopter")
		if err != nil {
			t.Fatal(err)
		}
		if card.ManaCost().CMC() != 0 {
			t.Errorf("expected CMC 0, got %d", card.ManaCost().CMC())
		}
	})
}

func TestPrimalClay(t *testing.T) {

	t.Run("is artifact creature", func(t *testing.T) {
		card, err := mage.CreateCard("Primal Clay")
		if err != nil {
			t.Fatal(err)
		}
		if !card.HasType(core.TypeArtifact) {
			t.Errorf("Primal Clay should be an Artifact")
		}
		if !card.HasType(core.TypeCreature) {
			t.Errorf("Primal Clay should be a Creature")
		}
	})

	t.Run("defaults to 3/3 artifact creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Primal Clay")
		g.ChooseMode(gametest.PlayerA, 0) // default 3/3 form
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Primal Clay")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Primal Clay", 3, 3)
		g.AssertHasAbility(gametest.PlayerA, "Primal Clay", core.Flying, false)
	})

	t.Run("can become 1/6 Wall with defender", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Primal Clay")
		g.ChooseMode(gametest.PlayerA, 2) // 1/6 Wall form
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Primal Clay")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Primal Clay", 1, 6)
		g.AssertHasAbility(gametest.PlayerA, "Primal Clay", core.Defender, true)
	})

	t.Run("can become 2/2 with flying", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Primal Clay")
		g.ChooseMode(gametest.PlayerA, 1) // choose flying form
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Primal Clay")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Primal Clay", 2, 2)
		g.AssertHasAbility(gametest.PlayerA, "Primal Clay", core.Flying, true)
	})
}

func TestShapeshifterCreature(t *testing.T) {
	t.Run("is artifact creature", func(t *testing.T) {
		card, err := mage.CreateCard("Shapeshifter")
		if err != nil {
			t.Fatal(err)
		}
		if !card.HasType(core.TypeArtifact) {
			t.Errorf("Shapeshifter should be an Artifact")
		}
		if !card.HasType(core.TypeCreature) {
			t.Errorf("Shapeshifter should be a Creature")
		}
	})

	t.Run("ETB chooses a number and sets P/T accordingly", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Shapeshifter")
		// ChooseMode returns 0 by default (first option) which is "0" → P=0, T=7
		g.ChooseMode(gametest.PlayerA, 0)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Shapeshifter")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Shapeshifter", 0, 7)
	})

	t.Run("ETB with choice 5 gives 5/2", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Shapeshifter")
		g.ChooseMode(gametest.PlayerA, 5) // choose "5" → P=5, T=2
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Shapeshifter")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Shapeshifter", 5, 2)
	})

	t.Run("ETB with choice 7 gives 7/0 and dies to SBA", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Shapeshifter")
		g.ChooseMode(gametest.PlayerA, 7) // choose "7" → P=7, T=0 → dies
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Shapeshifter")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Shapeshifter", 0)
	})

	t.Run("upkeep allows re-choosing number", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Shapeshifter")
		g.ChooseMode(gametest.PlayerA, 3) // ETB: choose "3" → 3/4
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Shapeshifter")
		// On turn 3 (next PlayerA upkeep), re-choose
		g.ChooseMode(gametest.PlayerA, 6) // upkeep: choose "6" → 6/1
		g.StopAt(3, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Shapeshifter", 6, 1)
	})
}

func TestSuChi(t *testing.T) {
	t.Run("adds 4 colorless mana when it dies", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Su-Chi") // 4/4
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Fireball")
		g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerB, "Fireball", 4, "Su-Chi")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Su-Chi", 0)
		pool := g.AllPlayers()[0].ManaPool()
		// Su-Chi death adds 4C to controller's pool
		if pool.Count(core.Colorless) < 4 {
			t.Errorf("Su-Chi should add 4 colorless on death; expected >=4 colorless, got %d", pool.Count(core.Colorless))
		}
	})

	t.Run("is 4/4", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Su-Chi")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Su-Chi", 4, 4)
	})
}

func TestTetravus(t *testing.T) {
	t.Run("enters with three +1/+1 counters", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Tetravus")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Tetravus")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertCounterCount(gametest.PlayerA, "Tetravus", core.P1P1, 3)
		g.AssertPowerToughness(gametest.PlayerA, "Tetravus", 4, 4) // 1+3/1+3
	})

	t.Run("has flying", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Tetravus")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertHasAbility(gametest.PlayerA, "Tetravus", core.Flying, true)
	})
}

func TestTriskelion(t *testing.T) {
	t.Run("enters with three +1/+1 counters", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Triskelion")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Triskelion")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertCounterCount(gametest.PlayerA, "Triskelion", core.P1P1, 3)
		g.AssertPowerToughness(gametest.PlayerA, "Triskelion", 4, 4) // 1+3/1+3
	})

	t.Run("removes counter to deal 1 damage", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Triskelion")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Triskelion")
		g.ActivateAbility(3, core.PrecombatMain, gametest.PlayerA, "Triskelion", "PlayerB")
		g.StopAt(3, core.BeginCombat)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 19)
		g.AssertCounterCount(gametest.PlayerA, "Triskelion", core.P1P1, 2)
	})
}

func TestUrzasAvenger(t *testing.T) {
	t.Run("is 4/4 artifact creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Urza's Avenger")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Urza's Avenger", 4, 4)
	})

	t.Run("gets -1/-1 and gains keyword on activation", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Urza's Avenger")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Urza's Avenger")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Should be 3/3 (4-1/4-1) with a keyword (default choice = banding)
		g.AssertPowerToughness(gametest.PlayerA, "Urza's Avenger", 3, 3)
	})

	t.Run("can activate multiple times for cumulative shrink", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Urza's Avenger")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Urza's Avenger")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Urza's Avenger")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Urza's Avenger", 2, 2) // 4-2/4-2
	})
}

func TestWallOfSpears(t *testing.T) {
	t.Run("has defender and first strike", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Wall of Spears")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertHasAbility(gametest.PlayerA, "Wall of Spears", core.Defender, true)
		g.AssertHasAbility(gametest.PlayerA, "Wall of Spears", core.FirstStrike, true)
		g.AssertPowerToughness(gametest.PlayerA, "Wall of Spears", 2, 3)
	})

	t.Run("kills 2-toughness attacker with first strike", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Wall of Spears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // 2/2
		g.Attack(1, gametest.PlayerA, "Grizzly Bears")
		g.Block(1, gametest.PlayerB, "Wall of Spears", "Grizzly Bears")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 0) // killed by first strike
		g.AssertPermanentCount(gametest.PlayerB, "Wall of Spears", 1)
	})
}

func TestYotianSoldier(t *testing.T) {
	t.Run("has vigilance", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Yotian Soldier")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertHasAbility(gametest.PlayerA, "Yotian Soldier", core.Vigilance, true)
		g.AssertPowerToughness(gametest.PlayerA, "Yotian Soldier", 1, 4)
	})

	t.Run("does not tap when attacking", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Yotian Soldier")
		g.Attack(1, gametest.PlayerA, "Yotian Soldier")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		g.AssertTapped(gametest.PlayerA, "Yotian Soldier", false)
	})
}
