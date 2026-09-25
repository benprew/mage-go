package antiquities

import (
	"fmt"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestArmageddonClockCounterSourceLifetime(t *testing.T) {
	for _, activator := range []gametest.PlayerRef{gametest.PlayerA, gametest.PlayerB} {
		for _, scenario := range []string{"present", "leaves", "returns", "empty"} {
			t.Run(fmt.Sprint(activator)+"/"+scenario, func(t *testing.T) {
				g := gametest.NewTestGame(t)
				source := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Armageddon Clock")
				card := g.FindPermanent(source).Card
				g.FindPermanent(source).AddCounter(core.Doom, 2)
				if scenario == "empty" {
					g.FindPermanent(source).RemoveCounter(core.Doom, 2)
				}
				initial := g.FindPermanent(source).Counters[core.Doom]
				g.FindPermanent(source).Tapped = true
				g.SetStep(core.Upkeep)
				g.SetActivePlayerIndex(1)
				player := g.GetPlayer(activator)
				player.ManaPool().Add(core.Green, 4)
				if err := g.ActivateAbilityByIndex(player.PlayerID(), source, 2, nil); err != nil {
					t.Fatal(err)
				}
				if player.ManaPool().TotalMana() != 0 || g.FindPermanent(source).Counters[core.Doom] != initial {
					t.Fatal("pay before removing the counter")
				}
				if g.StackSize() != 1 || len(g.StackPeek().Targets) != 0 || g.StackPeek().Controller != player.PlayerID() {
					t.Fatal("the activator controls the untargeted ability")
				}
				if scenario == "leaves" || scenario == "returns" {
					g.BouncePermanentToHand(g.FindPermanent(source))
					if scenario == "returns" {
						g.GetPlayer(gametest.PlayerA).RemoveFromHand(source)
						g.PutOnBattlefield(card, g.GetPlayer(gametest.PlayerA).PlayerID())
						g.FindPermanent(source).AddCounter(core.Doom, 2)
					}
				}
				g.ResolveStack()
				switch scenario {
				case "present":
					g.AssertCounterCount(gametest.PlayerA, "Armageddon Clock", core.Doom, 1)
				case "empty":
					g.AssertCounterCount(gametest.PlayerA, "Armageddon Clock", core.Doom, 0)
				case "returns":
					g.AssertCounterCount(gametest.PlayerA, "Armageddon Clock", core.Doom, 2)
				}
			})
		}
	}
}

func TestArmageddonClockActivationRestrictions(t *testing.T) {
	for _, activator := range []gametest.PlayerRef{gametest.PlayerA, gametest.PlayerB} {
		for _, active := range []int{0, 1} {
			t.Run(fmt.Sprintf("activator%d/active%d", activator, active), func(t *testing.T) {
				g := gametest.NewTestGame(t)
				source := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Armageddon Clock")
				g.FindPermanent(source).AddCounter(core.Doom, 2)
				g.SetActivePlayerIndex(active)
				player := g.GetPlayer(activator)
				player.ManaPool().Add(core.Blue, 3)
				g.SetStep(core.Upkeep)
				if err := g.ActivateAbilityByIndex(player.PlayerID(), source, 2, nil); err == nil || player.ManaPool().TotalMana() != 3 || g.StackSize() != 0 {
					t.Fatal("failed payment must leave mana and stack unchanged")
				}
				player.ManaPool().Add(core.Blue, 1)
				g.SetStep(core.PrecombatMain)
				if err := g.ActivateAbilityByIndex(player.PlayerID(), source, 2, nil); err == nil || player.ManaPool().TotalMana() != 4 || g.StackSize() != 0 {
					t.Fatal("reject activation outside upkeep before payment")
				}
				g.SetStep(core.Upkeep)
				if err := g.ActivateAbilityByIndex(player.PlayerID(), source, 2, nil); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				g.AssertCounterCount(gametest.PlayerA, "Armageddon Clock", core.Doom, 1)
			})
		}
	}
}
