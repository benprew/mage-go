package limited

import (
	"testing"

	_ "git.sr.ht/~cdcarter/mage-go/cards/arabian"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/gametest"
)

// TestWorldsNinetyFour recreates a game inspired by the 1994 Magic World
// Championship finals between Zak Dolan (Blue/White control) and Bertrand
// Lestree (Red/Green aggro). Dolan used Swords to Plowshares, Serra Angel,
// and Counterspell to overcome Lestree's aggressive creature and burn strategy.
//
// Historical context: Dolan won 2-1 to become the first Magic World Champion
// at GenCon '94 in Milwaukee. His control deck featured Serra Angel as the
// primary win condition backed by countermagic, while Lestree's Zoo deck
// relied on efficient creatures and burn spells to close games quickly.
func TestWorldsNinetyFour(t *testing.T) {
	g := gametest.NewTestGame(t)

	// --- Battlefield: Lestree (B) deployed early threats ---
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Taiga")         // Mountain Forest — powers up Kird Ape
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Kird Ape")      // 2/3 with Taiga
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Ironclaw Orcs") // 2/2

	// --- Hands ---
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Swords to Plowshares")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Serra Angel")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Counterspell")
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Fireball")

	// Turn 1 (Dolan): Exile Kird Ape with Swords to Plowshares.
	// Lestree gains life equal to Kird Ape's power (2, boosted by Taiga). B: 22
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Swords to Plowshares", "Kird Ape")

	// Turn 2 (Lestree): Lightning Bolt Dolan, then swing with Ironclaw Orcs.
	// A takes 3 (bolt) + 2 (orcs) = 5 damage. A: 15
	g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Lightning Bolt", "PlayerA")
	g.Attack(2, gametest.PlayerB, "Ironclaw Orcs")

	// Turn 3 (Dolan): Deploy Serra Angel — the iconic finisher.
	g.CastSpell(3, core.PrecombatMain, gametest.PlayerA, "Serra Angel")

	// Turn 4 (Lestree): Ironclaw Orcs charge in, but Serra Angel blocks.
	// Serra (4/4) kills Ironclaw (2/2). Serra takes 2 damage, heals at cleanup.
	g.Attack(4, gametest.PlayerB, "Ironclaw Orcs")
	g.Block(4, gametest.PlayerA, "Serra Angel", "Ironclaw Orcs")

	// Turn 5 (Dolan): Serra Angel takes flight — vigilance means she doesn't tap.
	// B: 22 - 4 = 18
	g.Attack(5, gametest.PlayerA, "Serra Angel")

	// Turn 6 (Lestree): Desperation Fireball (X=4) aimed at Serra Angel.
	// Dolan has Counterspell ready — the Fireball fizzles.
	g.CastSpellWithX(6, core.PrecombatMain, gametest.PlayerB, "Fireball", 4, "Serra Angel")
	g.CastInResponseTo(gametest.PlayerA, "Counterspell")

	// Turn 7 (Dolan): Serra Angel attacks again. B: 18 - 4 = 14
	g.Attack(7, gametest.PlayerA, "Serra Angel")

	g.StopAt(7, core.EndCombat)
	g.Execute()

	// --- Assertions ---
	// Dolan took a bolt and one Ironclaw swing
	g.AssertLife(gametest.PlayerA, 15)
	// Lestree gained 2 from StP, then took two Serra hits
	g.AssertLife(gametest.PlayerB, 14)
	// Serra Angel dominates the empty board
	g.AssertPermanentCount(gametest.PlayerA, "Serra Angel", 1)
	// Kird Ape was exiled — not in graveyard or on battlefield
	g.AssertPermanentCount(gametest.PlayerB, "Kird Ape", 0)
	g.AssertGraveyardCount(gametest.PlayerB, "Kird Ape", 0)
	// Ironclaw Orcs fell to Serra Angel in combat
	g.AssertPermanentCount(gametest.PlayerB, "Ironclaw Orcs", 0)
	g.AssertGraveyardCount(gametest.PlayerB, "Ironclaw Orcs", 1)
}

