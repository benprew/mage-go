package mage

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestForEachSavedCardDrawOrderVisitsAndClone(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	first, second := NewArtifact("First", ""), NewArtifact("Second", "")
	a.AddToLibrary(first)
	a.AddToLibrary(second)
	cards, member := BindCards("drawn"), BindCard("member")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := DrawFor(You, Fixed(2), DrawCardsInto(cards)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	var visited []uuid.UUID
	body := FuncEffect("inspect card", EffectProperties{}, func(game *Game, _, controller uuid.UUID, _ []uuid.UUID) error {
		card, ok := FromCard(member).resolveCard(&EffectContext{Game: game, Controller: controller})
		if ok {
			visited = append(visited, card.ID())
		}
		return nil
	})
	if _, ok := a.DiscardCard(second.ID()); !ok {
		t.Fatal("discard failed")
	}
	a.AddToHand(second)
	if err := ForEachSavedCard(cards, member, body).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(visited) != 1 || visited[0] != first.ID() {
		t.Fatalf("visited after return = %v", visited)
	}
	visited = nil
	if err := ForEachSavedCard(cards, member, body).Apply(&EffectContext{Game: clone, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if len(visited) != 2 || visited[0] != first.ID() || visited[1] != second.ID() {
		t.Fatalf("clone visits = %v", visited)
	}
	if len(g.resolution.cardBindings) != 0 || len(clone.resolution.cardBindings) != 0 {
		t.Fatal("loop member escaped")
	}
}

func TestForEachSavedCardScopeAndFailureRestoresFrame(t *testing.T) {
	cards, member := BindCards("drawn"), BindCard("member")
	valid := NewInstant("Valid", "", Spell(Sequence(DrawFor(You, Fixed(1), DrawCardsInto(cards)),
		ForEachSavedCard(cards, member, ReturnCardToHand(FromCard(member))))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		body ResolvingEffect
		want string
	}{
		{ForEachSavedCard(cards, member, ReturnCardToHand(FromCard(member))), "producer"},
		{Sequence(DrawFor(You, Fixed(1), DrawCardsInto(cards)), ForEachSavedCard(cards, member, ReturnCardToHand(FromCard(member))), ReturnCardToHand(FromCard(member))), "producer"},
		{Sequence(DrawFor(You, Fixed(1), DrawCardsInto(cards)), ForEachSavedCard(cards, nil, GainLifeFor(You, Fixed(1)))), "non-nil"},
		{Sequence(DrawFor(You, Fixed(1), DrawCardsInto(cards)), ForEachSavedCard(cards, BindCard(""), GainLifeFor(You, Fixed(1)))), "name"},
	} {
		card := NewInstant("Invalid", "", Spell(tc.body))
		if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("validation = %v, want %q", err, tc.want)
		}
	}
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	a.AddToLibrary(NewArtifact("One", ""))
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := DrawFor(You, Fixed(1), DrawCardsInto(cards)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	outer := BindCard("outer")
	g.resolution.cardBindings = map[*CardBinding]capturedCard{outer: {id: uuid.New(), visit: uuid.New()}}
	want := errors.New("body failed")
	fail := FuncEffect("fail", EffectProperties{}, func(game *Game, _, _ uuid.UUID, _ []uuid.UUID) error {
		delete(game.resolution.cardBindings, outer)
		return want
	})
	if err := ForEachSavedCard(cards, member, fail).Apply(ctx); !errors.Is(err, want) || len(g.resolution.cardBindings) != 1 || g.resolution.cardBindings[outer].id == uuid.Nil {
		t.Fatal("loop did not restore the outer card frame")
	}
}

func TestForEachSavedCardStackCopy(t *testing.T) {
	g, a, b := newSpellCopyGame()
	for range 4 {
		b.AddToLibrary(NewArtifact("Library card", ""))
	}
	cards, member := BindCards("drawn"), BindCard("member")
	spell := NewInstant("Synthetic loop", "", Spell(Sequence(
		DrawFor(You, Fixed(2), DrawCardsInto(cards)),
		ForEachSavedCard(cards, member, GainLifeFor(You, Fixed(1))),
	)))
	spell.SetOwner(b.PlayerID())
	b.AddToHand(spell)
	if err := g.CastSpellByName(b.PlayerID(), spell.Name(), nil, 0); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	copyObj := clone.CopyStackObjectDirect(clone.StackPeek(), b.PlayerID(), false)
	clone.ResolveStackObject(copyObj)
	if got := clone.GetPlayer(b.PlayerID()).Life(); got != 22 {
		t.Fatalf("copy life = %d, want 22", got)
	}
	if b.Life() != 20 || len(b.Hand()) != 0 {
		t.Fatal("copy changed the original game")
	}
	g.ResolveStack()
	if b.Life() != 22 || len(b.Hand()) != 2 || a.Life() != 20 {
		t.Fatal("original loop or draw failed")
	}
}
