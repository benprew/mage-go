package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestSourceBecomesTappedTransitionAndIdentity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(BecomesTapped(This),
		Sequence(GainLifeFor(You, Fixed(1)), Until(EndOfTurn, ModifyPT(This, Fixed(2), Fixed(2)))))))
	watcher.SetOwner(a.PlayerID())
	other := NewLand("Other")
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutOnBattlefield(other, b.PlayerID())
	g.TapPermanent(g.FindPermanent(other.ID()))
	g.TapPermanent(g.FindPermanent(watcher.ID()))
	g.TapPermanent(g.FindPermanent(watcher.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("tap transition queued %d triggers, want 1", g.StackSize())
	}
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.PutOnBattlefield(watcher, b.PlayerID())
	clone := g.Clone()
	clone.ResolveStack()
	if a.Life() != 20 || b.Life() != 20 {
		t.Fatal("clone changed original life totals")
	}
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 20 {
		t.Fatal("pending trigger changed controller after source return")
	}
	if g.FindPermanent(watcher.ID()).CurrentPower(g) != 1 {
		t.Fatal("old tap trigger modified returned incarnation")
	}
	if clone.GetPlayer(a.PlayerID()).Life() != 21 {
		t.Fatal("clone lost pending tap trigger")
	}
}

func TestSourceBecomesTappedAbsentPhasedAndCopy(t *testing.T) {
	a, b := newRecordingPlayer("A"), newRecordingPlayer("B")
	g := NewGame(a, b)
	target := DeclarePlayerTarget("recipient")
	watcher := NewLand("Watcher", WithAbility(WhenEvent(BecomesTapped(This),
		GainLifeFor(target, Fixed(2)), WithTargets(target))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	phase := g.PhaseOut(watcher.ID())
	g.FireEvent(GameEvent{Type: EvtTapped, SourceID: watcher.ID(), PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased source triggered")
	}
	g.PhaseIn(phase)
	a.chooseQueue = [][]uuid.UUID{{a.PlayerID()}}
	g.TapPermanent(g.FindPermanent(watcher.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 1 || len(g.StackPeek().Targets) != 1 {
		t.Fatal("tap trigger did not announce its target")
	}
	stackCopy := g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
	if stackCopy == nil || len(stackCopy.Targets) != 1 || stackCopy.Targets[0] != a.PlayerID() {
		t.Fatal("tap trigger copy lost its target")
	}
	g.ResolveStack()
	if a.Life() != 24 || b.Life() != 20 {
		t.Fatal("tap trigger copy changed target")
	}
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.FireEvent(GameEvent{Type: EvtTapped, SourceID: watcher.ID(), PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("absent source triggered")
	}
}

func TestSourceBecomesTappedDeclaration(t *testing.T) {
	card := NewLand("Valid", WithAbility(WhenEvent(BecomesTapped(This), GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	meta := InspectDefinition(card.Abilities()[0])
	if meta.Role != "event-trigger" || len(meta.Children) == 0 || meta.Children[0].Role != "event-pattern" {
		t.Fatal("source tap pattern missing from inspection")
	}
	spell := NewInstant("Invalid", "", WithAbility(WhenEvent(BecomesTapped(This), GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(spell.Name(), spell); err == nil || !strings.Contains(err.Error(), "battlefield") {
		t.Fatalf("spell source tap validation = %v", err)
	}
}

func TestSourceBecomesTappedDoesNotMatchEntryTapped(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	land := NewLand("Entry watcher", WithKeyword(EntersTapped),
		WithAbility(WhenEvent(BecomesTapped(This), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(land, a.PlayerID())
	g.PutTriggersOnStack()
	if g.StackSize() != 0 || a.Life() != 20 {
		t.Fatal("entering tapped caused a source tap trigger")
	}
	g.UntapPermanent(g.FindPermanent(land.ID()))
	g.TapPermanent(g.FindPermanent(land.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("later tap transition did not trigger")
	}
}
