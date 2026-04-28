package jumpstart

import (
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/gametest"

	_ "git.sr.ht/~cdcarter/mage-go/cards/limited"
)

// Tinybones, Trinket Thief: end-step half fires only when an opponent has
// discarded a card this turn. Drainpipe Vermin's death trigger forces the
// opponent to discard, which lights up the per-turn discard tracker.
func TestTinybonesTrinketThief_DrawsWhenOpponentDiscarded(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Tinybones, Trinket Thief")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Drainpipe Vermin")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Forest", 5)
	// Kill the vermin so its death trigger fires; pay {B} to make B discard.
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "Drainpipe Vermin")
	g.ChooseTarget(gametest.PlayerA, "PlayerB")
	g.StopAt(1, core.Cleanup)
	g.Execute()
	// End-step trigger: A draws a card and loses 1 life.
	g.AssertHandCount(gametest.PlayerA, "Forest", 1)
	g.AssertLife(gametest.PlayerA, 19)
}

// No opponent discard this turn: Tinybones end-step ability does nothing.
func TestTinybonesTrinketThief_NoDiscardNoDraw(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Tinybones, Trinket Thief")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Forest")
	g.StopAt(1, core.Cleanup)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Forest", 0)
	g.AssertLife(gametest.PlayerA, 20)
}

// Witch of the Moors: when controller gained life this turn, each opponent
// sacrifices a creature and the controller may return a creature card from
// their graveyard.
func TestWitchOfTheMoors_TriggersOnLifeGain(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Witch of the Moors")
	g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	// Make A gain life via a Healing Salve-style spell.
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Healing Salve")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Healing Salve")
	g.ChooseMode(gametest.PlayerA, 0)
	g.ChooseTarget(gametest.PlayerA, "PlayerA")
	// End-step trigger: B picks Hill Giant to sacrifice.
	g.ChoosePermanent(gametest.PlayerB, "Hill Giant")
	g.StopAt(1, core.Cleanup)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Hill Giant", 1)
	g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
}

// Without life gain, the trigger's intervening-if fails silently.
func TestWitchOfTheMoors_NoLifeGainNoEffect(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Witch of the Moors")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.StopAt(1, core.Cleanup)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerB, "Hill Giant", 1)
}

// Inferno Hellion: at the beginning of each end step, if it attacked this
// turn, it gets shuffled into its owner's library.
func TestInfernoHellion_ShufflesAfterAttacking(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Inferno Hellion")
	g.Attack(1, gametest.PlayerA, "Inferno Hellion")
	g.StopAt(1, core.Cleanup)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Inferno Hellion", 0)
	g.AssertGraveyardCount(gametest.PlayerA, "Inferno Hellion", 0)
	// Card returns to library; library count goes up by 1.
}

// Inferno Hellion that didn't attack/block stays put.
func TestInfernoHellion_StaysIfIdle(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Inferno Hellion")
	g.StopAt(1, core.Cleanup)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Inferno Hellion", 1)
}

// Allosaurus Shepherd: the spell itself can't be countered.
func TestAllosaurusShepherd_SelfUncounterable(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Allosaurus Shepherd")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Island", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Counterspell")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Allosaurus Shepherd")
	g.CastInResponseTo(gametest.PlayerB, "Counterspell", "Allosaurus Shepherd")
	g.StopAt(1, core.Cleanup)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Allosaurus Shepherd", 1)
	g.AssertGraveyardCount(gametest.PlayerB, "Counterspell", 1)
}

// Allosaurus Shepherd: while on the battlefield, other green spells the
// controller casts can't be countered.
func TestAllosaurusShepherd_ProtectsControllersGreenSpells(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Allosaurus Shepherd")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 2)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Island", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Counterspell")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Grizzly Bears")
	g.CastInResponseTo(gametest.PlayerB, "Counterspell", "Grizzly Bears")
	g.StopAt(1, core.Cleanup)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
	g.AssertGraveyardCount(gametest.PlayerB, "Counterspell", 1)
}

// Volcanic Fallout: this spell can't be countered.
func TestVolcanicFallout_Uncounterable(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Volcanic Fallout")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 3)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Island", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Counterspell")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Volcanic Fallout")
	g.CastInResponseTo(gametest.PlayerB, "Counterspell", "Volcanic Fallout")
	g.StopAt(1, core.Cleanup)
	g.Execute()
	// Fallout still resolves: 2 damage to each player.
	g.AssertLife(gametest.PlayerA, 18)
	g.AssertLife(gametest.PlayerB, 18)
	g.AssertGraveyardCount(gametest.PlayerB, "Counterspell", 1)
}
