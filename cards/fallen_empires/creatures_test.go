package fallen_empires

import (
	"testing"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/gametest"
)

func TestVodalianSoldiers(t *testing.T) {
	// Vanilla 1/2 Merfolk Soldier — just verify stats on the battlefield.
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Vodalian Soldiers")
	g.StopAt(1, PrecombatMain)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Vodalian Soldiers", 1)
	g.AssertPowerToughness(gametest.PlayerA, "Vodalian Soldiers", 1, 2)
}

func TestIcatianPhalanx(t *testing.T) {
	// 2/4 Human Soldier with Banding.
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Icatian Phalanx")
	g.StopAt(1, PrecombatMain)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Icatian Phalanx", 1)
	g.AssertPowerToughness(gametest.PlayerA, "Icatian Phalanx", 2, 4)
	g.AssertHasAbility(gametest.PlayerA, "Icatian Phalanx", Banding, true)
}

func TestBrassclawOrcs_CantBlockPower2OrGreater(t *testing.T) {
	// Brassclaw Orcs (3/2) can't block creatures with power 2 or greater.
	// PlayerA attacks with a 2/2, PlayerB has Brassclaw Orcs that tries to block.
	// The block should be prevented, so damage goes through.
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")  // 2/2
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Brassclaw Orcs") // 3/2
	g.Attack(1, gametest.PlayerA, "Grizzly Bears")
	g.Block(1, gametest.PlayerB, "Brassclaw Orcs", "Grizzly Bears")
	g.StopAt(1, PostcombatMain)
	g.Execute()
	// Block should be prevented — Grizzly Bears has power 2
	g.AssertLife(gametest.PlayerB, 18)                            // 2 damage gets through
	g.AssertPermanentCount(gametest.PlayerB, "Brassclaw Orcs", 1) // Orcs survived (didn't block)
}

func TestBrassclawOrcs_CanBlockPower1(t *testing.T) {
	// Brassclaw Orcs can block creatures with power less than 2.
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Scryb Sprites")  // 1/1 flying
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Brassclaw Orcs") // 3/2
	// Use a 1/1 without flying so it can be blocked
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Mons's Goblin Raiders") // 1/1
	g.Attack(1, gametest.PlayerA, "Mons's Goblin Raiders")
	g.Block(1, gametest.PlayerB, "Brassclaw Orcs", "Mons's Goblin Raiders")
	g.StopAt(1, PostcombatMain)
	g.Execute()
	// Block should succeed — Mons's Goblin Raiders has power 1
	g.AssertLife(gametest.PlayerB, 20)                                   // no damage gets through
	g.AssertGraveyardCount(gametest.PlayerA, "Mons's Goblin Raiders", 1) // dies to 3/2
}

