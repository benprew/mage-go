package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestSourceCounterCap(t *testing.T) {
	for _, batch := range []bool{false, true} {
		g, a, _ := newSpellCopyGame()
		donor := g.PutOnBattlefield(NewArtifact("Doubler", ""), a.PlayerID())
		g.AddCounterDoubler(donor.ID(), core.Dream, IsCreature)
		card := NewCreature("Cap", "", 1, 20, WithAbility(EntersWithNCounters(core.Dream, 7)), WithStaticAbility(SourceCounterCap(core.Dream, 7)))
		p := g.PutOnBattlefield(card, a.PlayerID())
		if p.Counters[core.Dream] != 7 {
			t.Fatal("entry replacement exceeded cap", p.Counters[core.Dream])
		}
		p.RemoveCounter(core.Dream, 3)
		clone := g.Clone()
		if batch {
			g.AddCountersTogether([]*Permanent{p, p}, core.Dream, 4, p.ID())
		} else {
			g.AddCountersWithReplacement(p, core.Dream, 4, p.ID(), false)
		}
		p = g.FindPermanent(p.ID())
		if p.Counters[core.Dream] != 7 || clone.FindPermanent(p.ID()).Counters[core.Dream] != 4 {
			t.Fatal("placement cap or clone isolation")
		}
		g.AddCountersWithReplacement(p, core.Charge, 10, p.ID(), false)
		if g.FindPermanent(p.ID()).Counters[core.Charge] != 10 {
			t.Fatal("cap changed other counter type")
		}
		phased := g.PhaseOut(p.ID())
		g.AddCountersTogether([]*Permanent{p}, core.Dream, 4, p.ID())
		if g.FindPermanentIncludingPhased(p.ID()).Counters[core.Dream] != 7 {
			t.Fatal("phased placement")
		}
		g.PhaseIn(phased)
		// A new visit must get its own entry counters and rule.
		p = g.FindPermanent(p.ID())
		g.RemoveFromBattlefield(p)
		returned := g.PutOnBattlefield(card, a.PlayerID())
		if returned.Counters[core.Dream] != 7 || returned.incarnationID == p.incarnationID {
			t.Fatal("return")
		}
		g.MutablePermanent(returned.ID()).RemoveCounter(core.Dream, 2)
		g.AddCountersWithReplacement(p, core.Dream, 2, p.ID(), false)
		if g.FindPermanent(returned.ID()).Counters[core.Dream] != 5 {
			t.Fatal("departed visit changed returned counters")
		}
		live := g.MutablePermanent(returned.ID())
		g.AddContinuousEffect(RemoveAllAbilities(live.ID(), core.Indefinite))
		g.ApplyContinuousEffects()
		g.AddCountersWithReplacement(live, core.Dream, 2, live.ID(), false)
		if g.FindPermanent(live.ID()).Counters[core.Dream] != 9 {
			t.Fatal("removed ability kept cap")
		}
	}
}

func TestSourceCounterCapDefinition(t *testing.T) {
	for _, tc := range []struct {
		counter core.CounterType
		limit   int
		valid   bool
	}{{core.Dream, 0, true}, {core.Dream, 7, true}, {core.Dream, 300, true}, {core.Dream, -1, false}, {core.NumCounters, 7, false}} {
		c := NewArtifact("Cap", "", WithStaticAbility(SourceCounterCap(tc.counter, tc.limit)))
		if err := ValidateCardDefinition(c.Name(), c); (err == nil) != tc.valid {
			t.Fatalf("definition %v: %v", tc, err)
		}
	}
}

func TestSourceCounterCapBatchAndMultipleRules(t *testing.T) {
	g, a, _ := newSpellCopyGame()
	capped := g.PutOnBattlefield(NewArtifact("Two caps", "", WithStaticAbility(SourceCounterCap(core.Charge, 7), SourceCounterCap(core.Charge, 3))), a.PlayerID())
	uncapped := g.PutOnBattlefield(NewArtifact("No cap", ""), a.PlayerID())
	g.AddCountersTogether([]*Permanent{capped, uncapped, capped}, core.Charge, 10, capped.ID())
	if g.FindPermanent(capped.ID()).Counters[core.Charge] != 3 || g.FindPermanent(uncapped.ID()).Counters[core.Charge] != 10 {
		t.Fatal("batch caps or duplicate visits")
	}
	zero := g.PutOnBattlefield(NewArtifact("Zero", "", WithAbility(EntersWithNCounters(core.Charge, 10)), WithStaticAbility(SourceCounterCap(core.Charge, 0))), a.PlayerID())
	if zero.Counters[core.Charge] != 0 {
		t.Fatal("zero cap")
	}
}

