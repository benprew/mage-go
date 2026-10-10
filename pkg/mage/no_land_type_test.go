package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestNoLandTypeStateContract(t *testing.T) {
	for _, scenario := range []string{"token", "type-loss", "control", "phase", "cancel", "batch"} {
		t.Run(scenario, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			g.SetTurn(1)
			g.PutOnBattlefield(NewLand("Foreign land"), b.PlayerID())
			card := Card(NewLand("Land"))
			if scenario == "token" {
				card = NewToken("Land token", 0, 0, []core.CardType{core.TypeLand}, nil)
			}
			land := g.PutOnBattlefield(card, a.PlayerID())
			source := g.PutOnBattlefield(NewCreature("Observer", "", 1, 1, WithAbility(WhenState(ControlsNone(IsLand), Sacrifice(This)))), a.PlayerID())
			g.CheckStateTriggers()
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("land did not satisfy condition")
			}
			switch scenario {
			case "type-loss":
				g.MutablePermanent(land.ID()).RevokeBaseAttr(core.AttrIsLand)
				g.MutablePermanent(land.ID()).GrantBaseAttr(core.AttrIsArtifact)
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: source.ID(), TargetID: land.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "phase":
				g.PhaseOut(land.ID())
			case "batch":
				other := g.PutOnBattlefield(NewLand("Other land"), a.PlayerID())
				if err := ReturnEachToHand(MatchingPermanents(IsLand)).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
					t.Fatal(err)
				}
				if g.FindPermanent(other.ID()) != nil {
					t.Fatal("batch did not finish")
				}
			default:
				g.BouncePermanentToHand(land)
			}
			g.CheckStateTriggers()
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatal("no-land trigger missing")
			}
			clone := g.Clone()
			g.CheckStateTriggers()
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatal("active trigger repeated")
			}
			if scenario == "cancel" {
				replacement := &testPrioritizedReplacement{matchFunc: func(a Action) bool { _, ok := a.(*SacrificePermanentAction); return ok }, replaceFunc: func(Action, *Game) Action { return nil }}
				g.effects.AddReplacement(replacement)
				g.ResolveTopOfStack()
				if g.FindPermanent(source.ID()) == nil {
					t.Fatal("canceled sacrifice moved source")
				}
				g.CheckStateTriggers()
				g.PutTriggersOnStack()
				if g.StackSize() != 1 {
					t.Fatal("resolved trigger did not rearm")
				}
				g.effects.replacements = nil
			}
			g.PutOnBattlefield(NewLand("Acquired"), a.PlayerID())
			g.ResolveStack()
			if g.FindPermanent(source.ID()) != nil {
				t.Fatal("acquired land canceled pending trigger")
			}
			if clone.FindPermanent(source.ID()) == nil {
				t.Fatal("clone shares source movement")
			}
			clone.ResolveStack()
			if clone.FindPermanent(source.ID()) != nil {
				t.Fatal("clone lost pending trigger")
			}
		})
	}
}

func TestNoLandTypeStatePhaseCycle(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	land := g.PutOnBattlefield(NewLand("Land"), a.PlayerID())
	source := g.PutOnBattlefield(NewCreature("Observer", "", 1, 1, WithAbility(WhenState(ControlsNone(IsLand), Sacrifice(This)))), a.PlayerID())
	g.PhaseOut(source.ID())
	g.BouncePermanentToHand(land)
	g.CheckStateTriggers()
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased source detected state")
	}
	g.PhaseIn([]uuid.UUID{source.ID()})
	g.CheckStateTriggers()
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("phase-in did not detect state")
	}
	g.ResolveStack()
	if g.FindPermanent(source.ID()) != nil {
		t.Fatal("state trigger did not resolve")
	}
}
