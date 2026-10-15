package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCounterDurationLifetime(t *testing.T) {
	for _, change := range []string{"none", "partial", "other-counter", "failed-remove", "remove-readd", "source-leave", "source-phase", "target-phase", "target-return", "cleanup", "control"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
			card := NewCreature("Target", "", 2, 3)
			target := g.PutOnBattlefield(card, b.PlayerID())
			target.AddCounter(core.Mire, 2)
			g.MutablePermanent(source.ID()).AttachedTo = target.ID()
			effect := UntilPermanentHasCounter(AttachedPermanent, core.Mire, ModifyPT(AttachedPermanent, Fixed(2), Fixed(1)), GrantKeywordModification(AttachedPermanent, core.Flying))
			if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			clone := g.Clone()
			switch change {
			case "partial":
				g.MutablePermanent(target.ID()).RemoveCounter(core.Mire, 1)
			case "other-counter":
				g.MutablePermanent(target.ID()).AddCounter(core.Charge, 1)
				g.MutablePermanent(target.ID()).RemoveCounter(core.Charge, 1)
			case "failed-remove":
				g.MutablePermanent(target.ID()).RemoveCounter(core.Mire, 3)
			case "remove-readd":
				g.MutablePermanent(target.ID()).RemoveCounter(core.Mire, 2)
				g.MutablePermanent(target.ID()).AddCounter(core.Mire, 1)
			case "source-leave":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			case "source-phase":
				g.PhaseIn(g.PhaseOut(source.ID()))
			case "target-phase":
				g.PhaseIn(g.PhaseOut(target.ID()))
			case "target-return":
				g.RemoveFromBattlefield(g.FindPermanent(target.ID()))
				target = g.PutOnBattlefield(card, b.PlayerID())
				target.AddCounter(core.Mire, 1)
			case "cleanup":
				g.effects.RemoveEndOfTurn()
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: source.ID(), TargetID: target.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			}
			g.ApplyContinuousEffects()
			active := change != "remove-readd" && change != "target-phase" && change != "target-return"
			want := 2
			if active {
				want = 4
			}
			live := g.FindPermanent(target.ID())
			if live.CurrentPower(g) != want || live.HasKeyword(core.Flying) != active {
				t.Fatalf("power=%d flying=%v, want %d/%v", live.CurrentPower(g), live.HasKeyword(core.Flying), want, active)
			}
			if clone.FindPermanent(target.ID()).CurrentPower(clone) != 4 {
				t.Fatal("original changed clone")
			}
			clone.MutablePermanent(target.ID()).RemoveCounter(core.Mire, 2)
			clone.ApplyContinuousEffects()
			if clone.FindPermanent(target.ID()).CurrentPower(clone) != 2 {
				t.Fatal("clone did not expire")
			}
		})
	}
}

func TestCounterDurationStartAndDeclarations(t *testing.T) {
	for _, start := range []string{"none", "counter", "phased"} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		target := g.PutOnBattlefield(NewCreature("Target", "", 2, 3), a.PlayerID())
		if start != "none" {
			target.AddCounter(core.Mire, 1)
		}
		if start == "phased" {
			g.PhaseOut(target.ID())
		}
		if err := ApplyEffect(g, UntilPermanentHasCounter(This, core.Mire, ModifyPT(This, Fixed(2), Fixed(0))), target.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if start == "none" {
			g.MutablePermanent(target.ID()).AddCounter(core.Mire, 1)
		}
		if start == "phased" {
			g.PhaseIn([]uuid.UUID{target.ID()})
		}
		g.ApplyContinuousEffects()
		want := 2
		if start == "counter" {
			want = 4
		}
		if g.FindPermanent(target.ID()).CurrentPower(g) != want {
			t.Fatal("wrong initial condition", start)
		}
	}
	for _, effect := range []ResolvingEffect{
		UntilPermanentHasCounter(nil, core.Mire, ModifyPT(This, Fixed(1), Fixed(0))),
		UntilPermanentHasCounter(This, core.CounterType(-1), ModifyPT(This, Fixed(1), Fixed(0))),
		UntilPermanentHasCounter(This, core.NumCounters, ModifyPT(This, Fixed(1), Fixed(0))),
		UntilPermanentHasCounter(This, core.Mire),
		UntilPermanentHasCounter(This, core.Mire, nil),
	} {
		card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), effect)))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("accepted invalid duration")
		}
	}
}

