package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCounterCapacity(t *testing.T) {
	for _, count := range []int{255, 256, 300, 65536} {
		g, a, _ := newSpellCopyGame()
		p := g.PutOnBattlefield(NewCreature("Counters", "", 0, 1, WithAbility(EntersWithNCounters(core.P1P1, count))), a.PlayerID())
		if p.Counters[core.P1P1] != count || p.CurrentPower(g) != count {
			t.Fatalf("entry %d: counters=%d power=%d", count, p.Counters[core.P1P1], p.CurrentPower(g))
		}
		clone := g.Clone()
		g.AddCountersWithReplacement(p, core.P1P1, count, p.ID(), false)
		p = g.MutablePermanent(p.ID())
		if p.Counters[core.P1P1] != 2*count {
			t.Fatal("addition truncated")
		}
		if p.RemoveCounter(core.P1P1, 2*count+1) || p.Counters[core.P1P1] != 2*count {
			t.Fatal("failed removal changed counters")
		}
		if !p.RemoveCounter(core.P1P1, count+1) || p.Counters[core.P1P1] != count-1 {
			t.Fatal("removal truncated")
		}
		g.PhaseIn(g.PhaseOut(p.ID()))
		if g.FindPermanent(p.ID()).Counters[core.P1P1] != count-1 || clone.FindPermanent(p.ID()).Counters[core.P1P1] != count {
			t.Fatal("phase or clone changed count")
		}
		p = g.FindPermanent(p.ID())
		visit := p.incarnationID
		g.BouncePermanentToHand(p)
		if g.zones.departedCounters[visit][core.P1P1] != count-1 {
			t.Fatal("departure snapshot truncated")
		}
		a.RemoveFromHand(p.ID())
		returned := g.PutOnBattlefield(p.Card, a.PlayerID())
		if returned.Counters[core.P1P1] != count || returned.incarnationID == visit {
			t.Fatal("return kept old counters")
		}
	}
}
