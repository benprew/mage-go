package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestControllerHandSizeIsActivation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewLand("Hand gate", WithAbility(Activate(Costs(TapCost(This)), DrawFor(You, Fixed(1)), ActivationIf(ControllerHandSizeIs(7)))))
	card.SetOwner(b.PlayerID())
	g.PutOnBattlefield(card, a.PlayerID())
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	for i := range 8 {
		a.AddToHand(NewArtifact("Hand card", ""))
		if i == 5 || i == 6 {
			want := i == 6
			if got := card.Abilities()[0].(*ActionDefinition).CanActivate(a.PlayerID(), g); got != want {
				t.Fatalf("hand size %d: can activate = %v, want %v", i+1, got, want)
			}
		}
	}
	if card.Abilities()[0].(*ActionDefinition).CanActivate(a.PlayerID(), g) {
		t.Fatal("eight cards allowed activation")
	}
	clone := g.Clone()
	if _, ok := clone.GetPlayer(a.PlayerID()).RemoveFromHand(a.Hand()[0].ID()); !ok {
		t.Fatal("clone hand removal failed")
	}
	if !clone.FindPermanent(card.ID()).Card.Abilities()[0].(*ActionDefinition).CanActivate(a.PlayerID(), clone) {
		t.Fatal("clone did not read its own hand")
	}
	if card.Abilities()[0].(*ActionDefinition).CanActivate(a.PlayerID(), g) {
		t.Fatal("clone changed original hand")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 0, nil); err == nil || g.FindPermanent(card.ID()).Tapped || g.StackSize() != 0 {
		t.Fatal("failed activation changed payment state")
	}
	if err := clone.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 0, nil); err != nil || !clone.FindPermanent(card.ID()).Tapped || clone.StackSize() != 1 {
		t.Fatal("seven-card activation did not pay and enter the stack", err)
	}
}

func TestControllerHandSizeIsStateAndValidation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	a.AddToHand(NewArtifact("Card", ""))
	condition := ControllerHandSizeIs(1)
	if !condition.checkState(g, StateContext{ControllerID: a.PlayerID()}) {
		t.Fatal("absent source changed controller hand check")
	}
	if condition.checkState(g, StateContext{ControllerID: b.PlayerID()}) || condition.checkState(g, StateContext{ControllerID: uuid.Nil}) {
		t.Fatal("condition used the wrong or missing controller")
	}
	if got := InspectDefinition(condition); got.Role != "state-condition" || got.Text != "controller hand size is 1" {
		t.Fatalf("inspection = %+v", got)
	}
	invalid := NewLand("Invalid", WithAbility(Activate(Costs(), DrawFor(You, Fixed(1)), ActivationIf(ControllerHandSizeIs(-1)))))
	if err := ValidateCardDefinition(invalid.Name(), invalid); err == nil {
		t.Fatal("negative hand size passed registration")
	}
}
