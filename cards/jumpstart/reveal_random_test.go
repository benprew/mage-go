package jumpstart

import (
	"math/rand"
	"testing"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/gametest"
)

// Tests for cards wired to the new reveal-and-pick / random primitives
// (Lurking Predators, Commune with Dinosaurs, Silhana Wayfinder, Muxus,
// Sin Prodder, Corpse Traders, Entomber Exarch, Goblin Lore, Charmbreaker
// Devils).

func TestLurkingPredators_CreatureRevealedHitsBattlefield(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Lurking Predators")
	g.AddCard(ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(ZoneHand, gametest.PlayerB, "Lightning Bolt")
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Mountain")
	g.CastSpell(1, PrecombatMain, gametest.PlayerB, "Lightning Bolt", "PlayerA")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
}

func TestLurkingPredators_NonCreatureMayGoToBottom(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Lurking Predators")
	// Top card is non-creature.
	g.AddCard(ZoneLibrary, gametest.PlayerA, "Mountain")
	g.AddCard(ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(ZoneHand, gametest.PlayerB, "Lightning Bolt")
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Mountain")
	g.CastSpell(1, PrecombatMain, gametest.PlayerB, "Lightning Bolt", "PlayerA")
	g.StopAt(1, EndStep)
	g.Execute()
	// Default ChooseMayAbility=true => Mountain goes to bottom.
	g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 0)
	g.AssertPermanentCount(gametest.PlayerA, "Mountain", 0)
	g.AssertLibraryCount(gametest.PlayerA, "Mountain", 1)
}

func TestCommuneWithDinosaurs_FindsDinosaur(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneHand, gametest.PlayerA, "Commune with Dinosaurs")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Forest")
	g.AddCard(ZoneLibrary, gametest.PlayerA, "Orazca Frillback")
	g.AddCard(ZoneLibrary, gametest.PlayerA, "Mox Ruby", 4)
	g.ChooseFromLibrary(gametest.PlayerA, "Orazca Frillback")
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Commune with Dinosaurs")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Orazca Frillback", 1)
	g.AssertLibraryCount(gametest.PlayerA, "Orazca Frillback", 0)
	g.AssertLibraryCount(gametest.PlayerA, "Mox Ruby", 4)
}

func TestSilhanaWayfinder_PutsCreatureOnTop(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneHand, gametest.PlayerA, "Silhana Wayfinder")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Forest", 2)
	g.AddCard(ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(ZoneLibrary, gametest.PlayerA, "Mox Ruby", 4)
	g.ChooseFromLibrary(gametest.PlayerA, "Grizzly Bears")
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Silhana Wayfinder")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertLibraryTop(gametest.PlayerA, "Grizzly Bears")
}

func TestMuxus_PutsGoblinsOntoBattlefield(t *testing.T) {
	rand.Seed(1)
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneHand, gametest.PlayerA, "Muxus, Goblin Grandee")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Mountain", 6)
	// Library top 6: 2 Goblin creatures (MV<=5), 4 Mox Rubies (artifact, not goblin)
	g.AddCard(ZoneLibrary, gametest.PlayerA, "Goblin Chieftain")
	g.AddCard(ZoneLibrary, gametest.PlayerA, "Goblin Instigator")
	g.AddCard(ZoneLibrary, gametest.PlayerA, "Mox Ruby", 4)
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Muxus, Goblin Grandee")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Goblin Chieftain", 1)
	g.AssertPermanentCount(gametest.PlayerA, "Goblin Instigator", 1)
}

func TestSinProdder_OpponentSendsToGraveyardForDamage(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Sin Prodder")
	g.AddCard(ZoneLibrary, gametest.PlayerA, "Lightning Bolt") // MV 1
	// Opponent (PlayerB) chooses YES on first ChooseMayAbility -> 1 damage to PlayerB.
	g.GetPlayer(gametest.PlayerB).QueueMayAbilityChoices(true)
	g.StopAt(2, Upkeep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerA, "Lightning Bolt", 1)
	g.AssertLife(gametest.PlayerB, 19)
}

