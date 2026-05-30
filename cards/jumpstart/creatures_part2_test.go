package jumpstart

import (
	"testing"

	_ "github.com/benprew/mage-go/cards/antiquities"
	_ "github.com/benprew/mage-go/cards/limited"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestCrowOfDarkTidings_MillsOnETBAndDeath(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Crow of Dark Tidings")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 3)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Mox Ruby", 6)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Crow of Dark Tidings")
	g.CastSpell(1, core.PostcombatMain, gametest.PlayerA, "Lightning Bolt", "Crow of Dark Tidings")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerA, "Mox Ruby", 4)
}
func TestDawntreaderElk_TutorsBasicLandTapped(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Dawntreader Elk")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Plains", 1)
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Dawntreader Elk")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Plains", 1)
	g.AssertGraveyardCount(gametest.PlayerA, "Dawntreader Elk", 1)
}
func TestDragonHatchling_BoostsForR(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Dragon Hatchling")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Dragon Hatchling")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Dragon Hatchling", 1, 1)
}
func TestDrainpipeVermin_DiscardOnDeath(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Drainpipe Vermin")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Hill Giant")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "Drainpipe Vermin")
	g.ChooseTarget(gametest.PlayerA, "PlayerB")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerB, "Hill Giant", 0)
	g.AssertGraveyardCount(gametest.PlayerB, "Hill Giant", 1)
}
func TestDranaLiberatorOfMalakir_BoostsAttackersOnHit(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Drana, Liberator of Malakir")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.Attack(3, gametest.PlayerA, "Drana, Liberator of Malakir", "Grizzly Bears")
	g.StopAt(3, core.EndStep)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Drana, Liberator of Malakir", 3, 4)
	g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 3, 3)
}
func TestDroverOfTheMighty_BoostsWhenDinosaur(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Drover of the Mighty")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Orazca Frillback")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Drover of the Mighty", 3, 3)
}
func TestDualcasterMage_CopiesSpellOnStack(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 5)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Dualcaster Mage")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Bathe in Dragonfire")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Bathe in Dragonfire", "Hill Giant")
	g.CastInResponseTo(gametest.PlayerA, "Dualcaster Mage", "Bathe in Dragonfire")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Hill Giant", 1)
	g.AssertPermanentCount(gametest.PlayerA, "Dualcaster Mage", 1)
}
func TestDutifulAttendant_ReturnsOnDeath(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Dutiful Attendant")
	g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "Dutiful Attendant")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
}
func TestDwynensElite_CreatesTokenIfElf(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Dwynen's Elite")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 2)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Leaf Gilder")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Dwynen's Elite")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Elf Warrior", 1)
}
func TestElvishArchdruid_BoostsOtherElves(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Elvish Archdruid")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Leaf Gilder")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Leaf Gilder", 3, 2)
	g.AssertPowerToughness(gametest.PlayerA, "Elvish Archdruid", 2, 2)
}
func TestEmancipationAngel(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Emancipation Angel")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 3)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Emancipation Angel", "Grizzly Bears")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
}
func TestEmielTheBlessed(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Emiel the Blessed")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 3)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.ActivateAbility(3, core.PrecombatMain, gametest.PlayerA, "Emiel the Blessed", "Grizzly Bears")
	g.StopAt(3, core.PostcombatMain)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
}
func TestEmielTheBlessed_NoTriggerOnSelfETB(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Emiel the Blessed")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 4)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Emiel the Blessed")
	g.StopAt(1, core.EndStep)
	g.Execute()
	// Trigger says "another creature you control" — Emiel itself shouldn't trigger.
	g.AssertCounterCount(gametest.PlayerA, "Emiel the Blessed", core.P1P1, 0)
}
func TestEmielTheBlessed_PayCounterOnNonUnicorn(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Emiel the Blessed")
	// One Forest pays the may-pay {G/W} when Grizzly Bears enters.
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Grizzly Bears", core.P1P1, 1)
}
func TestEmielTheBlessed_PayTwoCountersOnUnicorn(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Emiel the Blessed")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 3)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Mesa Unicorn")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Mesa Unicorn")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Mesa Unicorn", core.P1P1, 2)
}
func TestErraticVisionary(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Erratic Visionary")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 2)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Mountain", 10)
	g.ActivateAbility(3, core.PrecombatMain, gametest.PlayerA, "Erratic Visionary")
	g.StopAt(3, core.PostcombatMain)
	g.Execute()
	// Activation drew a Mountain then discarded a card.
	g.AssertGraveyardCount(gametest.PlayerA, "Mountain", 1)
}
func TestEternalTaskmaster_ReturnOnAttack(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Eternal Taskmaster")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 3)
	g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Hill Giant")
	g.Attack(3, gametest.PlayerA, "Eternal Taskmaster")
	g.StopAt(3, core.EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Hill Giant", 1)
}
func TestExclusionMage(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Exclusion Mage")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 3)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Exclusion Mage", "Grizzly Bears")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertHandCount(gametest.PlayerB, "Grizzly Bears", 1)
}
func TestFalkenrathNoble_DrainsOnAnyDeath(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Falkenrath Noble")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "Grizzly Bears")
	g.ChooseTarget(gametest.PlayerA, "PlayerB")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 21)
	g.AssertLife(gametest.PlayerB, 19)
}
func TestFanaticalFirebrand_PingsAndSacs(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Fanatical Firebrand")
	g.SetLife(gametest.PlayerB, 20)
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Fanatical Firebrand", "PlayerB")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 19)
	g.AssertPermanentCount(gametest.PlayerA, "Fanatical Firebrand", 0)
}
func TestFellSpecter_DiscardCausesLifeLoss(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Fell Specter")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Mind Twist")
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Mountain", 2)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 2)
	g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Mind Twist", 1, "PlayerB")
	g.ChooseDiscard(gametest.PlayerB, "Mountain")
	g.StopAt(1, core.EndStep)
	g.Execute()
	// Fell Specter's ETB ("target opponent discards a card") fires when added to
	// the battlefield, then Mind Twist X=1 discards another. Two discards → two
	// trigger fires of "Whenever an opponent discards a card, that player loses 2
	// life" (CR 701.8: each card discarded is its own event) → 4 life lost.
	g.AssertLife(gametest.PlayerB, 16)
}