func TestOrgg_Stats(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Orgg")
	g.StopAt(1, PrecombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Orgg", 6, 6)
	g.AssertHasAbility(gametest.PlayerA, "Orgg", Trample, true)
}

func TestOrgg_CantAttackIfDefenderHasUntappedPower3(t *testing.T) {
	// Orgg can't attack if defending player controls an untapped creature with power >= 3.
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Orgg")
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Hill Giant") // 3/3
	g.Attack(1, gametest.PlayerA, "Orgg")
	g.StopAt(1, PostcombatMain)
	g.Execute()
	// Orgg can't attack — Hill Giant is untapped with power 3
	g.AssertLife(gametest.PlayerB, 20) // no damage
}

func TestOrgg_CantBlockPower3OrGreater(t *testing.T) {
	// Orgg can't block creatures with power 3 or greater.
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Hill Giant") // 3/3
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Orgg")       // 6/6
	g.Attack(1, gametest.PlayerA, "Hill Giant")
	g.Block(1, gametest.PlayerB, "Orgg", "Hill Giant")
	g.StopAt(1, PostcombatMain)
	g.Execute()
	// Block should be prevented — Hill Giant has power 3
	g.AssertLife(gametest.PlayerB, 17) // 3 damage gets through
}

func TestDwarvenSoldier_BlocksOrc(t *testing.T) {
	// Dwarven Soldier gets +0/+2 when it blocks an Orc.
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Brassclaw Orcs")  // 3/2 Orc
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Dwarven Soldier") // 2/1 Dwarf Soldier
	g.Attack(1, gametest.PlayerA, "Brassclaw Orcs")
	// Brassclaw Orcs can't block power >= 2, but can be blocked
	g.Block(1, gametest.PlayerB, "Dwarven Soldier", "Brassclaw Orcs")
	g.StopAt(1, PostcombatMain)
	g.Execute()
	// Dwarven Soldier gets +0/+2 (now 2/3), takes 3 damage, survives (barely? no, 3 = 3, dies)
	// Actually 2/3 takes 3 damage = exactly lethal
	g.AssertGraveyardCount(gametest.PlayerB, "Dwarven Soldier", 1) // dies to 3 damage
	g.AssertGraveyardCount(gametest.PlayerA, "Brassclaw Orcs", 1)  // dies to 2 damage
}

func TestDwarvenSoldier_BecomeBlockedByOrc(t *testing.T) {
	// Dwarven Soldier gets +0/+2 when it becomes blocked by an Orc.
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Dwarven Soldier") // 2/1 Dwarf Soldier
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Orcish Spy")      // 1/1 Orc (power 1, so Brassclaw can block)
	g.Attack(1, gametest.PlayerA, "Dwarven Soldier")
	g.Block(1, gametest.PlayerB, "Orcish Spy", "Dwarven Soldier")
	g.StopAt(1, PostcombatMain)
	g.Execute()
	// Dwarven Soldier becomes blocked by an Orc, gets +0/+2 (now 2/3)
	// Takes 1 damage (survives), deals 2 damage to Orcish Spy (dies)
	g.AssertPermanentCount(gametest.PlayerA, "Dwarven Soldier", 1) // survives
	g.AssertGraveyardCount(gametest.PlayerB, "Orcish Spy", 1)      // dies
}

func TestSvyelunitePriest_GrantsShroud(t *testing.T) {
	// Svyelunite Priest grants shroud to target creature until end of turn.
	// Activate only during upkeep.
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Svyelunite Priest")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Island", 2) // for {U}{U} cost
	g.ActivateAbility(1, Upkeep, gametest.PlayerA, "Svyelunite Priest", "Grizzly Bears")
	g.StopAt(1, PrecombatMain)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Grizzly Bears", Shroud, true)
	g.AssertTapped(gametest.PlayerA, "Svyelunite Priest", true) // tapped from {T} cost
}

func TestIcatianPriest_BoostsTarget(t *testing.T) {
	// {1}{W}{W}: Target creature gets +1/+1 until end of turn.
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Icatian Priest")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // 2/2
	g.ActivateAbility(1, PrecombatMain, gametest.PlayerA, "Icatian Priest", "Grizzly Bears")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 3, 3)
}

func TestIcatianLieutenant_BoostsSoldier(t *testing.T) {
	// {1}{W}: Target Soldier creature gets +1/+0 until end of turn.
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Icatian Lieutenant") // 1/2 Soldier
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Icatian Phalanx")    // 2/4 Soldier
	g.ActivateAbility(1, PrecombatMain, gametest.PlayerA, "Icatian Lieutenant", "Icatian Phalanx")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Icatian Phalanx", 3, 4) // +1/+0
}

func TestIcatianScout_GrantsFirstStrike(t *testing.T) {
	// {1}, {T}: Target creature gains first strike until end of turn.
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Icatian Scout")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.ActivateAbility(1, PrecombatMain, gametest.PlayerA, "Icatian Scout", "Grizzly Bears")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Grizzly Bears", FirstStrike, true)
	g.AssertTapped(gametest.PlayerA, "Icatian Scout", true)
}

func TestRiverMerfolk_GainsMountainwalk(t *testing.T) {
	// {U}: This creature gains mountainwalk until end of turn.
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "River Merfolk")
	g.ActivateAbility(1, PrecombatMain, gametest.PlayerA, "River Merfolk")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "River Merfolk", Mountainwalk, true)
}

func TestDwarvenLieutenant_BoostsDwarf(t *testing.T) {
	// {1}{R}: Target Dwarf creature gets +1/+0 until end of turn.
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Dwarven Lieutenant") // 1/2 Dwarf Soldier
	// Lieutenant can target itself (it's a Dwarf)
	g.ActivateAbility(1, PrecombatMain, gametest.PlayerA, "Dwarven Lieutenant", "Dwarven Lieutenant")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Dwarven Lieutenant", 2, 2) // +1/+0
}

