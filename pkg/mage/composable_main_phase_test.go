package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestTypedMainPhaseTriggers(t *testing.T) {
	for _, first := range []bool{true, false} {
		pattern := YourFirstMainPhase
		if !first {
			pattern = YourPostcombatMainPhase
		}
		for _, modal := range []bool{false, true} {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			trigger := At(pattern, GainLifeFor(You, Fixed(2)))
			if modal {
				trigger = AtModes(pattern, ModeOf("Two", GainLifeFor(You, Fixed(2))), ModeOf("Three", GainLifeFor(You, Fixed(3))))
			}
			source := NewEnchantment("Main phase", "", WithAbility(trigger))
			source.SetOwner(a.PlayerID())
			if err := ValidateCardDefinition(source.Name(), source); err != nil {
				t.Fatal(err)
			}
			g.PutOnBattlefield(source, a.PlayerID())
			for _, event := range []GameEvent{
				{Type: EvtMainPhase, PlayerID: b.PlayerID(), Flag: first},
				{Type: EvtMainPhase, PlayerID: a.PlayerID(), Flag: !first},
				{Type: EvtUpkeep, PlayerID: a.PlayerID(), Flag: first},
			} {
				g.FireEvent(event)
			}
			if len(g.triggers.Pending()) != 0 {
				t.Fatal("unrelated phase matched")
			}
			g.FireEvent(GameEvent{Type: EvtMainPhase, PlayerID: a.PlayerID(), Flag: first})
			if len(g.triggers.Pending()) != 1 || a.Life() != 20 {
				t.Fatal("matching phase did not queue one trigger")
			}
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatal("main phase trigger did not use the stack")
			}
			g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			g.ResolveStack()
			if a.Life() != 22 {
				t.Fatalf("life after source departure = %d", a.Life())
			}
		}
	}
}

func TestTypedDelayedMainPhase(t *testing.T) {
	for _, first := range []bool{true, false} {
		pattern := YourFirstMainPhase
		if !first {
			pattern = YourPostcombatMainPhase
		}
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		event := GameEvent{Type: EvtMainPhase, PlayerID: a.PlayerID(), Flag: first}
		g.FireEvent(event)
		if err := ApplyEffect(g, AtNext(pattern, GainLifeFor(You, Fixed(2))), uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		for _, wrong := range []GameEvent{
			{Type: EvtMainPhase, PlayerID: b.PlayerID(), Flag: first},
			{Type: EvtMainPhase, PlayerID: a.PlayerID(), Flag: !first},
		} {
			g.FireEvent(wrong)
		}
		if len(g.DelayedTriggers()) != 1 || len(g.triggers.Pending()) != 0 {
			t.Fatal("wrong main phase consumed delayed trigger")
		}
		clone := g.Clone()
		clone.FireEvent(event)
		if len(clone.triggers.Pending()) != 1 || clone.StackSize() != 0 {
			t.Fatal("delayed trigger bypassed pending placement")
		}
		clone.ResolveStack()
		clone.FireEvent(event)
		if clone.GetPlayer(a.PlayerID()).Life() != 22 || len(clone.triggers.Pending()) != 0 || len(clone.DelayedTriggers()) != 0 {
			t.Fatal("delayed trigger did not fire exactly once")
		}
		if a.Life() != 20 || len(g.DelayedTriggers()) != 1 {
			t.Fatal("clone consumed original trigger")
		}
		g.FireEvent(event)
		g.ResolveStack()
		if a.Life() != 22 {
			t.Fatal("original delayed trigger did not resolve")
		}
	}
}
