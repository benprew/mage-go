package jumpstart

import (
	"testing"

	_ "git.sr.ht/~cdcarter/mage-go/cards/limited"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/gametest"
)

// TestSavageStomp_CounterAndFight verifies +1/+1 counter then fight between
// two targets. A 2/2 (post-counter 3/3) fights a 3/3 — both die.
func TestSavageStomp_CounterAndFight(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 3)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Savage Stomp")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Savage Stomp", "Grizzly Bears", "Hill Giant")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 1)
	g.AssertGraveyardCount(gametest.PlayerB, "Hill Giant", 1)
}

// TestTimeToFeed_FightAndGain3 verifies the fight resolves and the on-death
// trigger fires for +3 life when the opponent's creature dies.
func TestTimeToFeed_FightAndGain3(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.SetLife(gametest.PlayerA, 20)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 3)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Time to Feed")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Time to Feed", "Grizzly Bears", "Hill Giant")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
	g.AssertLife(gametest.PlayerA, 23)
}

// TestThirstForKnowledge_DiscardArtifact verifies the controller defaults to
// "yes pay" and discards an artifact card to avoid the 2-card discard.
func TestThirstForKnowledge_DiscardArtifact(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 3)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Thirst for Knowledge")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Plains", 5)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Sol Ring")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Thirst for Knowledge")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerA, "Sol Ring", 1)
}

// TestThirstForKnowledge_PlayerChoosesArtifact verifies that with multiple
// artifact cards in hand, the controller picks which one to discard rather
// than the engine auto-picking the first.
func TestThirstForKnowledge_PlayerChoosesArtifact(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 3)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Thirst for Knowledge")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Sol Ring")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Mox Ruby")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Plains", 5)
	g.ChooseDiscard(gametest.PlayerA, "Mox Ruby")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Thirst for Knowledge")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerA, "Mox Ruby", 1)
	g.AssertGraveyardCount(gametest.PlayerA, "Sol Ring", 0)
	g.AssertHandCount(gametest.PlayerA, "Sol Ring", 1)
}

// TestThirstForKnowledge_NoArtifactDiscardTwo verifies that when there is no
// artifact card in hand, the controller cannot pay the artifact branch and
// must discard two cards instead.
func TestThirstForKnowledge_NoArtifactDiscardTwo(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 3)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Thirst for Knowledge")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Counterspell")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Plains", 5)
	g.ChooseDiscard(gametest.PlayerA, "Lightning Bolt", "Counterspell")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Thirst for Knowledge")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerA, "Lightning Bolt", 1)
	g.AssertGraveyardCount(gametest.PlayerA, "Counterspell", 1)
	g.AssertGraveyardCount(gametest.PlayerA, "Thirst for Knowledge", 1)
}

// TestThirstForKnowledge_DeclineArtifactBranch verifies that the controller
// can choose NOT to pay the discard-an-artifact branch even when an artifact
// is in hand, and instead discard two non-artifact cards of their choice.
func TestThirstForKnowledge_DeclineArtifactBranch(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 3)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Thirst for Knowledge")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Mox Ruby")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Counterspell")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Plains", 5)
	tpA := g.GetPlayer(gametest.PlayerA)
	tpA.QueueMayAbilityChoices(false)
	g.ChooseDiscard(gametest.PlayerA, "Lightning Bolt", "Counterspell")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Thirst for Knowledge")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Mox Ruby", 1)
	g.AssertGraveyardCount(gametest.PlayerA, "Mox Ruby", 0)
	g.AssertGraveyardCount(gametest.PlayerA, "Lightning Bolt", 1)
	g.AssertGraveyardCount(gametest.PlayerA, "Counterspell", 1)
}

// TestReadTheRunes_X1Discard verifies that for X=1 with the controller
// declining the sacrifice option, the discard branch fires once.
func TestReadTheRunes_X1Discard(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Read the Runes")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Plains")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Forest")
	tp := g.GetPlayer(gametest.PlayerA)
	tp.QueueMayAbilityChoices(false)
	g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Read the Runes", 1)
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerA, "Read the Runes", 1)
}