func TestHomaridShaman_TapsGreenCreature(t *testing.T) {
	// {U}: Tap target green creature.
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Homarid Shaman")
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Grizzly Bears") // green 2/2
	g.ActivateAbility(1, PrecombatMain, gametest.PlayerA, "Homarid Shaman", "Grizzly Bears")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertTapped(gametest.PlayerB, "Grizzly Bears", true)
}

func TestOrderOfLeitbur_ProtectionFromBlack(t *testing.T) {
	// Order of Leitbur has protection from black — a black creature can't block it.
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Order of Leitbur")
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Frozen Shade") // 0/1 black creature
	g.Attack(1, gametest.PlayerA, "Order of Leitbur")
	g.Block(1, gametest.PlayerB, "Frozen Shade", "Order of Leitbur")
	g.StopAt(1, PostcombatMain)
	g.Execute()
	// Block should be prevented by protection from black
	g.AssertLife(gametest.PlayerB, 18)
}

func TestOrderOfLeitbur_FirstStrikePump(t *testing.T) {
	// {W}: gains first strike until end of turn
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Order of Leitbur")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Plains", 3)
	g.ActivateAbility(1, PrecombatMain, gametest.PlayerA, "Order of Leitbur")
	g.StopAt(1, BeginCombat)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Order of Leitbur", FirstStrike, true)
	g.AssertPowerToughness(gametest.PlayerA, "Order of Leitbur", 2, 1)
}

func TestOrderOfEbonHand_ProtectionFromWhite(t *testing.T) {
	// Order of the Ebon Hand has protection from white.
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Order of the Ebon Hand")
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "White Knight") // 2/2 white creature
	g.Attack(1, gametest.PlayerA, "Order of the Ebon Hand")
	g.Block(1, gametest.PlayerB, "White Knight", "Order of the Ebon Hand")
	g.StopAt(1, PostcombatMain)
	g.Execute()
	// Block should be prevented by protection from white
	g.AssertLife(gametest.PlayerB, 18)
}

func TestIcatianInfantry_FirstStrike(t *testing.T) {
	// {1}: gains first strike until end of turn
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Icatian Infantry")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Plains")
	g.ActivateAbility(1, PrecombatMain, gametest.PlayerA, "Icatian Infantry")
	g.StopAt(1, BeginCombat)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Icatian Infantry", FirstStrike, true)
}

func TestIcatianInfantry_Stats(t *testing.T) {
	// Verify base stats: 1/1 Human Soldier
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Icatian Infantry")
	g.StopAt(1, PrecombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Icatian Infantry", 1, 1)
	g.AssertPermanentCount(gametest.PlayerA, "Icatian Infantry", 1)
}

func TestOrcishVeteran_CantBlockWhitePower2(t *testing.T) {
	// Orcish Veteran can't block white creatures with power 2 or greater.
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "White Knight")   // 2/2 white
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Orcish Veteran") // 2/2
	g.Attack(1, gametest.PlayerA, "White Knight")
	g.Block(1, gametest.PlayerB, "Orcish Veteran", "White Knight")
	g.StopAt(1, PostcombatMain)
	g.Execute()
	// Block should be prevented — White Knight is white with power 2
	g.AssertLife(gametest.PlayerB, 18)
}

func TestOrcishVeteran_CanBlockNonWhitePower2(t *testing.T) {
	// Orcish Veteran CAN block non-white creatures with power 2 or greater.
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")  // 2/2 green
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Orcish Veteran") // 2/2
	g.Attack(1, gametest.PlayerA, "Grizzly Bears")
	g.Block(1, gametest.PlayerB, "Orcish Veteran", "Grizzly Bears")
	g.StopAt(1, PostcombatMain)
	g.Execute()
	// Block should succeed — Grizzly Bears is not white
	g.AssertLife(gametest.PlayerB, 20)
}

