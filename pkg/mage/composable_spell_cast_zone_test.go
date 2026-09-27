package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestSpellCastFromZoneUsesExactStackObject(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	caster := BindPlayer("caster")
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		SpellCast().FromZone(ZoneExile).ByOpponent().WithCaster(caster),
		GainLifeFor(FromPlayer(caster), Fixed(1)))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	card := NewInstant("Shared", "")
	card.SetOwner(b.PlayerID())
	old := &StackObject{ID: uuid.New(), Card: card, SourceID: card.ID(), Controller: b.PlayerID(), CastZone: ZoneExile}
	current := &StackObject{ID: uuid.New(), Card: card, SourceID: card.ID(), Controller: b.PlayerID(), CastZone: ZoneHand}
	g.pushStack(old)
	g.pushStack(current)
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: card.ID(), TargetID: current.ID, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("matched another cast of the same card: stack=%d", g.StackSize())
	}
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: card.ID(), TargetID: old.ID, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 3 {
		t.Fatalf("exile cast did not trigger: stack=%d", g.StackSize())
	}
	clone := g.Clone()
	clone.ResolveTopOfStack()
	if b.Life() != 20 {
		t.Fatal("clone changed original caster")
	}
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
		t.Fatal("trigger copy failed")
	}
	g.PhaseOut(watcher.ID())
	g.ResolveTopOfStack()
	g.ResolveTopOfStack()
	if b.Life() != 22 {
		t.Fatalf("copied event caster lost: life=%d", b.Life())
	}
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: card.ID(), TargetID: old.ID, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("phased source triggered: stack=%d", g.StackSize())
	}
}

func TestSpellCastFromZoneRealCastAndFailedCast(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		SpellCast().FromZone(ZoneHand), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	unpaid := NewInstant("Unpaid", "{U}")
	unpaid.SetOwner(b.PlayerID())
	b.AddToHand(unpaid)
	if err := g.CastSpellByName(b.PlayerID(), unpaid.Name(), nil); err == nil {
		t.Fatal("unpaid spell was cast")
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("failed cast produced a trigger")
	}
	free := NewInstant("Free", "")
	free.SetOwner(b.PlayerID())
	b.AddToHand(free)
	if err := g.CastSpellByName(b.PlayerID(), free.Name(), nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("hand cast stack=%d", g.StackSize())
	}
	g.ResolveStack()
	if a.Life() != 21 {
		t.Fatalf("hand cast life=%d", a.Life())
	}
}

func TestSpellCastFromZoneRealExileCast(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		SpellCast().FromZone(ZoneExile), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	card := NewInstant("Exiled", "{1}")
	card.SetOwner(b.PlayerID())
	g.ExileCard(card, uuid.Nil)
	if err := g.CastCardFromZoneWithoutPaying(b.PlayerID(), card.ID(), ZoneExile, nil, 0); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("exile cast stack=%d", g.StackSize())
	}
	g.ResolveStack()
	if a.Life() != 21 {
		t.Fatalf("exile cast life=%d", a.Life())
	}
}

func TestSpellCastFromZoneDeclaration(t *testing.T) {
	valid := NewArtifact("Valid", "", WithAbility(WhenEvent(SpellCast().FromZone(ZoneGraveyard), GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	for _, pattern := range []EventPattern{
		SpellCast().FromZone(ZoneAny),
		SpellCast().FromZone(ZoneStack),
		SpellCast().FromZone(Zone(-1)),
		SpellCast().FromZone(ZoneHand).FromZone(ZoneExile),
	} {
		bad := NewArtifact("Bad", "", WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(bad.Name(), bad); err == nil || !strings.Contains(err.Error(), "zone") {
			t.Fatalf("invalid cast zone accepted: %v", err)
		}
	}
	if got := InspectDefinition(SpellCast().FromZone(ZoneExile)); !strings.Contains(got.Text, "exile") {
		t.Fatalf("inspection=%+v", got)
	}
}
