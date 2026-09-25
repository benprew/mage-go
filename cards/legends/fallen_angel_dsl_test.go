package legends

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestFallenAngelTypedActivation(t *testing.T) {
	card, err := mage.CreateCard("Fallen Angel")
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
	if meta.Role != "until" || meta.Properties.PowerBoost != 2 || meta.Properties.ToughnessBoost != 1 || meta.Properties.Mass || meta.Properties.Outcome != mage.OutcomeBenefit {
		t.Fatalf("activation metadata = %+v", meta)
	}
}

func TestFallenAngelCanSacrificeItself(t *testing.T) {
	g := gametest.NewTestGame(t)
	a := g.GetPlayer(gametest.PlayerA)
	id := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Fallen Angel")
	g.MutablePermanent(id).Tapped = true
	g.SetStep(core.Upkeep)
	g.SetActivePlayerIndex(1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), id, 0, nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(id) != nil || g.StackSize() != 1 {
		t.Fatal("sacrifice must be paid before resolution")
	}
	g.AssertGraveyardCount(gametest.PlayerA, "Fallen Angel", 1)
	g.ResolveStack()
	g.AssertGraveyardCount(gametest.PlayerA, "Fallen Angel", 1)
	if g.StackSize() != 0 {
		t.Fatal("ability did not resolve after source sacrifice")
	}
}
