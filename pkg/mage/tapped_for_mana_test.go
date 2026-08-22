package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func addTappedForManaWatcher(g *Game, controller Player) {
	watcher := NewEnchantment("Tapped for Mana Watcher", "{0}",
		WithAbility(NewTriggered(EvtTappedForMana, false, GainLife(1))),
	)
	watcher.SetOwner(controller.PlayerID())
	g.PutOnBattlefield(watcher, controller.PlayerID())
}

func resolvePendingTriggers(g *Game) {
	g.PutTriggersOnStack()
	g.ResolveStack()
}

func TestTappedForManaEvent(t *testing.T) {
	t.Run("fires after a tap mana ability produces mana", func(t *testing.T) {
		g, player, _ := randomTestGame()
		addTappedForManaWatcher(g, player)
		land := NewLand("Test Forest", WithManaAbility(Green))
		land.SetOwner(player.PlayerID())
		perm := g.PutOnBattlefield(land, player.PlayerID())

		if err := g.TapForMana(player.PlayerID(), perm.ID()); err != nil {
			t.Fatalf("TapForMana: %v", err)
		}
		resolvePendingTriggers(g)

		if got := player.Life(); got != 21 {
			t.Fatalf("life = %d, want 21", got)
		}
	})

	t.Run("does not fire when a permanent is merely tapped", func(t *testing.T) {
		g, player, _ := randomTestGame()
		addTappedForManaWatcher(g, player)
		land := NewLand("Utility Land")
		land.SetOwner(player.PlayerID())
		perm := g.PutOnBattlefield(land, player.PlayerID())

		g.TapPermanent(perm)
		resolvePendingTriggers(g)

		if got := player.Life(); got != 20 {
			t.Fatalf("life = %d, want 20", got)
		}
	})

	t.Run("does not fire when the mana ability produces no mana", func(t *testing.T) {
		g, player, _ := randomTestGame()
		addTappedForManaWatcher(g, player)
		land := NewLand("Empty Mana Land", WithDynamicManaAbility(
			func(GameReader, uuid.UUID) []ManaProduction { return nil },
		))
		land.SetOwner(player.PlayerID())
		perm := g.PutOnBattlefield(land, player.PlayerID())

		if err := g.TapForMana(player.PlayerID(), perm.ID()); err != nil {
			t.Fatalf("TapForMana: %v", err)
		}
		resolvePendingTriggers(g)

		if got := player.Life(); got != 20 {
			t.Fatalf("life = %d, want 20", got)
		}
	})

	t.Run("fires for a tap activated ability that adds mana", func(t *testing.T) {
		g, player, _ := randomTestGame()
		addTappedForManaWatcher(g, player)
		land := NewLand("Activated Mana Land",
			WithActivatedAbility(AddMana(Colorless, 2), Tap()),
		)
		land.SetOwner(player.PlayerID())
		perm := g.PutOnBattlefield(land, player.PlayerID())

		if err := g.ActivateAbilityByIndex(player.PlayerID(), perm.ID(), 0, nil); err != nil {
			t.Fatalf("ActivateAbilityByIndex: %v", err)
		}
		resolvePendingTriggers(g)

		if got := player.Life(); got != 21 {
			t.Fatalf("life = %d, want 21", got)
		}
	})

	t.Run("fires when automatic payment taps a mana source", func(t *testing.T) {
		g, player, _ := randomTestGame()
		addTappedForManaWatcher(g, player)
		land := NewLand("Automatic Mana Land", WithManaAbility(Blue))
		land.SetOwner(player.PlayerID())
		g.PutOnBattlefield(land, player.PlayerID())

		if err := g.AutoTapForCost(player.PlayerID(), ManaCost{Blue: 1}); err != nil {
			t.Fatalf("AutoTapForCost: %v", err)
		}
		resolvePendingTriggers(g)

		if got := player.Life(); got != 21 {
			t.Fatalf("life = %d, want 21", got)
		}
	})

	t.Run("does not fire for a nonmana tap ability", func(t *testing.T) {
		g, player, _ := randomTestGame()
		addTappedForManaWatcher(g, player)
		land := NewLand("Utility Land",
			WithActivatedAbility(UntapSource(), Tap()),
		)
		land.SetOwner(player.PlayerID())
		perm := g.PutOnBattlefield(land, player.PlayerID())

		if err := g.ActivateAbilityByIndex(player.PlayerID(), perm.ID(), 0, nil); err != nil {
			t.Fatalf("ActivateAbilityByIndex: %v", err)
		}
		resolvePendingTriggers(g)

		if got := player.Life(); got != 20 {
			t.Fatalf("life = %d, want 20", got)
		}
	})
}
