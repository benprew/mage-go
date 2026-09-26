package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestSavedCardTypeBranch(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	land := NewLand("Forest")
	artifact := NewArtifact("Relic", "")
	a.AddToLibrary(land)
	b := BindCard("drawn")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	check := Sequence(
		DrawFor(You, Fixed(1), DrawCardInto(b)),
		IfCardHasType(FromCard(b), core.TypeLand, GainLifeFor(You, Fixed(2))),
		IfCardLacksType(FromCard(b), core.TypeLand, GainLifeFor(You, Fixed(3))),
	)
	if err := check.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if got := a.Life(); got != 22 {
		t.Fatalf("land life = %d", got)
	}
	a.AddToLibrary(artifact)
	if err := check.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if got := a.Life(); got != 25 {
		t.Fatalf("nonland life = %d", got)
	}
	if err := check.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if got := a.Life(); got != 25 {
		t.Fatalf("absent card matched negative condition: %d", got)
	}
}

func TestSavedCardTypeBranchVisitAndClone(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	card := NewLand("Forest")
	a.AddToLibrary(card)
	b := BindCard("drawn")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := DrawFor(You, Fixed(1), DrawCardInto(b)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	if _, ok := a.RemoveFromHand(card.ID()); !ok {
		t.Fatal("missing card")
	}
	a.AddToHand(card)
	branch := IfCardHasType(FromCard(b), core.TypeLand, GainLifeFor(You, Fixed(2)))
	if err := branch.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if got := a.Life(); got != 20 {
		t.Fatalf("old visit matched: %d", got)
	}
	if err := branch.Apply(&EffectContext{Game: clone, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if got := clone.GetPlayer(a.PlayerID()).Life(); got != 22 {
		t.Fatalf("clone life = %d", got)
	}
}

func TestSavedCardTypeBranchDeclaration(t *testing.T) {
	b := BindCard("drawn")
	inner := BindCard("inner")
	valid := NewInstant("Valid", "", Spell(Sequence(DrawFor(You, Fixed(1), DrawCardInto(b)), IfCardLacksType(FromCard(b), core.TypeLand, DiscardCard(FromCard(b))))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		effect ResolvingEffect
		want   string
	}{
		{IfCardHasType(nil, core.TypeLand, GainLifeFor(You, Fixed(1))), "card"},
		{IfCardLacksType(FromCard(b), core.TypeLand, GainLifeFor(You, Fixed(1))), "producer"},
		{IfCardHasType(FromCard(b), core.CardType(-1), GainLifeFor(You, Fixed(1))), "type"},
		{IfCardHasType(FromCard(b), core.TypeLand, nil), "then"},
		{Sequence(DrawFor(You, Fixed(1), DrawCardInto(b)), IfCardHasType(FromCard(b), core.TypeLand, DrawFor(You, Fixed(1), DrawCardInto(inner))), DiscardCard(FromCard(inner))), "producer"},
	} {
		card := NewInstant("Invalid", "", Spell(tc.effect))
		if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("validation = %v; want %q", err, tc.want)
		}
	}
	if got := InspectDefinition(IfCardLacksType(FromCard(b), core.TypeLand, DiscardCard(FromCard(b)))); got.Role != "resolution-branch" {
		t.Fatalf("role = %s", got.Role)
	}
}

func TestCardTypeBranchGraveyardTargetVisit(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	land := NewLand("Forest")
	a.AddToGraveyard(land)
	target := DeclareGraveyardCardTarget("card")
	ability := Spell(IfCardHasType(target, core.TypeLand, GainLifeFor(You, Fixed(2))), WithTargets(target))
	spell := NewInstant("Check", "", WithAbility(ability))
	spell.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
	obj := &StackObject{ID: uuid.New(), SourceID: spell.ID(), Card: spell, Controller: a.PlayerID(), Targets: []uuid.UUID{land.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	g.ResolveTopOfStack()
	if got := a.Life(); got != 22 {
		t.Fatalf("legal target life = %d", got)
	}
	obj = &StackObject{ID: uuid.New(), SourceID: spell.ID(), Card: spell, Controller: a.PlayerID(), Targets: []uuid.UUID{land.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	moved, ok := g.MoveFromGraveyard(a.PlayerID(), land.ID(), core.ZoneHand)
	if !ok {
		t.Fatal("move from graveyard failed")
	}
	a.AddToHand(moved)
	returned, ok := a.RemoveFromHand(land.ID())
	if !ok {
		t.Fatal("remove from hand failed")
	}
	a.AddToGraveyard(returned)
	g.ResolveTopOfStack()
	if got := a.Life(); got != 22 {
		t.Fatalf("returned target life = %d", got)
	}
}
