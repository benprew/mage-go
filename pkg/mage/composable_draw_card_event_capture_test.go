package mage

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestControllerDrawsCardIntoExactHandVisit(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	drawer, drawn := BindPlayer("drawer"), BindCard("drawn")
	source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		ControllerDrawsCardInto(drawer, drawn),
		IfCardHasType(FromCard(drawn), core.TypeLand, GainLifeFor(FromPlayer(drawer), Fixed(2))))))
	g.PutOnBattlefield(source, a.PlayerID())
	land := NewLand("Land")
	a.AddToLibrary(land)
	a.AddToLibrary(NewArtifact("Artifact", ""))
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	g.PlayerDrawCard(a)
	g.PlayerDrawCard(a)
	g.PutTriggersOnStack()
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 22 || a.Life() != 20 {
		t.Fatal("draw captures changed across clone")
	}
	if _, ok := a.DiscardCard(land.ID()); !ok {
		t.Fatal("drawn card absent from hand")
	}
	a.AddToHand(land)
	g.ResolveStack()
	if a.Life() != 20 {
		t.Fatal("draw capture followed a new hand visit")
	}
}

func TestControllerDrawsCardIntoDeclaration(t *testing.T) {
	drawer, drawn := BindPlayer("drawer"), BindCard("drawn")
	for _, tc := range []struct {
		name string
		card Card
		want string
	}{
		{"valid", NewEnchantment("Valid", "", WithAbility(WhenEvent(ControllerDrawsCardInto(drawer, drawn), IfCardHasType(FromCard(drawn), core.TypeLand, GainLifeFor(You, Fixed(1)))))), ""},
		{"nil", NewEnchantment("Nil", "", WithAbility(WhenEvent(ControllerDrawsCardInto(drawer, nil), GainLifeFor(You, Fixed(1))))), "non-nil"},
		{"missing", NewEnchantment("Missing", "", WithAbility(WhenEvent(ControllerDrawsCard(drawer), IfCardHasType(FromCard(drawn), core.TypeLand, GainLifeFor(You, Fixed(1)))))), "producer"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateCardDefinition(tc.card.Name(), tc.card)
			if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("validation = %v", err)
			}
		})
	}
}

func TestControllerDrawsCardIntoCopyAndAbsence(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	drawn := BindCard("drawn")
	source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		ControllerDrawsCardInto(BindPlayer("drawer"), drawn),
		IfCardHasType(FromCard(drawn), core.TypeLand, GainLifeFor(You, Fixed(2))))))
	g.PutOnBattlefield(source, a.PlayerID())
	a.AddToLibrary(NewLand("Land"))
	g.PlayerDrawCard(a)
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("draw did not create one trigger")
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], b.PlayerID(), false)
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 22 {
		t.Fatal("copied trigger lost its exact drawn card")
	}
	g.PlayerDrawCard(a)
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("empty-library draw created a trigger")
	}
	phased := g.PhaseOut(source.ID())
	a.AddToLibrary(NewLand("Later"))
	g.PlayerDrawCard(a)
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased source observed a draw")
	}
	g.PhaseIn(phased)
}
