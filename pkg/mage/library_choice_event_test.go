package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestLibraryChoiceEventsCloneAndSequence(t *testing.T) {
	for _, tc := range []struct {
		event EventType
		build func(ResolutionValue) ResolvingEffect
	}{{EvtScry, ScryAtResolution}, {EvtSurveil, SurveilAtResolution}} {
		t.Run(tc.event.String(), func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			g.PutOnBattlefield(NewEnchantment("Watcher", "", WithAbility(NewTriggered(tc.event, false, GainLifeFor(You, Fixed(1))))), a.PlayerID())
			clone := g.Clone()
			if err := ApplyEffect(clone, Sequence(tc.build(Fixed(1)), tc.build(Fixed(2))), uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("clone action queued an original trigger")
			}
			clone.PutTriggersOnStack()
			if clone.StackSize() != 2 {
				t.Fatalf("stack = %d, want two distinct actions", clone.StackSize())
			}
			clone.ResolveTopOfStack()
			clone.ResolveTopOfStack()
			if clone.GetPlayer(a.PlayerID()).Life() != 22 || a.Life() != 20 {
				t.Fatal("trigger resolution lost clone isolation")
			}
		})
	}
}

func TestLibraryChoiceAbsentPlayerHasNoEvent(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	for _, event := range []EventType{EvtScry, EvtSurveil} {
		g.PutOnBattlefield(NewEnchantment("Watcher", "", WithAbility(NewTriggered(event, false, GainLifeFor(You, Fixed(1))))), a.PlayerID())
	}
	if g.PerformScry(nil, 2) != 0 || g.PerformSurveil(nil, 2) != 0 {
		t.Fatal("absent player examined cards")
	}
	for _, effect := range []Effect{ScryAtResolution(Fixed(2)), SurveilAtResolution(Fixed(2))} {
		if err := ApplyEffect(g, effect, uuid.Nil, uuid.New(), nil); err == nil {
			t.Fatal("missing controller was accepted")
		}
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("absent player emitted an event")
	}
}
