package gametest

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// A blocker without banding keeps the choice with the attacking player.
// Each source can assign all its damage to the same blocker.
func TestBandingXVal_AttackerCanFreelyAssignDamageToBlockers(t *testing.T) {
	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Hill Giant")
	g.AddCard(core.ZoneBattlefield, PlayerA, "Benalish Hero")
	g.AddCard(core.ZoneBattlefield, PlayerB, "Wall of Wood")
	g.AddCard(core.ZoneBattlefield, PlayerB, "Craw Wurm")
	g.FormBand(1, PlayerA, "Hill Giant", "Benalish Hero")
	g.Attack(1, PlayerA, "Hill Giant", "Benalish Hero")
	g.Block(1, PlayerB, "Wall of Wood", "Hill Giant")
	g.Block(1, PlayerB, "Craw Wurm", "Benalish Hero")

	g.AssignCombatDamage("Hill Giant", map[string]int{"Craw Wurm": 3})
	g.AssignCombatDamage("Benalish Hero", map[string]int{"Craw Wurm": 1})
	g.ChooseBandingDistribution(PlayerA, map[string]int{
		"Hill Giant":    6,
		"Benalish Hero": 0,
	})
	g.StopAt(1, core.PostcombatMain)
	g.Execute()

	g.AssertGraveyardCount(PlayerB, "Craw Wurm", 1)
	g.AssertPermanentCount(PlayerB, "Wall of Wood", 1)
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