// Fell Specter: ETB makes target opponent discard.
func TestFellSpecter_ETBDiscard(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Fell Specter")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 4)
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Mountain", 3)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Fell Specter")
	g.ChooseDiscard(gametest.PlayerB, "Mountain")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerB, "Mountain", 2)
	g.AssertGraveyardCount(gametest.PlayerB, "Mountain", 1)
}
func TestFeralHydra_EntersWithXCounters(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Feral Hydra")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 4)
	g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Feral Hydra", 3)
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Feral Hydra", 3, 3)
}
func TestFeralProwler_DrawsOnDeath(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Feral Prowler")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Forest", 5)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Craw Wurm")
	g.Attack(1, gametest.PlayerB, "Craw Wurm")
	// Wait until B's turn
	g.StopAt(2, core.EndStep)
	g.Execute()
}
func TestFertilid_EntersWithCounters(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Fertilid")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Fertilid", 2, 2)
}
func TestFesteringNewt_DebuffOnDeath(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Festering Newt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "Festering Newt")
	g.ChoosePermanent(gametest.PlayerA, "Hill Giant")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerB, "Hill Giant", 2, 2)
}
func TestFlametongueKavu_DealsFourOnETB(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Flametongue Kavu")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 4)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Flametongue Kavu")
	g.ChooseTarget(gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
}
func TestForgeDevil_DealsOneToTargetAndYou(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Forge Devil")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mons's Goblin Raiders")
	g.SetLife(gametest.PlayerA, 20)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Forge Devil")
	g.ChooseTarget(gametest.PlayerA, "Mons's Goblin Raiders")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Mons's Goblin Raiders", 1)
	g.AssertLife(gametest.PlayerA, 19)
}
func TestFurnaceWhelp_BoostsForR(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Furnace Whelp")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Furnace Whelp")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Furnace Whelp", 3, 2)
}
func TestGargoyleSentinel_GainsFlying(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Gargoyle Sentinel")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 3)
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Gargoyle Sentinel")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Gargoyle Sentinel", core.Flying, true)
	g.AssertHasAbility(gametest.PlayerA, "Gargoyle Sentinel", core.Defender, false)
}
func TestGhoulraiser_ReturnsZombieFromGraveyard(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Ghoulraiser")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 3)
	g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Black Cat")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Ghoulraiser")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Black Cat", 1)
}
func TestGiftedAetherborn(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Gifted Aetherborn")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Gifted Aetherborn", core.Deathtouch, true)
	g.AssertHasAbility(gametest.PlayerA, "Gifted Aetherborn", core.Lifelink, true)
}
func TestGingerbrute_HasteAndSacGainsLife(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Gingerbrute")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 3)
	g.SetLife(gametest.PlayerA, 20)
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Gingerbrute")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 23)
	g.AssertGraveyardCount(gametest.PlayerA, "Gingerbrute", 1)
}
func TestGoblinChieftain_BoostsAndHaste(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Goblin Chieftain")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mons's Goblin Raiders")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Mons's Goblin Raiders", 2, 2)
	g.AssertHasAbility(gametest.PlayerA, "Mons's Goblin Raiders", core.Haste, true)
}
func TestGoblinCommando_DealsTwoOnETB(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Goblin Commando")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 5)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mons's Goblin Raiders")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Goblin Commando")
	g.ChooseTarget(gametest.PlayerA, "Mons's Goblin Raiders")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Mons's Goblin Raiders", 1)
}

