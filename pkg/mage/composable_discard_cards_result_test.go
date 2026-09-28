package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestDiscardCardsResultVisitsAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first, second := NewArtifact("First", ""), NewArtifact("Second", "")
	b.AddToHand(first)
	b.AddToHand(second)
	result := BindCards("discarded")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	effect := Discard(ActivePlayer, Fixed(3), DiscardCardsInto(result))
	g.SetActivePlayerIndex(1)
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if got := CountCards(FromCards(result)).resolveValue(ctx); got != 2 {
		t.Fatalf("discarded visits = %d", got)
	}
	clone := g.Clone()
	b.RemoveFromGraveyard(first.ID())
	b.AddToHand(first)
	if got := CountCards(FromCards(result)).resolveValue(ctx); got != 1 {
		t.Fatalf("returned card counted: %d", got)
	}
	b.RemoveFromHand(first.ID())
	b.AddToGraveyard(first)
	if got := CountCards(FromCards(result)).resolveValue(ctx); got != 1 {
		t.Fatalf("new graveyard visit counted: %d", got)
	}
	if got := CountCards(FromCards(result)).resolveValue(&EffectContext{Game: clone}); got != 2 {
		t.Fatalf("clone visits = %d", got)
	}
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if got := CountCards(FromCards(result)).resolveValue(ctx); got != 0 {
		t.Fatalf("empty hand kept result: %d", got)
	}
}

func TestDiscardCardsResultReplacementAndTarget(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	b.AddToHand(NewArtifact("First", ""))
	b.AddToHand(NewArtifact("Second", ""))
	g.effects.AddReplacement(&allDiscardToLibraryReplacement{})
	result := BindCards("discarded")
	target := DeclarePlayerTarget("player")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	effect := Discard(target, Fixed(2), DiscardCardsInto(result))
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(b.Hand()) != 2 || CountCards(FromCards(result)).resolveValue(ctx) != 0 {
		t.Fatal("absent target changed state")
	}
	ctx.targetBindings = map[*targetDeclaration][]capturedPermanent{target.targetDeclaration: {{id: b.PlayerID()}}}
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(b.Library()) != 2 || len(b.Graveyard()) != 0 || CountCards(FromCards(result)).resolveValue(ctx) != 0 {
		t.Fatal("redirected discards entered result")
	}
}

func TestDiscardCardsResultPrevention(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	b.AddToHand(NewArtifact("Held", ""))
	g.effects.AddReplacement(&allDiscardToLibraryReplacement{})
	g.effects.AddReplacement(&discardToHandReplacement{})
	g.SetActivePlayerIndex(1)
	result := BindCards("discarded")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := Discard(ActivePlayer, Fixed(2), DiscardCardsInto(result)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(b.Hand()) != 1 || CountCards(FromCards(result)).resolveValue(ctx) != 0 {
		t.Fatal("prevented discard changed result")
	}
}

func TestDiscardCardsResultStackCopy(t *testing.T) {
	g, a, b := newSpellCopyGame()
	for range 4 {
		b.AddToHand(NewArtifact("Held", ""))
	}
	result := BindCards("discarded")
	target := DeclarePlayerTarget("player")
	spell := NewInstant("Discard", "", WithAbility(Spell(Sequence(
		Discard(target, Fixed(2), DiscardCardsInto(result)),
		GainLifeFor(You, CountCards(FromCards(result))),
	), WithTargets(target))))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	copyObj := clone.CopyStackObjectDirect(clone.StackPeek(), a.PlayerID(), false)
	clone.ResolveStackObject(copyObj)
	if clone.GetPlayer(a.PlayerID()).Life() != 22 || a.Life() != 20 || len(clone.GetPlayer(b.PlayerID()).Hand()) != 2 {
		t.Fatal("copy result or clone isolation failed")
	}
	g.ResolveStack()
	if a.Life() != 22 || len(b.Hand()) != 2 {
		t.Fatal("original result changed")
	}
}

func TestDiscardCardsResultDeclaration(t *testing.T) {
	b := BindCards("discarded")
	valid := NewInstant("Valid", "", Spell(Sequence(Discard(You, Fixed(2), DiscardCardsInto(b)), GainLifeFor(You, CountCards(FromCards(b))))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		effect ResolvingEffect
		want   string
	}{
		{Discard(You, Fixed(2), DiscardCardsInto(nil)), "binding"},
		{Discard(You, Fixed(2), DiscardCardsInto(BindCards(""))), "binding name"},
		{Discard(You, Fixed(2), DiscardCardsInto(b), DiscardCardsInto(b)), "once"},
		{GainLifeFor(You, CountCards(FromCards(b))), "producer"},
		{Sequence(IfState(Controls(IsCreature), Discard(You, Fixed(2), DiscardCardsInto(b))), GainLifeFor(You, CountCards(FromCards(b)))), "producer"},
	} {
		card := NewInstant("Invalid", "", Spell(tc.effect))
		if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("validation = %v, want %q", err, tc.want)
		}
	}
}
