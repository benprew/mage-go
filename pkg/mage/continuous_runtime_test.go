package mage

import (
	"testing"
	"uuid"
)

func TestContinuousContextController(t *testing.T) {
	for _, tc := range []struct {
		name    string
		printed bool
		bound   bool
	}{
		{name: "unbound"},
		{name: "bound", bound: true},
		{name: "printed", printed: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := newPriorityTestGame()
			owner := g.players[0].PlayerID()
			other := g.players[1].PlayerID()
			source := g.PutOnBattlefield(NewCreature("Source", "{0}", 1, 1), owner)
			var controller uuid.UUID
			if tc.bound {
				controller = other
			}
			g.effects.AddWithController(nil, source.ID(), controller)
			e := g.effects.effects[len(g.effects.effects)-1]
			e.printed = tc.printed
			stored := e.controllerID
			for _, current := range []uuid.UUID{owner, other} {
				source.computedController = current
				want := stored
				if tc.printed {
					want = current
				}
				if got := e.context(g, true).Controller; got != want {
					t.Fatalf("context controller = %v, want %v", got, want)
				}
				if e.controllerID != stored {
					t.Fatal("context changed the stored controller")
				}
			}
		})
	}
}
