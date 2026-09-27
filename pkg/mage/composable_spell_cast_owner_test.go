package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestSpellCastNotOwnedByCasterUsesExactSpell(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		SpellCast().NotOwnedByCaster().ByOpponent().FromZone(ZoneExile),
		GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	card := NewInstant("Shared", "")
	card.SetOwner(a.PlayerID())
	foreign := &StackObject{ID: uuid.New(), Card: card, SourceID: card.ID(), Controller: b.PlayerID(), CastZone: ZoneExile}
	owned := &StackObject{ID: uuid.New(), Card: card, SourceID: card.ID(), Controller: a.PlayerID(), CastZone: ZoneExile}
	g.pushStack(foreign)
	g.pushStack(owned)
	ownedByB := NewInstant("Owned by caster", "")
	ownedByB.SetOwner(b.PlayerID())
	ownedByBObject := &StackObject{ID: uuid.New(), Card: ownedByB, SourceID: ownedByB.ID(), Controller: b.PlayerID(), CastZone: ZoneExile}
	g.pushStack(ownedByBObject)
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: ownedByB.ID(), TargetID: ownedByBObject.ID, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 3 {
		t.Fatalf("caster-owned spell matched: stack=%d", g.StackSize())
	}
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: card.ID(), TargetID: owned.ID, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 3 {
		t.Fatalf("owned spell matched: stack=%d", g.StackSize())
	}
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: ownedByB.ID(), TargetID: foreign.ID, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 3 {
		t.Fatalf("mismatched source matched: stack=%d", g.StackSize())
	}
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: card.ID(), TargetID: foreign.ID, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 4 {
		t.Fatalf("foreign spell missed: stack=%d", g.StackSize())
	}
	clone := g.Clone()
	clone.ResolveTopOfStack()
	if a.Life() != 20 {
		t.Fatal("clone changed original life")
	}
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
		t.Fatal("trigger copy failed")
	}
	g.PhaseOut(watcher.ID())
	g.ResolveTopOfStack()
	g.ResolveTopOfStack()
	if a.Life() != 22 {
		t.Fatalf("copied trigger lost caster ownership result: life=%d", a.Life())
	}
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: card.ID(), TargetID: foreign.ID, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 3 {
		t.Fatalf("phased watcher triggered: stack=%d", g.StackSize())
	}
}

func TestSpellCastNotOwnedByCasterRejectsInvalidDeclaration(t *testing.T) {
	pattern := SpellCast().NotOwnedByCaster()
	card := NewArtifact("Valid", "", WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if got := InspectDefinition(pattern); !strings.Contains(got.Text, "not owned") {
		t.Fatalf("inspection=%+v", got)
	}
	bad := NewArtifact("Bad", "", WithAbility(WhenEvent(pattern.NotOwnedByCaster(), GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(bad.Name(), bad); err == nil {
		t.Fatal("duplicate ownership filter accepted")
	}
}

func TestSpellCastNotOwnedByCasterRealCastAndRollback(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		SpellCast().NotOwnedByCaster(), GainLifeFor(You, Fixed(1)))))
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
	foreign := NewInstant("Foreign", "")
	foreign.SetOwner(a.PlayerID())
	g.ExileCard(foreign, uuid.Nil)
	if err := g.CastCardFromExileWithoutPaying(b.PlayerID(), foreign.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("foreign cast stack=%d", g.StackSize())
	}
	g.ResolveStack()
	if a.Life() != 21 {
		t.Fatalf("foreign cast life=%d", a.Life())
	}
}
