package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestDiscardEachTotalActualAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	a.AddToHand(NewArtifact("A1", ""))
	a.AddToHand(NewArtifact("A2", ""))
	b.AddToHand(NewArtifact("B1", ""))
	count := BindNumber("discarded")
	body := Sequence(DiscardEach(EachPlayer, DiscardTotalInto(count)), GainLifeFor(You, FromNumber(count)))
	clone := g.Clone()
	if err := ApplyEffect(clone, body, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if got := clone.GetPlayer(a.PlayerID()).Life(); got != 23 {
		t.Fatalf("life = %d", got)
	}
	if len(a.Hand()) != 2 || len(b.Hand()) != 1 || a.Life() != 20 {
		t.Fatal("clone changed original")
	}
	if err := ApplyEffect(g, body, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 23 {
		t.Fatalf("life = %d", a.Life())
	}
	if err := ApplyEffect(g, body, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 23 {
		t.Fatal("empty hands retained stale count")
	}
}

func TestDiscardEachTotalReplacementAndAbsence(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	a.AddToHand(NewArtifact("A", ""))
	b.AddToHand(NewArtifact("B", ""))
	g.effects.AddReplacement(&allDiscardToLibraryReplacement{})
	count := BindNumber("discarded")
	body := Sequence(DiscardEach(EachPlayer, DiscardTotalInto(count)), GainLifeFor(You, FromNumber(count)))
	if err := ApplyEffect(g, body, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 22 {
		t.Fatalf("redirected discard count: life = %d", a.Life())
	}
	a.AddToHand(NewArtifact("A2", ""))
	b.AddToHand(NewArtifact("B2", ""))
	g.effects.AddReplacement(&discardToHandReplacement{})
	if err := ApplyEffect(g, body, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 22 {
		t.Fatal("prevented discard counted")
	}
	absent := Sequence(DiscardEach(OpponentsOf(FromPlayer(BindPlayer("absent"))), DiscardTotalInto(count)), GainLifeFor(You, FromNumber(count)))
	if err := ApplyEffect(g, absent, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 22 {
		t.Fatal("absent set retained stale count")
	}
}

func TestDiscardEachTotalDeclaration(t *testing.T) {
	count := BindNumber("discarded")
	valid := NewSorcery("Valid", "", WithAbility(Spell(Sequence(DiscardEach(EachPlayer, DiscardTotalInto(count)), GainLifeFor(You, FromNumber(count))))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	for _, effect := range []ResolvingEffect{
		DiscardEach(EachPlayer, DiscardTotalInto(nil)),
		DiscardEach(EachPlayer, DiscardTotalInto(BindNumber(""))),
		DiscardEach(EachPlayer, DiscardTotalInto(count), DiscardTotalInto(count)),
		Sequence(GainLifeFor(You, FromNumber(count)), DiscardEach(EachPlayer, DiscardTotalInto(count))),
	} {
		card := NewSorcery("Invalid", "", WithAbility(Spell(effect)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatalf("accepted %T", effect)
		}
	}
}
