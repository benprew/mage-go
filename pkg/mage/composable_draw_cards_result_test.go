package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestDrawCardsResultVisitsAndReplacement(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	first, second := NewArtifact("First", ""), NewArtifact("Second", "")
	a.AddToLibrary(first)
	a.AddToLibrary(second)
	result := BindCards("drawn")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	effect := DrawFor(You, Fixed(3), DrawCardsInto(result))
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if got := CountCards(FromCards(result)).resolveValue(ctx); got != 2 {
		t.Fatalf("drawn cards = %d, want 2", got)
	}
	cards := FromCards(result).resolveCards(ctx)
	if len(cards) != 2 || cards[0].ID() != first.ID() || cards[1].ID() != second.ID() {
		t.Fatal("draw result lost the cards or their draw order")
	}
	clone := g.Clone()
	if _, ok := a.DiscardCard(first.ID()); !ok {
		t.Fatal("discard failed")
	}
	a.AddToHand(first)
	if got := CountCards(FromCards(result)).resolveValue(ctx); got != 1 {
		t.Fatalf("returned card counted: %d", got)
	}
	if got := CountCards(FromCards(result)).resolveValue(&EffectContext{Game: clone}); got != 2 {
		t.Fatalf("clone cards = %d, want 2", got)
	}
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if got := CountCards(FromCards(result)).resolveValue(ctx); got != 0 {
		t.Fatalf("empty draw retained cards: %d", got)
	}
}

func TestDrawCardsResultReplacedDraw(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	source := NewArtifact("Replacement", "")
	g.PutOnBattlefield(source, a.PlayerID())
	replaced := 0
	g.AddEmptyLibraryDrawReplacement(source.ID(), a.PlayerID(), func(_ *Game, _ uuid.UUID) { replaced++ })
	result := BindCards("drawn")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := DrawFor(You, Fixed(2), DrawCardsInto(result)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if replaced != 2 || CountCards(FromCards(result)).resolveValue(ctx) != 0 {
		t.Fatal("replaced draws saved cards")
	}
}

func TestDrawCardsResultTargetAndCopy(t *testing.T) {
	g, a, b := newSpellCopyGame()
	for range 4 {
		b.AddToLibrary(NewArtifact("Library card", ""))
	}
	result := BindCards("drawn")
	target := DeclarePlayerTarget("player")
	body := Sequence(DrawFor(target, Fixed(2), DrawCardsInto(result)), GainLifeFor(You, CountCards(FromCards(result))))
	spell := NewInstant("Synthetic draw", "", WithAbility(Spell(body, WithTargets(target))))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	copyObj := clone.CopyStackObjectDirect(clone.StackPeek(), a.PlayerID(), false)
	clone.ResolveStackObject(copyObj)
	if got := clone.GetPlayer(a.PlayerID()).Life(); got != 22 {
		t.Fatalf("copy life = %d, want 22", got)
	}
	if len(b.Hand()) != 0 {
		t.Fatal("copy changed original hand")
	}
	g.ResolveStack()
	if a.Life() != 22 || len(b.Hand()) != 2 {
		t.Fatal("original result or draw changed")
	}
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := DrawFor(target, Fixed(1), DrawCardsInto(result)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if got := CountCards(FromCards(result)).resolveValue(ctx); got != 0 {
		t.Fatalf("absent target retained cards: %d", got)
	}
}

func TestDrawCardsResultDeclaration(t *testing.T) {
	result := BindCards("drawn")
	valid := NewInstant("Valid", "", Spell(Sequence(DrawFor(You, Fixed(2), DrawCardsInto(result)), GainLifeFor(You, CountCards(FromCards(result))))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		body ResolvingEffect
		want string
	}{
		{DrawFor(You, Fixed(1), DrawCardsInto(nil)), "binding"},
		{DrawFor(You, Fixed(1), DrawCardsInto(BindCards(""))), "name"},
		{DrawFor(You, Fixed(1), DrawCardsInto(result), DrawCardsInto(result)), "once"},
		{DrawFor(You, Fixed(1), DrawCardsInto(result), DrawCardInto(BindCard("one"))), "one kind"},
		{GainLifeFor(You, CountCards(FromCards(result))), "producer"},
		{Sequence(IfState(Controls(IsCreature), DrawFor(You, Fixed(1), DrawCardsInto(result))), GainLifeFor(You, CountCards(FromCards(result)))), "producer"},
	} {
		card := NewInstant("Invalid", "", Spell(tc.body))
		if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("validation = %v, want %q", err, tc.want)
		}
	}
}
