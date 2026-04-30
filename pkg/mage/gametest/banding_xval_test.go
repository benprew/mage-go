package gametest

import (
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// Cross-validation tests against XMage behavior, focused on engine-level
// banding rules (CR 702.22 / formerly 702.21). XMage reference:
// ~/src/xmage/Mage/src/main/java/mage/game/combat/CombatGroup.java
//
// The mg-oracle banding combat path is in pkg/mage/combat.go:
//   - doBandedAttackDamage (combat.go:627)
//   - doBlockingBandDamage (combat.go:550)
//
// XMage's CombatGroup.attackerDamage/blockerDamage prompts the controlling
// player (per attackerAssignsCombatDamage / defenderAssignsCombatDamage) for
// a free distribution via getMultiAmountWithIndividualConstraints, ignoring
// the lethal-first ordering required by CR 510.1c.

// TestBandingXVal_AttackerCanFreelyAssignDamageToBlockers exercises CR 702.22:
// when the attacking band contains a creature with banding, the attacking
// player chooses how the band's combined combat damage is distributed across
// blockers, ignoring the assignment-order/lethal-first rule of CR 510.1c.
//
// Setup: a 3/3 + 1/1 band (Hill Giant + Benalish Hero, banding) — total power
// 4 — attacks. Two PlayerB blockers each block one band member: a Camel (0/1)
// and a Craw Wurm (6/4). The attacker assigns all 4 damage to Craw Wurm.
//
// Expected (XMage): Camel takes 0, lives; Craw Wurm takes 4 = lethal, dies.
// Current (mg-oracle): doBandedAttackDamage greedily fills blockers in order
// (combat.go:705-722) without consulting CombatDamageAssigner — so Camel
// takes 1 (lethal), Craw Wurm takes 3 (survives). The scripted assignment
// is silently ignored.
func TestBandingXVal_AttackerCanFreelyAssignDamageToBlockers(t *testing.T) {
	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Hill Giant")
	g.AddCard(core.ZoneBattlefield, PlayerA, "Benalish Hero")
	g.AddCard(core.ZoneBattlefield, PlayerB, "Camel")
	g.AddCard(core.ZoneBattlefield, PlayerB, "Craw Wurm")
	g.FormBand(1, PlayerA, "Hill Giant", "Benalish Hero")
	g.Attack(1, PlayerA, "Hill Giant", "Benalish Hero")
	g.Block(1, PlayerB, "Camel", "Hill Giant")
	g.Block(1, PlayerB, "Craw Wurm", "Benalish Hero")

	// Attacker (controller of the banding band) assigns all 4 damage to Craw
	// Wurm. Per CR 702.22 this is legal because banding suppresses the
	// lethal-first ordering for the attacking band's outgoing damage.
	g.AssignCombatDamage("Hill Giant", map[string]int{
		"Camel":     0,
		"Craw Wurm": 4,
	})
	// Distribute the blockers' incoming damage among band members so the
	// step doesn't crash / default. (Hill Giant absorbs all 6 from Craw Wurm.)
	g.ChooseBandingDistribution(PlayerA, map[string]int{
		"Hill Giant":    6,
		"Benalish Hero": 0,
	})
	g.StopAt(1, core.PostcombatMain)
	g.Execute()

	g.AssertGraveyardCount(PlayerB, "Craw Wurm", 1)
	g.AssertPermanentCount(PlayerB, "Camel", 1)
}

// TestBandingXVal_DefenderCanFreelyAssignDamageToAttackers exercises the
// other direction of CR 702.22: when blockers form a blocking band (any
// blocker has banding), the *defending* player chooses how the attacker's
// damage is distributed among blockers, ignoring lethal-first.
//
// Setup: PlayerA attacks with a 5/5 (Sedge Troll-style: use Hill Giant 3/3
// instead since available). Use Craw Wurm 6/4 attacker. Two PlayerB blockers,
// at least one with banding: Benalish Hero (1/1, banding) and Grizzly Bears
// (2/2). Defender assigns 6 attack damage all to Grizzly Bears.
//
// Expected (XMage): Bears take 6 (lethal, dead), Hero takes 0 (lives).
// Current (mg-oracle): doBlockingBandDamage *does* consult
// BandingDamageDistributor (combat.go:571), so this should already work —
// included as a regression / control test alongside the attacker-side bug.
func TestBandingXVal_DefenderCanFreelyAssignDamageToAttackers(t *testing.T) {
	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Craw Wurm")
	g.AddCard(core.ZoneBattlefield, PlayerB, "Benalish Hero")
	g.AddCard(core.ZoneBattlefield, PlayerB, "Grizzly Bears")
	g.Attack(1, PlayerA, "Craw Wurm")
	g.Block(1, PlayerB, "Benalish Hero", "Craw Wurm")
	g.Block(1, PlayerB, "Grizzly Bears", "Craw Wurm")

	// Blocking band has banding (Benalish Hero) — defender chooses
	// distribution of the attacker's 6 power among blockers.
	g.ChooseBandingDistribution(PlayerB, map[string]int{
		"Benalish Hero": 0,
		"Grizzly Bears": 6,
	})
	g.StopAt(1, core.PostcombatMain)
	g.Execute()

	g.AssertGraveyardCount(PlayerB, "Grizzly Bears", 1)
	g.AssertPermanentCount(PlayerB, "Benalish Hero", 1)
	// Attacker takes 1+2 = 3 damage (each blocker still deals its own damage),
	// not enough to kill the 4-toughness Wurm.
	g.AssertPermanentCount(PlayerA, "Craw Wurm", 1)
}
