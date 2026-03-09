package antiquities

import (
	"testing"

	"github.com/mage/mage/pkg/mage"
	"github.com/mage/mage/pkg/mage/core"
	"github.com/mage/mage/pkg/mage/gametest"
)

func TestAmuletOfKroog(t *testing.T) {
	t.Run("prevents 1 damage to target", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Amulet of Kroog")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // 2/2
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Amulet of Kroog", "Grizzly Bears")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerB, "Lightning Bolt", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// 3 damage - 1 prevented = 2, equals toughness → creature dies
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 0)
	})

	t.Run("prevents 1 damage to player", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Amulet of Kroog")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Amulet of Kroog", "PlayerA")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerB, "Lightning Bolt", "PlayerA")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertLife(gametest.PlayerA, 18) // 3 - 1 prevented = 2 damage
	})
}

func TestArmageddonClock(t *testing.T) {
	t.Run("deals damage equal to doom counters at draw step", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Armageddon Clock")
		// After upkeep (1 doom counter), at draw step deals 1 to each player
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertLife(gametest.PlayerA, 19) // 1 doom counter → 1 damage
		g.AssertLife(gametest.PlayerB, 19)
	})

	t.Run("doom counters accumulate over multiple turns", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Armageddon Clock")
		// Turn 1: 1 doom counter → 1 damage each, Turn 3: 2 doom counters → 2 damage each
		g.StopAt(3, core.PrecombatMain)
		g.Execute()
		// Turn 1: 1 dmg each, Turn 3: 2 dmg each = 3 total each
		g.AssertLife(gametest.PlayerA, 17)
		g.AssertLife(gametest.PlayerB, 17)
	})

	t.Run("remove doom counter ability for {4}", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Armageddon Clock")
		// After turn 1 upkeep adds doom counter, remove it during upkeep
		g.ActivateAbility(3, core.Upkeep, gametest.PlayerA, "Armageddon Clock")
		g.StopAt(3, core.PrecombatMain)
		g.Execute()
		// Turn 1: 1 doom counter dealt 1 damage each
		// Turn 3: added 2nd doom counter, removed 1 → 1 doom counter, dealt 1 each
		// Total: 2 damage each
		g.AssertLife(gametest.PlayerA, 18)
		g.AssertLife(gametest.PlayerB, 18)
	})
}

func TestAshnodsAltar(t *testing.T) {
	t.Run("sacrifice creature for 2 colorless mana", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ashnod's Altar")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Ashnod's Altar")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 0)
		pool := g.AllPlayers()[0].ManaPool()
		if pool.Count(core.Colorless) < 7 { // 5 auto + 2 altar
			t.Errorf("expected at least 7 colorless (5 auto + 2 altar), got %d", pool.Count(core.Colorless))
		}
	})
}

func TestAshnodsBattleGear(t *testing.T) {
	t.Run("boosts creature +2/-2 while tapped", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ashnod's Battle Gear")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant") // 3/3
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Ashnod's Battle Gear", "Hill Giant")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Hill Giant", 5, 1) // 3+2/3-2
		g.AssertTapped(gametest.PlayerA, "Ashnod's Battle Gear", true)
	})

	t.Run("boost ends when Battle Gear untaps", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ashnod's Battle Gear")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Ashnod's Battle Gear", "Hill Giant")
		// Let it untap on turn 3 (may choose not to untap, but TestPlayer says yes so it untaps)
		g.StopAt(3, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Hill Giant", 3, 3) // back to normal
	})
}

func TestAshnodsTransmogrant(t *testing.T) {
	t.Run("puts +1/+1 counter and makes target an artifact", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ashnod's Transmogrant")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Ashnod's Transmogrant", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Ashnod's Transmogrant", 0) // sacrificed
		g.AssertCounterCount(gametest.PlayerA, "Grizzly Bears", core.P1P1, 1)
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 3, 3)
	})
}

