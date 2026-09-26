package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestDrawCardResultVisitAndAbsence(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	card := NewArtifact("Drawn", "")
	a.AddToLibrary(card)
	binding := BindCard("drawn")
	effect := DrawFor(You, Fixed(1), DrawCardInto(binding))
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if got, ok := FromCard(binding).resolveCard(ctx); !ok || got.ID() != card.ID() {
		t.Fatal("drawn card missing")
	}
	clone := g.Clone()
	if _, ok := a.DiscardCard(card.ID()); !ok {
		t.Fatal("discard failed")
	}
	if _, ok := FromCard(binding).resolveCard(ctx); ok {
		t.Fatal("old hand visit survived departure")
	}
	a.AddToHand(card)
	if _, ok := FromCard(binding).resolveCard(ctx); ok {
		t.Fatal("old hand visit followed return")
	}
	if _, ok := FromCard(binding).resolveCard(&EffectContext{Game: clone}); !ok {
		t.Fatal("clone lost saved visit")
	}
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if _, ok := FromCard(binding).resolveCard(ctx); ok {
		t.Fatal("empty draw reused result")
	}
}

func TestDrawCardResultDeclaration(t *testing.T) {
	b := BindCard("drawn")
	valid := NewInstant("Valid", "", Spell(DrawFor(You, Fixed(1), DrawCardInto(b))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		effect ResolvingEffect
		want   string
	}{
		{DrawFor(You, Fixed(1), DrawCardInto(nil)), "binding"},
		{DrawFor(You, Fixed(2), DrawCardInto(b)), "one card"},
		{DrawFor(You, Fixed(1), DrawCardInto(b), DrawCardInto(b)), "once"},
		{DrawFor(You, Fixed(1), DrawCardInto(b), DrawAmountInto(BindNumber("count"))), "one kind"},
		{ReturnCardToHand(FromCard(b)), "producer"},
	} {
		card := NewInstant("Invalid", "", Spell(tc.effect))
		if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("validation = %v, want %q", err, tc.want)
		}
	}
}

func TestDrawCardResultReplacementAndTarget(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewArtifact("Replacement", "")
	g.PutOnBattlefield(source, a.PlayerID())
	g.AddEmptyLibraryDrawReplacement(source.ID(), a.PlayerID(), func(*Game, uuid.UUID) {})
	binding := BindCard("drawn")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := DrawFor(You, Fixed(1), DrawCardInto(binding)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if _, ok := FromCard(binding).resolveCard(ctx); ok {
		t.Fatal("replaced draw saved a card")
	}
	b.AddToLibrary(NewArtifact("Other", ""))
	target := DeclarePlayerTarget("player")
	if err := DrawFor(target, Fixed(1), DrawCardInto(binding)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if _, ok := FromCard(binding).resolveCard(ctx); ok {
		t.Fatal("absent target saved a card")
	}
	ctx.targetBindings = map[*targetDeclaration][]capturedPermanent{target.targetDeclaration: {{id: b.PlayerID()}}}
	if err := DrawFor(target, Fixed(1), DrawCardInto(binding)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if card, ok := FromCard(binding).resolveCard(ctx); !ok || card.Name() != "Other" {
		t.Fatal("legal target draw missing")
	}
}

type observeDrawCardEffect struct {
	binding *CardBinding
	seen    map[uuid.UUID]uuid.UUID
}

func (*observeDrawCardEffect) Text() string                 { return "observe drawn card" }
func (*observeDrawCardEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *observeDrawCardEffect) Apply(ctx *EffectContext) error {
	if card, ok := FromCard(e.binding).resolveCard(ctx); ok {
		e.seen[ctx.Controller] = card.ID()
	}
	return nil
}
func (e *observeDrawCardEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", bindingRead: e.binding, children: []definitionEdge{{"binding", e.binding}}}
}

func TestDrawCardResultStackCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	aCard, bCard := NewArtifact("A card", ""), NewArtifact("B card", "")
	a.AddToLibrary(aCard)
	b.AddToLibrary(bCard)
	binding := BindCard("drawn")
	seen := make(map[uuid.UUID]uuid.UUID)
	ability := Spell(Sequence(DrawFor(You, Fixed(1), DrawCardInto(binding)), &observeDrawCardEffect{binding, seen}))
	card := NewInstant("Synthetic draw", "", WithAbility(ability))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	obj := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Effects: ability.Effects()}
	g.pushStack(obj)
	if g.CopyStackObjectDirect(obj, b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStack()
	if seen[a.PlayerID()] != aCard.ID() || seen[b.PlayerID()] != bCard.ID() {
		t.Fatalf("copy results = %v", seen)
	}
	if len(g.resolution.cardBindings) != 0 {
		t.Fatal("card result escaped resolution")
	}
}