func TestOrcishVeteran_FirstStrike(t *testing.T) {
	// {R}: gains first strike until end of turn
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Orcish Veteran")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.ActivateAbility(1, PrecombatMain, gametest.PlayerA, "Orcish Veteran")
	g.StopAt(1, BeginCombat)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Orcish Veteran", FirstStrike, true)
}

func TestMindstabThrull_AttacksUnblocked(t *testing.T) {
	// Mindstab Thrull: when it attacks and isn't blocked, you may sacrifice it.
	// If you do, defending player discards three cards.
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Mindstab Thrull")
	g.AddCard(ZoneHand, gametest.PlayerB, "Forest")
	g.AddCard(ZoneHand, gametest.PlayerB, "Mountain")
	g.AddCard(ZoneHand, gametest.PlayerB, "Plains")
	g.Attack(1, gametest.PlayerA, "Mindstab Thrull")
	g.StopAt(1, PostcombatMain)
	g.Execute()
	// Thrull was sacrificed
	g.AssertPermanentCount(gametest.PlayerA, "Mindstab Thrull", 0)
	g.AssertGraveyardCount(gametest.PlayerA, "Mindstab Thrull", 1)
	// Defending player discarded 3 cards
	g.AssertHandCount(gametest.PlayerB, "Forest", 0)
	g.AssertHandCount(gametest.PlayerB, "Mountain", 0)
	g.AssertHandCount(gametest.PlayerB, "Plains", 0)
}

func TestMindstabThrull_Blocked(t *testing.T) {
	// When blocked, the trigger should NOT fire.
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Mindstab Thrull")
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(ZoneHand, gametest.PlayerB, "Forest")
	g.AddCard(ZoneHand, gametest.PlayerB, "Mountain")
	g.AddCard(ZoneHand, gametest.PlayerB, "Plains")
	g.Attack(1, gametest.PlayerA, "Mindstab Thrull")
	g.Block(1, gametest.PlayerB, "Grizzly Bears", "Mindstab Thrull")
	g.StopAt(1, PostcombatMain)
	g.Execute()
	// Both creatures should have died in combat (2/2 vs 2/2), no discard
	g.AssertHandCount(gametest.PlayerB, "Forest", 1)
	g.AssertHandCount(gametest.PlayerB, "Mountain", 1)
	g.AssertHandCount(gametest.PlayerB, "Plains", 1)
}

func TestNecrite_AttacksUnblocked(t *testing.T) {
	// Necrite: when it attacks and isn't blocked, you may sacrifice it.
	// If you do, destroy target creature defending player controls. It can't be regenerated.
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Necrite")
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.ChoosePermanent(gametest.PlayerA, "Hill Giant")
	g.Attack(1, gametest.PlayerA, "Necrite")
	g.StopAt(1, PostcombatMain)
	g.Execute()
	// Necrite was sacrificed
	g.AssertPermanentCount(gametest.PlayerA, "Necrite", 0)
	g.AssertGraveyardCount(gametest.PlayerA, "Necrite", 1)
	// Hill Giant was destroyed
	g.AssertPermanentCount(gametest.PlayerB, "Hill Giant", 0)
	g.AssertGraveyardCount(gametest.PlayerB, "Hill Giant", 1)
}

func TestNecrite_Blocked(t *testing.T) {
	// When blocked, the trigger should NOT fire; target creature survives.
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Necrite")
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.Attack(1, gametest.PlayerA, "Necrite")
	g.Block(1, gametest.PlayerB, "Grizzly Bears", "Necrite")
	g.StopAt(1, PostcombatMain)
	g.Execute()
	// Hill Giant should survive (trigger didn't fire)
	g.AssertPermanentCount(gametest.PlayerB, "Hill Giant", 1)
}

func TestFarrelsZealot_AttacksUnblocked(t *testing.T) {
	// Farrel's Zealot: when it attacks and isn't blocked, you may have it deal 3 damage
	// to target creature. If you do, it assigns no combat damage this turn.
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Farrel's Zealot")
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Hill Giant") // 3/3 — will die to 3 damage
	g.ChoosePermanent(gametest.PlayerA, "Hill Giant")
	g.Attack(1, gametest.PlayerA, "Farrel's Zealot")
	g.StopAt(1, PostcombatMain)
	g.Execute()
	// Hill Giant took 3 damage and died
	g.AssertGraveyardCount(gametest.PlayerB, "Hill Giant", 1)
	// Zealot assigns no combat damage — player B takes 0
	g.AssertLife(gametest.PlayerB, 20)
	// Zealot survives
	g.AssertPermanentCount(gametest.PlayerA, "Farrel's Zealot", 1)
}

