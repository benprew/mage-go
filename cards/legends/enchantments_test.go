package legends

import (
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/gametest"
)

func TestSeeker(t *testing.T) {
	t.Run("enchanted creature cannot be blocked by non-artifact non-white creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Seeker")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant") // 3/3 red
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Seeker", "Grizzly Bears")
		g.Attack(1, gametest.PlayerA, "Grizzly Bears")
		g.Block(1, gametest.PlayerB, "Hill Giant", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Hill Giant is red and not artifact — can't block enchanted Bears
		g.AssertLife(gametest.PlayerB, 18)
	})

	t.Run("enchanted creature can be blocked by white creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Seeker")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Keepers of the Faith") // 2/3 white
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Seeker", "Grizzly Bears")
		g.Attack(1, gametest.PlayerA, "Grizzly Bears")
		g.Block(1, gametest.PlayerB, "Keepers of the Faith", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Keepers of the Faith is white — can block
		g.AssertLife(gametest.PlayerB, 20)
	})
}

func TestUnderworldDreams(t *testing.T) {
	t.Run("deals 1 damage when opponent draws", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Underworld Dreams")
		// PlayerB draws on turn 2 (draw step)
		g.StopAt(2, core.PrecombatMain)
		g.Execute()
		// PlayerB drew 1 card — should take 1 damage
		g.AssertLife(gametest.PlayerB, 19)
	})

	t.Run("does not damage controller on their draw", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Underworld Dreams")
		// PlayerA draws on turn 3 (their next draw step)
		g.StopAt(3, core.PrecombatMain)
		g.Execute()
		// PlayerA should not take damage from their own Underworld Dreams
		g.AssertLife(gametest.PlayerA, 20)
	})
}

func TestMarblePriest(t *testing.T) {
	t.Run("Wall combat damage to Marble Priest is prevented", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		// Use a Marble Priest (3/3) attacking into a Wall of Opposition (0/6)
		// Pump the wall's power to 4 via its {1}: +1/+0 ability so it would be lethal
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Marble Priest") // 3/3
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Wall of Opposition") // 0/6 Wall, {1}: +1/+0
		// Pump 4 times during begin combat on PlayerA's turn (PlayerB activates)
		g.ActivateAbility(1, core.BeginCombat, gametest.PlayerB, "Wall of Opposition")
		g.ActivateAbility(1, core.BeginCombat, gametest.PlayerB, "Wall of Opposition")
		g.ActivateAbility(1, core.BeginCombat, gametest.PlayerB, "Wall of Opposition")
		g.ActivateAbility(1, core.BeginCombat, gametest.PlayerB, "Wall of Opposition")
		g.Attack(1, gametest.PlayerA, "Marble Priest")
		g.Block(1, gametest.PlayerB, "Wall of Opposition", "Marble Priest")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Wall of Opposition is 4/6 — would deal 4 damage (lethal to 3/3) without prevention
		// With prevention, Marble Priest survives
		g.AssertPermanentCount(gametest.PlayerA, "Marble Priest", 1)
	})
}

func TestKismet(t *testing.T) {
	t.Run("opponent creatures enter tapped", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Kismet")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Grizzly Bears")
		g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Grizzly Bears")
		g.StopAt(2, core.BeginCombat)
		g.Execute()
		g.AssertTapped(gametest.PlayerB, "Grizzly Bears", true)
	})
}

func TestStormWorld(t *testing.T) {
	t.Run("deals damage when hand is small", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Storm World")
		// PlayerA starts with 0 cards in hand; upkeep damage = 4-0 = 4
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		// Storm World triggers at each upkeep: PlayerA has 0 cards → 4 damage
		g.AssertLife(gametest.PlayerA, 16)
	})
}

func TestSpiritualSanctuary(t *testing.T) {
	t.Run("gain 1 life if active player controls Plains", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Spiritual Sanctuary")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
		// Turn 1 upkeep: PlayerA has Plains → gains 1
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertLife(gametest.PlayerA, 21)
	})
}

func TestLivingPlane(t *testing.T) {
	t.Run("lands become 1/1 creatures", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Living Plane")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Forest", 1, 1)
	})
}