func TestSinProdder_OpponentDeclinesGoesToHand(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Sin Prodder")
	g.AddCard(ZoneLibrary, gametest.PlayerA, "Lightning Bolt")
	g.GetPlayer(gametest.PlayerB).QueueMayAbilityChoices(false)
	g.StopAt(2, Upkeep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Lightning Bolt", 1)
	g.AssertLife(gametest.PlayerB, 20)
}

func TestCorpseTraders_OpponentDiscardsChosenCard(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Corpse Traders")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Swamp", 3)
	g.AddCard(ZoneHand, gametest.PlayerB, "Lightning Bolt")
	g.AddCard(ZoneHand, gametest.PlayerB, "Mountain")
	g.ChooseFromLibrary(gametest.PlayerA, "Lightning Bolt")
	g.ActivateAbility(1, PrecombatMain, gametest.PlayerA, "Corpse Traders", "PlayerB")
	g.ChoosePermanent(gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Lightning Bolt", 1)
}

func TestEntomberExarch_Mode1ReturnsCreatureFromGraveyard(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneHand, gametest.PlayerA, "Entomber Exarch")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Swamp", 4)
	g.AddCard(ZoneGraveyard, gametest.PlayerA, "Grizzly Bears")
	g.ChooseMode(gametest.PlayerA, 0)
	g.ChooseFromLibrary(gametest.PlayerA, "Grizzly Bears")
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Entomber Exarch")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
}

func TestEntomberExarch_Mode2OpponentDiscardsNoncreature(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneHand, gametest.PlayerA, "Entomber Exarch")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Swamp", 4)
	g.AddCard(ZoneHand, gametest.PlayerB, "Lightning Bolt")
	g.AddCard(ZoneHand, gametest.PlayerB, "Grizzly Bears")
	g.ChooseMode(gametest.PlayerA, 1)
	g.ChooseFromLibrary(gametest.PlayerA, "Lightning Bolt")
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Entomber Exarch")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Lightning Bolt", 1)
	g.AssertHandCount(gametest.PlayerB, "Grizzly Bears", 1)
}

func TestGoblinLore_DrawFourDiscardThreeAtRandom(t *testing.T) {
	rand.Seed(42)
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneHand, gametest.PlayerA, "Goblin Lore")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Mountain", 2)
	g.AddCard(ZoneLibrary, gametest.PlayerA, "Mox Ruby", 4)
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Goblin Lore")
	g.StopAt(1, EndStep)
	g.Execute()
	// Drew 4 (Mox Ruby x4) and discarded 3 at random (could include any of the 4).
	g.AssertGraveyardCount(gametest.PlayerA, "Goblin Lore", 1)
	// 4 drawn - 3 discarded = 1 in hand from library; 0 starting hand.
	g.AssertHandCount(gametest.PlayerA, "Mox Ruby", 1)
	g.AssertGraveyardCount(gametest.PlayerA, "Mox Ruby", 3)
}

func TestCharmbreakerDevils_RandomReturnFromGraveyard(t *testing.T) {
	rand.Seed(7)
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Charmbreaker Devils")
	g.AddCard(ZoneGraveyard, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(ZoneGraveyard, gametest.PlayerA, "Grizzly Bears")
	g.StopAt(2, Upkeep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Lightning Bolt", 1)
	// Creature card stays in graveyard
	g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 1)
}

func TestBogbrewWitch_SearchForNewt(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Bogbrew Witch")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Swamp", 2)
	g.AddCard(ZoneLibrary, gametest.PlayerA, "Festering Newt")
	g.AddCard(ZoneLibrary, gametest.PlayerA, "Mox Ruby", 3)
	g.ChooseFromLibrary(gametest.PlayerA, "Festering Newt")
	g.ActivateAbility(1, PrecombatMain, gametest.PlayerA, "Bogbrew Witch")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Festering Newt", 1)
	g.AssertTapped(gametest.PlayerA, "Festering Newt", true)
}

func TestCharmbreakerDevils_BoostOnInstantOrSorcery(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Charmbreaker Devils")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Mountain", 1)
	g.AddCard(ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Lightning Bolt", "PlayerB")
	g.StopAt(1, BeginCombat)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Charmbreaker Devils", 8, 4)
}
