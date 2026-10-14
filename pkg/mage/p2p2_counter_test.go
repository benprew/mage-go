package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestP2P2CounterLifecycle(t *testing.T) {
	g, a, _ := newSpellCopyGame()
	p := g.PutOnBattlefield(NewCreature("Counter guest", "", 1, 3), a.PlayerID())
	g.AddCounterDoubler(p.ID(), core.P2P2, PermanentFilter{})
	g.AddCountersWithReplacement(p, core.P2P2, 1, p.ID(), false)
	p = g.MutablePermanent(p.ID())
	if p.Counters[core.P2P2] != 2 || p.CurrentPower(g) != 5 || p.CurrentToughness(g) != 7 {
		t.Fatal("counter replacement or P/T failed")
	}
	p.AddCounter(core.M1M1, 1)
	g.CheckStateBasedActions()
	p = g.MutablePermanent(p.ID())
	if p.Counters[core.P2P2] != 2 || p.Counters[core.M1M1] != 1 || p.CurrentPower(g) != 4 || p.CurrentToughness(g) != 6 {
		t.Fatal("different counter kinds canceled")
	}
	clone := g.Clone()
	p = g.MutablePermanent(p.ID())
	if p.RemoveCounter(core.P2P2, 3) || p.Counters[core.P2P2] != 2 {
		t.Fatal("failed removal changed counters")
	}
	if !p.RemoveCounter(core.P2P2, 1) || clone.FindPermanent(p.ID()).Counters[core.P2P2] != 2 {
		t.Fatal("clone shared counter storage")
	}
	g.PhaseIn(g.PhaseOut(p.ID()))
	p = g.FindPermanent(p.ID())
	if p.Counters[core.P2P2] != 1 {
		t.Fatal("phasing lost counters")
	}
	visit := p.incarnationID
	g.BouncePermanentToHand(p)
	if g.zones.departedCounters[visit][core.P2P2] != 1 {
		t.Fatal("departure lost counters")
	}
	a.RemoveFromHand(p.ID())
	returned := g.PutOnBattlefield(p.Card, a.PlayerID())
	if returned.Counters[core.P2P2] != 0 || returned.CurrentPower(g) != 1 || returned.incarnationID == visit {
		t.Fatal("return kept counters or visit")
	}
}

func TestP2P2CounterLayers(t *testing.T) {
	g, a, _ := newSpellCopyGame()
	p := g.PutOnBattlefield(NewCreature("Layer guest", "", 1, 3), a.PlayerID())
	p.AddCounter(core.P2P2, 3)
	if err := ApplyEffect(g, Until(core.EndOfTurn, SetBasePowerToughness(This, Fixed(2), Fixed(5)), ModifyPT(This, Fixed(1), Fixed(2)), SwitchPowerToughness(This)), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.ApplyContinuousEffects()
	p = g.FindPermanent(p.ID())
	if p.CurrentPower(g) != 13 || p.CurrentToughness(g) != 9 {
		t.Fatal("counters must follow base values and precede switching")
	}
	if core.P2P2.String() != "+2/+2" || core.P2P2 >= core.NumCounters {
		t.Fatal("counter metadata or array bound failed")
	}
}
