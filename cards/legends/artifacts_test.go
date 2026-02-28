package legends

import (
	"testing"

	"github.com/mage/mage/pkg/mage/core"
	"github.com/mage/mage/pkg/mage/gametest"
)

func TestRelicBarrier(t *testing.T) {
	t.Run("tap to tap target artifact", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Relic Barrier")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Black Mana Battery")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Relic Barrier", "Black Mana Battery")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Relic Barrier tapped itself (tap cost) and tapped the Black Mana Battery
		g.AssertTapped(gametest.PlayerA, "Relic Barrier", true)
		g.AssertTapped(gametest.PlayerB, "Black Mana Battery", true)
	})
}

func TestHornOfDeafening(t *testing.T) {
	t.Run("prevents combat damage from target creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Horn of Deafening")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Craw Wurm") // 6/4
		// Prevent Craw Wurm's damage on turn 2
		g.ActivateAbility(2, core.BeginCombat, gametest.PlayerA, "Horn of Deafening", "Craw Wurm")
		g.Attack(2, gametest.PlayerB, "Craw Wurm")
		g.StopAt(2, core.EndStep)
		g.Execute()
		// Craw Wurm's combat damage prevented
		g.AssertLife(gametest.PlayerA, 20)
	})
}

func TestMirrorUniverse(t *testing.T) {
	t.Run("exchanges life totals", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mirror Universe")
		g.SetLife(gametest.PlayerA, 5)
		g.SetLife(gametest.PlayerB, 18)
		// Activate during PlayerA's upkeep (turn 3 — need to wait since it enters tapped? No, it's on battlefield)
		g.ActivateAbility(1, core.Upkeep, gametest.PlayerA, "Mirror Universe")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		// Life totals should be swapped
		g.AssertLife(gametest.PlayerA, 18)
		g.AssertLife(gametest.PlayerB, 5)
		// Mirror Universe should be sacrificed
		g.AssertPermanentCount(gametest.PlayerA, "Mirror Universe", 0)
	})
}

func TestSerpentGenerator(t *testing.T) {
	t.Run("creates snake token", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Serpent Generator")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Serpent Generator")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Should have a 1/1 Snake token
		g.AssertPermanentCount(gametest.PlayerA, "Snake", 1)
		g.AssertPowerToughness(gametest.PlayerA, "Snake", 1, 1)
	})

	t.Run("snake token gives poison counter on damage", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Serpent Generator")
		// Create the token on turn 1
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Serpent Generator")
		// Attack with the Snake on turn 3 (needs to lose summoning sickness)
		g.Attack(3, gametest.PlayerA, "Snake")
		g.StopAt(3, core.EndStep)
		g.Execute()
		// Snake dealt 1 combat damage + gave poison counter
		g.AssertLife(gametest.PlayerB, 19)
		g.AssertPoisonCounters(gametest.PlayerB, 1)
	})
}

func TestAlAbarasCarpet(t *testing.T) {
	t.Run("prevents damage from non-flying attackers", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Al-abara's Carpet")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears") // 2/2 no flying
		// Activate Carpet before combat on PlayerB's turn
		g.ActivateAbility(2, core.BeginCombat, gametest.PlayerA, "Al-abara's Carpet")
		g.Attack(2, gametest.PlayerB, "Grizzly Bears")
		g.StopAt(2, core.EndStep)
		g.Execute()
		// Bears' damage should be prevented (non-flying attacker)
		g.AssertLife(gametest.PlayerA, 20)
	})

	t.Run("does not prevent damage from flying attackers", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Al-abara's Carpet")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Azure Drake") // 2/4 flying
		// Activate Carpet before combat on PlayerB's turn
		g.ActivateAbility(2, core.BeginCombat, gametest.PlayerA, "Al-abara's Carpet")
		g.Attack(2, gametest.PlayerB, "Azure Drake")
		g.StopAt(2, core.EndStep)
		g.Execute()
		// Azure Drake has flying — damage should NOT be prevented
		g.AssertLife(gametest.PlayerA, 18)
	})
}

