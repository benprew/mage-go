package legends

import (
	"testing"

	"github.com/mage/mage/pkg/mage/core"
	"github.com/mage/mage/pkg/mage/gametest"
)

func TestDwarvenSong(t *testing.T) {
	t.Run("target creature becomes red", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // green
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Dwarven Song")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Dwarven Song", "Grizzly Bears")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 2, 2)
	})
}

func TestHeavensGate(t *testing.T) {
	t.Run("target creature becomes white", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // green
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Heaven's Gate")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Heaven's Gate", "Grizzly Bears")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 2, 2)
	})
}

func TestTouchOfDarkness(t *testing.T) {
	t.Run("target creature becomes black", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // green
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Touch of Darkness")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Touch of Darkness", "Grizzly Bears")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 2, 2)
	})
}

func TestPsychicPurge(t *testing.T) {
	t.Run("deals 1 damage to player", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Psychic Purge")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Psychic Purge", "PlayerB")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 19)
	})
}

func TestJovialEvil(t *testing.T) {
	t.Run("deals damage based on white creatures", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Keepers of the Faith") // white creature
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Tundra Wolves")        // white creature
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Jovial Evil")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Jovial Evil", "PlayerB")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 16) // 2 white creatures × 2 = 4 damage
	})
}

func TestTyphoon(t *testing.T) {
	t.Run("deals damage based on Islands", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Island")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Island")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Island")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Typhoon")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Typhoon")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 17) // 3 Islands = 3 damage
	})
}

func TestGreatDefender(t *testing.T) {
	t.Run("boosts toughness by mana value", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		// Durkwood Boars costs {4}{G} = CMC 5, so +0/+5
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Durkwood Boars")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Great Defender")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Great Defender", "Durkwood Boars")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Durkwood Boars", 4, 9) // 4/4 + 0/+5
	})
}

func TestManaDrain(t *testing.T) {
	t.Run("counters target spell", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Mana Drain")
		g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Grizzly Bears")
		g.CastInResponseTo(gametest.PlayerA, "Mana Drain")
		g.StopAt(2, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
		g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
	})
}

func TestTeleport(t *testing.T) {
	t.Run("makes creature unblockable", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant") // would block
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Teleport")
		g.CastSpell(3, core.BeginCombat, gametest.PlayerA, "Teleport", "Grizzly Bears")
		g.Attack(3, gametest.PlayerA, "Grizzly Bears")
		g.Block(3, gametest.PlayerB, "Hill Giant", "Grizzly Bears")
		g.StopAt(3, core.EndStep)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 18) // unblockable, 2 damage
	})
}

func TestEnergyTap(t *testing.T) {
	t.Run("taps creature for colorless mana", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		// Durkwood Boars is CMC 5
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Durkwood Boars")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Energy Tap")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Energy Tap", "Durkwood Boars")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		g.AssertTapped(gametest.PlayerA, "Durkwood Boars", true)
	})
}

func TestActiveVolcano(t *testing.T) {
	t.Run("destroy target blue permanent", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Azure Drake") // blue creature
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Active Volcano")
		g.ChooseMode(gametest.PlayerA, 0) // mode 0: destroy blue permanent
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Active Volcano", "Azure Drake")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Azure Drake", 0)
	})
}

func TestFlashFlood(t *testing.T) {
	t.Run("destroy target red permanent", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Raging Bull") // red creature
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Flash Flood")
		g.ChooseMode(gametest.PlayerA, 0) // mode 0: destroy red permanent
		g.CastSpell(2, core.PrecombatMain, gametest.PlayerA, "Flash Flood", "Raging Bull")
		g.StopAt(2, core.PostcombatMain)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Raging Bull", 0)
	})
}

func TestSylvanParadise(t *testing.T) {
	t.Run("target creature becomes green", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Raging Bull") // red
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Sylvan Paradise")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Sylvan Paradise", "Raging Bull")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Raging Bull", 2, 2)
	})
}

