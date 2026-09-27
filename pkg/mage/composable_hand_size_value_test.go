package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestHandSizeOfReadsCurrentSelectedPlayer(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	a.AddToHand(NewArtifact("Own card", ""))
	for range 3 {
		b.AddToHand(NewArtifact("Opposing card", ""))
	}
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if got := HandSizeOf(You).resolveValue(ctx); got != 1 {
		t.Fatalf("controller hand size = %d, want 1", got)
	}
	if got := HandSizeOf(ActivePlayer).resolveValue(ctx); got != 1 {
		t.Fatalf("active player hand size = %d, want 1", got)
	}
	g.SetActivePlayerIndex(1)
	if got := HandSizeOf(ActivePlayer).resolveValue(ctx); got != 3 {
		t.Fatalf("opponent hand size = %d, want 3", got)
	}
	b.AddToHand(NewArtifact("Late card", ""))
	if got := HandSizeOf(ActivePlayer).resolveValue(ctx); got != 4 {
		t.Fatalf("late hand size = %d, want 4", got)
	}
	clone := g.Clone()
	clone.GetPlayer(b.PlayerID()).RemoveFromHand(b.Hand()[0].ID())
	if got := HandSizeOf(ActivePlayer).resolveValue(&EffectContext{Game: clone, Controller: a.PlayerID()}); got != 3 {
		t.Fatalf("clone hand size = %d, want 3", got)
	}
	if got := HandSizeOf(ActivePlayer).resolveValue(ctx); got != 4 {
		t.Fatalf("original hand size = %d, want 4", got)
	}
	if got := HandSizeOf(ControllerOf(This)).resolveValue(&EffectContext{Game: g, SourceID: uuid.New(), Controller: a.PlayerID()}); got != 0 {
		t.Fatalf("absent player hand size = %d, want 0", got)
	}
}

func TestHandSizeOfTargetAndDeclaration(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	a.AddToHand(NewArtifact("Discard", ""))
	for range 3 {
		b.AddToHand(NewArtifact("Opposing card", ""))
		a.AddToLibrary(NewArtifact("Draw", ""))
	}
	target := DeclarePlayerTarget("opponent", TargetOpponentOnly())
	ability := Spell(Sequence(DiscardEntireHand(You), DrawFor(You, HandSizeOf(target))), WithTargets(target))
	card := NewSorcery("Synthetic hand spell", "", WithAbility(ability))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	info := InspectDefinition(HandSizeOf(target))
	if info.Role != "resolution-value" || len(info.Children) != 1 {
		t.Fatalf("inspection = %+v", info)
	}
	obj := &StackObject{ID: card.ID(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{b.PlayerID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	clone := g.Clone()
	clone.GetPlayer(b.PlayerID()).RemoveFromHand(b.Hand()[0].ID())
	clone.ResolveStack()
	if got := len(clone.GetPlayer(a.PlayerID()).Hand()); got != 2 {
		t.Fatalf("clone drew %d cards, want 2", got)
	}
	g.ResolveStack()
	if got := len(a.Hand()); got != 3 {
		t.Fatalf("original drew %d cards, want 3", got)
	}
	for _, subject := range []PlayerRef{nil, DeclarePlayerTarget("undeclared"), FromPlayer(BindPlayer("missing"))} {
		invalid := NewSorcery("Invalid", "", WithAbility(Spell(DrawFor(You, HandSizeOf(subject)))))
		if err := ValidateCardDefinition(invalid.Name(), invalid); err == nil {
			t.Fatalf("accepted invalid player reference %T", subject)
		}
	}
}
