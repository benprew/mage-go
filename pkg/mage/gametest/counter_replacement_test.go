package gametest

import (
	"sync"
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"github.com/google/uuid"
)

// Engine tests for the counter-placement replacement pipeline (CR 614.1c):
//   - counter-doubling effects (Doubling Season / Branching Evolution-style)
//   - "enters with an additional counter" effects (Oona's Blackguard-style)
// and tests for the may-pay-mana effect used in trigger resolution.

var counterReplacementRegistered sync.Once

func registerCounterReplacementTestCards() {
	counterReplacementRegistered.Do(func() {
		// A vanilla 2/2 creature target.
		if !mage.CardRegistered("Counter Bear") {
			mage.Register("Counter Bear", func() mage.Card {
				return mage.NewCreature("Counter Bear", "{1}{G}", 2, 2,
					mage.WithSubTypes("Bear"),
				)
			})
		}

		// A 1/1 Rogue creature for ETB-additional-counter testing.
		if !mage.CardRegistered("Counter Rogue") {
			mage.Register("Counter Rogue", func() mage.Card {
				return mage.NewCreature("Counter Rogue", "{B}", 1, 1,
					mage.WithSubTypes("Rogue"),
				)
			})
		}

		// A 1/1 non-Rogue creature to verify the Rogue filter excludes
		// non-matching permanents.
		if !mage.CardRegistered("Counter Wizard") {
			mage.Register("Counter Wizard", func() mage.Card {
				return mage.NewCreature("Counter Wizard", "{U}", 1, 1,
					mage.WithSubTypes("Wizard"),
				)
			})
		}

		// A creature with EntersWithNCounters (fixed). Used to test ETB
		// counter doubling on the entering permanent itself.
		if !mage.CardRegistered("Counter Hydra Fixed") {
			mage.Register("Counter Hydra Fixed", func() mage.Card {
				return mage.NewCreature("Counter Hydra Fixed", "{2}{G}", 0, 0,
					mage.WithSubTypes("Hydra"),
					mage.WithAbility(mage.EntersWithNCounters(core.P1P1, 3)),
				)
			})
		}
	})
}

// ===== Counter doubling =====

// TestCounterDoublerOnAddCounters verifies that a registered counter-doubler
// replaces a 1-counter placement with a 2-counter placement on a matching
// permanent.
func TestCounterDoublerOnAddCounters(t *testing.T) {
	registerCounterReplacementTestCards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Counter Bear")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()

	bear := findPermanentByName(g, "Counter Bear")
	if bear == nil {
		t.Fatal("Counter Bear not on battlefield")
	}
	playerA := g.AllPlayers()[0]

	// Register a doubler scoped to the controller's permanents.
	g.AddCounterDoubler(bear.ID(), core.P1P1, mage.ControlledBy(playerA.PlayerID()))

	// Place 1 counter — should become 2.
	g.AddCountersWithReplacement(bear, core.P1P1, 1, uuid.Nil, false)
	g.AssertCounterCount(PlayerA, "Counter Bear", core.P1P1, 2)

	// A subsequent placement of 3 — should become 6.
	g.AddCountersWithReplacement(bear, core.P1P1, 3, uuid.Nil, false)
	g.AssertCounterCount(PlayerA, "Counter Bear", core.P1P1, 8)
}

// TestCounterDoublerFilterScope verifies the doubler ignores permanents the
// filter rejects (opponent's creatures don't get doubled).
func TestCounterDoublerFilterScope(t *testing.T) {
	registerCounterReplacementTestCards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Counter Bear")
	g.AddCard(core.ZoneBattlefield, PlayerB, "Counter Wizard")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()

	playerA := g.AllPlayers()[0]
	bear := findPermanentByName(g, "Counter Bear")
	wizard := findPermanentByName(g, "Counter Wizard")
	g.AddCounterDoubler(bear.ID(), core.P1P1, mage.ControlledBy(playerA.PlayerID()))

	g.AddCountersWithReplacement(bear, core.P1P1, 1, uuid.Nil, false)
	g.AddCountersWithReplacement(wizard, core.P1P1, 1, uuid.Nil, false)

	g.AssertCounterCount(PlayerA, "Counter Bear", core.P1P1, 2)   // doubled
	g.AssertCounterCount(PlayerB, "Counter Wizard", core.P1P1, 1) // not doubled
}

// TestCounterDoublerCounterTypeScope verifies the doubler only fires for the
// configured counter type.
func TestCounterDoublerCounterTypeScope(t *testing.T) {
	registerCounterReplacementTestCards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Counter Bear")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()

	bear := findPermanentByName(g, "Counter Bear")
	g.AddCounterDoubler(bear.ID(), core.P1P1, mage.PermanentFilter{})

	g.AddCountersWithReplacement(bear, core.M1M1, 2, uuid.Nil, false)
	g.AssertCounterCount(PlayerA, "Counter Bear", core.M1M1, 2)
	g.AssertCounterCount(PlayerA, "Counter Bear", core.P1P1, 0)
}