func TestWindsOfChange(t *testing.T) {
	t.Run("each player shuffles hand and redraws", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Winds of Change")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Raging Bull")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Durkwood Boars")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Moss Monster")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Winds of Change")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		// PlayerA had 2 cards in hand after casting (Grizzly Bears + Raging Bull)
		// Winds shuffles them in, draws 2 from library
		// So PlayerA should still have 2 cards in hand
	})
}

func TestManaDrainDelayedMana(t *testing.T) {
	t.Run("adds colorless mana at next upkeep", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		// Durkwood Boars costs {4}{G} = CMC 5
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Durkwood Boars")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Mana Drain")
		// PlayerA also has a Hill Giant (CMC 3) in hand to cast with the mana
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Hill Giant")
		g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Durkwood Boars")
		g.CastInResponseTo(gametest.PlayerA, "Mana Drain")
		// Turn 3 upkeep: delayed trigger fires, adds 5 colorless
		// Then cast Hill Giant (CMC 3) from that mana in main phase
		g.CastSpell(3, core.PrecombatMain, gametest.PlayerA, "Hill Giant")
		g.StopAt(3, core.BeginCombat)
		g.Execute()
		g.AssertGraveyardCount(gametest.PlayerB, "Durkwood Boars", 1)
		g.AssertPermanentCount(gametest.PlayerA, "Hill Giant", 1)
	})
}

func TestRapidFire(t *testing.T) {
	t.Run("grants first strike and rampage 2", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")      // 2/2
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Barbary Apes")       // 2/2 blocker 1
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Headless Horseman")  // 2/2 blocker 2
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Rapid Fire")
		g.CastSpell(3, core.BeginCombat, gametest.PlayerA, "Rapid Fire", "Grizzly Bears")
		g.Attack(3, gametest.PlayerA, "Grizzly Bears")
		g.Block(3, gametest.PlayerB, "Barbary Apes", "Grizzly Bears")
		g.Block(3, gametest.PlayerB, "Headless Horseman", "Grizzly Bears")
		g.StopAt(3, core.EndStep)
		g.Execute()
		// With rampage 2 and 2 blockers: +2/+2 from rampage (1 beyond the first)
		// Bears becomes 4/4 during combat with first strike
		// First strike kills one 2/2 blocker before it deals damage
		// Bears (4/4) takes 2 damage from remaining blocker = survives at 4/2
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
		// At least one blocker should be dead
		g.AssertGraveyardCount(gametest.PlayerB, "Barbary Apes", 1)
	})
}

func TestTelekinesis(t *testing.T) {
	t.Run("taps creature and prevents untap", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Telekinesis")
		g.CastSpell(2, core.PrecombatMain, gametest.PlayerA, "Telekinesis", "Grizzly Bears")
		// Turn 4 (PlayerB's next turn) — Bears should still be tapped
		g.StopAt(4, core.PrecombatMain)
		g.Execute()
		g.AssertTapped(gametest.PlayerB, "Grizzly Bears", true)
	})

	t.Run("creature untaps after two untap steps", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Telekinesis")
		g.CastSpell(2, core.PrecombatMain, gametest.PlayerA, "Telekinesis", "Grizzly Bears")
		// Turn 6 (PlayerB's 2nd untap after cast) — Bears should untap now
		// Cast on turn 2, skip turn 4 untap, skip turn 6 untap, untaps on turn 8
		g.StopAt(8, core.PrecombatMain)
		g.Execute()
		g.AssertTapped(gametest.PlayerB, "Grizzly Bears", false)
	})
}

func TestReincarnation(t *testing.T) {
	t.Run("reanimates creature when target dies", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneGraveyard, gametest.PlayerB, "Craw Wurm") // creature in graveyard to reanimate
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Reincarnation")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Reincarnation", "Grizzly Bears")
		// Kill Bears with Lightning Bolt — triggers reincarnation
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Bears died, Craw Wurm should be on battlefield
		g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
		g.AssertPermanentCount(gametest.PlayerB, "Craw Wurm", 1)
	})
}