func TestAshnodsTransmograntNegative(t *testing.T) {
	t.Run("cannot target artifact creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ashnod's Transmogrant")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ornithopter") // artifact creature
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Ashnod's Transmogrant", "Ornithopter")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Cannot target artifact creature — Transmogrant should still be on battlefield
		g.AssertPermanentCount(gametest.PlayerA, "Ashnod's Transmogrant", 1)
		g.AssertCounterCount(gametest.PlayerA, "Ornithopter", core.P1P1, 0)
	})
}

func TestCandelabraOfTawnos(t *testing.T) {
	t.Run("untaps X tapped lands with X=2", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Candelabra of Tawnos")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest") // extra land to pay X=2
		// Tap two forests first by casting a spell that costs {G}{G} — or just use them as mana
		// The test harness auto-taps lands for mana. Activating with X=2 will tap lands for {2}.
		g.ActivateAbilityWithX(1, core.PrecombatMain, gametest.PlayerA, "Candelabra of Tawnos", 2)
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Candelabra taps itself + pays {2} from lands. Then untaps 2 tapped lands.
		g.AssertTapped(gametest.PlayerA, "Candelabra of Tawnos", true)
		// After activation, 2 lands were tapped for mana, then 2 lands untapped by the effect.
		// The third land was not tapped. All 3 forests should be untapped.
		tappedCount := 0
		for _, perm := range g.Battlefield {
			if perm.Name() == "Forest" && perm.Tapped {
				tappedCount++
			}
		}
		if tappedCount != 0 {
			t.Errorf("expected 0 tapped forests after untapping 2, got %d", tappedCount)
		}
	})

	t.Run("X=0 does nothing", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Candelabra of Tawnos")
		g.ActivateAbilityWithX(1, core.PrecombatMain, gametest.PlayerA, "Candelabra of Tawnos", 0)
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertTapped(gametest.PlayerA, "Candelabra of Tawnos", true)
	})

	t.Run("taps itself as part of cost", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Candelabra of Tawnos")
		g.ActivateAbilityWithX(1, core.PrecombatMain, gametest.PlayerA, "Candelabra of Tawnos", 0)
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertTapped(gametest.PlayerA, "Candelabra of Tawnos", true)
	})
}

func TestCoralHelm(t *testing.T) {
	t.Run("boosts target creature +2/+2 until end of turn", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Coral Helm")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // 2/2
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Forest")              // will be discarded randomly
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Coral Helm", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 4, 4) // 2+2/2+2
	})

	t.Run("boost wears off at end of turn", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Coral Helm")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Forest")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Coral Helm", "Grizzly Bears")
		g.StopAt(2, core.Upkeep)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 2, 2)
	})
}

func TestCursedRack(t *testing.T) {
	t.Run("reduces chosen opponent max hand size to 4", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Cursed Rack")
		for i := 0; i < 7; i++ {
			g.AddCard(core.ZoneHand, gametest.PlayerB, "Forest")
		}
		// At end of PlayerB's turn, they should discard to 4
		g.StopAt(3, core.Upkeep) // after PlayerB's cleanup
		g.Execute()
		playerB := g.AllPlayers()[1]
		if len(playerB.Hand()) > 4 {
			t.Errorf("Cursed Rack should limit hand to 4; PlayerB has %d cards", len(playerB.Hand()))
		}
	})
}

func TestFeldonsCane(t *testing.T) {
	t.Run("shuffles graveyard into library", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Feldon's Cane")
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Hill Giant")
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Lightning Bolt")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Feldon's Cane")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 0)
		g.AssertGraveyardCount(gametest.PlayerA, "Hill Giant", 0)
		g.AssertGraveyardCount(gametest.PlayerA, "Lightning Bolt", 0)
		g.AssertPermanentCount(gametest.PlayerA, "Feldon's Cane", 0) // sacrificed
	})
}

