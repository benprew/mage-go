package ron

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONDelayedManaInspection(t *testing.T) {
	for _, state := range []string{"normal", "uncounterable", "illegal", "clone", "copy"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			victim := mage.NewInstant("Victim", "{3}")
			if state == "uncounterable" {
				mage.WithUncounterable()(victim)
			}
			target := castRONSpell(t, g, b, victim)
			obj := castRONSpell(t, g, a, repositoryCard(t, "LEG", "Mana Drain"), target.ID)
			a.ManaPool().Clear()
			b.ManaPool().Clear()
			if state == "illegal" {
				g.CounterStackObject(target.ID)
			}
			if state == "copy" {
				obj = g.CopyStackObjectDirect(obj, b.PlayerID(), false)
			}
			if state == "clone" {
				g = g.Clone()
				obj = g.StackPeek()
			}
			before := g.StackSize()
			for range 2 {
				if certain, known := g.InspectSpellDestruction(obj, nil); certain || !known {
					t.Fatal("complete Mana Drain inspection unknown")
				}
			}
			if g.StackSize() != before || len(g.DelayedTriggers()) != 0 || g.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 0 || g.GetPlayer(b.PlayerID()).ManaPool().TotalMana() != 0 {
				t.Fatal("inspection changed state")
			}
			g.ResolveTopOfStack()
			want := 1
			if state == "illegal" {
				want = 0
			}
			if len(g.DelayedTriggers()) != want {
				t.Fatal("wrong schedule count")
			}
			if want == 0 {
				return
			}
			controller := a.PlayerID()
			if state == "copy" {
				controller = b.PlayerID()
			}
			g.FireEvent(core.GameEvent{Type: core.EvtMainPhase, PlayerID: controller})
			g.PutTriggersOnStack()
			if g.GetPlayer(controller).ManaPool().TotalMana() != 0 {
				t.Fatal("delayed body ran before resolution")
			}
			g.ResolveTopOfStack()
			if g.GetPlayer(controller).ManaPool().Count(core.Colorless) != 3 {
				t.Fatal("saved spell value lost")
			}
		})
	}
}