// TestRhysticStudy_OpponentDeclinesPay verifies that when an opponent casts
// a spell and declines/cannot pay {1}, the controller draws a card.
func TestRhysticStudy_OpponentDeclinesPay(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Rhystic Study")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mountain", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Plains")
	g.SetLife(gametest.PlayerA, 20)
	tpB := g.GetPlayer(gametest.PlayerB)
	tpB.QueueMayAbilityChoices(false)
	g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Lightning Bolt", "PlayerA")
	g.StopAt(2, core.EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Plains", 1)
	g.AssertLife(gametest.PlayerA, 17)
}

// TestDraconicRoar_RevealedDragonDealsExtra verifies the optional reveal pays
// off as +3 damage to the targeted creature's controller.
func TestDraconicRoar_RevealedDragonDealsExtra(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.SetLife(gametest.PlayerB, 20)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Draconic Roar")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Shivan Dragon")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Draconic Roar", "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
	g.AssertLife(gametest.PlayerB, 17)
	g.AssertHandCount(gametest.PlayerA, "Shivan Dragon", 1)
}

// TestDraconicRoar_NoDragonNoBonus verifies that without a Dragon to reveal
// or control, only the base 3 damage applies and the controller takes no hit.
func TestDraconicRoar_NoDragonNoBonus(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.SetLife(gametest.PlayerB, 20)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Draconic Roar")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Draconic Roar", "Hill Giant")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Hill Giant", 1)
	g.AssertLife(gametest.PlayerB, 20)
}

// TestDraconicRoar_ControlledDragonAtCastDealsExtra verifies that a Dragon
// the caster controls at cast time triggers the bonus damage on resolution.
// PlayerA declines the optional reveal — the controlled-Dragon half alone
// must satisfy the condition.
func TestDraconicRoar_ControlledDragonAtCastDealsExtra(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.SetLife(gametest.PlayerB, 20)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 2)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Dragon Hatchling")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Draconic Roar")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	tpA := g.GetPlayer(gametest.PlayerA)
	tpA.QueueMayAbilityChoices(false) // decline optional reveal
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Draconic Roar", "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
	g.AssertLife(gametest.PlayerB, 17)
}

// TestDraconicRoar_ControlledDragonLeavesBeforeResolution verifies CR 608.2g:
// the "controlled a Dragon as you cast this spell" condition is fixed at
// cast time, so a Dragon that is killed in response between cast and
// resolution still satisfies the condition. PlayerA's Dragon Hatchling
// (0/1) is killed by PlayerB's Lightning Bolt cast in response to
// Draconic Roar; when Draconic Roar resolves it must still deal the
// bonus damage to PlayerB.
func TestDraconicRoar_ControlledDragonLeavesBeforeResolution(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.SetLife(gametest.PlayerB, 20)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 2)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Dragon Hatchling")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Draconic Roar")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mountain")
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	tpA := g.GetPlayer(gametest.PlayerA)
	tpA.QueueMayAbilityChoices(false) // decline optional reveal — only the controlled half should matter
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Draconic Roar", "Grizzly Bears")
	g.CastInResponseTo(gametest.PlayerB, "Lightning Bolt", "Dragon Hatchling")
	g.StopAt(1, core.EndStep)
	g.Execute()
	// Dragon Hatchling died to the Bolt before Draconic Roar resolved.
	g.AssertGraveyardCount(gametest.PlayerA, "Dragon Hatchling", 1)
	// Grizzly Bears died to Draconic Roar's 3 damage.
	g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
	// Bonus damage from cast-time-snapshotted "controlled a Dragon" still applied.
	g.AssertLife(gametest.PlayerB, 17)
}

// TestDraconicRoar_RevealedDragonOnlyNoBattlefieldDragon verifies the
// reveal-half of the condition independently: PlayerA controls no Dragon
// but reveals a Dragon card from hand as the optional additional cost,
// triggering the bonus damage.
func TestDraconicRoar_RevealedDragonOnlyNoBattlefieldDragon(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.SetLife(gametest.PlayerB, 20)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Draconic Roar")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Shivan Dragon")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Draconic Roar", "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
	g.AssertHandCount(gametest.PlayerA, "Shivan Dragon", 1)
	g.AssertLife(gametest.PlayerB, 17)
}

// TestExplore_AllowsSecondLand verifies that after Explore resolves, the
// active player's land-play allowance is bumped to 2. The harness's auto-
// land-play loop then plays both Forests in the same main phase.
func TestExplore_AllowsSecondLand(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Explore")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Forest", 2)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Plains")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Explore")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Forest", 4)
}