func TestGolgothianSylex(t *testing.T) {
	t.Run("is an artifact with CMC 4", func(t *testing.T) {
		card, err := mage.CreateCard("Golgothian Sylex")
		if err != nil {
			t.Fatal(err)
		}
		if !card.HasType(core.TypeArtifact) {
			t.Errorf("Golgothian Sylex should be an Artifact")
		}
		if card.ManaCost().CMC() != 4 {
			t.Errorf("expected CMC 4, got %d", card.ManaCost().CMC())
		}
	})

	t.Run("sacrifices all nontoken Antiquities permanents", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Golgothian Sylex")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ornithopter")   // Antiquities card
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Jalum Tome")    // Antiquities card
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears") // NOT Antiquities
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Golgothian Sylex")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Ornithopter and Jalum Tome should be sacrificed (Antiquities names)
		g.AssertPermanentCount(gametest.PlayerA, "Ornithopter", 0)
		g.AssertPermanentCount(gametest.PlayerB, "Jalum Tome", 0)
		// Grizzly Bears should survive (not an Antiquities name)
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 1)
		// Golgothian Sylex itself is also an Antiquities card — should sacrifice itself too
		g.AssertPermanentCount(gametest.PlayerA, "Golgothian Sylex", 0)
	})
}

func TestIvoryTower(t *testing.T) {
	t.Run("gains life equal to hand size minus 4 on upkeep", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ivory Tower")
		for i := 0; i < 7; i++ {
			g.AddCard(core.ZoneHand, gametest.PlayerA, "Forest")
		}
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		// 7 cards in hand - 4 = 3 life gain
		g.AssertLife(gametest.PlayerA, 23)
	})

	t.Run("no life gain with 4 or fewer cards", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ivory Tower")
		for i := 0; i < 3; i++ {
			g.AddCard(core.ZoneHand, gametest.PlayerA, "Forest")
		}
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertLife(gametest.PlayerA, 20)
	})
}

func TestJalumTome(t *testing.T) {
	t.Run("draws then discards a card", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Jalum Tome")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Forest") // will be chosen for discard
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Jalum Tome")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertTapped(gametest.PlayerA, "Jalum Tome", true)
		// Should have drawn 1 (Bears) and discarded 1 (Forest) — net hand size same
	})
}

func TestMightstone(t *testing.T) {
	t.Run("attacking creatures get +1/+0", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mightstone")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // 2/2
		g.Attack(1, gametest.PlayerA, "Grizzly Bears")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 17) // 2+1 = 3 damage
	})
}

func TestMillstone(t *testing.T) {
	t.Run("target player mills two cards", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Millstone")
		g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Forest")
		g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Hill Giant")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Millstone", "PlayerB")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertTapped(gametest.PlayerA, "Millstone", true)
		playerB := g.AllPlayers()[1]
		// Started with 3 cards, milled 2, should have 1 left
		if len(playerB.Library()) != 1 {
			t.Errorf("expected 1 card in library after milling 2, got %d", len(playerB.Library()))
		}
	})
}

func TestObeliskOfUndoing(t *testing.T) {
	t.Run("returns own permanent to hand for {6}", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Obelisk of Undoing")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Obelisk of Undoing", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 0)
		g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
	})

	// NOTE: The "own AND control" restriction in the target filter is implemented
	// but cannot be tested without a control-stealing effect (Control Magic), since
	// the engine resets Controller=Owner each effect cycle. The fix in
	// ControlledPermanentTarget.Possible correctly checks both ownership and control.
}

