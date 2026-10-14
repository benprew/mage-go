package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestGraveyardBatchExileEvents(t *testing.T) {
	g, a, b := newObjectColorGame()
	first, second := NewArtifact("First", ""), NewCreature("Second", "", 1, 1)
	for _, card := range []Card{first, second} {
		card.SetOwner(b.PlayerID())
		b.AddToGraveyard(card)
	}
	added := NewLand("Added after movement")
	added.SetOwner(b.PlayerID())
	zones, batches := 0, 0
	observer := NewTriggered(core.EvtZoneChange, false).SetCondition(func(event *core.GameEvent, r GameReader, _, _ uuid.UUID) bool {
		if event.FromZone != core.ZoneGraveyard || event.ToZone != core.ZoneExile {
			return false
		}
		zones++
		if event.PlayerID != b.PlayerID() || g.FindExiledCard(first.ID()) == nil || g.FindExiledCard(second.ID()) == nil {
			t.Fatal("zone observation preceded complete movement")
		}
		if zones == 1 {
			b.AddToGraveyard(added)
		}
		return false
	})
	batch := NewTriggered(core.EvtCardsLeftGraveyard, false).SetCondition(func(event *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		batches++
		if event.PlayerID != b.PlayerID() || event.Amount != 2 {
			t.Fatal("wrong graveyard batch")
		}
		return false
	})
	g.PutOnBattlefield(NewArtifact("Observer", "", WithAbility(observer), WithAbility(batch)), a.PlayerID())
	source := uuid.New()
	if err := ApplyEffect(g, ExileGraveyardFor(You), source, b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if zones != 2 || batches != 1 || len(b.Graveyard()) != 1 || b.Graveyard()[0].ID() != added.ID() {
		t.Fatal("wrong simultaneous batch or later card joined it")
	}
	for _, card := range []Card{first, second} {
		exiled := g.FindExiledCard(card.ID())
		if exiled.FaceDown || exiled.ExiledBy != source || exiled.Card.Owner() != b.PlayerID() {
			t.Fatal("exile lost visibility, attribution or ownership")
		}
	}
}

func TestGraveyardBatchExileSavedVisits(t *testing.T) {
	g, a, b := newObjectColorGame()
	first, second := NewArtifact("Returned", ""), NewArtifact("Present", "")
	for _, card := range []Card{first, second} {
		card.SetOwner(b.PlayerID())
		b.AddToGraveyard(card)
	}
	selected := []capturedCard{
		{id: first.ID(), visit: g.graveyardVisit(first.ID()), zone: core.ZoneGraveyard},
		{id: second.ID(), visit: g.graveyardVisit(second.ID()), zone: core.ZoneGraveyard},
		{id: second.ID(), visit: g.graveyardVisit(second.ID()), zone: core.ZoneGraveyard},
		{id: uuid.New(), visit: uuid.New(), zone: core.ZoneGraveyard},
	}
	b.RemoveFromGraveyard(first.ID())
	b.AddToGraveyard(first)
	clone := g.Clone()
	g.exileGraveyardVisitsTogether(b.PlayerID(), selected, a.PlayerID())
	if g.FindExiledCard(first.ID()) != nil || g.FindExiledCard(second.ID()) == nil || len(g.GetExile()) != 1 || len(b.Graveyard()) != 1 {
		t.Fatal("saved visit followed a return or duplicated movement")
	}
	if len(clone.GetPlayer(b.PlayerID()).Graveyard()) != 2 || len(clone.GetExile()) != 0 {
		t.Fatal("movement changed clone")
	}
	clone.exileGraveyardVisitsTogether(b.PlayerID(), selected, a.PlayerID())
	if clone.FindExiledCard(first.ID()) != nil || clone.FindExiledCard(second.ID()) == nil {
		t.Fatal("clone lost saved visits")
	}
}

func TestGraveyardBatchExileDeclaration(t *testing.T) {
	for _, ref := range []PlayerRef{nil, FromPlayer(nil), FromPlayer(BindPlayer("missing"))} {
		card := NewSorcery("Invalid", "", WithAbility(Spell(ExileGraveyardFor(ref))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid player accepted")
		}
	}
	g, a, _ := newObjectColorGame()
	for _, ref := range []PlayerRef{You, ActivePlayer, FromPlayer(BindPlayer("absent"))} {
		if err := ApplyEffect(g, ExileGraveyardFor(ref), uuid.New(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if len(g.GetExile()) != 0 {
		t.Fatal("empty or missing player supplied cards")
	}
}
