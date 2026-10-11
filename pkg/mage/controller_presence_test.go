package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestControllerPresenceStateBatchAndReplacement(t *testing.T) {
	for _, scenario := range []string{"batch", "cancel", "exile"} {
		t.Run(scenario, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			dwarf := g.PutOnBattlefield(NewCreature("Dwarf", "", 1, 1, WithSubTypes("Dwarf")), a.PlayerID())
			source := g.PutOnBattlefield(NewCreature("Observer", "", 1, 1, WithAbility(WhenState(Controls(HasSubType("Dwarf")), Sacrifice(This)))), a.PlayerID())
			other := g.PutOnBattlefield(NewCreature("Other observer", "", 1, 1, WithAbility(WhenState(Controls(HasSubType("Dwarf")), Sacrifice(This)))), a.PlayerID())
			for range 3 {
				g.CheckStateTriggers()
			}
			g.PutTriggersOnStack()
			if g.StackSize() != 2 {
				t.Fatal("state instances collided")
			}
			if scenario == "batch" {
				if err := ReturnEachToHand(MatchingPermanents(HasSubType("Dwarf"))).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
					t.Fatal(err)
				}
			}
			if scenario == "cancel" {
				replacement := &testPrioritizedReplacement{matchFunc: func(action Action) bool { _, ok := action.(*SacrificePermanentAction); return ok }, replaceFunc: func(Action, *Game) Action { return nil }}
				g.effects.AddReplacement(replacement)
				g.ResolveTopOfStack()
				if g.FindPermanent(other.ID()) == nil {
					t.Fatal("canceled sacrifice moved subject")
				}
				g.CheckStateTriggers()
				g.PutTriggersOnStack()
				if g.StackSize() != 2 {
					t.Fatal("resolved state trigger did not rearm")
				}
				g.effects.replacements = nil
			}
			if scenario == "exile" {
				g.AddExileIfWouldGoToGraveyardThisTurn(source.ID(), dwarf.ID())
			}
			g.AddRegenerationShield(source.ID())
			g.MutablePermanent(source.ID()).GrantBaseAttr(core.Indestructible)
			clone := g.Clone()
			g.ResolveStack()
			if g.FindPermanent(source.ID()) != nil || g.FindPermanent(other.ID()) != nil {
				t.Fatal("sacrifice used destruction or rechecked state")
			}
			if scenario == "exile" && g.FindExiledCard(source.ID()) == nil {
				t.Fatal("replacement destination lost")
			}
			if clone.FindPermanent(source.ID()) == nil {
				t.Fatal("clone shared departure")
			}
			clone.ResolveStack()
			if clone.FindPermanent(source.ID()) != nil || clone.FindPermanent(other.ID()) != nil {
				t.Fatal("clone lost pending captures")
			}
		})
	}
}
