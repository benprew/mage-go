package gametest

import (
	"sync"
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

var graveActOnce sync.Once

func registerGraveyardActivatedCards() {
	graveActOnce.Do(func() {
		// Sanitarium-Skeleton-style: {2}{B}: Return this card from your
		// graveyard to your hand. Activate only as a sorcery.
		if !mage.CardRegistered("Test Skeleton") {
			mage.Register("Test Skeleton", func() mage.Card {
				return mage.NewCreature("Test Skeleton", "{B}", 1, 2,
					mage.WithSubTypes("Skeleton"),
					mage.WithGraveyardActivatedAbility(
						mage.ReturnSourceToHand(),
						mage.ManaCostOf("{2}{B}"),
					),
				)
			})
		}
		// Ghoulcaller's-Accomplice-style: {3}{B}, Exile this card from your
		// graveyard: Create a 2/2 black Zombie creature token.
		if !mage.CardRegistered("Test Accomplice") {
			mage.Register("Test Accomplice", func() mage.Card {
				return mage.NewCreature("Test Accomplice", "{1}{B}", 2, 2,
					mage.WithSubTypes("Human", "Rogue"),
					mage.WithGraveyardActivatedAbility(
						mage.GainLife(2),
						mage.ManaCostOf("{3}{B}"),
						mage.WithCost(mage.ExileSelfFromGraveyardCost()),
					),
				)
			})
		}
	})
}

// TestGraveyardActivated_ReturnSelf verifies a graveyard-zone activated
// ability can be invoked while the card is in the graveyard, paying mana
// and putting the effect on the stack.
func TestGraveyardActivated_ReturnSelf(t *testing.T) {
	registerGraveyardActivatedCards()

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneGraveyard, PlayerA, "Test Skeleton")

	pid := tg.getPlayerID(PlayerA)

	tg.StopAt(1, core.PrecombatMain)
	tg.Execute()

	pool := tg.GetPlayer(PlayerA).ManaPool()
	pool.Add(core.Black, 5)
	pool.Add(core.Colorless, 5)

	skel, idx, ok := tg.FindGraveyardActivatableCard(pid, "Test Skeleton")
	if !ok {
		t.Fatal("graveyard activatable not found")
	}
	if err := tg.ActivateGraveyardAbility(pid, skel, idx, nil); err != nil {
		t.Fatalf("ActivateGraveyardAbility: %v", err)
	}
	tg.ResolveStack()

	tg.AssertHandCount(PlayerA, "Test Skeleton", 1)
	tg.AssertGraveyardCount(PlayerA, "Test Skeleton", 0)
}

// TestGraveyardActivated_ExileSelfCost verifies the
// ExileSelfFromGraveyardCost moves the card to exile and the effect
// resolves.
func TestGraveyardActivated_ExileSelfCost(t *testing.T) {
	registerGraveyardActivatedCards()

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneGraveyard, PlayerA, "Test Accomplice")
	pid := tg.getPlayerID(PlayerA)

	tg.StopAt(1, core.PrecombatMain)
	tg.Execute()

	pool := tg.GetPlayer(PlayerA).ManaPool()
	pool.Add(core.Black, 5)
	pool.Add(core.Colorless, 5)

	acc, idx, ok := tg.FindGraveyardActivatableCard(pid, "Test Accomplice")
	if !ok {
		t.Fatal("graveyard activatable not found")
	}
	if err := tg.ActivateGraveyardAbility(pid, acc, idx, nil); err != nil {
		t.Fatalf("ActivateGraveyardAbility: %v", err)
	}
	tg.ResolveStack()

	tg.AssertExileCount("Test Accomplice", 1)
	tg.AssertGraveyardCount(PlayerA, "Test Accomplice", 0)
	tg.AssertLife(PlayerA, 22)
}
