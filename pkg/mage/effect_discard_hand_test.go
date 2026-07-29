package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestDiscardHandDiscardsSelectedPlayersHand(t *testing.T) {
	g, a, _ := anteEffectTestGame(t)
	one := ownedTestCard(a.PlayerID(), "one")
	two := ownedTestCard(a.PlayerID(), "two")
	a.SetHand([]Card{one, two})

	if err := DiscardHand().Apply(&EffectContext{Game: g, Controller: a.PlayerID(), SourceID: uuid.New()}); err != nil {
		t.Fatal(err)
	}
	if len(a.Hand()) != 0 || len(a.Graveyard()) != 2 {
		t.Fatalf("hand/graveyard sizes = %d/%d, want 0/2", len(a.Hand()), len(a.Graveyard()))
	}
}
