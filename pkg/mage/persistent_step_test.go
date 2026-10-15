package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPersistentStepLifecycle(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewArtifact("Source", "")
	card.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(card, a.PlayerID())
	amount := BindNumber("amount")
	effect := AtEach(YourUpkeep, GainLifeFor(You, FromNumber(amount)))
	if err := applyEffectWithCardSetCaptures(g, effect, source.ID(), a.PlayerID(), nil, map[*NumberBinding]int{amount: 3}, nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 20 || len(g.DelayedTriggers()) != 1 {
		t.Fatal("subscription executed at creation")
	}
	g.DestroyPermanent(source)
	g.ResolveStack()
	clone := g.Clone()
	for i := range 3 {
		g.triggers.ClearEndOfTurn()
		g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
		g.PutTriggersOnStack()
		if g.StackSize() != 0 {
			t.Fatal("wrong upkeep player")
		}
		g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
		if a.Life() != 20+i*3 {
			t.Fatal("body executed during detection")
		}
		g.PutTriggersOnStack()
		if g.StackSize() != 1 {
			t.Fatal("missing recurring trigger")
		}
		g.ResolveStack()
		if a.Life() != 23+i*3 || len(g.DelayedTriggers()) != 1 {
			t.Fatal("subscription did not repeat")
		}
	}
	if clone.GetPlayer(a.PlayerID()).Life() != 20 {
		t.Fatal("clone shared results")
	}
	clone.RunStepWithPriority(core.Upkeep)
	if clone.GetPlayer(a.PlayerID()).Life() != 23 {
		t.Fatal("clone lost saved amount")
	}
}

func TestPersistentStepDeclarations(t *testing.T) {
	for _, effect := range []ResolvingEffect{AtEach(nil, GainLifeFor(You, Fixed(1))), AtEach(YourUpkeep, nil)} {
		card := NewInstant("Invalid", "", Spell(effect))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid subscription accepted")
		}
	}
}

func TestPersistentStepIndependentFrames(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewArtifact("Source", "")
	card.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(card, a.PlayerID())
	amount := BindNumber("amount")
	effect := AtEach(YourUpkeep, GainLifeFor(You, FromNumber(amount)))
	for i, player := range []Player{a, b} {
		if err := applyEffectWithCardSetCaptures(g, effect, source.ID(), player.PlayerID(), nil, map[*NumberBinding]int{amount: i + 2}, nil, nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	g.PhaseOut(source.ID())
	for range 2 {
		g.RunStepWithPriority(core.Cleanup)
		g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
		g.PutTriggersOnStack()
		if g.StackSize() != 1 {
			t.Fatal("source change altered subscription")
		}
		if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
			t.Fatal("copy failed")
		}
		g.ResolveStack()
		g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
		g.ResolveStack()
	}
	if a.Life() != 24 || b.Life() != 30 || len(g.DelayedTriggers()) != 2 {
		t.Fatal("controllers, saved amounts or subscriptions were shared")
	}
}

func TestPersistentStepSourceLKI(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Source", "", 4, 4)
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	effect := AtEach(YourUpkeep, GainLifeFor(You, PowerOfWithLKI(This)))
	if err := ApplyEffect(g, effect, p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	original := g.CaptureDamageSource(p.ID())
	g.BouncePermanentToHand(p)
	a.RemoveFromHand(card.ID())
	g.PutOnBattlefield(card, b.PlayerID())
	for i := range 2 {
		g.RunStepWithPriority(core.Cleanup)
		sources := g.WaitingDelayedPermanentSources()
		if len(sources) != 1 || !sources[0].SameObject(original) || a.Life() != 20+i*4 {
			t.Fatal("waiting query changed state or lost the saved source")
		}
		g.SetActivePlayerIndex(0)
		g.RunStepWithPriority(core.Upkeep)
		if a.Life() != 24+i*4 {
			t.Fatalf("source LKI did not survive return and cleanup: life=%d", a.Life())
		}
	}
}
