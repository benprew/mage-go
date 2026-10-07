package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestGraveyardOnlyShuffleBatch(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		a := &librarySearchPlayer{BasePlayer: NewBasePlayer("A")}
		b := NewBasePlayer("B")
		g := NewGame(a, b)
		hand := NewLand("Hand")
		hand.SetOwner(a.PlayerID())
		a.AddToHand(hand)
		for range 2 {
			c := NewLand("Graveyard")
			c.SetOwner(a.PlayerID())
			a.AddToGraveyard(c)
		}
		foreign := NewLand("Foreign")
		foreign.SetOwner(b.PlayerID())
		b.AddToGraveyard(foreign)
		zones, batches := 0, 0
		observer := NewTriggered(core.EvtZoneChange, false).SetCondition(func(e *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
			if e.FromZone == core.ZoneGraveyard && e.ToZone == core.ZoneLibrary {
				zones++
				if len(a.Graveyard()) != 0 || len(a.Library()) != 2 {
					t.Error("partial batch")
				}
			}
			return false
		})
		batch := NewTriggered(core.EvtCardsLeftGraveyard, false).SetCondition(func(e *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
			batches++
			if e.Amount != 2 {
				t.Error("batch amount")
			}
			return false
		})
		g.PutOnBattlefield(NewArtifact("Observer", "", WithAbility(observer), WithAbility(batch)), a.PlayerID())
		var effect = ShuffleGraveyardFor(You)
		if legacy {
			effect = ShuffleGraveyardIntoLibrary()
		}
		clone := g.Clone()
		if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if zones != 2 || batches != 1 || a.shuffles != 1 || len(a.Hand()) != 1 || len(b.Graveyard()) != 1 || len(clone.GetPlayer(a.PlayerID()).Graveyard()) != 2 {
			t.Fatal("events, scope, shuffle, or clone")
		}
		if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if a.shuffles != 2 || zones != 2 || batches != 1 {
			t.Fatal("empty shuffle")
		}
	}
	card := NewSorcery("Invalid", "", Spell(ShuffleGraveyardFor(nil)))
	if err := ValidateCardDefinition(card.Name(), card); err == nil {
		t.Fatal("nil subjects")
	}
}