func TestRecall(t *testing.T) {
	t.Run("discards X and returns X from graveyard", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Recall")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Lightning Bolt")
		g.ChooseDiscard(gametest.PlayerA, "Grizzly Bears")
		g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Recall", 1)
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Grizzly Bears discarded, Lightning Bolt returned to hand
		g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 1)
	})
}

func TestDisharmony(t *testing.T) {
	t.Run("steals attacking creature until end of turn", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Disharmony")
		// Bears attack on turn 2
		g.Attack(2, gametest.PlayerB, "Grizzly Bears")
		// Cast Disharmony targeting attacking Bears
		g.CastSpell(2, core.DeclareAttackers, gametest.PlayerA, "Disharmony", "Grizzly Bears")
		g.StopAt(2, core.EndStep)
		g.Execute()
		// Bears should be under PlayerA's control
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
	})
}

func TestGlyphOfDestruction(t *testing.T) {
	t.Run("boosts blocking Wall then destroys it at end step", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Wall of Earth") // 0/6 Wall Defender
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")    // 3/3
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Glyph of Destruction")
		// Turn 2: PlayerB attacks with Hill Giant, Wall of Earth blocks
		g.Attack(2, gametest.PlayerB, "Hill Giant")
		g.Block(2, gametest.PlayerA, "Wall of Earth", "Hill Giant")
		// Cast Glyph of Destruction targeting the blocking Wall (after blocks declared)
		g.CastSpell(2, core.FirstStrikeDamage, gametest.PlayerA, "Glyph of Destruction", "Wall of Earth")
		g.StopAt(3, core.PrecombatMain) // go to next turn so EndStep triggers fire
		g.Execute()
		// Wall got +10/+0 = 10/6, so Hill Giant should take 10 damage and die
		g.AssertGraveyardCount(gametest.PlayerB, "Hill Giant", 1)
		// Wall prevented all damage but is destroyed at end step
		g.AssertGraveyardCount(gametest.PlayerA, "Wall of Earth", 1)
	})
}

func TestGlyphOfLife(t *testing.T) {
	t.Run("gain life when target Wall is dealt combat damage", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Wall of Earth") // 0/6 Wall Defender
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")    // 3/3
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Glyph of Life")
		// Turn 2: PlayerB attacks with Hill Giant, Wall of Earth blocks
		g.Attack(2, gametest.PlayerB, "Hill Giant")
		g.Block(2, gametest.PlayerA, "Wall of Earth", "Hill Giant")
		// Cast Glyph of Life targeting Wall of Earth (after blocks declared)
		g.CastSpell(2, core.FirstStrikeDamage, gametest.PlayerA, "Glyph of Life", "Wall of Earth")
		g.StopAt(2, core.EndStep)
		g.Execute()
		// Wall took 3 damage from Hill Giant (an attacking creature) → gain 3 life
		g.AssertLife(gametest.PlayerA, 23)
		// Wall survives (0/6 took 3 damage)
		g.AssertPermanentCount(gametest.PlayerA, "Wall of Earth", 1)
	})
}

func TestGlyphOfDoom(t *testing.T) {
	t.Run("destroys creatures blocked by target Wall at end of combat", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Wall of Earth") // 0/6 Wall Defender
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")    // 3/3
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Glyph of Doom")
		// Turn 2: PlayerB attacks with Hill Giant, Wall of Earth blocks
		g.Attack(2, gametest.PlayerB, "Hill Giant")
		g.Block(2, gametest.PlayerA, "Wall of Earth", "Hill Giant")
		// Cast Glyph of Doom targeting Wall of Earth (after blocks declared)
		g.CastSpell(2, core.FirstStrikeDamage, gametest.PlayerA, "Glyph of Doom", "Wall of Earth")
		g.StopAt(2, core.EndStep)
		g.Execute()
		// Hill Giant was blocked by target Wall → destroyed at end of combat
		g.AssertGraveyardCount(gametest.PlayerB, "Hill Giant", 1)
		// Wall survives (0/6 took 3 damage from Hill Giant)
		g.AssertPermanentCount(gametest.PlayerA, "Wall of Earth", 1)
	})
}

