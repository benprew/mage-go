package astral

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestPrismaticDragonUpkeepChangesColorPermanently(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Prismatic Dragon")
	g.SetRandomResults([]int{1, 4})
	g.StopAt(2, core.PrecombatMain)
	g.Execute()

	g.AssertHasColor(gametest.PlayerA, "Prismatic Dragon", core.Blue, true)
	g.AssertHasColor(gametest.PlayerA, "Prismatic Dragon", core.White, false)
}

func TestPrismaticDragonActivatedAbilityChangesColorPermanently(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Prismatic Dragon")
	g.SetRandomResults([]int{0, 2})
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Prismatic Dragon")
	g.StopAt(2, core.PrecombatMain)
	g.Execute()

	g.AssertHasColor(gametest.PlayerA, "Prismatic Dragon", core.Black, true)
	g.AssertHasColor(gametest.PlayerA, "Prismatic Dragon", core.White, false)
}

func TestRainbowKnightsEntersWithPermanentRandomProtection(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.SetRandomResults([]int{3})
	id := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Rainbow Knights")
	g.ResolveStack()

	perm := g.FindPermanent(id)
	if perm == nil {
		t.Fatal("Rainbow Knights not on battlefield")
	}
	var colors []core.Color
	for _, ability := range perm.RuntimeAbilities {
		if protection, ok := mage.UnwrapAbility(ability).(*mage.ProtectionAbility); ok {
			colors = append(colors, protection.FromColors...)
		}
	}
	if len(colors) != 1 || colors[0] != core.Red {
		t.Fatalf("Rainbow Knights protection colors = %v, want [Red]", colors)
	}

	g.StopAt(2, core.PrecombatMain)
	g.Execute()
	perm = g.FindPermanent(id)
	colors = nil
	for _, ability := range perm.RuntimeAbilities {
		if protection, ok := mage.UnwrapAbility(ability).(*mage.ProtectionAbility); ok {
			colors = append(colors, protection.FromColors...)
		}
	}
	if len(colors) != 1 || colors[0] != core.Red {
		t.Fatalf("Rainbow Knights protection colors after end of turn = %v, want [Red]", colors)
	}
}

func TestRainbowKnightsFirstStrikeUntilEndOfTurn(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.SetRandomResults([]int{0})
	id := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Rainbow Knights")
	g.ResolveStack()
	g.GetPlayer(gametest.PlayerA).ManaPool().Add(core.Colorless, 1)

	if err := g.ActivateAbilityByIndex(g.GetPlayer(gametest.PlayerA).PlayerID(), id, 1, nil); err != nil {
		t.Fatalf("activate Rainbow Knights first strike: %v", err)
	}
	g.ResolveStack()
	g.AssertHasAbility(gametest.PlayerA, "Rainbow Knights", core.FirstStrike, true)

	g.StopAt(2, core.PrecombatMain)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Rainbow Knights", core.FirstStrike, false)
}

func TestRainbowKnightsRandomPowerUntilEndOfTurn(t *testing.T) {
	for roll := range 3 {
		t.Run(string(rune('0'+roll)), func(t *testing.T) {
			g := gametest.NewTestGame(t)
			g.SetRandomResults([]int{0, roll})
			id := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Rainbow Knights")
			g.ResolveStack()
			g.GetPlayer(gametest.PlayerA).ManaPool().Add(core.White, 2)

			if err := g.ActivateAbilityByIndex(g.GetPlayer(gametest.PlayerA).PlayerID(), id, 2, nil); err != nil {
				t.Fatalf("activate Rainbow Knights power ability: %v", err)
			}
			g.ResolveStack()
			g.AssertPowerToughness(gametest.PlayerA, "Rainbow Knights", 2+roll, 1)

			g.StopAt(2, core.PrecombatMain)
			g.Execute()
			g.AssertPowerToughness(gametest.PlayerA, "Rainbow Knights", 2, 1)
		})
	}
}