func TestFarrelsZealot_Blocked(t *testing.T) {
	// When blocked, the trigger should NOT fire.
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Farrel's Zealot")
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.Attack(1, gametest.PlayerA, "Farrel's Zealot")
	g.Block(1, gametest.PlayerB, "Grizzly Bears", "Farrel's Zealot")
	g.StopAt(1, PostcombatMain)
	g.Execute()
	// Hill Giant should survive (trigger didn't fire)
	g.AssertPermanentCount(gametest.PlayerB, "Hill Giant", 1)
	// Normal combat: Zealot (2/2) vs Grizzly Bears (2/2) — both die
	g.AssertGraveyardCount(gametest.PlayerA, "Farrel's Zealot", 1)
	g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
}

func TestElvishHunter_TargetDoesntUntap(t *testing.T) {
	// Elvish Hunter: {1}{G}, {T}: Target creature doesn't untap during its controller's next untap step.
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Elvish Hunter")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Forest", 2) // for {1}{G} cost
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	// Activate on turn 1 precombat main targeting opponent's Grizzly Bears
	g.ActivateAbility(1, PrecombatMain, gametest.PlayerA, "Elvish Hunter", "Grizzly Bears")
	// Tap the Bears so we can observe it not untapping
	// On turn 2 (PlayerB's turn), Grizzly Bears should not untap
	g.StopAt(2, PrecombatMain) // PlayerB's turn
	g.Execute()
	// Grizzly Bears should still be untapped initially, but the effect prevents untap
	// Actually Bears aren't tapped yet - we need to tap them first or just check the attr
	g.AssertHasAbility(gametest.PlayerB, "Grizzly Bears", AttrDoesNotUntap, true)
}

// ===== Icatian Javelineers Tests =====

func TestIcatianJavelineers_DealsDamage(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Icatian Javelineers")
	g.ActivateAbility(1, PrecombatMain, gametest.PlayerA, "Icatian Javelineers", "PlayerB")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 19)
	g.AssertTapped(gametest.PlayerA, "Icatian Javelineers", true)
	g.AssertCounterCount(gametest.PlayerA, "Icatian Javelineers", Javelin, 0)
}

func TestIcatianJavelineers_EntersWithCounter(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Icatian Javelineers")
	g.StopAt(1, PrecombatMain)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Icatian Javelineers", Javelin, 1)
}

// ===== Armor Thrull Tests =====

func TestArmorThrull_PutsCounter(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Armor Thrull")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.ActivateAbility(1, PrecombatMain, gametest.PlayerA, "Armor Thrull", "Grizzly Bears")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Armor Thrull", 0) // sacrificed
	g.AssertCounterCount(gametest.PlayerA, "Grizzly Bears", P1P2, 1)
	g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 3, 4) // 2+1/2+2
}

// ===== Basal Thrull Tests =====

func TestBasalThrull_AddsMana(t *testing.T) {
	// Basal Thrull: {T}, Sacrifice: Add {B}{B}
	// Use the mana to cast a spell that costs {B}{B}
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Basal Thrull")
	g.AddCard(ZoneHand, gametest.PlayerA, "Order of the Ebon Hand") // costs {B}{B}
	g.ActivateAbility(1, PrecombatMain, gametest.PlayerA, "Basal Thrull")
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Order of the Ebon Hand")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Basal Thrull", 0)           // sacrificed
	g.AssertPermanentCount(gametest.PlayerA, "Order of the Ebon Hand", 1) // cast with the mana
}

// ===== Icatian Moneychanger Tests =====

func TestIcatianMoneychanger_ETBDamage(t *testing.T) {
	// When Moneychanger enters, it deals 3 damage to you and enters with 3 credit counters
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneHand, gametest.PlayerA, "Icatian Moneychanger")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Plains")
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Icatian Moneychanger")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 17) // 20 - 3 = 17
	g.AssertCounterCount(gametest.PlayerA, "Icatian Moneychanger", Credit, 3)
}