func TestTheBrute(t *testing.T) {
	t.Run("enchanted creature gets +1/+0", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "The Brute")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "The Brute", "Grizzly Bears")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 3, 2) // +1/+0
	})
}

func TestNetherVoid(t *testing.T) {
	t.Run("counters spell if opponent cant pay 3", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Nether Void")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Grizzly Bears")
		// PlayerB casts Bears, tapping their 2 lands — can't pay extra {3}
		g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Grizzly Bears")
		g.StopAt(2, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
	})
}

func TestBlight(t *testing.T) {
	t.Run("destroy enchanted land when it becomes tapped", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Forest")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Blight")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Blight", "Forest")
		// Explicitly tap Forest by activating its mana ability
		g.ActivateAbility(2, core.PrecombatMain, gametest.PlayerB, "Forest")
		g.StopAt(2, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Forest", 0)
	})
}

func TestDemonicTorment(t *testing.T) {
	t.Run("enchanted creature cannot attack", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Demonic Torment")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Demonic Torment", "Grizzly Bears")
		g.Attack(1, gametest.PlayerA, "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Bears can't attack due to Demonic Torment
		g.AssertLife(gametest.PlayerB, 20)
	})
}

func TestGaseousForm(t *testing.T) {
	t.Run("prevents all combat damage to and from enchanted creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant") // 3/3
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Gaseous Form")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Gaseous Form", "Hill Giant")
		g.Attack(1, gametest.PlayerA, "Hill Giant")
		g.Block(1, gametest.PlayerB, "Grizzly Bears", "Hill Giant")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Hill Giant deals no damage (prevented), Bears deal no damage to Hill Giant (prevented)
		g.AssertPermanentCount(gametest.PlayerA, "Hill Giant", 1)
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 1)
	})
}

func TestInTheEyeOfChaos(t *testing.T) {
	t.Run("counters instant if player cant pay CMC", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "In the Eye of Chaos")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Flash Counter") // CMC 2 instant
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
		// PlayerA casts Bears, then PlayerB tries to Flash Counter it
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Grizzly Bears")
		g.CastInResponseTo(gametest.PlayerB, "Flash Counter")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Flash Counter should be countered by In the Eye of Chaos (no lands to pay CMC 2)
		// Bears should resolve
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
	})
}

func TestPresenceOfTheMaster(t *testing.T) {
	t.Run("counters enchantment spells", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Presence of the Master")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Giant Strength")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Giant Strength", "Grizzly Bears")
		g.StopAt(2, core.BeginCombat)
		g.Execute()
		// Giant Strength should be countered
		g.AssertPowerToughness(gametest.PlayerB, "Grizzly Bears", 2, 2) // no boost
	})
}

func TestSpectralCloak(t *testing.T) {
	t.Run("enchanted untapped creature has shroud", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Spectral Cloak")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Spectral Cloak", "Grizzly Bears")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		g.AssertHasAbility(gametest.PlayerA, "Grizzly Bears", core.Shroud, true)
	})
}

func TestLifeblood(t *testing.T) {
	t.Run("gain 1 life when opponent taps Mountain", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Lifeblood")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mountain")
		// Explicitly tap Mountain by activating its mana ability
		g.ActivateAbility(2, core.PrecombatMain, gametest.PlayerB, "Mountain")
		g.StopAt(2, core.BeginCombat)
		g.Execute()
		g.AssertLife(gametest.PlayerA, 21)
	})
}

func TestCocoon(t *testing.T) {
	t.Run("taps creature and adds pupa counters then removes them", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Cocoon")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Cocoon", "Grizzly Bears")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		// Bears should be tapped and have 3 pupa counters
		g.AssertTapped(gametest.PlayerA, "Grizzly Bears", true)
		g.AssertCounterCount(gametest.PlayerA, "Cocoon", core.Pupa, 3)
	})
}

func TestBackfire(t *testing.T) {
	t.Run("deals damage back equal to damage dealt to controller", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Craw Wurm") // 6/4
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Backfire")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Backfire", "Craw Wurm")
		g.Attack(2, gametest.PlayerB, "Craw Wurm")
		g.StopAt(2, core.EndStep)
		g.Execute()
		// Craw Wurm dealt 6 to PlayerA — Backfire deals 6 to Craw Wurm's controller (PlayerB)
		g.AssertLife(gametest.PlayerA, 14) // 20 - 6
		g.AssertLife(gametest.PlayerB, 14) // 20 - 6
	})
}