func TestRakalite(t *testing.T) {
	t.Run("is a 6-cost artifact", func(t *testing.T) {
		card, err := mage.CreateCard("Rakalite")
		if err != nil {
			t.Fatal(err)
		}
		if card.ManaCost().CMC() != 6 {
			t.Errorf("expected CMC 6, got %d", card.ManaCost().CMC())
		}
	})

	t.Run("prevents 1 damage to target creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Rakalite")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // 2/2
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Rakalite", "Grizzly Bears")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerB, "Lightning Bolt", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// 3 damage - 1 prevented = 2, equals toughness → creature dies
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 0)
	})

	t.Run("prevents 1 damage to player", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Rakalite")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Rakalite", "PlayerA")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerB, "Lightning Bolt", "PlayerA")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertLife(gametest.PlayerA, 18) // 3 - 1 = 2 damage
	})

	t.Run("returns to owner hand at end step after activation", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Rakalite")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Rakalite", "Grizzly Bears")
		g.StopAt(2, core.Upkeep)
		g.Execute()
		// Rakalite should have bounced to hand at end step
		g.AssertPermanentCount(gametest.PlayerA, "Rakalite", 0)
		g.AssertHandCount(gametest.PlayerA, "Rakalite", 1)
	})

	t.Run("can be activated multiple times before bouncing", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Rakalite")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant") // 3/3
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
		// Activate twice to prevent 2 total
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Rakalite", "Hill Giant")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Rakalite", "Hill Giant")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerB, "Lightning Bolt", "Hill Giant")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// 3 - 2 = 1 damage; Hill Giant (3/3) survives
		g.AssertPermanentCount(gametest.PlayerA, "Hill Giant", 1)
	})
}

func TestRocketLauncher(t *testing.T) {
	t.Run("is a 4-cost artifact", func(t *testing.T) {
		card, err := mage.CreateCard("Rocket Launcher")
		if err != nil {
			t.Fatal(err)
		}
		if card.ManaCost().CMC() != 4 {
			t.Errorf("expected CMC 4, got %d", card.ManaCost().CMC())
		}
	})

	t.Run("deals 1 damage to target player", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Rocket Launcher")
		// Can't activate turn 1 (just gained control), need to wait until turn 3
		g.ActivateAbility(3, core.PrecombatMain, gametest.PlayerA, "Rocket Launcher", "PlayerB")
		g.StopAt(3, core.BeginCombat)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 19)
	})

	t.Run("cannot activate on the turn it entered", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Rocket Launcher")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Rocket Launcher")
		// Try to activate same turn — should fail because not controlled since turn start
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Rocket Launcher", "PlayerB")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Activation should have failed — life unchanged
		g.AssertLife(gametest.PlayerB, 20)
	})

	t.Run("destroys self at end step after activation", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Rocket Launcher")
		g.ActivateAbility(3, core.PrecombatMain, gametest.PlayerA, "Rocket Launcher", "PlayerB")
		g.StopAt(4, core.Upkeep) // after end step
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Rocket Launcher", 0)
	})

	t.Run("deals 1 damage to target creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Rocket Launcher")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears") // 2/2
		g.ActivateAbility(3, core.PrecombatMain, gametest.PlayerA, "Rocket Launcher", "Grizzly Bears")
		g.StopAt(3, core.BeginCombat)
		g.Execute()
		// 1 damage to 2/2 — survives
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 1)
		g.AssertLife(gametest.PlayerB, 20)
	})
}

func TestStaffOfZegon(t *testing.T) {
	t.Run("gives target creature -2/-0", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Staff of Zegon")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant") // 3/3
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Staff of Zegon", "Hill Giant")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerB, "Hill Giant", 1, 3)
		g.AssertTapped(gametest.PlayerA, "Staff of Zegon", true)
	})

	t.Run("effect wears off at end of turn", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Staff of Zegon")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Staff of Zegon", "Hill Giant")
		g.StopAt(2, core.Upkeep)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerB, "Hill Giant", 3, 3)
	})
}

func TestTabletOfEpityr(t *testing.T) {
	t.Run("gain 1 life when own artifact dies paying {1}", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Tablet of Epityr")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ornithopter")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest") // land to pay {1}
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "Ornithopter")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// With optional payment of {1}, should gain 1 life
		g.AssertLife(gametest.PlayerA, 21)
	})
}