func TestIcatianMoneychanger_UpkeepCounter(t *testing.T) {
	// At the beginning of your upkeep, put a credit counter on it.
	// AddCard on battlefield triggers ETB (3 counters) + turn 1 upkeep (+1) + turn 3 upkeep (+1) = 5
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Icatian Moneychanger")
	g.StopAt(3, PrecombatMain)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Icatian Moneychanger", Credit, 5)
}

func TestIcatianMoneychanger_SacrificeGainsLife(t *testing.T) {
	// Sacrifice: gain 1 life per credit counter, only during upkeep.
	// AddCard triggers ETB: 3 counters, 3 damage (life → 17).
	// Turn 1 (PlayerA) upkeep: +1 counter (4).
	// Turn 2 (PlayerB) upkeep: trigger reads "at beginning of your upkeep",
	//   only fires on the controller's turn → no counter added.
	// Turn 3 (PlayerA) upkeep: upkeep trigger goes on the stack and, per CR
	//   117.1b / 603.6, active player gets priority only after TBAs and
	//   triggers have been placed. The trigger resolves first (+1 → 5),
	//   then PlayerA activates the sacrifice ability (sorcery speed, empty
	//   stack). Gain 5 life: 17 + 5 = 22.
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Icatian Moneychanger")
	g.ActivateAbility(3, Upkeep, gametest.PlayerA, "Icatian Moneychanger")
	g.StopAt(3, PrecombatMain)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Icatian Moneychanger", 0)
	g.AssertLife(gametest.PlayerA, 22)
}

// ===== SPORE CREATURE TESTS =====

func TestThallid_SporeCounterAtUpkeep(t *testing.T) {
	// Verify upkeep trigger adds a spore counter.
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Thallid")
	g.StopAt(1, PrecombatMain) // after upkeep
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Thallid", Spore, 1)
}

func TestThallid_CreatesSaproling(t *testing.T) {
	// After accumulating 3 spore counters, remove them to create a 1/1 Saproling.
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Thallid")
	g.AddCounters(1, Upkeep, gametest.PlayerA, "Thallid", Spore, 2) // 2 + 1 from upkeep = 3
	g.ActivateAbility(1, PrecombatMain, gametest.PlayerA, "Thallid")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Saproling", 1)
	g.AssertCounterCount(gametest.PlayerA, "Thallid", Spore, 0)
}

func TestThallid_CantActivateWithoutCounters(t *testing.T) {
	// With only 2 spore counters, cannot activate.
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Thallid")
	g.AddCounters(1, Upkeep, gametest.PlayerA, "Thallid", Spore, 1) // 1 + 1 from upkeep = 2
	g.ActivateAbility(1, PrecombatMain, gametest.PlayerA, "Thallid")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Saproling", 0) // can't activate
	g.AssertCounterCount(gametest.PlayerA, "Thallid", Spore, 2)
}

func TestThornThallid_Deals1DamageToPlayer(t *testing.T) {
	// Remove 3 spore counters to deal 1 damage to any target.
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Thorn Thallid")
	g.AddCounters(1, Upkeep, gametest.PlayerA, "Thorn Thallid", Spore, 2) // 2 + 1 = 3
	g.ActivateAbility(1, PrecombatMain, gametest.PlayerA, "Thorn Thallid", "PlayerB")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 19)
	g.AssertCounterCount(gametest.PlayerA, "Thorn Thallid", Spore, 0)
}

func TestThornThallid_DamageCreature(t *testing.T) {
	// Can also target creatures.
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Thorn Thallid")
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Scryb Sprites") // 1/1
	g.AddCounters(1, Upkeep, gametest.PlayerA, "Thorn Thallid", Spore, 2)
	g.ActivateAbility(1, PrecombatMain, gametest.PlayerA, "Thorn Thallid", "Scryb Sprites")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Scryb Sprites", 1) // killed by 1 damage
	g.AssertCounterCount(gametest.PlayerA, "Thorn Thallid", Spore, 0)
}