func TestSourceCounterCapStateBased(t *testing.T) {
	for _, phased := range []bool{false, true} {
		g, a, _ := newSpellCopyGame()
		p := g.PutOnBattlefield(NewArtifact("Cap", "", WithStaticAbility(SourceCounterCap(core.Charge, 7))), a.PlayerID())
		donor := g.PutOnBattlefield(NewArtifact("Remover", ""), a.PlayerID())
		removal := RemoveAllAbilities(p.ID(), core.WhileOnBattlefield)
		removal.SetSourceID(donor.ID())
		g.AddContinuousEffect(removal)
		g.AddCountersWithReplacement(p, core.Charge, 10, p.ID(), false)
		clone := g.Clone()
		if phased {
			g.PhaseOut(p.ID())
		}
		g.RemoveFromBattlefield(donor)
		g.CheckStateBasedActions()
		want := 7
		if phased {
			want = 10
		}
		if g.FindPermanentIncludingPhased(p.ID()).Counters[core.Charge] != want || clone.FindPermanent(p.ID()).Counters[core.Charge] != 10 {
			t.Fatal("state-based cap or phasing/clone")
		}
		if phased {
			g.PhaseIn([]uuid.UUID{p.ID()})
			g.ApplyContinuousEffects()
			g.CheckStateBasedActions()
			if g.FindPermanent(p.ID()).Counters[core.Charge] != 7 {
				t.Fatal("phase-in cap")
			}
		}
	}
}

type capReturnChooser struct {
	*BasePlayer
	onChoice func()
}

func (p *capReturnChooser) ChooseMode([]string, string) int {
	if p.onChoice != nil {
		choice := p.onChoice
		p.onChoice = nil
		choice()
	}
	return 0
}

func TestSourceCounterCapReplacementReturn(t *testing.T) {
	for _, batch := range []bool{false, true} {
		a := &capReturnChooser{BasePlayer: NewBasePlayer("A")}
		b := NewBasePlayer("B")
		g := NewGame(a, b)
		first := g.PutOnBattlefield(NewArtifact("Doubler 1", ""), a.PlayerID())
		second := g.PutOnBattlefield(NewArtifact("Doubler 2", ""), a.PlayerID())
		g.AddCounterDoubler(first.ID(), core.Charge, AnyPermanent)
		g.AddCounterDoubler(second.ID(), core.Charge, AnyPermanent)
		card := NewArtifact("Cap", "", WithStaticAbility(SourceCounterCap(core.Charge, 7)))
		p := g.PutOnBattlefield(card, a.PlayerID())
		a.onChoice = func() { g.RemoveFromBattlefield(g.FindPermanent(p.ID())); g.PutOnBattlefield(card, a.PlayerID()) }
		if batch {
			g.AddCountersTogether([]*Permanent{p}, core.Charge, 1, first.ID())
		} else {
			g.AddCountersWithReplacement(p, core.Charge, 1, first.ID(), false)
		}
		if g.FindPermanent(p.ID()).Counters[core.Charge] != 0 {
			t.Fatal("replacement choice followed returned visit")
		}
	}
}

func TestSourceCounterCapCopiedEntry(t *testing.T) {
	a := &optionalEntryPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	plan := entryCopyPlan(t, "Donor", func() []CardOption {
		return []CardOption{WithAbility(EntersWithNCounters(core.Dream, 10)), WithStaticAbility(SourceCounterCap(core.Dream, 7))}
	})
	donor, err := plan.Build()
	if err != nil {
		t.Fatal(err)
	}
	source := g.PutOnBattlefield(donor, b.PlayerID())
	source.RemoveCounter(core.Dream, 2)
	physical, err := NewCardDefinitionPlan(func() Card { return NewCreature("Physical", "", 1, 1, WithAbility(MayCopyAsEnters(IsCreature))) })
	if err != nil {
		t.Fatal(err)
	}
	card, err := physical.Build()
	if err != nil {
		t.Fatal(err)
	}
	copied := g.PutOnBattlefield(card, a.PlayerID())
	if copied.Name() != "Donor" || copied.Counters[core.Dream] != 7 || source.Counters[core.Dream] != 5 {
		t.Fatal("copied cap or entry counters")
	}
	g.AddCountersWithReplacement(copied, core.Dream, 10, copied.ID(), false)
	if g.FindPermanent(copied.ID()).Counters[core.Dream] != 7 || source.Counters[core.Dream] != 5 {
		t.Fatal("copied cap not independent")
	}
}
