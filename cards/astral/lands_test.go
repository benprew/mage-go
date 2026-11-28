package astral

import (
	"testing"

	_ "github.com/benprew/mage-go/cards/fourthedition" // Register Fellwar Stone.
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func putGemBazaarOntoBattlefield(t *testing.T, g *gametest.TestGame) *mage.Permanent {
	t.Helper()
	player := g.GetPlayer(gametest.PlayerA)
	card, err := mage.CreateCard("Gem Bazaar")
	if err != nil {
		t.Fatalf("create Gem Bazaar: %v", err)
	}
	card.SetOwner(player.PlayerID())
	return g.PutOnBattlefield(card, player.PlayerID())
}

func TestGemBazaar_TriggeredRandomColorThenUsesAndChangesIt(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.SetRandomResults([]int{1, 3, 4})
	gem := putGemBazaarOntoBattlefield(t, g)
	player := g.GetPlayer(gametest.PlayerA)

	if gem.ChosenColor != core.Colorless {
		t.Fatalf("ETB color choice happened before its trigger resolved: got %s", gem.ChosenColor)
	}
	g.ResolveStack()
	if gem.ChosenColor != core.Blue {
		t.Fatalf("first random color = %s, want Blue", gem.ChosenColor)
	}

	if err := g.TapForMana(player.PlayerID(), gem.ID()); err != nil {
		t.Fatalf("first Gem Bazaar activation: %v", err)
	}
	if got := player.ManaPool().Count(core.Blue); got != 1 {
		t.Fatalf("first activation produced %d blue mana, want 1", got)
	}
	if gem.ChosenColor != core.Red {
		t.Fatalf("color after first activation = %s, want Red", gem.ChosenColor)
	}
	if got := g.GetStack().Size(); got != 0 {
		t.Fatalf("mana ability used the stack: size = %d", got)
	}

	gem.Tapped = false
	if err := g.TapForMana(player.PlayerID(), gem.ID()); err != nil {
		t.Fatalf("second Gem Bazaar activation: %v", err)
	}
	if got := player.ManaPool().Count(core.Red); got != 1 {
		t.Fatalf("second activation produced %d red mana, want 1", got)
	}
	if gem.ChosenColor != core.Green {
		t.Fatalf("color after second activation = %s, want Green", gem.ChosenColor)
	}
}

func TestGemBazaar_CanAffordOnlyCurrentChosenColor(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.SetRandomResults([]int{2})
	gem := putGemBazaarOntoBattlefield(t, g)
	g.ResolveStack()
	player := g.GetPlayer(gametest.PlayerA)

	if gem.ChosenColor != core.Black {
		t.Fatalf("chosen color = %s, want Black", gem.ChosenColor)
	}
	if !g.CanAfford(player.PlayerID(), core.ManaCost{Black: 1}, nil) {
		t.Fatal("solver did not recognize Gem Bazaar's current black production")
	}
	if g.CanAfford(player.PlayerID(), core.ManaCost{White: 1}, nil) {
		t.Fatal("solver treated Gem Bazaar as producing an unchosen color")
	}
}

func TestGemBazaarUpdatesFellwarStone(t *testing.T) {
	t.Run("when it enters", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.SetRandomResults([]int{1})
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Fellwar Stone")
		putGemBazaarOntoBattlefield(t, g)
		g.ResolveStack()

		stoneController := g.GetPlayer(gametest.PlayerB).PlayerID()
		if !g.CanAfford(stoneController, core.ManaCost{Blue: 1}, nil) {
			t.Fatal("Fellwar Stone cannot produce Gem Bazaar's first chosen color")
		}
	})

	t.Run("after producing mana", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.SetRandomResults([]int{1, 3})
		putGemBazaarOntoBattlefield(t, g)
		g.ResolveStack()
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Fellwar Stone")

		gemController := g.GetPlayer(gametest.PlayerA).PlayerID()
		stoneController := g.GetPlayer(gametest.PlayerB).PlayerID()
		if !g.CanAfford(stoneController, core.ManaCost{Blue: 1}, nil) {
			t.Fatal("Fellwar Stone cannot produce blue before Gem Bazaar changes color")
		}
		if err := g.AutoTapForCost(gemController, core.ManaCost{Blue: 1}); err != nil {
			t.Fatalf("auto-tap Gem Bazaar for blue: %v", err)
		}
		if g.CanAfford(stoneController, core.ManaCost{Blue: 1}, nil) {
			t.Error("Fellwar Stone still offers blue after Gem Bazaar changes color")
		}
		if err := g.AutoTapForCost(stoneController, core.ManaCost{Red: 1}); err != nil {
			t.Fatalf("auto-tap Fellwar Stone for Gem Bazaar's new color: %v", err)
		}
	})
}
