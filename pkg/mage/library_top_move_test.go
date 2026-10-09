package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestLibraryTopMoveHand(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first, second := NewLand("First"), NewLand("Second")
	first.SetOwner(b.PlayerID())
	second.SetOwner(a.PlayerID())
	a.SetLibrary([]Card{first, second})
	result := BindCard("reveal")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := RevealLibraryTopInto(You, result).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	visit := a.libraryVisit(second.ID())
	zones, draws := 0, 0
	observer := NewTriggered(core.EvtZoneChange, false).SetCondition(func(e *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		if e.SourceID == first.ID() {
			zones++
			if e.FromZone != core.ZoneLibrary || e.ToZone != core.ZoneHand || e.PlayerID != b.PlayerID() || len(a.Library()) != 1 || len(b.Hand()) != 1 {
				t.Error("zone event before completed owner movement")
			}
		}
		return false
	})
	drawObserver := NewTriggered(core.EvtCardDrawn, false).SetCondition(func(*core.GameEvent, GameReader, uuid.UUID, uuid.UUID) bool { draws++; return false })
	g.PutOnBattlefield(NewArtifact("Observer", "", WithAbility(observer), WithAbility(drawObserver)), a.PlayerID())
	if err := PutLibraryTopInHand(You).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if zones != 1 || draws != 0 || a.LastDrawnCardID() != uuid.Nil || a.libraryVisit(second.ID()) != visit {
		t.Fatal("move used draw or changed retained visit")
	}
	if _, ok := FromCard(result).resolveCard(ctx); ok {
		t.Fatal("departure kept library reference")
	}
	if len(clone.GetPlayer(a.PlayerID()).Library()) != 2 || len(clone.GetPlayer(b.PlayerID()).Hand()) != 0 {
		t.Fatal("move changed clone")
	}
	if err := PutLibraryTopInHand(You).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(a.Hand()) != 1 || a.Hand()[0].ID() != second.ID() {
		t.Fatal("move did not read current top")
	}
	if err := PutLibraryTopInHand(You).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if a.drewFromEmpty {
		t.Fatal("empty movement counted as empty draw")
	}
	if err := PutLibraryTopInHand(FromPlayer(BindPlayer("absent"))).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if err := ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), PutLibraryTopInHand(nil))))); err == nil {
		t.Fatal("nil player accepted")
	}
}

func TestLibraryTopMoveWaiting(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewArtifact("Source", "")
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	a.AddToLibrary(NewLand("Top"))
	if err := ApplyEffect(g, AtNext(EachEndStep, PutLibraryTopInHand(ControllerOfWithLKI(This))), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	saved := g.CaptureDamageSource(p.ID())
	for range 2 {
		refs := g.WaitingDelayedPermanentSources()
		if len(refs) != 1 || !refs[0].SameObject(saved) || len(a.Hand()) != 0 || len(a.Library()) != 1 {
			t.Fatal("query lost source or moved card")
		}
	}
	g.DestroyPermanent(p)
	g.PutOnBattlefield(card, b.PlayerID())
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
		game.PutTriggersOnStack()
		game.ResolveStack()
		if len(game.GetPlayer(a.PlayerID()).Hand()) != 1 || len(game.GetPlayer(b.PlayerID()).Hand()) != 0 {
			t.Fatal("returned source replaced saved controller")
		}
	}
}

func TestLibraryTopMoveGraveyard(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first, second, third := NewLand("First"), NewLand("Second"), NewLand("Third")
	for _, c := range []Card{first, second, third} {
		c.SetOwner(b.PlayerID())
		b.AddToLibrary(c)
	}
	count, deaths, draws := 0, 0, 0
	observer := NewTriggered(core.EvtZoneChange, false).SetCondition(func(e *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		if e.FromZone == core.ZoneLibrary && e.ToZone == core.ZoneGraveyard {
			count++
			if e.PlayerID != b.PlayerID() || len(b.Library()) != 2 || len(b.Graveyard()) != 1 {
				t.Error("movement event saw incomplete state")
			}
		}
		return false
	})
	death := NewTriggered(core.EvtZoneChange, false).SetCondition(func(e *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		if e.FromZone == core.ZoneBattlefield && e.ToZone == core.ZoneGraveyard {
			deaths++
		}
		return false
	})
	draw := NewTriggered(core.EvtCardDrawn, false).SetCondition(func(*core.GameEvent, GameReader, uuid.UUID, uuid.UUID) bool { draws++; return false })
	source := g.PutOnBattlefield(NewArtifact("Observer", "", WithAbility(observer), WithAbility(death), WithAbility(draw)), a.PlayerID())
	g.AddMillReplacement(source.ID(), DoubleMillForOpponents(source.ID(), a.PlayerID()))
	ctx := &EffectContext{Game: g, Controller: b.PlayerID()}
	if err := PutLibraryTopInGraveyard(You).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if count != 1 || deaths != 0 || draws != 0 || b.Library()[0].ID() != second.ID() {
		t.Fatal("top graveyard move used mill modifiers or missed zone events", count)
	}
}