func TestLifeChisel(t *testing.T) {
	t.Run("sacrifice creature gains life equal to toughness", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Life Chisel")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Craw Wurm") // 6/4
		g.SetLife(gametest.PlayerA, 10)
		// Activate during upkeep (turn 1)
		g.ActivateAbility(1, core.Upkeep, gametest.PlayerA, "Life Chisel")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		// Sacrificed Craw Wurm (toughness 4), gained 4 life: 10 + 4 = 14
		g.AssertLife(gametest.PlayerA, 14)
		g.AssertPermanentCount(gametest.PlayerA, "Craw Wurm", 0)
	})
}

func TestKryShield(t *testing.T) {
	t.Run("prevents damage from creature and boosts toughness", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Kry Shield")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // 2/2, CMC 2
		// Use Kry Shield on Bears — prevents Bears from dealing damage + gives +0/+2
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Kry Shield", "Grizzly Bears")
		// Bears attack — they deal 0 damage (prevented)
		g.Attack(1, gametest.PlayerA, "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Bears' damage should be prevented — PlayerB takes 0
		g.AssertLife(gametest.PlayerB, 20)
		// Bears should have +0/+2 from CMC boost: 2/4
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 2, 4)
	})
}

func TestArenaOfTheAncients(t *testing.T) {
	t.Run("taps legendary creatures on ETB", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Sol'kanar the Swamp King")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Arena of the Ancients")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Arena of the Ancients")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Sol'kanar is legendary — should be tapped
		g.AssertTapped(gametest.PlayerA, "Sol'kanar the Swamp King", true)
		// Grizzly Bears is not legendary — should not be tapped
		g.AssertTapped(gametest.PlayerB, "Grizzly Bears", false)
	})

	t.Run("legendary creatures do not untap", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Arena of the Ancients")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Sol'kanar the Swamp King")
		// Sol'kanar attacks on turn 1, becomes tapped
		g.Attack(1, gametest.PlayerA, "Sol'kanar the Swamp King")
		// On turn 3 (PlayerA's next turn), Sol'kanar should still be tapped
		g.StopAt(3, core.PrecombatMain)
		g.Execute()
		g.AssertTapped(gametest.PlayerA, "Sol'kanar the Swamp King", true)
	})
}

func TestBlackManaBattery(t *testing.T) {
	t.Run("adds charge counter", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Black Mana Battery")
		// Charge on turn 1 (first ability: {2}, {T}: put a charge counter)
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Black Mana Battery")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertCounterCount(gametest.PlayerA, "Black Mana Battery", core.Charge, 1)
		g.AssertTapped(gametest.PlayerA, "Black Mana Battery", true)
	})
}

func TestManaMatrix(t *testing.T) {
	t.Run("reduces instant spell cost by 2", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mana Matrix")
		// Lightning Bolt is {R} (1 CMC) — cost reduced to {R} (generic is 0, can't reduce below 0)
		// Use a bigger spell: Fireball {X}{R} — cast with X=3 costs {3}{R}. With Mana Matrix, costs {1}{R}.
		// Actually, let's use a simpler test. Holy Day is {W} (1 CMC). Flash Counter is {1}{U} (2 CMC).
		// Flash Counter {1}{U}: with Mana Matrix, generic drops from 1 to 0, so it costs {U}.
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Flash Counter")
		// Flash Counter targets an instant spell — we need an instant to counter.
		// Let's test with a sorcery instead. Chain Lightning is {R} (1 CMC sorcery).
		// The cost reduction applies to the spell itself, not to what it targets.
		// Let's just test that a spell with generic cost can be cast with less mana.
		// Holy Day {W} is instant, costs {W}. With Mana Matrix, still {W} (no generic).
		// Use Divine Offering: {1}{W} instant. With Mana Matrix, costs {W} only.
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Divine Offering")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Black Mana Battery") // artifact target
		// Cast Divine Offering (normally {1}{W}, with Matrix costs {W})
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Divine Offering", "Black Mana Battery")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Divine Offering resolved — Black Mana Battery destroyed
		g.AssertPermanentCount(gametest.PlayerB, "Black Mana Battery", 0)
	})
}

func TestPlanarGate(t *testing.T) {
	t.Run("reduces creature spell cost by 2", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Planar Gate")
		// Hill Giant is {3}{R} (4 CMC). With Planar Gate, costs {1}{R}.
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Hill Giant")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Hill Giant")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Hill Giant", 1)
	})
}