// TestOldSchoolCreatureBrawl recreates a game inspired by Old School 93/94
// tournament play — the kind of back-and-forth creature battles that defined
// early Magic. Green/Red ground aggro races against Blue/White flyers in a
// slugfest featuring removal, bounce, and a clutch Giant Growth.
//
// Historical context: Old School 93/94 tournaments restrict cards to sets
// printed in 1993–1994 (Alpha through The Dark). These events celebrate the
// raw creature combat and simple spells of Magic's earliest days, where
// Air Elemental was a top-tier finisher and Giant Growth decided games.
func TestOldSchoolCreatureBrawl(t *testing.T) {
	g := gametest.NewTestGame(t)

	// --- Battlefield: early drops already deployed ---
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")    // 2/2
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hurloon Minotaur") // 2/3
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Air Elemental")    // 4/4 flying

	// --- Hands ---
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Giant Growth")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Swords to Plowshares")
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Unsummon")

	// Turn 1 (A): Ground assault — Bears and Minotaur swing for 4.
	// B: 20 - 4 = 16
	g.Attack(1, gametest.PlayerA, "Grizzly Bears", "Hurloon Minotaur")

	// Turn 2 (B): StP exiles Minotaur (A gains 2 life from its power).
	// Then Air Elemental flies over for 4.
	// A: 20 + 2 - 4 = 18
	g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Swords to Plowshares", "Hurloon Minotaur")
	g.Attack(2, gametest.PlayerB, "Air Elemental")

	// Turn 3 (A): Bolt to the face, then Bears attacks unblocked.
	// B: 16 - 3 - 2 = 11
	g.CastSpell(3, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "PlayerB")
	g.Attack(3, gametest.PlayerA, "Grizzly Bears")

	// Turn 4 (B): Air Elemental swings again, then Unsummon bounces Bears.
	// A: 18 - 4 = 14
	g.Attack(4, gametest.PlayerB, "Air Elemental")
	g.CastSpell(4, core.PostcombatMain, gametest.PlayerB, "Unsummon", "Grizzly Bears")

	// Turn 5 (A): Replay Bears from hand. Summoning sickness — can't attack.
	g.CastSpell(5, core.PrecombatMain, gametest.PlayerA, "Grizzly Bears")

	// Turn 6 (B): Air Elemental keeps the pressure on.
	// A: 14 - 4 = 10
	g.Attack(6, gametest.PlayerB, "Air Elemental")

	// Turn 7 (A): Giant Growth makes Bears a 5/5, then swing for the fences.
	// B: 11 - 5 = 6
	g.CastSpell(7, core.PrecombatMain, gametest.PlayerA, "Giant Growth", "Grizzly Bears")
	g.Attack(7, gametest.PlayerA, "Grizzly Bears")

	g.StopAt(7, core.EndCombat)
	g.Execute()

	// --- Assertions ---
	// A took three Air Elemental hits, partially offset by StP life gain
	g.AssertLife(gametest.PlayerA, 10)
	// B took ground attacks, a bolt, and a Giant Growth-boosted swing
	g.AssertLife(gametest.PlayerB, 6)
	// Both sides still have creatures in play
	g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
	g.AssertPermanentCount(gametest.PlayerB, "Air Elemental", 1)
	// Hurloon Minotaur was exiled by StP — not in graveyard
	g.AssertPermanentCount(gametest.PlayerA, "Hurloon Minotaur", 0)
	g.AssertGraveyardCount(gametest.PlayerA, "Hurloon Minotaur", 0)
}

// TestSengirsFeast demonstrates Sengir Vampire's signature ability — growing
// stronger by devouring creatures in combat. Each flying blocker that falls
// to the Vampire feeds it a +1/+1 counter, turning a 4/4 into an unstoppable
// 6/6 that dominates the skies once all opposition is consumed.
//
// Historical context: Sengir Vampire was one of Alpha's most feared creatures.
// In the early days of Magic, a 4/4 flyer that grew was nearly unbeatable —
// especially in Limited formats where flying blockers were scarce.
func TestSengirsFeast(t *testing.T) {
	g := gametest.NewTestGame(t)

	// --- Battlefield ---
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Sengir Vampire")  // 4/4 flying
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mesa Pegasus")    // 1/1 flying banding
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Phantom Monster") // 3/3 flying
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")   // 2/2 (can't block flyers)

	// Turn 1 (A): Sengir attacks. B chump-blocks with Mesa Pegasus.
	// Sengir (4/4) kills Pegasus (1/1) → gains +1/+1 counter → 5/5.
	g.Attack(1, gametest.PlayerA, "Sengir Vampire")
	g.Block(1, gametest.PlayerB, "Mesa Pegasus", "Sengir Vampire")

	// Turn 2 (B): Bears swing alone — B holds Phantom Monster back as a blocker.
	// A: 20 - 2 = 18
	g.Attack(2, gametest.PlayerB, "Grizzly Bears")

	// Turn 3 (A): Sengir (5/5) attacks. B intercepts with Phantom Monster (3/3 flying).
	// Sengir kills Phantom (5 > 3) → gains another +1/+1 counter → 6/6.
	// Sengir takes 3 damage, heals at cleanup.
	g.Attack(3, gametest.PlayerA, "Sengir Vampire")
	g.Block(3, gametest.PlayerB, "Phantom Monster", "Sengir Vampire")

	// Turn 4 (B): Only Grizzly Bears left. Attacks alone.
	// A: 18 - 2 = 16
	g.Attack(4, gametest.PlayerB, "Grizzly Bears")

	// Turn 5 (A): Sengir (6/6 flying) — no flyers left to block.
	// B: 20 - 6 = 14
	g.Attack(5, gametest.PlayerA, "Sengir Vampire")

	g.StopAt(5, core.EndCombat)
	g.Execute()

	// Sengir grew from 4/4 to 6/6 by devouring two flyers
	g.AssertLife(gametest.PlayerA, 16)
	g.AssertLife(gametest.PlayerB, 14)
	g.AssertPermanentCount(gametest.PlayerA, "Sengir Vampire", 1)
	g.AssertCounterCount(gametest.PlayerA, "Sengir Vampire", core.P1P1, 2)
	g.AssertPowerToughness(gametest.PlayerA, "Sengir Vampire", 6, 6)
	// Both flyers consumed
	g.AssertGraveyardCount(gametest.PlayerB, "Mesa Pegasus", 1)
	g.AssertGraveyardCount(gametest.PlayerB, "Phantom Monster", 1)
	// Bears survived
	g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 1)
}

