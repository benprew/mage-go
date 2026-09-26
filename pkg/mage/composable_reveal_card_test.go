package mage

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRevealSavedHandCardPublicRecord(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewLand("Forest")
	a.AddToLibrary(card)
	binding := BindCard("drawn")
	ctx := &EffectContext{Game: g, Controller: b.PlayerID()}
	target := DeclarePlayerTarget("drawer")
	ctx.targetBindings = map[*targetDeclaration][]capturedPermanent{target.targetDeclaration: {{id: a.PlayerID()}}}
	if err := Sequence(DrawFor(target, Fixed(1), DrawCardInto(binding)), RevealCard(FromCard(binding))).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	reveals := g.PublicReveals()
	if len(reveals) != 1 || reveals[0].CardID != card.ID() || reveals[0].Owner != a.PlayerID() || reveals[0].Name != "Forest" || len(reveals[0].Types) != 1 || reveals[0].Types[0] != core.TypeLand {
		t.Fatalf("public reveal = %+v", reveals)
	}
	if len(a.Hand()) != 1 || a.Hand()[0].ID() != card.ID() {
		t.Fatal("reveal moved the card")
	}
	reveals[0].Types[0] = core.TypeArtifact
	reveals[0].Card.AddType(core.TypeArtifact)
	if g.PublicReveals()[0].Types[0] != core.TypeLand || g.PublicReveals()[0].Card.HasType(core.TypeArtifact) {
		t.Fatal("caller mutated public history")
	}
	clone := g.Clone()
	if err := RevealCard(FromCard(binding)).Apply(&EffectContext{Game: clone, Controller: b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if len(g.PublicReveals()) != 1 || len(clone.PublicReveals()) != 2 {
		t.Fatal("clone reveal history was shared")
	}
}

func TestRevealSavedHandCardAbsenceAndVisit(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	card := NewArtifact("Drawn", "")
	a.AddToLibrary(card)
	binding := BindCard("drawn")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := RevealCard(FromCard(binding)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(g.PublicReveals()) != 0 {
		t.Fatal("absent binding revealed a card")
	}
	if err := DrawFor(You, Fixed(1), DrawCardInto(binding)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if _, ok := a.RemoveFromHand(card.ID()); !ok {
		t.Fatal("drawn card missing")
	}
	a.AddToHand(card)
	if err := RevealCard(FromCard(binding)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(g.PublicReveals()) != 0 {
		t.Fatal("old hand visit was revealed")
	}
	if err := DrawFor(You, Fixed(1), DrawCardInto(binding)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if err := RevealCard(FromCard(binding)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(g.PublicReveals()) != 0 {
		t.Fatal("empty draw reused an old card")
	}
}

func TestRevealCardDeclaration(t *testing.T) {
	binding := BindCard("drawn")
	valid := NewInstant("Valid", "", Spell(Sequence(DrawFor(You, Fixed(1), DrawCardInto(binding)), RevealCard(FromCard(binding)))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		effect ResolvingEffect
		want   string
	}{
		{RevealCard(nil), "card"},
		{RevealCard(FromCard(binding)), "producer"},
		{Sequence(IfState(Controls(IsCreature), DrawFor(You, Fixed(1), DrawCardInto(binding))), RevealCard(FromCard(binding))), "producer"},
	} {
		card := NewInstant("Invalid", "", Spell(tc.effect))
		if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("validation = %v; want %q", err, tc.want)
		}
	}
	info := InspectDefinition(RevealCard(FromCard(binding)))
	if info.Role != "resolving-effect" {
		t.Fatalf("inspection = %+v", info)
	}
}