// Goblin Goon: can't attack unless attacker controls strictly more
// creatures than the defending player.
func TestGoblinGoon_CantAttackEqualCount(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Goblin Goon")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.Attack(1, gametest.PlayerA, "Goblin Goon")
	g.StopAt(1, core.EndCombat)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 20)
}

// Goblin Goon: can't block unless its controller has strictly more
// creatures than attacking player.
func TestGoblinGoon_CantBlockEqualCount(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Goblin Goon")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.Attack(1, gametest.PlayerA, "Grizzly Bears")
	g.Block(1, gametest.PlayerB, "Goblin Goon", "Grizzly Bears")
	g.StopAt(1, core.EndCombat)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 18)
}
func TestGoblinInstigator_CreatesGoblinToken(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Goblin Instigator")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 2)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Goblin Instigator")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Goblin", 1)
}
func TestGoblinShortcutter_PreventsBlocking(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Goblin Shortcutter")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 2)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.SetLife(gametest.PlayerB, 20)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Goblin Shortcutter", "Grizzly Bears")
	g.Attack(1, gametest.PlayerA, "Hill Giant")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 17)
}
func TestGraveBramble_HasDefenderAndProtection(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grave Bramble")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Grave Bramble", core.Defender, true)
}
func TestGravewaker_ReanimatesTapped(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Gravewaker")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 7)
	g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Grizzly Bears")
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Gravewaker", "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
	g.AssertTapped(gametest.PlayerA, "Grizzly Bears", true)
}
func TestGrimLavamancer_PingForTwo(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grim Lavamancer")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Plains", 2)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.ActivateAbility(2, core.PrecombatMain, gametest.PlayerA, "Grim Lavamancer", "Grizzly Bears")
	g.StopAt(2, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
}
func TestGristleGrinner_BoostOnDeath(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Gristle Grinner")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "Grizzly Bears")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Gristle Grinner", 5, 5)
}
func TestHamletbackGoliath_AddsCountersOnETB(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hamletback Goliath")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Hill Giant")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 4)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Hill Giant")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Hamletback Goliath", core.P1P1, 3)
}
func TestHarvesterOfSouls_DrawsOnNontokenDeath(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Harvester of Souls")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Mox Ruby", 2)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Mox Ruby", 1)
}
func TestHealersHawk(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Healer's Hawk")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Healer's Hawk", core.Flying, true)
	g.AssertHasAbility(gametest.PlayerA, "Healer's Hawk", core.Lifelink, true)
}
func TestHellrider_DealsOneOnAttack(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hellrider")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.SetLife(gametest.PlayerB, 20)
	g.Attack(3, gametest.PlayerA, "Hellrider", "Grizzly Bears")
	g.StopAt(3, core.EndStep)
	g.Execute()
	// 3+2 combat damage + 1+1 from Hellrider's trigger (fires per declared attacker
	// you control, including Hellrider itself).
	g.AssertLife(gametest.PlayerB, 13)
}
func TestHighSentinelsOfArashin(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "High Sentinels of Arashin")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCounters(1, core.PrecombatMain, gametest.PlayerA, "Grizzly Bears", core.P1P1, 1)
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "High Sentinels of Arashin", 4, 5)
}
func TestInniazTheGaleForce_DoesNotPumpNonAttackers(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Inniaz, the Gale Force")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Healer's Hawk")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 3)
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Inniaz, the Gale Force")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Healer's Hawk", 1, 1)
}

