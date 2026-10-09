package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestAnteTopBatch(t *testing.T) {
	g, a, b := anteEffectTestGame(t)
	topA, nextA := ownedTestCard(a.PlayerID(), "top A"), ownedTestCard(a.PlayerID(), "next A")
	topB := ownedTestCard(b.PlayerID(), "top B")
	a.SetLibrary([]Card{topA, nextA})
	b.SetLibrary([]Card{topB})
	events := 0
	watcher := NewTriggered(core.EvtZoneChange, false).SetCondition(func(e *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		if e.ToZone == core.ZoneAnte {
			events++
			if len(a.Library()) != 1 || len(b.Library()) != 0 || len(a.Ante()) != 1 || len(b.Ante()) != 1 {
				t.Error("event observed partial batch")
			}
		}
		return false
	})
	g.PutOnBattlefield(NewArtifact("observer", "", WithAbility(watcher)), a.PlayerID())
	if err := AnteTopEach(EachPlayer).Apply(&EffectContext{Game: g}); err != nil {
		t.Fatal(err)
	}
	if events != 2 || a.Library()[0].ID() != nextA.ID() || a.Ante()[0].ID() != topA.ID() || b.Ante()[0].ID() != topB.ID() {
		t.Fatal("wrong cards or events")
	}
}

func TestAnteTopLegality(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		var g *Game
		if enabled {
			g, _ = NewGameWithAnte(a, b, nil, nil)
		} else {
			g = NewGame(a, b)
		}
		wrong := ownedTestCard(b.PlayerID(), "wrong owner")
		top := ownedTestCard(b.PlayerID(), "legal")
		a.SetLibrary([]Card{wrong})
		b.SetLibrary([]Card{top})
		if err := AnteTopEach(EachPlayer).Apply(&EffectContext{Game: g}); err != nil {
			t.Fatal(err)
		}
		if len(a.Library()) != 1 || (enabled && len(b.Ante()) != 1) || (!enabled && len(b.Library()) != 1) {
			t.Fatal("ante legality")
		}
		b.SetLibrary(nil)
		if err := AnteTopFor(You).Apply(&EffectContext{Game: g, Controller: b.PlayerID()}); err != nil {
			t.Fatal(err)
		}
	}
}

func TestAnteTopBatchIdentityAndClone(t *testing.T) {
	g, a, b := anteEffectTestGame(t)
	first, second := ownedTestCard(a.PlayerID(), "first"), ownedTestCard(a.PlayerID(), "second")
	token := NewToken("token", 1, 1, []core.CardType{core.TypeCreature}, nil)
	token.SetOwner(b.PlayerID())
	a.SetLibrary([]Card{first, second})
	b.SetLibrary([]Card{token})
	clone := g.Clone()
	clone.AnteLibraryTops([]uuid.UUID{uuid.New(), a.PlayerID(), a.PlayerID(), b.PlayerID()})
	if len(a.Library()) != 2 || len(a.Ante()) != 0 || len(clone.GetPlayer(a.PlayerID()).Ante()) != 1 || clone.GetPlayer(a.PlayerID()).Ante()[0].ID() != first.ID() || len(clone.GetPlayer(b.PlayerID()).Library()) != 1 {
		t.Fatal("duplicate, token or clone identity")
	}
	g.anteSettled = true
	g.AnteLibraryTops([]uuid.UUID{a.PlayerID()})
	if len(a.Library()) != 2 {
		t.Fatal("settled game moved a card")
	}
}

func TestAnteTopDefinition(t *testing.T) {
	for _, effect := range []ResolvingEffect{AnteTopFor(nil), AnteTopEach(nil)} {
		card := NewInstant("invalid", "", WithAbility(Spell(effect)))
		if err := ValidateCardDefinition("invalid", card); err == nil {
			t.Fatal("nil subject accepted")
		}
	}
}
