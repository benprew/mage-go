package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type cardSetChoicePlayer struct {
	*BasePlayer
	accept bool
	choice uuid.UUID
	seen   []uuid.UUID
}

func (p *cardSetChoicePlayer) ChooseMayAbility(string) bool { return p.accept }
func (p *cardSetChoicePlayer) ChooseCardFromGraveyard(cards []Card, _ string, _ GameReader) Card {
	p.seen = nil
	for _, card := range cards {
		p.seen = append(p.seen, card.ID())
		if card.ID() == p.choice {
			return card
		}
	}
	return nil
}

func TestChooseUpToOneCardCapturedSet(t *testing.T) {
	baseA, b := NewBasePlayer("A"), NewBasePlayer("B")
	a := &cardSetChoicePlayer{BasePlayer: baseA, accept: true}
	g := NewGame(a, b)
	land := NewLand("Land")
	artifact := NewArtifact("Artifact", "")
	other := NewLand("Other")
	for _, card := range []Card{land, artifact, other} {
		card.SetOwner(b.PlayerID())
		b.AddToGraveyard(card)
	}
	set, picked := BindCards("discarded"), BindCard("picked")
	g.resolution.cardsBindings = map[*CardsBinding][]capturedCard{set: {
		{id: land.ID(), visit: g.graveyardVisit(land.ID()), zone: core.ZoneGraveyard},
		{id: artifact.ID(), visit: g.graveyardVisit(artifact.ID()), zone: core.ZoneGraveyard},
	}}
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	choose := ChooseUpToOneCard(You, FromCards(set), IsLandCard, picked)
	a.choice = land.ID()
	if err := choose.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(a.seen) != 1 || a.seen[0] != land.ID() {
		t.Fatalf("offered %v", a.seen)
	}
	if card, ok := FromCard(picked).resolveCard(ctx); !ok || card.ID() != land.ID() {
		t.Fatal("wrong chosen visit")
	}
	clone := g.Clone()
	b.RemoveFromGraveyard(land.ID())
	b.AddToHand(land)
	b.RemoveFromHand(land.ID())
	b.AddToGraveyard(land)
	if _, ok := FromCard(picked).resolveCard(ctx); ok {
		t.Fatal("old visit survived return")
	}
	if card, ok := FromCard(picked).resolveCard(&EffectContext{Game: clone, Controller: a.PlayerID()}); !ok || card.ID() != land.ID() {
		t.Fatal("clone lost chosen visit")
	}
	a.choice = other.ID()
	if err := choose.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if _, ok := FromCard(picked).resolveCard(ctx); ok {
		t.Fatal("out-of-set choice was saved")
	}
	a.accept = false
	if err := choose.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if _, ok := FromCard(picked).resolveCard(ctx); ok {
		t.Fatal("declined choice kept old result")
	}
}

func TestChooseUpToOneCardDeclaration(t *testing.T) {
	set, picked := BindCards("discarded"), BindCard("picked")
	valid := NewSorcery("Choice", "", WithAbility(Spell(Sequence(
		Discard(You, Fixed(2), DiscardCardsInto(set)),
		ChooseUpToOneCard(You, FromCards(set), IsLandCard, picked),
		ReturnCardToHand(FromCard(picked)),
	))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		effect ResolvingEffect
		want   string
	}{
		{ChooseUpToOneCard(You, FromCards(set), IsLandCard, picked), "producer"},
		{ChooseUpToOneCard(nil, FromCards(set), IsLandCard, picked), "non-nil"},
		{ChooseUpToOneCard(You, nil, IsLandCard, picked), "non-nil"},
		{ChooseUpToOneCard(You, FromCards(set), IsLandCard, nil), "non-nil"},
		{Sequence(Discard(You, Fixed(2), DiscardCardsInto(set)), ChooseUpToOneCard(You, FromCards(set), IsLandCard, BindCard(""))), "name"},
	} {
		card := NewSorcery("Invalid", "", WithAbility(Spell(tc.effect)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("validation = %v, want %q", err, tc.want)
		}
	}
}

func TestChooseUpToOneCardSpellCopy(t *testing.T) {
	a := &cardSetChoicePlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	land := NewLand("Land")
	land.SetOwner(b.PlayerID())
	b.AddToHand(land)
	artifact := NewArtifact("Artifact", "")
	artifact.SetOwner(b.PlayerID())
	b.AddToHand(artifact)
	a.choice = land.ID()
	set, picked := BindCards("discarded"), BindCard("picked")
	target := DeclarePlayerTarget("player")
	spell := NewInstant("Choice", "", WithAbility(Spell(Sequence(
		Discard(target, Fixed(2), DiscardCardsInto(set)),
		ChooseUpToOneCard(You, FromCards(set), IsLandCard, picked),
		ReturnCardToHand(FromCard(picked)),
	), WithTargets(target))))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	copyObj := g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false)
	g.ResolveStackObject(copyObj)
	if len(b.Hand()) != 1 || len(clone.GetPlayer(b.PlayerID()).Hand()) != 2 {
		t.Fatal("copy choice or clone isolation failed")
	}
	g.ResolveStack()
	if len(b.Hand()) != 1 || b.Hand()[0].ID() != land.ID() {
		t.Fatal("spell did not choose the discarded land")
	}
}
