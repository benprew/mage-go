package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestLeastPowerComparisonSelection(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first := g.PutOnBattlefield(NewCreature("First", "", -2, 4), a.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", -2, 4), b.PlayerID())
	high := g.PutOnBattlefield(NewCreature("High", "", 3, 4), a.PlayerID())
	set := PowerEqualToLeastOf(PermanentsControlledBy(You, IsCreature), MatchingPermanents(IsCreature))
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	check := func(want ...*Permanent) {
		t.Helper()
		got := set.resolvePermanents(ctx)
		if len(got) != len(want) {
			t.Fatalf("members=%d want=%d", len(got), len(want))
		}
		for i, p := range got {
			if p.ID() != want[i].ID() {
				t.Fatal("wrong comparison member")
			}
		}
	}
	check(first)
	g.MutablePermanent(first.ID()).AddCounter(core.P1P1, 1)
	check()
	g.PhaseOut(second.ID())
	check(first)
	g.PhaseOut(first.ID())
	check(high)
	g.PhaseOut(high.ID())
	check()
	clone := g.Clone()
	clone.PhaseIn([]uuid.UUID{second.ID()})
	if got := set.resolvePermanents(&EffectContext{Game: clone, Controller: b.PlayerID()}); len(got) != 1 || got[0].ID() != second.ID() {
		t.Fatal("clone selection")
	}
	check()
}

func TestLeastPowerComparisonDeclaration(t *testing.T) {
	for _, sets := range [][2]PermanentSet{
		{nil, MatchingPermanents(IsCreature)}, {MatchingPermanents(IsCreature), nil},
		{MatchingPermanents(IsCreature), DeclarePermanentTargets("undeclared", 1, 1, IsCreature)},
	} {
		c := NewInstant("Invalid", "", WithAbility(Spell(TapEach(PowerEqualToLeastOf(sets[0], sets[1])))))
		if ValidateCardDefinition(c.Name(), c) == nil {
			t.Fatal("invalid comparison accepted")
		}
	}
}

func TestLeastPowerComparisonWaitingSource(t *testing.T) {
	for _, side := range []string{"subjects", "compared"} {
		t.Run(side, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			c := NewCreature("Source", "", -1, 4)
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			saved := g.CaptureDamageSource(p.ID())
			calls := 0
			filter := NewPermanentFilter("query must not run", func(*Permanent, *Game) bool { calls++; return true })
			self := FilterPermanentSet(SingletonPermanent(This), filter)
			subjects, compared := MatchingPermanents(filter), self
			if side == "subjects" {
				subjects, compared = self, MatchingPermanents(filter)
			}
			if err := ApplyEffect(g, AtNext(EachEndStep, TapEach(PowerEqualToLeastOf(subjects, compared))), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.BouncePermanentToHand(p)
			a.RemoveFromHand(c.ID())
			returned := g.PutOnBattlefield(c, b.PlayerID())
			clone := g.Clone()
			for _, game := range []*Game{g, clone} {
				got := game.WaitingDelayedPermanentSources()
				if len(got) != 1 || !got[0].SameObject(saved) || calls != 0 {
					t.Fatal("waiting query evaluated selection or lost source")
				}
			}
			clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			clone.PutTriggersOnStack()
			clone.ResolveStack()
			if clone.FindPermanent(returned.ID()).Tapped || len(clone.WaitingDelayedPermanentSources()) != 0 || len(g.WaitingDelayedPermanentSources()) != 1 {
				t.Fatal("return or clone lifetime changed")
			}
		})
	}
}

func TestLeastPowerComparisonChoiceIdentity(t *testing.T) {
	for _, change := range []string{"control", "phase", "return"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			c := NewCreature("Chosen", "", -1, 4)
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			binding := BindPermanent("chosen")
			ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
			if err := ChoosePermanent(You, PowerEqualToLeastOf(MatchingPermanents(IsCreature), MatchingPermanents(IsCreature)), binding).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			clone := g.Clone()
			switch change {
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: p.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "phase":
				g.PhaseOut(p.ID())
			case "return":
				g.BouncePermanentToHand(p)
				a.RemoveFromHand(c.ID())
				g.PutOnBattlefield(c, b.PlayerID())
			}
			if err := DestroyWithoutRegeneration(From(binding)).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			if (g.FindPermanentIncludingPhased(p.ID()) == nil) != (change == "control") {
				t.Fatal("choice lost exact visit")
			}
			if err := DestroyWithoutRegeneration(From(binding)).Apply(&EffectContext{Game: clone, Controller: a.PlayerID()}); err != nil || clone.FindPermanent(p.ID()) != nil {
				t.Fatal("clone choice", err)
			}
			if change == "phase" {
				g.PhaseIn([]uuid.UUID{p.ID()})
				if err := DestroyWithoutRegeneration(From(binding)).Apply(ctx); err != nil || g.FindPermanent(p.ID()) != nil {
					t.Fatal("phase cycle lost choice", err)
				}
			}
		})
	}
}