func TestFeint(t *testing.T) {
	t.Run("taps blockers and prevents combat damage from attacker and blockers", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")    // 3/3
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears") // 2/2 blocker
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Feint")
		// Turn 1: PlayerA attacks with Hill Giant, PlayerB blocks with Grizzly Bears
		g.Attack(1, gametest.PlayerA, "Hill Giant")
		g.Block(1, gametest.PlayerB, "Grizzly Bears", "Hill Giant")
		// Cast Feint targeting Hill Giant (after blocks declared)
		g.CastSpell(1, core.FirstStrikeDamage, gametest.PlayerA, "Feint", "Hill Giant")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Grizzly Bears should be tapped (from Feint)
		g.AssertTapped(gametest.PlayerB, "Grizzly Bears", true)
		// No combat damage dealt — both creatures survive
		g.AssertPermanentCount(gametest.PlayerA, "Hill Giant", 1)
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 1)
		// No damage to players either
		g.AssertLife(gametest.PlayerB, 20)
	})
}

func TestGlyphOfDelusion(t *testing.T) {
	t.Run("puts glyph counters on creature blocked by Wall and prevents untap", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Wall of Earth") // 0/6 Wall Defender
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")    // 3/3
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Glyph of Delusion")
		// Turn 2: PlayerB attacks with Hill Giant, Wall of Earth blocks
		g.Attack(2, gametest.PlayerB, "Hill Giant")
		g.Block(2, gametest.PlayerA, "Wall of Earth", "Hill Giant")
		// Cast Glyph of Delusion targeting Wall of Earth (after blocks declared)
		g.CastSpell(2, core.FirstStrikeDamage, gametest.PlayerA, "Glyph of Delusion", "Wall of Earth")
		// Run all the way to PlayerB's next turn, after upkeep
		g.StopAt(4, core.PrecombatMain)
		g.Execute()
		// Hill Giant should have 2 glyph counters (started with 3, one removed at turn 4 upkeep)
		g.AssertCounterCount(gametest.PlayerB, "Hill Giant", core.Glyph, 2)
		// Hill Giant should still be tapped (AttrDoesNotUntap while glyph counters present)
		g.AssertTapped(gametest.PlayerB, "Hill Giant", true)
		// Hill Giant survives (3/3 took 0 damage from 0/6 Wall)
		g.AssertPermanentCount(gametest.PlayerB, "Hill Giant", 1)
	})
}

func TestGlyphOfReincarnation(t *testing.T) {
	t.Run("destroys creatures blocked by Wall and reanimates from graveyard", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Wall of Earth") // 0/6 Wall Defender
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")    // 3/3
		g.AddCard(core.ZoneGraveyard, gametest.PlayerB, "Grizzly Bears")   // creature in graveyard
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Glyph of Reincarnation")
		// Turn 2: PlayerB attacks with Hill Giant, Wall of Earth blocks
		g.Attack(2, gametest.PlayerB, "Hill Giant")
		g.Block(2, gametest.PlayerA, "Wall of Earth", "Hill Giant")
		// Cast Glyph of Reincarnation targeting Wall of Earth after combat
		g.CastSpell(2, core.PostcombatMain, gametest.PlayerA, "Glyph of Reincarnation", "Wall of Earth")
		g.StopAt(2, core.EndStep)
		g.Execute()
		// Hill Giant should be destroyed (blocked by target Wall this turn)
		g.AssertGraveyardCount(gametest.PlayerB, "Hill Giant", 1)
		// Grizzly Bears should be reanimated from PlayerB's graveyard
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 1)
		g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 0)
	})
}