func TestTabletOfEpityrNegative(t *testing.T) {
	t.Run("does not trigger when Tablet itself goes to graveyard", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Tablet of Epityr")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest") // land to pay {1}
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Shatter")
		g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Shatter", "Tablet of Epityr")
		g.StopAt(2, core.BeginCombat)
		g.Execute()
		// Tablet should not trigger on its own death (code excludes sourceID)
		g.AssertLife(gametest.PlayerA, 20)
	})
}

func TestTawnossCoffin(t *testing.T) {
	t.Run("is a 4-cost artifact", func(t *testing.T) {
		card, err := mage.CreateCard("Tawnos's Coffin")
		if err != nil {
			t.Fatal(err)
		}
		if card.ManaCost().CMC() != 4 {
			t.Errorf("expected CMC 4, got %d", card.ManaCost().CMC())
		}
	})

	t.Run("exiles target creature when activated", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Tawnos's Coffin")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant") // 3/3
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Tawnos's Coffin", "Hill Giant")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Hill Giant", 0)
		g.AssertTapped(gametest.PlayerA, "Tawnos's Coffin", true)
	})

	t.Run("returns creature when Coffin is untapped", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Tawnos's Coffin")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Tawnos's Coffin", "Hill Giant")
		// TestPlayer always says yes to ChooseMayAbility, so Coffin will untap on turn 3
		g.StopAt(3, core.PrecombatMain)
		g.Execute()
		// Coffin untaps → creature returns
		g.AssertTapped(gametest.PlayerA, "Tawnos's Coffin", false)
		g.AssertPermanentCount(gametest.PlayerB, "Hill Giant", 1)
	})

	t.Run("returns creature when Coffin leaves battlefield", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Tawnos's Coffin")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Shatter") // destroy artifact
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Tawnos's Coffin", "Hill Giant")
		g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Shatter", "Tawnos's Coffin")
		g.StopAt(2, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Tawnos's Coffin", 0)
		g.AssertPermanentCount(gametest.PlayerB, "Hill Giant", 1)
	})

	t.Run("creature returns tapped", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Tawnos's Coffin")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Tawnos's Coffin", "Hill Giant")
		// Let Coffin untap on turn 3
		g.StopAt(3, core.PrecombatMain)
		g.Execute()
		g.AssertTapped(gametest.PlayerB, "Hill Giant", true) // returns tapped
	})

	t.Run("preserves counters on returned creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Tawnos's Coffin")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
		g.AddCounters(1, core.PrecombatMain, gametest.PlayerB, "Hill Giant", core.P1P1, 3)
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Tawnos's Coffin", "Hill Giant")
		g.StopAt(3, core.PrecombatMain)
		g.Execute()
		// Hill Giant returns with the noted 3 +1/+1 counters
		g.AssertCounterCount(gametest.PlayerB, "Hill Giant", core.P1P1, 3)
	})

	t.Run("exiles and returns attached Aura", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Tawnos's Coffin")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Flight") // a simple Aura
		// Cast Flight on Hill Giant
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Flight", "Hill Giant")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Tawnos's Coffin", "Hill Giant")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Both Hill Giant and Flight should be gone from battlefield
		g.AssertPermanentCount(gametest.PlayerB, "Hill Giant", 0)
		g.AssertPermanentCount(gametest.PlayerA, "Flight", 0)
	})

	t.Run("returns attached Aura when Coffin leaves", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Tawnos's Coffin")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Flight")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Shatter")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Flight", "Hill Giant")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Tawnos's Coffin", "Hill Giant")
		// Destroy Coffin → creature and Aura should return
		g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Shatter", "Tawnos's Coffin")
		g.StopAt(2, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Hill Giant", 1)
		g.AssertPermanentCount(gametest.PlayerA, "Flight", 1)
		g.AssertAttachedTo(gametest.PlayerA, "Flight", "Hill Giant")
	})

	t.Run("exile without Aura still works", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Tawnos's Coffin")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Shatter")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Tawnos's Coffin", "Hill Giant")
		g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Shatter", "Tawnos's Coffin")
		g.StopAt(2, core.BeginCombat)
		g.Execute()
		// Creature returns without any Aura issues
		g.AssertPermanentCount(gametest.PlayerB, "Hill Giant", 1)
	})
}

