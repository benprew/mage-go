package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestDiscardEachSimultaneousAndContinuation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	a.AddToHand(NewArtifact("A card", ""))
	b.AddToHand(NewArtifact("B card", ""))
	seen := 0
	watcher := NewTriggered(core.EvtDiscard, false, DrawFor(You, Fixed(1))).SetCondition(func(_ *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		seen++
		if len(a.Hand()) != 0 || len(b.Hand()) != 0 {
			t.Errorf("discard event observed partial hands: %d/%d", len(a.Hand()), len(b.Hand()))
		}
		return false
	})
	g.PutOnBattlefield(NewArtifact("Watcher", "", WithAbility(watcher)), a.PlayerID())
	if err := ApplyEffect(g, Sequence(DiscardEach(EachPlayer), GainLifeFor(You, Fixed(1))), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if seen != 2 || len(a.Graveyard()) != 1 || len(b.Graveyard()) != 1 || a.Life() != 21 {
		t.Fatalf("events %d, graves %d/%d, life %d", seen, len(a.Graveyard()), len(b.Graveyard()), a.Life())
	}
}

func TestDiscardEachDeclarationAndClone(t *testing.T) {
	valid := NewSorcery("Valid", "", WithAbility(Spell(Sequence(DiscardEach(EachPlayer), DrawEach(EachPlayer, Fixed(7))))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	for _, set := range []PlayerSet{nil, OpponentsOf(FromPlayer(BindPlayer("missing")))} {
		card := NewSorcery("Invalid", "", WithAbility(Spell(DiscardEach(set))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid set accepted")
		}
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	a.AddToHand(NewArtifact("A card", ""))
	b.AddToHand(NewArtifact("B card", ""))
	clone := g.Clone()
	if err := ApplyEffect(clone, DiscardEach(OpponentsOf(You)), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(clone.GetPlayer(b.PlayerID()).Hand()) != 0 || len(clone.GetPlayer(a.PlayerID()).Hand()) != 1 || len(b.Hand()) != 1 {
		t.Fatal("wrong selected player or clone changed original")
	}
}

func TestDiscardEachReplacementPreparedBeforeMoves(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	a.AddToHand(NewArtifact("A card", ""))
	b.AddToHand(NewArtifact("B card", ""))
	g.effects.AddReplacement(&allDiscardToLibraryReplacement{})
	seen := 0
	watcher := NewTriggered(core.EvtDiscard, false, DrawFor(You, Fixed(1))).SetCondition(func(_ *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		seen++
		if len(a.Hand()) != 0 || len(b.Hand()) != 0 || len(a.Library()) != 1 || len(b.Library()) != 1 {
			t.Error("event saw partial replacement outcomes")
		}
		return false
	})
	g.PutOnBattlefield(NewArtifact("Watcher", "", WithAbility(watcher)), a.PlayerID())
	if err := ApplyEffect(g, DiscardEach(EachPlayer), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if seen != 2 || len(a.Graveyard()) != 0 || len(b.Graveyard()) != 0 {
		t.Fatal("replacement or discard events lost")
	}
}

func TestDiscardEachAbsentSetAndPreventedActions(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	a.AddToHand(NewArtifact("A card", ""))
	b.AddToHand(NewArtifact("B card", ""))
	g.effects.AddReplacement(&allDiscardToLibraryReplacement{})
	g.effects.AddReplacement(&discardToHandReplacement{})
	if err := ApplyEffect(g, DiscardEach(EachPlayer), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(a.Hand()) != 1 || len(b.Hand()) != 1 || g.PlayerDiscardCountThisTurn(a.PlayerID()) != 0 || g.PlayerDiscardCountThisTurn(b.PlayerID()) != 0 {
		t.Fatal("prevented discard moved a card or sent an event")
	}
	if err := ApplyEffect(g, DiscardEach(OpponentsOf(FromPlayer(BindPlayer("absent")))), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(a.Hand()) != 1 || len(b.Hand()) != 1 {
		t.Fatal("absent player set changed a hand")
	}
}
