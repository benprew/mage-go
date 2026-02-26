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
	// XXX: doom counters + draw step damage + any-player activation
	t.Run("is a 6-cost artifact", func(t *testing.T) {
		card, err := mage.CreateCard("Armageddon Clock")
		if err != nil {
			t.Fatal(err)
		}
		if card.ManaCost().CMC() != 6 {
			t.Errorf("expected CMC 6, got %d", card.ManaCost().CMC())
		}
	})

	t.Run("deals damage equal to doom counters at draw step", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Armageddon Clock")
		// After upkeep (1 doom counter), at draw step deals 1 to each player
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertLife(gametest.PlayerA, 19) // 1 doom counter → 1 damage
		g.AssertLife(gametest.PlayerB, 19)
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
	// XXX: continuous boost while source remains tapped
	t.Run("is a 2-cost artifact", func(t *testing.T) {
		card, err := mage.CreateCard("Ashnod's Battle Gear")
		if err != nil {
			t.Fatal(err)
		}
		if card.ManaCost().CMC() != 2 {
			t.Errorf("expected CMC 2, got %d", card.ManaCost().CMC())
		}
	})
}

func TestAshnodsTransmogrant(t *testing.T) {
	// XXX: type addition + counter
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

func TestCandelabraOfTawnos(t *testing.T) {
	// XXX: X-targeting for untap lands
	t.Run("is a 1-cost artifact", func(t *testing.T) {
		card, err := mage.CreateCard("Candelabra of Tawnos")
		if err != nil {
			t.Fatal(err)
		}
		if card.ManaCost().CMC() != 1 {
			t.Errorf("expected CMC 1, got %d", card.ManaCost().CMC())
		}
	})
}

func TestCoralHelm(t *testing.T) {
	// XXX: discard at random as cost
	t.Run("is a 3-cost artifact", func(t *testing.T) {
		card, err := mage.CreateCard("Coral Helm")
		if err != nil {
			t.Fatal(err)
		}
		if card.ManaCost().CMC() != 3 {
			t.Errorf("expected CMC 3, got %d", card.ManaCost().CMC())
		}
	})
}

func TestCursedRack(t *testing.T) {
	// XXX: max hand size reduction for chosen opponent
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
	// XXX: own-and-control restriction on targeting
	t.Run("is a 1-cost artifact", func(t *testing.T) {
		card, err := mage.CreateCard("Obelisk of Undoing")
		if err != nil {
			t.Fatal(err)
		}
		if card.ManaCost().CMC() != 1 {
			t.Errorf("expected CMC 1, got %d", card.ManaCost().CMC())
		}
	})

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
	// XXX: conditional trigger on own artifact death with optional payment
	t.Run("is a 1-cost artifact", func(t *testing.T) {
		card, err := mage.CreateCard("Tablet of Epityr")
		if err != nil {
			t.Fatal(err)
		}
		if card.ManaCost().CMC() != 1 {
			t.Errorf("expected CMC 1, got %d", card.ManaCost().CMC())
		}
	})

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

func TestTawnossCoffin(t *testing.T) {
	// XXX: complex exile/return with counter/aura tracking
	t.Run("is a 4-cost artifact", func(t *testing.T) {
		card, err := mage.CreateCard("Tawnos's Coffin")
		if err != nil {
			t.Fatal(err)
		}
		if card.ManaCost().CMC() != 4 {
			t.Errorf("expected CMC 4, got %d", card.ManaCost().CMC())
		}
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

func TestTawnossWeaponry(t *testing.T) {
	// XXX: continuous boost while source remains tapped
	t.Run("is a 2-cost artifact", func(t *testing.T) {
		card, err := mage.CreateCard("Tawnos's Weaponry")
		if err != nil {
			t.Fatal(err)
		}
		if card.ManaCost().CMC() != 2 {
			t.Errorf("expected CMC 2, got %d", card.ManaCost().CMC())
		}
	})
}

func TestTheRack(t *testing.T) {
	// XXX: chosen opponent + hand-size based damage
	t.Run("deals 3 minus hand size damage to chosen player", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "The Rack")
		// PlayerB has 0 cards in hand → 3 - 0 = 3 damage
		g.StopAt(2, core.PrecombatMain)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 17)
	})
}

func TestUrzasChalice(t *testing.T) {
	// XXX: trigger on artifact spell cast with optional payment
	t.Run("is a 1-cost artifact", func(t *testing.T) {
		card, err := mage.CreateCard("Urza's Chalice")
		if err != nil {
			t.Fatal(err)
		}
		if card.ManaCost().CMC() != 1 {
			t.Errorf("expected CMC 1, got %d", card.ManaCost().CMC())
		}
	})

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
	// XXX: non-sacrifice artifact death trigger with optional payment
	t.Run("is a 3-cost artifact", func(t *testing.T) {
		card, err := mage.CreateCard("Urza's Miter")
		if err != nil {
			t.Fatal(err)
		}
		if card.ManaCost().CMC() != 3 {
			t.Errorf("expected CMC 3, got %d", card.ManaCost().CMC())
		}
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
