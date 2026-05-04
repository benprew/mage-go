package gametest

// Hybrid mana costs (CR 107.4d, 117.7, 202.2d, 202.3f).
//
// A two-color hybrid mana symbol like {W/U} can be paid with mana of either
// of its two colors. CMC of a hybrid symbol is 1, and the symbol's color is
// both of its colors for purposes of CR 202.2c.

import (
	"sync"
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

var hybridOnce sync.Once

func registerHybridCards() {
	hybridOnce.Do(func() {
		reg := func(name string, f func() mage.Card) {
			if !mage.CardRegistered(name) {
				mage.Register(name, f)
			}
		}
		reg("Hybrid WU Bear", func() mage.Card {
			return mage.NewCreature("Hybrid WU Bear", "{W/U}", 1, 1,
				mage.WithSubTypes("Bear"))
		})
		reg("Hybrid Two WU Bear", func() mage.Card {
			return mage.NewCreature("Hybrid Two WU Bear", "{W/U}{W/U}", 2, 2,
				mage.WithSubTypes("Bear"))
		})
		reg("Hybrid Mixed Bear", func() mage.Card {
			return mage.NewCreature("Hybrid Mixed Bear", "{1}{G}{G/W}", 3, 3,
				mage.WithSubTypes("Bear"))
		})
	})
}

func TestCR107_4d_HybridManaCost_Parses(t *testing.T) {
	mc := core.ParseManaCost("{1}{W/U}{B/R}")
	if mc.Generic != 1 {
		t.Errorf("expected 1 generic, got %d", mc.Generic)
	}
	if got, want := len(mc.Hybrid), 2; got != want {
		t.Fatalf("expected %d hybrid symbols, got %d", want, got)
	}
	if mc.Hybrid[0].A != core.White || mc.Hybrid[0].B != core.Blue {
		t.Errorf("first hybrid: got %v/%v", mc.Hybrid[0].A, mc.Hybrid[0].B)
	}
	if mc.Hybrid[1].A != core.Black || mc.Hybrid[1].B != core.Red {
		t.Errorf("second hybrid: got %v/%v", mc.Hybrid[1].A, mc.Hybrid[1].B)
	}
}

func TestCR202_3f_HybridManaValue_IsOnePerSymbol(t *testing.T) {
	mc := core.ParseManaCost("{2}{W/U}{B/R}")
	if got, want := mc.CMC(), 4; got != want {
		t.Errorf("expected CMC %d, got %d", want, got)
	}
}

func TestCR202_2c_HybridSymbolContributesBothColors(t *testing.T) {
	mc := core.ParseManaCost("{W/U}")
	colors := mc.Colors()
	hasW, hasU := false, false
	for _, c := range colors {
		if c == core.White {
			hasW = true
		}
		if c == core.Blue {
			hasU = true
		}
	}
	if !hasW || !hasU {
		t.Errorf("expected both white and blue in colors, got %v", colors)
	}
}

func TestCR107_4d_HybridCost_PayWithFirstColor(t *testing.T) {
	registerHybridCards()
	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Plains")
	g.AddCard(core.ZoneHand, PlayerA, "Hybrid WU Bear")

	g.CastSpell(1, core.PrecombatMain, PlayerA, "Hybrid WU Bear")
	g.StopAt(1, core.EndStep)
	g.Execute()

	g.AssertPermanentCount(PlayerA, "Hybrid WU Bear", 1)
}

func TestCR107_4d_HybridCost_PayWithSecondColor(t *testing.T) {
	registerHybridCards()
	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Island")
	g.AddCard(core.ZoneHand, PlayerA, "Hybrid WU Bear")

	g.CastSpell(1, core.PrecombatMain, PlayerA, "Hybrid WU Bear")
	g.StopAt(1, core.EndStep)
	g.Execute()

	g.AssertPermanentCount(PlayerA, "Hybrid WU Bear", 1)
}

func TestCR107_4d_HybridCost_TwoSymbolsBothColorsAvailable(t *testing.T) {
	registerHybridCards()
	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Plains")
	g.AddCard(core.ZoneBattlefield, PlayerA, "Island")
	g.AddCard(core.ZoneHand, PlayerA, "Hybrid Two WU Bear")

	g.CastSpell(1, core.PrecombatMain, PlayerA, "Hybrid Two WU Bear")
	g.StopAt(1, core.EndStep)
	g.Execute()

	g.AssertPermanentCount(PlayerA, "Hybrid Two WU Bear", 1)
}

func TestCR107_4d_HybridCost_TwoSymbolsOneColorOnly(t *testing.T) {
	registerHybridCards()
	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Island")
	g.AddCard(core.ZoneBattlefield, PlayerA, "Island")
	g.AddCard(core.ZoneHand, PlayerA, "Hybrid Two WU Bear")

	g.CastSpell(1, core.PrecombatMain, PlayerA, "Hybrid Two WU Bear")
	g.StopAt(1, core.EndStep)
	g.Execute()

	g.AssertPermanentCount(PlayerA, "Hybrid Two WU Bear", 1)
}

func TestCR107_4d_HybridCost_MixedWithGenericAndColored(t *testing.T) {
	registerHybridCards()
	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Forest")
	g.AddCard(core.ZoneBattlefield, PlayerA, "Forest")
	g.AddCard(core.ZoneBattlefield, PlayerA, "Forest")
	g.AddCard(core.ZoneHand, PlayerA, "Hybrid Mixed Bear")

	g.CastSpell(1, core.PrecombatMain, PlayerA, "Hybrid Mixed Bear")
	g.StopAt(1, core.EndStep)
	g.Execute()

	g.AssertPermanentCount(PlayerA, "Hybrid Mixed Bear", 1)
}
