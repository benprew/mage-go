package gametest

import (
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

// Engine tests for the may-pay-mana effect used in trigger resolution
// (Cradle of Vitality, Kels Fight Fixer, Emiel the Blessed-style abilities).

var mayPayManaRegistered sync.Once

func registerMayPayManaTestCards() {
	mayPayManaRegistered.Do(func() {
		// A 2/2 Bear used as a counter target.
		if !mage.CardRegistered("MayPay Bear") {
			mage.Register("MayPay Bear", func() mage.Card {
				return mage.NewCreature("MayPay Bear", "{1}{G}", 2, 2,
					mage.WithSubTypes("Bear"),
				)
			})
		}
	})
}

// applyMayPay is a small helper that invokes a MayPayMana effect with the
// given source/controller/targets so each test can exercise pay/decline paths.
func applyMayPay(t *testing.T, g *TestGame, e mage.Effect, controller uuid.UUID) {
	t.Helper()
	if err := mage.ApplyEffect(g.Game, e, uuid.Nil, controller, nil); err != nil {
		t.Fatalf("MayPayMana apply: %v", err)
	}
}

// TestMayPayMana_PayAccepted verifies the inner effect runs when the
// controller accepts AND the cost is payable.
func TestMayPayMana_PayAccepted(t *testing.T) {
	registerMayPayManaTestCards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "MayPay Bear")
	// Two basic Plains supply {W}{W}; the may-pay cost we test is {1}{W}.
	g.AddCard(core.ZoneBattlefield, PlayerA, "Plains", 2)
	g.StopAt(1, core.PrecombatMain)
	g.Execute()

	bear := findPermanentByName(g, "MayPay Bear")
	playerA := g.AllPlayers()[0]

	inner := mage.FuncEffect("put a +1/+1 counter on bear", mage.EffectProperties{},
		func(gg *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
			gg.AddCountersWithReplacement(bear, core.P1P1, 1, sourceID, false)
			return nil
		})
	eff := mage.MayPayMana("{1}{W}", "put a +1/+1 counter on Counter Bear", inner)

	// Default ChooseMayAbility=true; lands are available.
	applyMayPay(t, g, eff, playerA.PlayerID())

	g.AssertCounterCount(PlayerA, "MayPay Bear", core.P1P1, 1)
	// All 2 plains should be tapped paying {1}{W}.
	tappedPlains := 0
	for _, p := range g.AllBattlefield() {
		if p.Name() == "Plains" && p.Tapped {
			tappedPlains++
		}
	}
	if tappedPlains != 2 {
		t.Errorf("expected 2 tapped Plains, got %d", tappedPlains)
	}
}

// TestMayPayMana_PlayerDeclines verifies the inner effect does NOT run when
// the controller declines, and lands stay untapped.
func TestMayPayMana_PlayerDeclines(t *testing.T) {
	registerMayPayManaTestCards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "MayPay Bear")
	g.AddCard(core.ZoneBattlefield, PlayerA, "Plains", 2)
	g.StopAt(1, core.PrecombatMain)
	g.Execute()

	bear := findPermanentByName(g, "MayPay Bear")
	playerA := g.AllPlayers()[0]
	tpA := g.GetPlayer(PlayerA)
	tpA.QueueMayAbilityChoices(false)

	inner := mage.FuncEffect("put a +1/+1 counter on bear", mage.EffectProperties{},
		func(gg *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
			gg.AddCountersWithReplacement(bear, core.P1P1, 1, sourceID, false)
			return nil
		})
	eff := mage.MayPayMana("{1}{W}", "put a +1/+1 counter on MayPay Bear", inner)

	applyMayPay(t, g, eff, playerA.PlayerID())

	g.AssertCounterCount(PlayerA, "MayPay Bear", core.P1P1, 0)
	for _, p := range g.AllBattlefield() {
		if p.Name() == "Plains" && p.Tapped {
			t.Errorf("Plains should not have been tapped when may-pay declined")
		}
	}
}

// TestMayPayMana_CannotAffordCost verifies that even when the controller
// accepts, if the cost cannot be paid (no lands), the inner effect does
// not run and no partial payment is made.
func TestMayPayMana_CannotAffordCost(t *testing.T) {
	registerMayPayManaTestCards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "MayPay Bear")
	// Only one Forest available; cost is {1}{W} which Forest can't pay.
	g.AddCard(core.ZoneBattlefield, PlayerA, "Forest")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()

	bear := findPermanentByName(g, "MayPay Bear")
	playerA := g.AllPlayers()[0]

	inner := mage.FuncEffect("put a +1/+1 counter on bear", mage.EffectProperties{},
		func(gg *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
			gg.AddCountersWithReplacement(bear, core.P1P1, 1, sourceID, false)
			return nil
		})
	eff := mage.MayPayMana("{1}{W}", "put a +1/+1 counter", inner)

	applyMayPay(t, g, eff, playerA.PlayerID())

	g.AssertCounterCount(PlayerA, "MayPay Bear", core.P1P1, 0)
}
