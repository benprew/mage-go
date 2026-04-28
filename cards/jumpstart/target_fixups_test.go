package jumpstart

import (
	"testing"

	_ "git.sr.ht/~cdcarter/mage-go/cards/limited"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/gametest"
)

// Brightmare's "tap up to one target creature" should resolve when no
// creatures exist on the battlefield. Without the up-to-one target, casting
// would fail with no legal targets.
func TestBrightmareNoTargetsAvailable(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Brightmare")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 3)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Brightmare")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Brightmare", 1)
	g.AssertLife(gametest.PlayerA, 20)
}

// Departed Deckhand's {3}{U} ability targets "another target creature you
// control" — it must not be able to target itself, and the target filter
// must exclude opponent's creatures.
func TestDepartedDeckhandActivatedTargetsAnotherYouControl(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Departed Deckhand")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 4)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Wall of Wood")
	g.ActivateAbility(3, core.PrecombatMain, gametest.PlayerA, "Departed Deckhand", "Grizzly Bears")
	g.Attack(3, gametest.PlayerA, "Grizzly Bears")
	g.Block(3, gametest.PlayerB, "Wall of Wood", "Grizzly Bears")
	g.StopAt(3, core.PostcombatMain)
	g.Execute()
	// Wall of Wood (not a Spirit) couldn't legally block Grizzly Bears.
	g.AssertLife(gametest.PlayerB, 18)
}