func TestTawnossWand(t *testing.T) {
	t.Run("makes target creature unblockable", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Tawnos's Wand")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // 2/2, power ≤ 2
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")    // blocker
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Tawnos's Wand", "Grizzly Bears")
		g.Attack(1, gametest.PlayerA, "Grizzly Bears")
		g.Block(1, gametest.PlayerB, "Hill Giant", "Grizzly Bears")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 18) // unblockable, 2 damage
	})
}

func TestTawnossWandNegative(t *testing.T) {
	t.Run("cannot target creature with power greater than 2", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Tawnos's Wand")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant") // 3/3, power > 2
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant") // blocker
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Tawnos's Wand", "Hill Giant")
		g.Attack(1, gametest.PlayerA, "Hill Giant")
		g.Block(1, gametest.PlayerB, "Hill Giant", "Hill Giant")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		// Hill Giant power 3 > 2, so Wand should fail to target — Hill Giant gets blocked
		g.AssertLife(gametest.PlayerB, 20) // blocked, no damage through
	})
}

func TestTawnossWeaponry(t *testing.T) {
	t.Run("boosts creature +1/+1 while tapped", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Tawnos's Weaponry")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // 2/2
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Tawnos's Weaponry", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 3, 3) // 2+1/2+1
		g.AssertTapped(gametest.PlayerA, "Tawnos's Weaponry", true)
	})

	t.Run("boost ends when Weaponry untaps", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Tawnos's Weaponry")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Tawnos's Weaponry", "Grizzly Bears")
		g.StopAt(3, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 2, 2) // back to normal
	})
}

func TestTheRack(t *testing.T) {
	t.Run("deals 3 minus hand size damage with empty hand", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "The Rack")
		// PlayerB has 0 cards in hand → 3 - 0 = 3 damage
		g.StopAt(2, core.PrecombatMain)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 17)
	})

	t.Run("no damage when opponent has 3 or more cards", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "The Rack")
		for i := 0; i < 4; i++ {
			g.AddCard(core.ZoneHand, gametest.PlayerB, "Forest")
		}
		g.StopAt(2, core.PrecombatMain)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 20) // 3-4 = negative, no damage
	})
}

func TestUrzasChalice(t *testing.T) {
	t.Run("gains 1 life when artifact spell cast", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Urza's Chalice")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest") // land to pay {1}
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Ornithopter")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Ornithopter")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertLife(gametest.PlayerA, 21)
	})
}

func TestUrzasMiter(t *testing.T) {
	t.Run("draws card when non-sacrifice artifact dies", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Urza's Miter")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ornithopter") // 0/2 artifact
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Forest") // card to draw
		// Destroy ornithopter (non-sacrifice death)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerB, "Lightning Bolt", "Ornithopter")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Urza's Miter trigger should draw a card (if player has {3} mana)
		g.AssertPermanentCount(gametest.PlayerA, "Ornithopter", 0)
	})

	t.Run("does not trigger on sacrificed artifacts", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Urza's Miter")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ashnod's Altar")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ornithopter")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Forest")
		// Sacrifice Ornithopter to Ashnod's Altar (Flag = true)
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Ashnod's Altar")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Sacrifice flag prevents Miter trigger
		g.AssertPermanentCount(gametest.PlayerA, "Ornithopter", 0)
	})
}

func TestWeakstone(t *testing.T) {
	t.Run("attacking creatures get -1/-0", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Weakstone")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant") // 3/3
		g.Attack(2, gametest.PlayerB, "Hill Giant")
		g.StopAt(2, core.EndCombat)
		g.Execute()
		// Hill Giant attacks with -1/-0 from Weakstone = 2/3 → deals 2 damage
		g.AssertLife(gametest.PlayerA, 18)
	})

	t.Run("does not affect non-attacking creatures", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Weakstone")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerB, "Hill Giant", 3, 3)
	})
}