// TestCounterDoublerOnETBCounters verifies that fixed-N ETB counters
// (EntersWithNCounters) flow through the replacement pipeline and are
// doubled when a matching doubler is in play.
func TestCounterDoublerOnETBCounters(t *testing.T) {
	registerCounterReplacementTestCards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Counter Bear")
	bear := findPermanentByName(g, "Counter Bear")
	playerA := g.AllPlayers()[0]
	g.AddCounterDoubler(bear.ID(), core.P1P1, mage.ControlledBy(playerA.PlayerID()))

	// Drop a Counter Hydra Fixed onto the battlefield directly so the ETB
	// fixed-N counters flow through the replacement pipeline. The hydra has
	// EntersWithNCounters(P1P1, 3); the doubler scoped to PlayerA's
	// permanents should turn that into 6.
	hydra, err := mage.CreateCard("Counter Hydra Fixed")
	if err != nil {
		t.Fatalf("CreateCard: %v", err)
	}
	hydra.SetOwner(playerA.PlayerID())
	g.PutOnBattlefield(hydra, playerA.PlayerID())
	g.AssertCounterCount(PlayerA, "Counter Hydra Fixed", core.P1P1, 6)
}

// ===== ETB additional counters =====

// TestETBAdditionalCounters verifies that "enters with an additional +1/+1
// counter" places the extra counter on top of the normal placement.
func TestETBAdditionalCounters(t *testing.T) {
	registerCounterReplacementTestCards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Counter Bear")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()

	bear := findPermanentByName(g, "Counter Bear")

	// Register an ETB-additional replacement that adds 1 extra counter to
	// every creature you control on entry.
	playerA := g.AllPlayers()[0]
	g.AddETBAdditionalCounters(bear.ID(), core.P1P1, 1,
		mage.And(mage.IsCreature, mage.ControlledBy(playerA.PlayerID())), false)

	// On entry: 1 -> 2.
	g.AddCountersWithReplacement(bear, core.P1P1, 1, uuid.Nil, true)
	g.AssertCounterCount(PlayerA, "Counter Bear", core.P1P1, 2)

	// Outside ETB: no boost.
	g.AddCountersWithReplacement(bear, core.P1P1, 1, uuid.Nil, false)
	g.AssertCounterCount(PlayerA, "Counter Bear", core.P1P1, 3)
}

// TestETBAdditionalCountersExcludeSelf verifies excludeSelf=true keeps the
// source permanent from buffing itself (Oona's Blackguard says "Each *other*
// Rogue creature ... enters with an additional +1/+1 counter").
func TestETBAdditionalCountersExcludeSelf(t *testing.T) {
	registerCounterReplacementTestCards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Counter Bear")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()

	bear := findPermanentByName(g, "Counter Bear")
	// Source is the bear itself.
	g.AddETBAdditionalCounters(bear.ID(), core.P1P1, 1, mage.PermanentFilter{}, true)

	// Self ETB placement with extra=1 and excludeSelf=true: NO bonus.
	g.AddCountersWithReplacement(bear, core.P1P1, 1, uuid.Nil, true)
	g.AssertCounterCount(PlayerA, "Counter Bear", core.P1P1, 1)
}

// TestETBAdditionalCountersOnlyOnEntry verifies the replacement matches only
// when OnEntry=true (per CR 614.1c "as it enters").
func TestETBAdditionalCountersOnlyOnEntry(t *testing.T) {
	registerCounterReplacementTestCards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Counter Bear")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()

	bear := findPermanentByName(g, "Counter Bear")
	g.AddETBAdditionalCounters(bear.ID(), core.P1P1, 2, mage.PermanentFilter{}, false)

	// Non-entry placement: NO extra counters.
	g.AddCountersWithReplacement(bear, core.P1P1, 1, uuid.Nil, false)
	g.AssertCounterCount(PlayerA, "Counter Bear", core.P1P1, 1)
}

// TestDoublerStacksWithETBAdditional verifies that a doubler and an
// ETB-additional replacement both fire on the same ETB placement (each at
// most once per event, in the order the pipeline picks them up). The result
// is (1 + 1) * 2 = 4 counters from a 1-counter placement when extra=1 and
// the doubler doubles all +1/+1 placements.
func TestDoublerStacksWithETBAdditional(t *testing.T) {
	registerCounterReplacementTestCards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Counter Bear")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()

	bear := findPermanentByName(g, "Counter Bear")
	g.AddETBAdditionalCounters(bear.ID(), core.P1P1, 1, mage.PermanentFilter{}, false)
	g.AddCounterDoubler(bear.ID(), core.P1P1, mage.PermanentFilter{})

	g.AddCountersWithReplacement(bear, core.P1P1, 1, uuid.Nil, true)
	g.AssertCounterCount(PlayerA, "Counter Bear", core.P1P1, 4)
}