// TestWrathAndRecovery demonstrates the classic control strategy of surviving
// an aggro onslaught, resetting the board with Wrath of God, then using
// Resurrection to rebuild faster than the opponent.
//
// Historical context: "Wrathurrection" was a staple control play pattern.
// White mages would take early damage, wipe the board, then bring back their
// best creature from the graveyard — often Serra Angel — to close the game.
func TestWrathAndRecovery(t *testing.T) {
	g := gametest.NewTestGame(t)

	// --- Battlefield: A has White Weenie start, B has fatties ---
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Savannah Lions") // 2/1
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "White Knight")   // 2/2 first strike, pro-black
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Craw Wurm")      // 6/4
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")  // 2/2

	// --- Hands ---
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Serra Angel")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Resurrection")
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Wrath of God")
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")

	// Turn 1 (A): Lions + White Knight charge in. B blocks Lions with Bears.
	// Lions (2/1) and Bears (2/2) trade. White Knight goes unblocked → B: 18.
	g.Attack(1, gametest.PlayerA, "Savannah Lions", "White Knight")
	g.Block(1, gametest.PlayerB, "Grizzly Bears", "Savannah Lions")

	// Turn 2 (B): Craw Wurm swings for 6. A: 14.
	g.Attack(2, gametest.PlayerB, "Craw Wurm")

	// Turn 3 (A): Deploy Serra Angel. White Knight attacks again → B: 16.
	g.CastSpell(3, core.PrecombatMain, gametest.PlayerA, "Serra Angel")
	g.Attack(3, gametest.PlayerA, "White Knight")

	// Turn 4 (B): Wrath of God destroys everything — Serra, White Knight, Craw Wurm.
	// Then Lightning Bolt finishes the pain. A: 11.
	g.CastSpell(4, core.PrecombatMain, gametest.PlayerB, "Wrath of God")
	g.CastSpell(4, core.PostcombatMain, gametest.PlayerB, "Lightning Bolt", "PlayerA")

	// Turn 5 (A): Resurrection brings Serra Angel back from the grave.
	g.CastSpell(5, core.PrecombatMain, gametest.PlayerA, "Resurrection", "Serra Angel")

	// Turn 7 (A): Serra Angel attacks into the empty board. B: 12.
	g.Attack(7, gametest.PlayerA, "Serra Angel")

	g.StopAt(7, core.EndCombat)
	g.Execute()

	// A survived the Wrath and rebuilt
	g.AssertLife(gametest.PlayerA, 11)
	g.AssertLife(gametest.PlayerB, 12)
	// Serra Angel rose from the dead to dominate
	g.AssertPermanentCount(gametest.PlayerA, "Serra Angel", 1)
	// Everything else is gone
	g.AssertPermanentCount(gametest.PlayerA, "White Knight", 0)
	g.AssertPermanentCount(gametest.PlayerB, "Craw Wurm", 0)
	// Wrath casualties in graveyards
	g.AssertGraveyardCount(gametest.PlayerA, "White Knight", 1)
	g.AssertGraveyardCount(gametest.PlayerB, "Craw Wurm", 1)
}

