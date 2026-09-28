package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestDiscardCardResultIdentityAndAbsence(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first := NewArtifact("First", "")
	second := NewArtifact("Second", "")
	b.AddToHand(first)
	b.AddToHand(second)
	binding := BindCard("discarded")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	effect := Discard(ActivePlayer, Fixed(1), DiscardCardInto(binding))
	g.SetActivePlayerIndex(1)
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	got, ok := FromCard(binding).resolveCard(ctx)
	if !ok || got.ID() != first.ID() {
		t.Fatal("wrong discarded card")
	}
	clone := g.Clone()
	b.RemoveFromGraveyard(first.ID())
	b.AddToHand(first)
	if _, ok := FromCard(binding).resolveCard(ctx); ok {
		t.Fatal("binding followed returned card")
	}
	if _, ok := FromCard(binding).resolveCard(&EffectContext{Game: clone}); !ok {
		t.Fatal("clone lost discard result")
	}
	if err := Discard(FromPlayer(BindPlayer("absent")), Fixed(1), DiscardCardInto(binding)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if _, ok := FromCard(binding).resolveCard(ctx); ok {
		t.Fatal("absent player retained result")
	}
}

func TestDiscardCardResultReplacementAndTarget(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	b.AddToHand(NewArtifact("Library card", ""))
	g.effects.AddReplacement(&allDiscardToLibraryReplacement{})
	binding := BindCard("discarded")
	target := DeclarePlayerTarget("player")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	effect := Discard(target, Fixed(1), DiscardCardInto(binding))
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(b.Hand()) != 1 {
		t.Fatal("absent target changed hand")
	}
	ctx.targetBindings = map[*targetDeclaration][]capturedPermanent{target.targetDeclaration: {{id: b.PlayerID()}}}
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(b.Library()) != 1 || len(b.Graveyard()) != 0 {
		t.Fatal("replacement did not move card")
	}
	if _, ok := FromCard(binding).resolveCard(ctx); ok {
		t.Fatal("library replacement produced a graveyard card")
	}
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if _, ok := FromCard(binding).resolveCard(ctx); ok {
		t.Fatal("empty hand reused card")
	}
}

func TestDiscardCardResultStackCopy(t *testing.T) {
	g, a, b := newSpellCopyGame()
	first := NewArtifact("First", "")
	second := NewArtifact("Second", "")
	b.AddToHand(first)
	b.AddToHand(second)
	binding := BindCard("discarded")
	target := DeclarePlayerTarget("player")
	spell := NewInstant("Discard", "", WithAbility(Spell(Sequence(Discard(target, Fixed(1), DiscardCardInto(binding)), ReturnCardToHand(FromCard(binding))), WithTargets(target))))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	copyObj := clone.CopyStackObjectDirect(clone.StackPeek(), a.PlayerID(), false)
	clone.ResolveStackObject(copyObj)
	if len(clone.GetPlayer(b.PlayerID()).Hand()) != 2 || len(b.Hand()) != 2 {
		t.Fatal("copy result or clone isolation failed")
	}
	g.ResolveStack()
	if len(b.Hand()) != 2 {
		t.Fatal("original result did not resolve")
	}
}

func TestDiscardCardResultDeclaration(t *testing.T) {
	b := BindCard("discarded")
	valid := NewInstant("Valid", "", Spell(Sequence(Discard(You, Fixed(1), DiscardCardInto(b)), ReturnCardToHand(FromCard(b)))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		effect ResolvingEffect
		want   string
	}{
		{Discard(You, Fixed(1), DiscardCardInto(nil)), "binding"},
		{Discard(You, Fixed(2), DiscardCardInto(b)), "one card"},
		{Discard(You, Fixed(1), DiscardCardInto(b), DiscardCardInto(b)), "once"},
		{ReturnCardToHand(FromCard(b)), "producer"},
	} {
		card := NewInstant("Invalid", "", Spell(tc.effect))
		if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("validation = %v, want %q", err, tc.want)
		}
	}
}
