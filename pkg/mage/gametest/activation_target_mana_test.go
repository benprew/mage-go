package gametest

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestActivationTargetManaIntegration(t *testing.T) {
	tg := NewTestGame(t)
	a, b := tg.GetPlayer(PlayerA), tg.GetPlayer(PlayerB)
	targets := dsl.TargetPermanents("creatures", 0, 3, dsl.IsCreature)
	card := dsl.NewArtifact("Synthetic Target Payment", "",
		dsl.WithAbility(dsl.Activate(
			dsl.Costs(dsl.Mana("{2}"), dsl.TapCost(dsl.This)),
			dsl.TapEach(targets),
			dsl.WithTargets(targets), dsl.ActivationManaPerTarget(core.Red, 1))))
	source := tg.PutOnBattlefield(card, a.PlayerID())
	victims := make([]uuid.UUID, 0, 2)
	for _, name := range []string{"First", "Second"} {
		victims = append(victims, tg.PutOnBattlefield(dsl.NewCreature(name, "", 1, 1), b.PlayerID()).ID())
	}
	a.ManaPool().Add(core.Colorless, 2)
	a.ManaPool().Add(core.Red, 2)
	if err := tg.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, victims); err != nil {
		t.Fatal(err)
	}
	tg.ResolveTopOfStack()
	for _, id := range victims {
		if !tg.FindPermanent(id).Tapped {
			t.Fatal("legal target was not tapped")
		}
	}
	if !tg.FindPermanent(source.ID()).Tapped || a.ManaPool().TotalMana() != 0 {
		t.Fatal("wrong combined activation payment")
	}
}