func TestCounterDurationSeparateSubjectAndStateActions(t *testing.T) {
	for _, change := range []string{"recipient-phase", "watched-phase", "annihilate"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			watched := g.PutOnBattlefield(NewArtifact("Watched", ""), a.PlayerID())
			target := g.PutOnBattlefield(NewCreature("Target", "", 2, 3), b.PlayerID())
			watched.AddCounter(core.P1P1, 1)
			g.MutablePermanent(watched.ID()).AttachedTo = target.ID()
			effect := UntilPermanentHasCounter(This, core.P1P1, ModifyPT(AttachedPermanent, Fixed(1), Fixed(0)))
			if err := ApplyEffect(g, effect, watched.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "recipient-phase":
				g.PhaseIn(g.PhaseOut(target.ID()))
			case "watched-phase":
				g.PhaseIn(g.PhaseOut(watched.ID()))
			case "annihilate":
				g.MutablePermanent(watched.ID()).AddCounter(core.M1M1, 1)
				g.CheckStateBasedActions()
				g.MutablePermanent(watched.ID()).AddCounter(core.P1P1, 1)
			}
			g.ApplyContinuousEffects()
			want := 2
			if change == "recipient-phase" {
				want = 3
			}
			if g.FindPermanent(target.ID()).CurrentPower(g) != want {
				t.Fatal("wrong tracked subject lifetime")
			}
		})
	}
}

func TestCounterDurationWaiting(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	p.AddCounter(core.Mire, 1)
	body := UntilPermanentHasCounter(This, core.Mire, ModifyEachPT(MatchingPermanents(IsCreature), Fixed(1), Fixed(0)))
	if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		if len(g.WaitingDelayedPermanentSources()) != 1 || p.Counters[core.Mire] != 1 {
			t.Fatal("query lost duration source or changed counters")
		}
	}
}

func TestCounterDurationPlacementReplacement(t *testing.T) {
	for _, prevent := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		p := g.PutOnBattlefield(NewCreature("Subject", "", 2, 3), a.PlayerID())
		g.AddReplacementEffect(&testPrioritizedReplacement{
			replacementBase: replacementBase{duration: core.Indefinite},
			matchFunc:       func(action Action) bool { _, ok := action.(*AddCountersAction); return ok },
			replaceFunc: func(action Action, _ *Game) Action {
				if prevent {
					return nil
				}
				return action.(*AddCountersAction).WithAmount(2)
			},
		})
		body := Sequence(AddCountersTo(This, core.Mire, Fixed(1)), UntilPermanentHasCounter(This, core.Mire, ModifyPT(This, Fixed(1), Fixed(0))))
		if err := ApplyEffect(g, body, p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		want := 3
		if prevent {
			want = 2
		}
		if g.FindPermanent(p.ID()).CurrentPower(g) != want {
			t.Fatal("duration ignored actual placement")
		}
	}
}

func TestCounterDurationPaymentPreparation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Subject", "", 2, 3), a.PlayerID())
	p.AddCounter(core.Mire, 1)
	if err := ApplyEffect(g, UntilPermanentHasCounter(This, core.Mire, ModifyPT(This, Fixed(1), Fixed(0))), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	spec := costPaymentSpec{SourceID: p.ID(), Controller: a.PlayerID(), Costs: []Cost{RemoveCountersCost(core.Mire, 1), SacrificeMatchingCost(IsLand, "land")}}
	if _, err := g.prepareCostPaymentTransaction(spec, nil); err == nil {
		t.Fatal("unpayable cost accepted")
	}
	g.ApplyContinuousEffects()
	if g.FindPermanent(p.ID()).CurrentPower(g) != 3 || g.FindPermanent(p.ID()).Counters[core.Mire] != 1 {
		t.Fatal("failed payment ended live duration")
	}
	spec.Costs = spec.Costs[:1]
	tx, err := g.prepareCostPaymentTransaction(spec, nil)
	if err != nil {
		t.Fatal(err)
	}
	g.ApplyContinuousEffects()
	if g.FindPermanent(p.ID()).CurrentPower(g) != 3 {
		t.Fatal("payment preparation ended duration")
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	g.MutablePermanent(p.ID()).AddCounter(core.Mire, 1)
	g.ApplyContinuousEffects()
	if g.FindPermanent(p.ID()).CurrentPower(g) != 2 {
		t.Fatal("paid last counter did not end duration")
	}
}

func TestCounterDurationInstructionTiming(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Subject", "", 2, 3), a.PlayerID())
	p.AddCounter(core.Mire, 1)
	if err := ApplyEffect(g, UntilPermanentHasCounter(This, core.Mire, ModifyPT(This, Fixed(1), Fixed(0))), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if err := ApplyEffect(g, Sequence(RemoveCountersFrom(This, core.Mire, Fixed(1)), GainLifeFor(You, PowerOfWithLKI(This))), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 22 {
		t.Fatalf("next instruction read expired bonus: life=%d", a.Life())
	}
}

func TestCounterDurationPhaseTiming(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watched := g.PutOnBattlefield(NewArtifact("Watched", ""), a.PlayerID())
	target := g.PutOnBattlefield(NewCreature("Target", "", 2, 3), a.PlayerID())
	watched.AddCounter(core.Mire, 1)
	watched.AttachedTo = target.ID()
	if err := ApplyEffect(g, UntilPermanentHasCounter(This, core.Mire, ModifyPT(AttachedPermanent, Fixed(1), Fixed(0))), watched.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.PhaseOut(watched.ID())
	if g.FindPermanent(target.ID()).CurrentPower(g) != 2 {
		t.Fatal("phase-out did not end the tracked duration immediately")
	}
}
