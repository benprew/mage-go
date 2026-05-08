package jumpstart

import (
	"testing"

	_ "git.sr.ht/~cdcarter/mage-go/cards/limited"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/gametest"
)

// Neyith of the Dire Hunt {2}{G}{G}
// Legendary Creature — Human Warrior  3/3
// Whenever one or more creatures you control fight or become blocked, draw a card.
// At the beginning of combat on your turn, you may pay {2}{R/G}.
// If you do, double target creature's power until end of turn. That creature
// must be blocked this combat if able. ({R/G} can be paid with either {R} or {G}.)

func TestNeyith_StatsAndTypes(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Neyith of the Dire Hunt")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Neyith of the Dire Hunt", 1)
	g.AssertPowerToughness(gametest.PlayerA, "Neyith of the Dire Hunt", 3, 3)
}

// One of your creatures becomes blocked → draw a card.
func TestNeyith_BecomeBlocked_DrawsCard(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Neyith of the Dire Hunt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	// Non-land library so the card stays in hand for counting.
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Lightning Bolt", 5)
	// Decline the begin-of-combat may-pay so the test isolates the blocked-trigger.
	g.GetPlayer(gametest.PlayerA).QueueMayAbilityChoices(false)
	g.Attack(3, gametest.PlayerA, "Grizzly Bears")
	g.Block(3, gametest.PlayerB, "Hill Giant", "Grizzly Bears")
	g.StopAt(3, core.EndCombat)
	g.Execute()
	// Turn 3 draw step on PlayerA = 1 card; Neyith blocked-trigger = +1.
	g.AssertHandCount(gametest.PlayerA, "Lightning Bolt", 2)
}

// CR 603.2c — multiple creatures become blocked in the same dispatch
// window: trigger fires once, draw exactly one card.
func TestNeyith_MultipleBlocked_DrawsOnce(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Neyith of the Dire Hunt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Lightning Bolt", 5)
	g.GetPlayer(gametest.PlayerA).QueueMayAbilityChoices(false)
	g.Attack(3, gametest.PlayerA, "Grizzly Bears", "Hill Giant")
	g.Block(3, gametest.PlayerB, "Grizzly Bears", "Grizzly Bears")
	g.Block(3, gametest.PlayerB, "Hill Giant", "Hill Giant")
	g.StopAt(3, core.EndCombat)
	g.Execute()
	// Turn 3 draw step (+1) + ONE Neyith aggregating both blocks (+1) = 2.
	g.AssertHandCount(gametest.PlayerA, "Lightning Bolt", 2)
}

// A creature you control fights → draw a card.
func TestNeyith_Fight_DrawsCard(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Neyith of the Dire Hunt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 3)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Savage Stomp")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Lightning Bolt", 3)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Savage Stomp", "Hill Giant", "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	// Turn 1 PlayerA does not draw on its draw step (CR 103.7a). The only
	// card drawn is from Neyith's fight trigger.
	g.AssertHandCount(gametest.PlayerA, "Lightning Bolt", 1)
}

// Beginning of combat: pay {2}{R/G}, target creature gets doubled power and
// must be blocked. Single legal blocker is forced into the block.
func TestNeyith_CombatAbility_DoubleAndForceBlock(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Neyith of the Dire Hunt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // 2/2 → becomes 4/2
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 3)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant") // 3/3 — sole legal blocker
	// EvtBeginCombat fires on each of PlayerA's turns — queue a target name
	// for both turn 1 and turn 3 (the turn 2 BeginCombat is PlayerB's).
	g.ChooseTarget(gametest.PlayerA, "Grizzly Bears")
	g.ChooseTarget(gametest.PlayerA, "Grizzly Bears")
	g.Attack(3, gametest.PlayerA, "Grizzly Bears")
	g.StopAt(3, core.EndCombat)
	g.Execute()
	// Bears (now 4/2) is blocked by Hill Giant (must-be-blocked). Both die.
	g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 1)
	g.AssertGraveyardCount(gametest.PlayerB, "Hill Giant", 1)
}

// Decline the cost: no doubled power, no must-be-blocked, no opponent forced
// to block.
func TestNeyith_DeclineMayPay(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Neyith of the Dire Hunt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 3)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.GetPlayer(gametest.PlayerA).QueueMayAbilityChoices(false)
	g.Attack(3, gametest.PlayerA, "Grizzly Bears")
	g.StopAt(3, core.EndCombat)
	g.Execute()
	// PlayerB was not forced to block. Bears deals 2 to PlayerB; both
	// creatures still on the battlefield.
	g.AssertLife(gametest.PlayerB, 18)
	g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
	g.AssertPermanentCount(gametest.PlayerB, "Hill Giant", 1)
}