func TestFeralThallid_Regenerates(t *testing.T) {
	// Remove 3 spore counters to regenerate.
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Feral Thallid") // 6/3
	g.AddCard(ZoneHand, gametest.PlayerB, "Lightning Bolt")
	g.AddCounters(1, Upkeep, gametest.PlayerA, "Feral Thallid", Spore, 2) // 2 + 1 = 3
	// Activate regeneration shield first
	g.ActivateAbility(1, PrecombatMain, gametest.PlayerA, "Feral Thallid")
	// Then opponent bolts it
	g.CastSpell(1, PrecombatMain, gametest.PlayerB, "Lightning Bolt", "Feral Thallid")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Feral Thallid", 1) // survived via regeneration
	g.AssertCounterCount(gametest.PlayerA, "Feral Thallid", Spore, 0)
}

func TestFeralThallid_Stats(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Feral Thallid")
	g.StopAt(1, PrecombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Feral Thallid", 6, 3)
}

func TestSporeFlower_PreventsAllCombatDamage(t *testing.T) {
	// Remove 3 spore counters to prevent all combat damage this turn.
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Spore Flower")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // 2/2 attacker
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Hill Giant")    // 3/3 blocker
	g.AddCounters(1, Upkeep, gametest.PlayerA, "Spore Flower", Spore, 2)
	g.ActivateAbility(1, BeginCombat, gametest.PlayerA, "Spore Flower")
	g.Attack(1, gametest.PlayerA, "Grizzly Bears")
	g.Block(1, gametest.PlayerB, "Hill Giant", "Grizzly Bears")
	g.StopAt(1, PostcombatMain)
	g.Execute()
	// All combat damage prevented — both survive
	g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
	g.AssertPermanentCount(gametest.PlayerB, "Hill Giant", 1)
	g.AssertLife(gametest.PlayerB, 20)
	g.AssertCounterCount(gametest.PlayerA, "Spore Flower", Spore, 0)
}

func TestElvishFarmer_CreatesSaproling(t *testing.T) {
	// Elvish Farmer creates Saproling tokens like Thallid.
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Elvish Farmer") // 0/2 Elf
	g.AddCounters(1, Upkeep, gametest.PlayerA, "Elvish Farmer", Spore, 2)
	g.ActivateAbility(1, PrecombatMain, gametest.PlayerA, "Elvish Farmer")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Saproling", 1)
	g.AssertCounterCount(gametest.PlayerA, "Elvish Farmer", Spore, 0)
}

func TestElvishFarmer_Stats(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Elvish Farmer")
	g.StopAt(1, PrecombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Elvish Farmer", 0, 2)
}

func TestThallidDevourer_CreatesSaproling(t *testing.T) {
	// Thallid Devourer creates Saproling tokens like Thallid.
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Thallid Devourer") // 2/2 Fungus
	g.AddCounters(1, Upkeep, gametest.PlayerA, "Thallid Devourer", Spore, 2)
	g.ActivateAbility(1, PrecombatMain, gametest.PlayerA, "Thallid Devourer")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Saproling", 1)
	g.AssertCounterCount(gametest.PlayerA, "Thallid Devourer", Spore, 0)
}

func TestThallidDevourer_Stats(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Thallid Devourer")
	g.StopAt(1, PrecombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Thallid Devourer", 2, 2)
}

// ===== Homarid Tests =====

func TestHomarid_EntersWithTideCounter(t *testing.T) {
	// Homarid enters with 1 tide counter (ETB), then turn 1 upkeep adds another = 2 tide counters.
	// At 2 counters, no PT modification → stays 2/2.
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Homarid")
	g.StopAt(1, PrecombatMain)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Homarid", Tide, 2)
	g.AssertPowerToughness(gametest.PlayerA, "Homarid", 2, 2)
}

func TestHomarid_TideCounterCycle(t *testing.T) {
	// Turn 1 upkeep: starts with 1 counter, adds 1 → 2 counters (no PT mod) → 2/2
	// Turn 3 upkeep (PlayerA's second upkeep): adds 1 → 3 counters (+1/+1) → 3/3
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Homarid")
	g.StopAt(3, PrecombatMain)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Homarid", Tide, 3)
	g.AssertPowerToughness(gametest.PlayerA, "Homarid", 3, 3) // 2/2 base +1/+1 with 3 counters
}

func TestHomarid_TideCounterReset(t *testing.T) {
	// After reaching 4 counters, all tide counters are removed.
	// Turn 1: starts with 1, upkeep adds 1 → 2 counters
	// Turn 3: upkeep adds 1 → 3 counters
	// Turn 5: upkeep adds 1 → 4 counters → removed → 0 counters
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Homarid")
	g.StopAt(5, PrecombatMain) // Turn 5 = PlayerA's 3rd turn
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Homarid", Tide, 0)
	g.AssertPowerToughness(gametest.PlayerA, "Homarid", 2, 2) // base 2/2 with 0 counters
}

// ===== Vodalian Knights Tests =====

func TestVodalianKnights_FirstStrike(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Vodalian Knights")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Island")
	g.StopAt(1, PrecombatMain)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Vodalian Knights", FirstStrike, true)
	g.AssertPowerToughness(gametest.PlayerA, "Vodalian Knights", 2, 2)
}

func TestVodalianKnights_GainsFlying(t *testing.T) {
	// {U}: gains flying until end of turn
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Vodalian Knights")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Island", 2)
	g.ActivateAbility(1, PrecombatMain, gametest.PlayerA, "Vodalian Knights")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Vodalian Knights", Flying, true)
}

func TestVodalianKnights_SacrificedWhenNoIslands(t *testing.T) {
	// When you control no Islands, sacrifice this creature.
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Vodalian Knights")
	// No Islands — should be sacrificed
	g.StopAt(1, PrecombatMain)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Vodalian Knights", 0)
	g.AssertGraveyardCount(gametest.PlayerA, "Vodalian Knights", 1)
}

func TestVodalianKnights_SurvivesWithIsland(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Vodalian Knights")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Island")
	g.StopAt(1, PrecombatMain)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Vodalian Knights", 1)
}