// TestEfreetTempo recreates the frenetic tempo races of Old School Magic,
// where Serendib Efreet's raw stats (3/4 flying for 3 mana) came at the cost
// of 1 damage each upkeep — a Faustian bargain that defined blue tempo decks.
// The Efreet player must win fast before their own creature kills them.
//
// Historical context: Serendib Efreet from Arabian Nights was the premier
// blue aggro creature. At 3 mana for a 3/4 flyer, it was absurdly efficient,
// but the upkeep damage meant every turn was a race against your own clock.
func TestEfreetTempo(t *testing.T) {
	g := gametest.NewTestGame(t)

	// --- Battlefield ---
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Serendib Efreet") // 3/4 flying, 1 dmg/upkeep
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")      // 3/3
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Gray Ogre")       // 2/2

	// --- Hands ---
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Giant Growth")

	// Turn 1 (A): Upkeep — Serendib bites A for 1. A: 19.
	// Bolt to B's face. Serendib flies over for 3.
	// B: 20 - 3 (bolt) - 3 (efreet) = 14
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "PlayerB")
	g.Attack(1, gametest.PlayerA, "Serendib Efreet")

	// Turn 2 (B): Hill Giant + Gray Ogre swing back. A: 14.
	g.Attack(2, gametest.PlayerB, "Hill Giant", "Gray Ogre")

	// Turn 3 (A): Upkeep — Serendib bites again. A: 13.
	// Serendib attacks. B: 11.
	g.Attack(3, gametest.PlayerA, "Serendib Efreet")

	// Turn 4 (B): Giant Growth on Hill Giant (→ 6/6). Both swing.
	// A: 13 - 6 - 2 = 5.
	g.CastSpell(4, core.PrecombatMain, gametest.PlayerB, "Giant Growth", "Hill Giant")
	g.Attack(4, gametest.PlayerB, "Hill Giant", "Gray Ogre")

	// Turn 5 (A): Upkeep — Serendib bites once more. A: 4.
	// Serendib attacks for what might be the last time. B: 8.
	g.Attack(5, gametest.PlayerA, "Serendib Efreet")

	g.StopAt(5, core.EndCombat)
	g.Execute()

	// Both players battered — A's own Efreet is a ticking clock
	g.AssertLife(gametest.PlayerA, 4)
	g.AssertLife(gametest.PlayerB, 8)
	// Serendib still alive but bleeding its controller
	g.AssertPermanentCount(gametest.PlayerA, "Serendib Efreet", 1)
	// B's ground forces intact
	g.AssertPermanentCount(gametest.PlayerB, "Hill Giant", 1)
	g.AssertPermanentCount(gametest.PlayerB, "Gray Ogre", 1)
}

// TestSerraAngelLethal plays a full game where Serra Angel (4/4 flying vigilance)
// attacks an 8-life opponent until lethal. With no flying blockers, the Angel
// finishes the game in two swings.
func TestSerraAngelLethal(t *testing.T) {
	g := gametest.NewTestGame(t)

	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Serra Angel") // 4/4 flying vigilance
	g.SetLife(gametest.PlayerB, 8)

	// Serra Angel attacks every odd turn (A's turns).
	// Turn 1: B: 8 - 4 = 4
	g.Attack(1, gametest.PlayerA, "Serra Angel")
	// Turn 3: B: 4 - 4 = 0 → dead
	g.Attack(3, gametest.PlayerA, "Serra Angel")

	g.PlayToEnd()

	g.AssertGameOver(true)
	g.AssertWinner(gametest.PlayerA)
	g.AssertLife(gametest.PlayerB, 0)
	g.AssertPermanentCount(gametest.PlayerA, "Serra Angel", 1)
}

// TestBurnRace tests a burn + creature race to lethal. PlayerA bolts B's face
// and attacks with a creature, while B fights back with their own creature and
// burn. A's burst damage wins the race.
func TestBurnRace(t *testing.T) {
	g := gametest.NewTestGame(t)

	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Gray Ogre")     // 2/2
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears") // 2/2

	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")

	g.SetLife(gametest.PlayerA, 6)
	g.SetLife(gametest.PlayerB, 7)

	// Turn 1 (A): Bolt to face + Gray Ogre attacks. B: 7 - 3 - 2 = 2
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "PlayerB")
	g.Attack(1, gametest.PlayerA, "Gray Ogre")

	// Turn 2 (B): Grizzly Bears swings back. A: 6 - 2 = 4
	g.Attack(2, gametest.PlayerB, "Grizzly Bears")

	// Turn 3 (A): Second Bolt finishes B off. B: 2 - 3 = -1 → dead
	g.CastSpell(3, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "PlayerB")

	g.PlayToEnd()

	g.AssertGameOver(true)
	g.AssertWinner(gametest.PlayerA)
	g.AssertLife(gametest.PlayerA, 4)
}