// Non-flying attackers don't count toward the threshold.
func TestInniazTheGaleForce_NonFlyersDoNotCountTowardThreshold(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Inniaz, the Gale Force")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Healer's Hawk")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mesa Pegasus")
	// 2 flyers + 1 non-flyer attacking — flying count is 2, no trigger.
	g.Attack(3, gametest.PlayerA, "Inniaz, the Gale Force", "Healer's Hawk", "Grizzly Bears")
	g.StopAt(3, core.DeclareBlockers)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerB, "Mesa Pegasus", 1)
	g.AssertPermanentCount(gametest.PlayerA, "Mesa Pegasus", 0)
	g.AssertPermanentCount(gametest.PlayerA, "Healer's Hawk", 1)
}

// If the opponent controls only lands, no permanent transfers from B to A;
// the other assignment (A -> B) still happens.
func TestInniazTheGaleForce_NoNonlandOnOpposingSideSkipsAssignment(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Inniaz, the Gale Force")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Healer's Hawk")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mesa Pegasus")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mountain")
	// Only the second assignment has a candidate: A's nonland -> B.
	g.ChoosePermanent(gametest.PlayerA, "Mesa Pegasus")
	g.Attack(3, gametest.PlayerA, "Inniaz, the Gale Force", "Healer's Hawk", "Mesa Pegasus")
	g.StopAt(3, core.DeclareBlockers)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerB, "Mesa Pegasus", 1)
	g.AssertPermanentCount(gametest.PlayerA, "Mesa Pegasus", 0)
	g.AssertPermanentCount(gametest.PlayerA, "Healer's Hawk", 1)
	g.AssertPermanentCount(gametest.PlayerB, "Mountain", 1)
}
func TestInniazTheGaleForce_PumpAttackingFlyersWithU(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Inniaz, the Gale Force")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Healer's Hawk")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 3)
	g.Attack(3, gametest.PlayerA, "Healer's Hawk")
	g.ActivateAbility(3, core.DeclareAttackers, gametest.PlayerA, "Inniaz, the Gale Force")
	g.StopAt(3, core.DeclareBlockers)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Healer's Hawk", 2, 2)
}
func TestInniazTheGaleForce_PumpAttackingFlyersWithW(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Inniaz, the Gale Force")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Healer's Hawk")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 3)
	g.Attack(3, gametest.PlayerA, "Healer's Hawk")
	g.ActivateAbility(3, core.DeclareAttackers, gametest.PlayerA, "Inniaz, the Gale Force")
	g.StopAt(3, core.DeclareBlockers)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Healer's Hawk", 2, 2)
}

// Three flying attackers controlled by Inniaz's controller fires the
// trigger; in 2-player, "the player to their right" collapses to the
// unique opponent, so the controller picks for both assignments.
func TestInniazTheGaleForce_ThreeFlyersSwapsPermanents(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Inniaz, the Gale Force")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Healer's Hawk")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mesa Pegasus")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mountain")
	// Inniaz's controller (A) makes both selections in order:
	//   1) of B's nonland permanents -> A gains Grizzly Bears
	//   2) of A's nonland permanents -> B gains Healer's Hawk
	g.ChoosePermanent(gametest.PlayerA, "Grizzly Bears")
	g.ChoosePermanent(gametest.PlayerA, "Healer's Hawk")
	g.Attack(3, gametest.PlayerA, "Inniaz, the Gale Force", "Healer's Hawk", "Mesa Pegasus")
	g.StopAt(3, core.DeclareBlockers)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
	g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
	g.AssertPermanentCount(gametest.PlayerB, "Healer's Hawk", 1)
	g.AssertPermanentCount(gametest.PlayerA, "Healer's Hawk", 0)
}

// Only two flying attackers — does not meet the "three or more" threshold.
func TestInniazTheGaleForce_TwoFlyersDoesNotTrigger(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Inniaz, the Gale Force")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Healer's Hawk")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.Attack(3, gametest.PlayerA, "Inniaz, the Gale Force", "Healer's Hawk")
	g.StopAt(3, core.DeclareBlockers)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 1)
	g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 0)
	g.AssertPermanentCount(gametest.PlayerA, "Healer's Hawk", 1)
}
