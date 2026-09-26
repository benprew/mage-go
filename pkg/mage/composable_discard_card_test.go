package mage

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestDiscardCardSavedDrawAndReplacement(t *testing.T) {
	a := &acceptingMayPlayer{NewBasePlayer("A")}
	g := NewGame(a, NewBasePlayer("B"))
	drawn, other := NewArtifact("Drawn", ""), NewArtifact("Other", "")
	a.AddToLibrary(drawn)
	a.AddToHand(other)
	binding := BindCard("drawn")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := Sequence(DrawFor(You, Fixed(1), DrawCardInto(binding)), DiscardCard(FromCard(binding))).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(a.Hand()) != 1 || a.Hand()[0].ID() != other.ID() || len(a.Graveyard()) != 1 || a.Graveyard()[0].ID() != drawn.ID() {
		t.Fatalf("saved draw: hand=%s graveyard=%s want hand=%s graveyard=%s", a.Hand()[0].Name(), a.Graveyard()[0].Name(), other.Name(), drawn.Name())
	}
	if err := DiscardCard(FromCard(binding)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(a.Graveyard()) != 1 {
		t.Fatal("departed card was discarded again")
	}

	replaced := NewArtifact("Replaced", "")
	a.AddToLibrary(replaced)
	if err := DrawFor(You, Fixed(1), DrawCardInto(binding)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(NewArtifact("Library", "", WithStaticAbility(DiscardToLibraryReplacement())), a.PlayerID())
	if err := DiscardCard(FromCard(binding)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(a.Library()) != 1 || a.Library()[0].ID() != replaced.ID() || len(a.Hand()) != 1 || len(a.Graveyard()) != 1 {
		t.Fatal("saved discard bypassed replacement or event service")
	}
}

func TestDiscardCardVisitAndClone(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	card := NewArtifact("Drawn", "")
	a.AddToLibrary(card)
	binding := BindCard("drawn")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := DrawFor(You, Fixed(1), DrawCardInto(binding)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	if _, ok := a.RemoveFromHand(card.ID()); !ok {
		t.Fatal("missing card")
	}
	a.AddToHand(card)
	if err := DiscardCard(FromCard(binding)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(a.Hand()) != 1 || len(a.Graveyard()) != 0 {
		t.Fatal("old binding followed a new hand visit")
	}
	if err := DiscardCard(FromCard(binding)).Apply(&EffectContext{Game: clone, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if len(clone.GetPlayer(a.PlayerID()).Graveyard()) != 1 || len(a.Graveyard()) != 0 {
		t.Fatal("clone did not retain an independent binding")
	}
}

func TestDiscardCardOwnerAndZone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewArtifact("B's card", "")
	b.AddToLibrary(card)
	binding := BindCard("drawn")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	target := DeclarePlayerTarget("player")
	ctx.targetBindings = map[*targetDeclaration][]capturedPermanent{target.targetDeclaration: {{id: b.PlayerID()}}}
	if err := DrawFor(target, Fixed(1), DrawCardInto(binding)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if err := DiscardCard(FromCard(binding)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(a.Graveyard()) != 0 || len(b.Graveyard()) != 1 || b.Graveyard()[0].ID() != card.ID() {
		t.Fatal("discard used the ability controller instead of the card owner")
	}
	ctx.Game.resolution.cardBindings[binding] = capturedCard{id: card.ID(), visit: g.graveyardVisit(card.ID()), zone: core.ZoneGraveyard}
	if err := DiscardCard(FromCard(binding)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(b.Graveyard()) != 1 {
		t.Fatal("graveyard reference was treated as a hand card")
	}
}

func TestDiscardCardDefinition(t *testing.T) {
	binding := BindCard("drawn")
	valid := NewInstant("Valid", "", Spell(Sequence(DrawFor(You, Fixed(1), DrawCardInto(binding)), DiscardCard(FromCard(binding)))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		effect ResolvingEffect
		want   string
	}{
		{DiscardCard(nil), "card"},
		{DiscardCard(FromCard(binding)), "producer"},
		{Sequence(IfState(Controls(IsCreature), DrawFor(You, Fixed(1), DrawCardInto(binding))), DiscardCard(FromCard(binding))), "producer"},
	} {
		invalid := NewInstant("Invalid", "", Spell(tc.effect))
		if err := ValidateCardDefinition(invalid.Name(), invalid); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("validation = %v; want %q", err, tc.want)
		}
	}
	info := InspectDefinition(DiscardCard(FromCard(binding)))
	if info.Role != "resolving-effect" || info.Properties.Outcome != OutcomeDetriment {
		t.Fatalf("inspection = %+v", info)
	}
}
