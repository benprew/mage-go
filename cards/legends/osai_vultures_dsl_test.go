package legends

import (
	"fmt"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestOsaiVulturesTypedActivation(t *testing.T) {
	card, err := mage.CreateCard("Osai Vultures")
	if err != nil {
		t.Fatal(err)
	}
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	var ability mage.ActivatedAbility
	for _, candidate := range card.Abilities() {
		if activated, ok := mage.UnwrapAbility(candidate).(mage.ActivatedAbility); ok {
			ability = activated
		}
	}
	if ability == nil || ability.SorcerySpeed() || len(ability.Targets()) != 0 {
		t.Fatal("expected an untargeted instant-speed activation")
	}
	meta := mage.InspectDefinition(ability.Effects()[0])
	if meta.Role != "until" || meta.Properties.PowerBoost != 1 || meta.Properties.ToughnessBoost != 1 || meta.Properties.Mass || meta.Properties.Outcome != mage.OutcomeBenefit {
		t.Fatalf("activation metadata = %+v", meta)
	}
}

func TestOsaiVulturesCounterPayment(t *testing.T) {
	for _, counters := range []int{0, 1, 2, 4} {
		t.Run(fmt.Sprint(counters), func(t *testing.T) {
			g := gametest.NewTestGame(t)
			a := g.GetPlayer(gametest.PlayerA)
			id := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Osai Vultures")
			source := g.MutablePermanent(id)
			source.AddCounter(core.Carrion, counters)
			source.AddCounter(core.Charge, 2)
			source.Tapped = true
			g.SetStep(core.Upkeep)
			g.SetActivePlayerIndex(1)
			for paid := 0; paid <= counters/2; paid++ {
				err := g.ActivateAbilityByIndex(a.PlayerID(), id, 1, nil)
				if paid == counters/2 {
					if err == nil || g.StackSize() != paid {
						t.Fatal("activation with insufficient carrion counters changed the stack")
					}
					break
				}
				if err != nil {
					t.Fatal(err)
				}
				g.AssertCounterCount(gametest.PlayerA, "Osai Vultures", core.Carrion, counters-2*(paid+1))
				g.AssertPowerToughness(gametest.PlayerA, "Osai Vultures", 1, 1)
			}
			g.AssertCounterCount(gametest.PlayerA, "Osai Vultures", core.Carrion, counters%2)
			g.AssertCounterCount(gametest.PlayerA, "Osai Vultures", core.Charge, 2)
			g.ResolveStack()
			g.AssertPowerToughness(gametest.PlayerA, "Osai Vultures", 1+counters/2, 1+counters/2)
			g.GetEffects().RemoveEndOfTurn()
			g.ApplyContinuousEffects()
			g.AssertPowerToughness(gametest.PlayerA, "Osai Vultures", 1, 1)
		})
	}
}