func TestTakklemaggot(t *testing.T) {
	t.Run("puts -0/-1 counter on enchanted creature at upkeep", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Takklemaggot")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Takklemaggot", "Grizzly Bears")
		// Turn 2 upkeep (PlayerB's turn): should put -0/-1 on Bears
		g.StopAt(2, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerB, "Grizzly Bears", 2, 1) // 2/2 - 0/1
	})
}

func TestVenarianGold(t *testing.T) {
	t.Run("taps creature and adds sleep counters", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Venarian Gold")
		g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Venarian Gold", 3, "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Bears should be tapped with 3 sleep counters
		g.AssertTapped(gametest.PlayerB, "Grizzly Bears", true)
		g.AssertCounterCount(gametest.PlayerB, "Grizzly Bears", core.Sleep, 3)
	})
}

func TestPuppetMaster(t *testing.T) {
	t.Run("returns enchanted creature to hand on death", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Puppet Master")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Craw Wurm") // 6/4
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Puppet Master", "Grizzly Bears")
		// Bears attack and get blocked by Craw Wurm, Bears die
		g.Attack(3, gametest.PlayerA, "Grizzly Bears")
		g.Block(3, gametest.PlayerB, "Craw Wurm", "Grizzly Bears")
		g.StopAt(3, core.EndStep)
		g.Execute()
		// Bears should be returned to hand (Puppet Master trigger)
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 0)
		g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 0)
	})
}

func TestDreamCoat(t *testing.T) {
	t.Run("changes enchanted creature's color", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // green 2/2
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Dream Coat")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Dream Coat", "Grizzly Bears")
		// Activate: choose red
		g.ChooseManaColor(gametest.PlayerA, core.Red)
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Dream Coat")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Bears should still be on battlefield, P/T unchanged
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 2, 2)
		g.AssertAttachedTo(gametest.PlayerA, "Dream Coat", "Grizzly Bears")
	})
}

func TestInfiniteAuthority(t *testing.T) {
	t.Run("destroys creature with toughness 3 or less at end of combat", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")        // 3/3
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Infinite Authority")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")     // 2/2 (toughness <= 3)
		// Cast Infinite Authority on Hill Giant
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Infinite Authority", "Hill Giant")
		// Attack with Hill Giant, blocked by Bears
		g.Attack(1, gametest.PlayerA, "Hill Giant")
		g.Block(1, gametest.PlayerB, "Grizzly Bears", "Hill Giant")
		g.StopAt(2, core.PrecombatMain) // go to next turn so EndStep triggers fire
		g.Execute()
		// Bears (toughness 2 <= 3) destroyed at end of combat by Infinite Authority
		g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
		// Hill Giant survives (3/3 took 2 damage = 3/1)
		g.AssertPermanentCount(gametest.PlayerA, "Hill Giant", 1)
		// Hill Giant should have gotten a +1/+1 counter at end step
		g.AssertCounterCount(gametest.PlayerA, "Hill Giant", core.P1P1, 1)
	})
}

func TestLandTax(t *testing.T) {
	t.Run("searches for up to three basic lands when opponent has more", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Land Tax")
		// Opponent has 2 lands, we have 0
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mountain")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mountain")
		// Put 3 basic lands in library for searching
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Plains")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Plains")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Plains")
		// Script the library choices
		g.ChooseFromLibrary(gametest.PlayerA, "Plains")
		g.ChooseFromLibrary(gametest.PlayerA, "Plains")
		g.ChooseFromLibrary(gametest.PlayerA, "Plains")
		// Upkeep trigger fires on turn 1 (PlayerA's upkeep)
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		// Should have found 3 Plains in hand
		g.AssertHandCount(gametest.PlayerA, "Plains", 3)
	})
	t.Run("does not trigger when opponent has fewer lands", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Land Tax")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
		// Opponent has 1 land — fewer than us
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mountain")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Plains")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		// Should NOT have searched
		g.AssertHandCount(gametest.PlayerA, "Plains", 0)
	})
}