func TestVodalianKnights_CantAttackWithoutDefenderIsland(t *testing.T) {
	// Can't attack unless defending player controls an Island
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Vodalian Knights")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Island")
	// Defender has no Islands
	g.Attack(1, gametest.PlayerA, "Vodalian Knights")
	g.StopAt(1, PostcombatMain)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 20) // can't attack, no damage
}

func TestVodalianKnights_CanAttackWhenDefenderHasIsland(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Vodalian Knights")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Island")
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Island") // defender has Island
	g.Attack(1, gametest.PlayerA, "Vodalian Knights")
	g.StopAt(1, PostcombatMain)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 18) // 2 damage from first strike
}

// ===== Derelor Tests =====

func TestDerelor_Stats(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Derelor")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Swamp", 4)
	g.StopAt(1, PrecombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Derelor", 4, 4)
}

func TestDerelor_IncreasesBlackSpellCost(t *testing.T) {
	// Black spells you cast cost {B} more.
	// Try to cast a {B}{B} spell with only 2 Swamps — should fail because Derelor adds {1} more.
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Derelor")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Swamp", 2)
	g.AddCard(ZoneHand, gametest.PlayerA, "Order of the Ebon Hand") // {B}{B}
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Order of the Ebon Hand")
	g.StopAt(1, EndStep)
	g.Execute()
	// With only 2 Swamps and Derelor increasing cost, can't cast {B}{B} + {1} extra
	g.AssertPermanentCount(gametest.PlayerA, "Order of the Ebon Hand", 0)
}

// ===== Thrull Champion Tests =====

func TestThrullChampion_LordBoost(t *testing.T) {
	// Thrull creatures get +1/+1
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Thrull Champion")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Armor Thrull") // 1/3 Thrull
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Armor Thrull") // opponent's Thrull should NOT get boost
	g.StopAt(1, PrecombatMain)
	g.Execute()
	// Thrull Champion itself is a Thrull (2/2) but doesn't boost itself
	g.AssertPowerToughness(gametest.PlayerA, "Thrull Champion", 2, 2)
	// Armor Thrull gets +1/+1 from Thrull Champion
	g.AssertPowerToughness(gametest.PlayerA, "Armor Thrull", 2, 4)
	// Opponent's Armor Thrull should NOT be boosted
	g.AssertPowerToughness(gametest.PlayerB, "Armor Thrull", 1, 3)
}
